<!-- SPDX-License-Identifier: EPL-2.0 -->

# From VoIP: reconstructing audio files from RTP payloads

Design notes for a planned capability: turning captured RTP payloads into audio
files, with correct timing, one output channel per call direction, and support
for several codecs.

**Status: design only.** Nothing described here is implemented. This document
exists to record the analysis and the decisions that need making before code is
written.

---

## Contents

- [Scope](#scope)
- [Why this is not just another decoder](#why-this-is-not-just-another-decoder)
- [What the library has and lacks today](#what-the-library-has-and-lacks-today)
- [The timing model](#the-timing-model)
- [Codec notes](#codec-notes)
- [What to implement](#what-to-implement)
- [Suggested phasing](#suggested-phasing)
- [Testing strategy](#testing-strategy)
- [Decisions still needed](#decisions-still-needed)
- [References](#references)

---

## Scope

### In scope

Given a set of RTP **payloads** that have already been extracted from the wire,
together with their RTP headers or equivalent metadata, produce:

- A stereo WAV file where each channel carries one direction of the call
- A mono WAV file of a single direction, or of both directions mixed
- Raw companded files (`.ulaw`, `.alaw`) for direct use by Asterisk and
  FreeSWITCH
- Output that is **time-accurate**: silence gaps preserved at their true
  durations, and the two directions aligned with each other

Codecs to support: **G.711** (µ-law and A-law), **Opus**, and **G.729**.

### Out of scope

- Parsing RTP off a socket, or any live/real-time path. See
  [Why this is not just another decoder](#why-this-is-not-just-another-decoder)
  for why that distinction matters structurally.
- Jitter buffering as a runtime concern. Reordering is handled as a
  preprocessing step over a known, complete set of packets.
- RTCP parsing, beyond optionally consuming Sender Reports for cross-stream
  alignment.
- Encryption (SRTP). Payloads are assumed already decrypted.

### Why batch-only matters

This restriction is what makes the whole feature tractable, and it is worth
being explicit about. In a batch setting:

- `audio.Source`'s pull-based, reliable, terminating contract is a good fit,
  because data comes from memory or disk rather than a socket.
- The resampler's internal block size is irrelevant. It buffers roughly one
  second of 8 kHz audio before emitting its first sample — measured at 8194
  frames, or 52 RTP packets of 20 ms — which would be fatal for a live call and
  is invisible when writing a file.
- Packet loss and reordering are known quantities up front, not something to be
  guessed at against a deadline.

A live transcoding path would need a different interface than `audio.Source`,
and is deliberately not attempted here.

---

## Why this is not just another decoder

Every existing decoder in `formats/` reads a **container**: a self-describing
byte stream with a header announcing sample rate, channel count and bit depth,
followed by contiguous audio. `audio.Decoder` reflects that:

```go
type Decoder interface {
    Decode(r io.Reader) (Source, error)
}
```

RTP payloads differ in four ways that this interface cannot express:

1. **No container.** There is no header. Sample rate and channel count come from
   the SDP negotiation or from the RTP payload type, not from the data.
2. **Framed, not streamed.** For G.729 and Opus, a payload is a discrete codec
   frame (or several). Frame boundaries carry meaning and cannot be recovered
   from a flat `io.Reader`. G.711 is the exception — it is a plain byte stream.
3. **Discontinuous.** Packets may be missing, and the gaps are real time that
   must appear in the output. An `io.Reader` has no way to say "40 ms is absent
   here".
4. **Timestamped.** Position in the output is determined by RTP timestamps, not
   by accumulated byte count.

Consequently this feature needs a **frame-oriented decode interface** alongside
the existing stream-oriented one, plus a component that owns the timeline. It is
not a matter of adding another `formats/` package in the current shape.

---

## What the library has and lacks today

Verified against the current tree.

### Usable as-is

| Component | Notes |
|---|---|
| `audio.Source` | Fits batch reconstruction well |
| `audio.Resampler` | Needed only when legs have differing sample rates |
| `audio.MonoMixer` | Averages rather than sums, so a two-direction mixdown will not clip |
| `utils.Float32ToInt16` | Correct conversion including full-scale clamping |

### Missing or blocking

| Gap | Detail |
|---|---|
| **No G.711 codec** | No µ-law or A-law companding anywhere in the tree |
| **No Opus or G.729** | Not present in any form |
| **WAV writer is mono-only** | `WriteWAV16` hardcodes `numChannels := uint16(1)`; it is a literal, not a parameter. The stereo output this feature exists to produce is impossible today. |
| **No interleaver** | Every channel operation goes one way — `SplitToMonoSources`, `SplitChannels`, `ExtractChannels`, `MonoMixer` all split or downmix. Nothing combines N mono streams into one interleaved multi-channel stream, which is exactly what two-direction recording needs. |
| **No timeline concept** | `audio.Source` has no notion of position or timestamp. Gap reconstruction has no home in the current model. |
| **No raw output writers** | The only writer is `wav.WriteWAV16`. Raw `.ulaw`/`.alaw` output needs new code. |

### A trap to avoid

`utils.BitDepth8` and `utils.SampleScale8 = 128.0` exist and look like 8-bit
support. They describe **linear** 8-bit PCM. G.711 is **logarithmically
companded** — dividing a µ-law byte by 128 yields garbage, not quiet audio.
Do not reuse those constants for G.711.

---

## The timing model

This is the substance of the feature. The codecs are comparatively mechanical;
the timing is where correctness is won or lost.

### Within one direction

RTP timestamps advance at the codec's **clock rate**. Converting a timestamp
delta into a sample count requires knowing that clock rate — and it is not
always the audio sample rate (see [Opus](#opus-dynamic-payload-type) below).

For a stream where clock rate equals sample rate, gap length is exact:

```
gap_samples = (ts[n] - ts[n-1]) - samples_in_packet[n-1]
```

A positive result is a gap to be filled with silence. Zero means contiguous
audio. Negative means overlap, which indicates a duplicate or a reordered
packet.

### Across the two directions — the alignment trap

**Each direction is an independent RTP stream with its own randomly chosen
starting timestamp and its own SSRC.** RTP timestamps carry no defined
relationship between streams. Comparing the caller's timestamp to the callee's
is meaningless; do it and the two channels end up offset by an arbitrary amount.

Alignment requires one of:

- **Wall-clock arrival timestamps.** A pcap capture provides these per packet.
  Simplest and usually sufficient. Accuracy is limited by capture-point jitter,
  which is normally well under one packet interval.
- **RTCP Sender Reports.** These map an RTP timestamp to an NTP wall-clock time
  for each stream, allowing exact alignment. Requires RTCP to have been captured
  and adds real parsing work.

**If the capture source preserves neither, stereo alignment is not
recoverable.** This constrains the whole design and should be settled before any
code is written — see [Decisions still needed](#decisions-still-needed).

### Gaps: loss versus intentional silence

A gap does not imply packet loss. Common causes:

- **Silence suppression / VAD.** Many endpoints simply stop transmitting during
  silence. This is normal and produces large, legitimate gaps.
- **Comfort noise** (RFC 3389, payload type 13). A single packet describes a
  noise level to synthesise for an interval.
- **DTX.** Opus and G.729 Annex B both have discontinuous transmission modes.
- **Actual loss.**

For file reconstruction all of these become "fill this interval", but the fill
differs: silence, synthesised comfort noise, or codec-specific packet loss
concealment. Distinguishing them is optional for a first version; treating
everything as silence is acceptable and honest.

**The failure mode to avoid:** concatenating payloads and ignoring timestamps.
Each direction then shrinks by the total duration of its own silence, the two
directions drift apart progressively, and the recording sounds like the parties
talking over each other. This is the classic call-recorder bug and it is
entirely a timing error, not a codec error.

### Packet loss and codec state

Loss behaves very differently by codec, and this shapes the concealment design:

- **G.711 is stateless.** Each byte decodes independently. Insert silence for a
  lost packet and everything after it is unaffected.
- **G.729 and Opus are stateful.** Both are predictive: the decoder carries
  state across frames. A lost frame both leaves a hole and degrades subsequent
  frames until the state reconverges. A proper decoder exposes packet loss
  concealment for exactly this reason, and it must be *invoked* for lost frames
  rather than skipped — otherwise state diverges from the encoder's.

This is why the frame decode interface below has a distinct `Conceal` method.

### Other details that will bite

- **Timestamp wraparound.** The RTP timestamp is 32 bits. At 8 kHz it wraps
  every ~6 hours; at Opus's 48 kHz clock, every ~25 hours. Long or trunk-side
  recordings must handle it. Track an extended 64-bit timestamp.
- **Sequence number wraparound.** 16 bits — wraps every 65536 packets, roughly
  22 minutes at 20 ms per packet. Much more likely to be hit than timestamp
  wrap.
- **Duplicates and reordering.** Deduplicate by sequence number and sort before
  building the timeline.
- **Clock drift.** Endpoint sample clocks are not exactly nominal. Over a long
  call the two directions drift relative to each other. Usually ignored for
  recording; worth knowing before someone reports it as a bug.
- **Payload type changes mid-stream.** Legal, and it happens — for example a
  switch to comfort noise or to `telephone-event`.
- **DTMF as events.** RFC 4733 `telephone-event` packets (commonly payload type
  101, negotiated) carry DTMF as events, not audio. They occupy timestamp space
  but contain no waveform. Decide whether to ignore them, leaving a gap where
  the tone was, or to synthesise tones.

---

## Codec notes

### G.711 — payload types 0 (PCMU, µ-law) and 8 (PCMA, A-law)

| Property | Value |
|---|---|
| Sample rate | 8 kHz, fixed |
| Channels | 1 |
| RTP clock rate | 8000 — equals sample rate |
| Payload | 1 byte per sample, no framing |
| Typical packet | 160 bytes = 20 ms |
| State | Stateless |

The simplest case in every respect. Timestamp delta equals sample count
directly. Decoding is a 256-entry lookup table per variant; encoding needs the
segment-and-mantissa logic, which is still modest.

µ-law is standard in North America and Japan, A-law in Europe and most of the
rest of the world. Both must be supported.

Idle/silence bytes: **`0xFF` for µ-law, `0xD5` for A-law.** Useful for gap
filling in the raw-output path.

**The raw output path needs no codec at all.** Writing `.ulaw` or `.alaw` at
8 kHz is byte passthrough plus silence fill. No decode, no resample, no
conversion. Worth implementing first as it validates the entire timeline layer
independently of any DSP.

In a WAV container G.711 uses format tags 6 (A-law) and 7 (µ-law). The current
WAV decoder accepts only tag 1, so such files are rejected today — relevant if
input may arrive as WAV rather than raw payloads.

### Opus — dynamic payload type

| Property | Value |
|---|---|
| Sample rate | Decoder output is caller-chosen: 8, 12, 16, 24 or 48 kHz |
| Channels | 1 or 2 |
| RTP clock rate | **48000 always**, regardless of audio sample rate (RFC 7587) |
| Payload | One or more frames; 2.5/5/10/20/40/60 ms |
| State | Stateful, with built-in PLC |

**The 48 kHz clock rate is the detail that shapes the design.** For G.711 and
G.729 the RTP timestamp advances one unit per audio sample. For Opus it advances
at 48 kHz no matter what rate you decode to. A timeline built in "samples" will
be wrong for Opus by a factor of up to six.

This is the reason the timeline layer must work in a **rate-agnostic time
base** and convert to samples only at the end, rather than accumulating sample
counts as it goes.

Note also that a mixed-codec call — say a G.711 leg and an Opus leg — produces
two directions at different sample rates. One must be resampled to match before
interleaving, at which point `audio.Resampler` enters the picture, along with
the anti-aliasing limitation documented in the README. Downsampling a 48 kHz
Opus leg to 8 kHz is a 6:1 decimation, which is precisely the case where the
present one-pole filter is weakest.

Implementation options: a pure Go decoder exists (`github.com/pion/opus`) but
its completeness should be assessed before committing; otherwise cgo bindings to
libopus, which conflicts with this module's native-Go-first goal. Writing an
Opus decoder from scratch is a very large undertaking and is not recommended.

### G.729 — payload type 18

| Property | Value |
|---|---|
| Sample rate | 8 kHz, fixed |
| Channels | 1 |
| RTP clock rate | 8000 — equals sample rate |
| Payload | 10 bytes per 10 ms frame (8 kbit/s); packets often carry 2 frames |
| State | Stateful (CELP with adaptive codebook) |

Timestamp advances 80 per 10 ms frame. A 20 ms packet is 20 bytes and advances
the timestamp by 160.

G.729 Annex B adds VAD/DTX/CNG, in which SID (silence descriptor) frames are
2 bytes and some intervals are not transmitted at all. A decoder that ignores
Annex B will mis-parse payloads from an endpoint that uses it, since frame size
is no longer a constant 10 bytes. Frame type must be inferred from payload
length.

**Scale of work.** G.729 is ACELP: analysis-by-synthesis with an adaptive
codebook, a fixed algebraic codebook, LSP quantisation and a post-filter. The
**decoder is substantially simpler than the encoder** and is all that file
reconstruction requires. Building the decoder first, and only adding an encoder
if there is a concrete need, is strongly advised. This is still a serious DSP
effort measured in weeks rather than days, and it is by far the largest item in
this document.

**On licensing.** The core G.729 patents have expired, which is what makes an
independent implementation practical. Two things remain worth separating:

- **Patents** versus **the ITU-T reference source code**, which carries its own
  licence independent of patent status. A clean-room implementation written from
  the published Recommendation text avoids the reference-code licence question
  entirely; deriving from the reference C does not.
- **Annex B** (VAD/DTX/CNG) was covered by additional patents whose status
  should be confirmed separately from the core codec.

Worth confirming both against your own jurisdiction and risk posture before
starting. Noted here so the distinction is not discovered late.

### Comparison

| | G.711 | G.729 | Opus |
|---|---|---|---|
| Payload type | 0, 8 (static) | 18 (static) | dynamic |
| Sample rate | 8 kHz | 8 kHz | 8–48 kHz, chosen |
| RTP clock | 8 kHz | 8 kHz | **48 kHz always** |
| Frame size | none (byte stream) | 10 ms / 10 bytes | 2.5–60 ms, variable |
| Stateful | no | yes | yes |
| Needs PLC | no | yes | yes |
| Implementation effort | small | **large** | large, or use a library |

---

## What to implement

### 1. A frame-oriented decode interface

`audio.Decoder` stays as it is, for containers. Add a sibling for codecs that
arrive as discrete frames:

```go
// PacketDecoder decodes discrete codec frames such as those carried in RTP
// payloads. Unlike Decoder, it has no container to parse and no byte stream to
// follow: each call decodes one payload, and lost payloads are signalled
// explicitly so that stateful codecs can conceal them.
type PacketDecoder interface {
    // Decode decodes one payload into dst, returning the number of float32
    // values written. A payload may contain several codec frames.
    Decode(payload []byte, dst []float32) (int, error)

    // Conceal produces samples for one lost frame of duration d. Stateless
    // codecs may emit silence; predictive codecs should run their own packet
    // loss concealment so decoder state stays aligned with the encoder's.
    Conceal(d time.Duration, dst []float32) (int, error)

    // SampleRate of the decoded output in Hz.
    SampleRate() int

    // Channels in the decoded output.
    Channels() int

    // ClockRate is the RTP timestamp clock in Hz. This is NOT always
    // SampleRate: Opus always uses 48000 regardless of output rate.
    ClockRate() int

    // MaxFrameSamples bounds the dst buffer a caller must supply for one
    // payload, so callers can size buffers once.
    MaxFrameSamples() int

    Close() error
}
```

`Conceal` taking a duration rather than a sample count keeps callers out of the
clock-rate conversion business.

### 2. Codec packages

```
formats/g711/     — µ-law and A-law; PacketDecoder plus an encoder
formats/g729/     — decoder first; encoder only if needed
formats/opus/     — wrapper over a chosen implementation
```

Each should also expose a plain `audio.Source` over a contiguous byte stream
where that makes sense, so a raw `.ulaw` file on disk can be read without going
through the RTP machinery.

### 3. A packet-type registry

The existing `audio.Registry` is keyed by a format string (`"wav"`, `"mp3"`).
RTP dispatch is keyed by **payload type**, an integer, with dynamic types
resolved from SDP rather than fixed. These are different concerns and should not
share one map. A small separate registry mapping payload type to
`PacketDecoder`, with the static assignments from RFC 3551 preregistered and
dynamic types registered per session, is cleaner than overloading the existing
one.

### 4. The timeline assembler

The core new component. Responsibilities:

- Accept packets as `(sequence, timestamp, arrivalTime, payload)`
- Deduplicate by sequence number; sort into order
- Track extended 64-bit sequence and timestamp to survive wraparound
- Compute gaps in a **rate-agnostic time base**, converting to samples only when
  emitting
- Fill gaps via `PacketDecoder.Conceal`, or with silence
- Expose the result as an `audio.Source`, so everything downstream is reused
  unchanged

Suggested shape:

```go
// Packet is one RTP payload with the metadata needed to place it in time.
type Packet struct {
    Sequence  uint16
    Timestamp uint32        // RTP timestamp, in the codec's clock rate
    Arrival   time.Time     // wall clock; zero if unavailable
    Payload   []byte
}

// Timeline reconstructs a continuous audio stream from timestamped packets,
// filling absences so that output duration matches real elapsed time.
type Timeline struct { /* ... */ }

func NewTimeline(dec PacketDecoder, opts ...TimelineOption) *Timeline

func (t *Timeline) Add(p Packet) error   // may be called out of order
func (t *Timeline) Source() audio.Source // after all packets are added
func (t *Timeline) Stats() TimelineStats // gaps, duplicates, concealed frames
```

`Stats` matters more than it looks: a recording that silently drops 30% of a
call is worse than one that reports it. Surface gap count, total concealed
duration, duplicates and reordering.

**Alignment across directions** is a separate concern from a single timeline and
should be its own step, since it needs a common reference:

```go
// AlignedPair holds two directions of a call on a shared time base.
func AlignPair(a, b *Timeline, ref AlignmentRef) (audio.Source, error)
```

where `AlignmentRef` selects wall-clock arrival, RTCP Sender Reports, or an
explicit caller-supplied offset.

### 5. An interleaver

The missing inverse of `SplitToMonoSources`:

```go
// NewInterleaver combines mono sources into one interleaved multi-channel
// source, in the order given. All sources must share a sample rate; shorter
// sources are padded with silence so every channel spans the full duration.
func NewInterleaver(sources ...Source) (Source, error)
```

Belongs in `audio/`, next to the splitter it complements. Useful well beyond
this feature.

### 6. Multi-channel and raw writers

`WriteWAV16` hardcodes mono. Keep it — it is a documented API — and add:

```go
// WriteWAVN writes interleaved 16-bit PCM with the given channel count.
func WriteWAVN(w io.Writer, sampleRate, channels int, samples []int16) error
```

`WriteWAV16` then becomes a one-line call with `channels == 1`.

Raw writers for the passthrough path:

```go
// formats/g711
func WriteULaw(w io.Writer, payload []byte) error
func WriteALaw(w io.Writer, payload []byte) error
```

These are thin, but having them named makes the passthrough path explicit and
gives the silence-fill constants a home.

### 7. A high-level entry point

Mirroring `ResampleToMono16` in spirit:

```go
// RecordingToWAV writes a stereo WAV in which each channel carries one
// direction of a call, reconstructed with correct timing.
func RecordingToWAV(w io.Writer, inbound, outbound *voip.Timeline, opts ...RecordingOption) error
```

with options for target sample rate, mono mixdown, and how to handle a missing
alignment reference.

---

## Suggested phasing

Ordered so that each phase is independently useful and testable, and so the
timing layer is proven before any hard DSP is attempted.

**Phase 1 — timeline, proven without any codec**

Raw `.ulaw`/`.alaw` output. Byte passthrough plus silence fill, no decoding.
This exercises deduplication, ordering, wraparound, gap computation and stats
with zero DSP in the way. If the timeline is wrong, it is obvious here.

**Phase 2 — G.711 and stereo output**

`formats/g711` decode, `WriteWAVN`, `NewInterleaver`, cross-direction alignment.
At the end of this phase the headline use case works end to end for the most
common codec in telephony.

**Phase 3 — Opus**

Forces the clock-rate abstraction to be correct, since Opus is the codec where
clock rate and sample rate diverge. Also the first case where two legs may have
different sample rates, bringing the resampler in. Assess `pion/opus` before
deciding build-versus-adopt.

**Phase 4 — G.729 decoder**

Largest single item. Deliberately last: by this point the timeline, alignment,
interleaving, writers and PLC plumbing are all proven, so the work is confined
to the codec itself.

**Phase 5 — encoders, if needed**

G.711 encode for producing `.ulaw`/`.alaw` from arbitrary sources. G.729 encode
only if there is a concrete requirement.

---

## Testing strategy

Two lessons from the resampler performance work in this repository apply
directly, and both were learned the hard way.

**Test against real data, not mocks.** The resampler's dominant cost was
invisible to benchmarks built on an in-memory source, and an entire AIFF decode
path passed its whole test suite with the byte order inverted, because the tests
only exercised a mock. For this feature that means capturing real pcap-derived
payload fixtures, not just synthesising them.

**Assert on the property that actually matters.** For timing, that property is
duration and alignment, not sample values. Specific suggestions:

- **Duration accuracy.** Construct a packet set with known gaps; assert output
  duration equals the intended wall-clock span to within one packet interval.
  This is the assertion that catches the concatenate-and-drift bug.
- **Alignment.** Put an impulse at a known time in each direction; assert their
  positions in the output differ by the expected offset. A drift bug shows up
  immediately.
- **Wraparound.** Synthesise packet sets that cross both the 16-bit sequence
  and 32-bit timestamp boundaries. These are unreachable in short fixtures and
  certain to appear in production.
- **Loss and reordering.** Shuffle and drop packets; assert output duration is
  unchanged and stats report accurately.
- **Codec vectors.** G.711 has well-known companding tables; test the full
  256-value round trip exhaustively rather than sampling. For G.729, the ITU-T
  test vectors are the reference — noting that using the vectors is a separate
  question from deriving code from the reference implementation.
- **Silence-fill values.** Assert the exact idle bytes (`0xFF`, `0xD5`) in raw
  output, since a wrong constant produces quiet-but-not-silent output that is
  easy to miss by ear.

Also mirror the existing guards: a `NeverOverruns` test per decoder, and
allocation assertions on any hot path, following the pattern already established
in `formats/wav` and `formats/aiff`.

---

## Decisions still needed

Listed roughly in the order they block work.

1. **What does the input look like?** Does the caller hand over parsed
   `(seq, ts, arrival, payload)` tuples, or should this module read pcap
   directly? Reading pcap pulls in a dependency and a lot of scope; taking
   tuples keeps the module focused and pushes capture concerns to the caller.
   **Recommendation: take tuples.**

2. **Is an alignment reference available?** Wall-clock arrival times, RTCP
   Sender Reports, or neither. If neither, stereo alignment is not recoverable
   and the feature reduces to per-direction files. This determines whether the
   headline use case is achievable at all and should be answered first.

3. **How are gaps filled by default?** Silence is simplest and predictable.
   Codec PLC is better for short losses but makes output dependent on decoder
   internals. **Recommendation: silence by default, PLC opt-in.**

4. **What happens on codec mismatch between directions?** Resample to the
   higher rate, the lower rate, or refuse? Note the anti-aliasing caveat when
   downsampling.

5. **Opus: build or adopt?** Assess `pion/opus` maturity against the
   native-Go-first goal. A cgo binding to libopus would be the pragmatic choice
   but conflicts with that goal.

6. **G.729 scope.** Decoder only, or encoder too? Annex B support or core only?
   Annex B is not optional in practice if inputs come from endpoints that use
   it, because frame sizes stop being constant.

7. **Where does this live?** A `voip/` package at the top level, or under
   `formats/`? The timeline and alignment logic is not a format, so `voip/`
   with `formats/g711` etc. beneath or beside it seems more honest.

---

## References

- **RFC 3550** — RTP: A Transport Protocol for Real-Time Applications
- **RFC 3551** — RTP Profile for Audio and Video Conferences with Minimal
  Control (static payload type assignments, including 0/PCMU, 8/PCMA, 18/G729)
- **RFC 7587** — RTP Payload Format for the Opus Speech and Audio Codec
  (specifies the invariant 48 kHz clock rate)
- **RFC 3389** — RTP Payload for Comfort Noise (payload type 13)
- **RFC 4733** — RTP Payload for DTMF Digits, Telephony Tones and Signals
- **ITU-T G.711** — Pulse code modulation (PCM) of voice frequencies
- **ITU-T G.729** — Coding of speech at 8 kbit/s using CS-ACELP
- **ITU-T G.729 Annex B** — Silence compression scheme

Related documents in this repository:

- [README.markdown](README.markdown) — including the anti-aliasing limitation
  relevant to any mixed-sample-rate call
- [examples/profile_resampler/PROFILING_GUIDE.md](examples/profile_resampler/PROFILING_GUIDE.md)
  — the resampler's block-read design, its one-second startup buffering, and the
  testing lessons referenced above
