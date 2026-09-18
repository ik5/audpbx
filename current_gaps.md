<!-- SPDX-License-Identifier: EPL-2.0 -->

# Current gaps

An inventory of what is missing, incomplete or likely to need changing in
`audpbx`, compiled ahead of a v1 roadmap.

**Why this document exists.** The project is pre-v1 (latest tag `v0.0.3`). v1
commits to a stable API for the whole v1 series, so the urgent question is not
"what features are missing" — most missing features are purely additive and can
land in v1.1, v1.2 and so on — but **"what API shapes would require a breaking
change to fix later?"** Those must be settled before v1 is cut. The inventory
below is therefore classified by API impact rather than by perceived
importance.

**Status: analysis only.** Nothing here is an agreed change proposal, and
nothing has been implemented. The roadmap is a separate exercise.

---

## Scope: what this project is, and is not

`audpbx` is an **audio manipulation library**. It decodes, transforms and
encodes audio, and provides the abstractions that let audio flow through a
pipeline.

**It does not implement codecs.** Codec implementations are external
dependencies, imported and wrapped — as WAV, MP3, Ogg Vorbis and AIFF already
are today, via `go-audio`, `hajimehoshi/go-mp3` and `jfreymuth/oggvorbis`.

This distinction determines how every gap in this document should be read:

| In scope | Out of scope |
|---|---|
| Resampling, filtering, mixing, gain, dynamics, analysis | Codec compression and decompression algorithms |
| Container and header parsing, metadata, tags | The signal processing inside a codec |
| Tone generation and detection | |
| Composition — concatenation, trimming, interleaving | |
| The abstractions codecs plug into (`Decoder`, `Encoder`, `PacketDecoder`) | |
| Thin wrappers adapting third-party codecs to those abstractions | |

So "add G.729 support" means *integrate a G.729 implementation*, never *write a
CS-ACELP codec*. Where no suitable Go implementation exists, the options are a
cgo binding, a separate standalone project that `audpbx` imports, or leaving
the codec unsupported — and that is a dependency decision, not an
implementation task for this module.

Two consequences worth stating plainly:

- **Codec effort estimates do not belong in this project's planning.** What
  matters per codec is whether a usable implementation exists, under what
  licence, and how much wrapper code it needs.
- **Codec quality and performance are largely not this project's to fix.** The
  MP3 and Ogg throughput gap documented in
  [README.markdown](README.markdown#performance) sits inside the upstream
  decoders; the remedy is a different dependency, not local optimisation.

Tone generation, filtering and resampling are a deliberate exception to the
"do not implement DSP" reading: they are audio manipulation, which is precisely
what this library is for.

---

## Contents

- [Scope: what this project is, and is not](#scope-what-this-project-is-and-is-not)
- [How to read this](#how-to-read-this)
- [Summary index](#summary-index)
- [A. API stability](#a-api-stability-v1-critical)
- [B. Format and codec coverage](#b-format-and-codec-coverage)
- [C. Resampling, DSP and composition](#c-resampling-dsp-and-composition)
- [D. Observability: progress, timing, cancellation](#d-observability-progress-timing-cancellation)
- [E. VoIP and packet-based audio](#e-voip-and-packet-based-audio)
- [F. Testing, CI and hygiene](#f-testing-ci-and-hygiene)
- [G. Documentation](#g-documentation)
- [H. Telephony signalling and tones](#h-telephony-signalling-and-tones)
- [I. Filters, dynamics and enhancement](#i-filters-dynamics-and-enhancement)
- [J. PBX application support](#j-pbx-application-support)
- [K. Robustness and resource limits](#k-robustness-and-resource-limits)
- [L. Dependency health](#l-dependency-health)
- [Decisions required before v1](#decisions-required-before-v1)
- [Explicitly not gaps](#explicitly-not-gaps)
- [How these findings were verified](#how-these-findings-were-verified)

---

## How to read this

Every gap carries an **API impact** classification. This is the field that
determines whether it must be resolved before v1 or can wait:

| Class | Meaning | Deadline |
|---|---|---|
| **BREAKING** | Fixing this after v1 requires a v2. Includes adding a method to an exported interface, changing a signature, or removing an exported identifier. | **Before v1** |
| **ADDITIVE** | Can be introduced during v1.x without breaking existing callers: new packages, new functions, new variadic options. | Any time |
| **BEHAVIOURAL** | No signature change, but output or performance characteristics change. Needs a documented policy on what callers may rely on. | Policy before v1 |
| **INTERNAL** | No exported surface affected. | Any time |

Gap IDs are stable so they can be referenced from the roadmap and from commit
messages. The numbering is not a priority order.

Read every entry against
[the scope note](#scope-what-this-project-is-and-is-not): a "missing codec" is
a missing *integration*, and the work is a wrapper plus a dependency decision,
not a codec implementation.

Two markers appear throughout:

- **[user N]** — one of the six gaps raised when this review was commissioned,
  using that original numbering.
- **[requested]** — raised subsequently as a specific feature request, with no
  number attached.

---

## Summary index

### Must be settled before v1

| ID | Gap | Impact |
|---|---|---|
| [API-1](#api-1-source-cannot-report-length-or-duration) | `Source` cannot report length or duration | BREAKING |
| [API-2](#api-2-no-cancellation-mechanism) | No cancellation mechanism anywhere | BREAKING |
| [API-3](#api-3-readsamples-contract-is-under-specified) | `ReadSamples` EOF contract under-specified | BREAKING |
| [API-4](#api-4-channeloption-is-a-closed-extension-point) | `ChannelOption` is a closed extension point | BREAKING |
| [API-5](#api-5-no-encoder-abstraction) | No `Encoder` abstraction at all | BREAKING |
| [API-6](#api-6-writewav16-is-permanently-mono) | `WriteWAV16` is permanently mono | ADDITIVE (wart is permanent) |
| [API-7](#api-7-source-exposes-no-format-metadata) | `Source` exposes no format metadata | BREAKING |
| [API-8](#api-8-bufsize-is-vestigial) | `Source.BufSize()` is vestigial | BREAKING to remove |
| [API-9](#api-9-registry-is-string-keyed-and-its-zero-value-panics) | `Registry` string-keyed; zero value panics | BREAKING / INTERNAL |
| [API-10](#api-10-channelsource-value-and-pointer-asymmetry) | `ChannelSource` value/pointer asymmetry | BREAKING |
| [API-11](#api-11-two-different-processchannels-functions) | Two different `ProcessChannels` functions | BREAKING |
| [API-12](#api-12-dead-exported-error-values) | Dead exported error values | BREAKING to remove |
| [API-13](#api-13-inconsistent-error-construction) | Inconsistent error construction | BREAKING |
| [API-14](#api-14-resampletomono16-signature-and-memory-model) | `ResampleToMono16` signature and memory model | BREAKING |
| [API-15](#api-15-no-documented-concurrency-guarantees) | No documented concurrency guarantees | BREAKING (contract) |
| [API-16](#api-16-utils-is-a-grab-bag-package) | `utils` is a grab-bag package | BREAKING to rename |
| [API-17](#api-17-decoder-cannot-express-framed-codecs) | `Decoder` cannot express framed codecs | ADDITIVE if designed now |
| [API-18](#api-18-source-cannot-seek) | `Source` cannot seek | BREAKING |
| [API-19](#api-19-tag-manipulation-has-no-home-in-the-api) | Tag manipulation has no home in the API | BREAKING if on `Source` |
| [API-20](#api-20-source-cannot-report-position) | `Source` cannot report position | BREAKING |
| [API-21](#api-21-channel-is-uint32-capping-the-library-at-32-channels) | `Channel` is `uint32`, capping the library at 32 channels | BREAKING |
| [OBS-4](#obs-4-no-task-abstraction) | No task abstraction for progress, cancellation or lifecycle | BREAKING |
| [DSP-2](#dsp-2-no-resampler-algorithm-selection-user-4) | No resampler algorithm selection **[user 4]** | ADDITIVE if designed now |
| [DEP-1](#dep-1-five-of-seven-dependencies-are-archived) | **Five of seven dependencies are archived** | BREAKING in practice |

### Can land during v1.x

| ID | Gap | Impact |
|---|---|---|
| [RES-1](#res-1-a-62-byte-file-can-force-gigabytes-of-allocation) | **A 62-byte file can force gigabytes of allocation** | ADDITIVE |
| [RES-2](#res-2-untrusted-input-is-read-into-memory-without-bound) | Untrusted input read into memory without bound | ADDITIVE |
| [RES-3](#res-3-channel-splitting-buffers-unread-channels-without-bound) | Channel splitting buffers unread channels without bound | ADDITIVE |
| [FMT-1](#fmt-1-encoders-exist-for-one-format-in-one-configuration-user-1) | Encoders for all supported formats **[user 1]** | ADDITIVE |
| [FMT-2](#fmt-2-wav-decoding-is-narrow-user-5) | WAV: `WAVE_FORMAT_EXTENSIBLE`, other bit depths **[user 5]** | ADDITIVE |
| [FMT-3](#fmt-3-aiff-decoding-is-16-bit-only) | AIFF: 16-bit only, no AIFF-C | ADDITIVE |
| [FMT-4](#fmt-4-telco-and-voip-codecs-absent-user-2) | Telco/VoIP codecs **[user 2]** | ADDITIVE |
| [FMT-5](#fmt-5-aac-support-user-6) | AAC via build-tagged bindings **[user 6]** | ADDITIVE |
| [FMT-6](#fmt-6-no-headerless-raw-formats) | Headerless/raw formats (`.sln`, `.ulaw`, `.alaw`) | ADDITIVE |
| [FMT-7](#fmt-7-no-flac) | No FLAC | ADDITIVE |
| [FMT-8](#fmt-8-mono-mp3-is-decoded-as-stereo) | Mono MP3 decoded as stereo | BEHAVIOURAL |
| [FMT-9](#fmt-9-no-container-format-detection) | No content-based format detection | ADDITIVE |
| [FMT-10](#fmt-10-no-metadata-or-tag-handling) | **No metadata/tag handling** (ID3v1/v2, Vorbis comments, RIFF `INFO`, `bext`) | ADDITIVE |
| [FMT-11](#fmt-11-non-pcm-payloads-inside-the-wav-container) | Non-PCM payloads inside WAV (A-law, µ-law, GSM, ADPCM) | ADDITIVE |
| [FMT-12](#fmt-12-aiff-c-compression-types) | AIFF-C types, notably little-endian `sowt` | ADDITIVE |
| [FMT-13](#fmt-13-asterisk-and-freeswitch-native-file-formats) | **Asterisk/FreeSWITCH native file formats** | ADDITIVE |
| [FMT-14](#fmt-14-opus-in-ogg-is-not-the-same-as-vorbis-in-ogg) | Opus in Ogg (`.opus`) and WebM | ADDITIVE |
| [FMT-15](#fmt-15-no-large-file-or-extended-containers) | Large-file containers (RF64, W64, CAF) | ADDITIVE |
| [FMT-16](#fmt-16-in-file-markers-and-regions-are-discarded) | In-file markers and regions discarded | ADDITIVE |
| [FMT-17](#fmt-17-mp3-container-details) | MP3: Xing/VBRI duration, MPEG-2/2.5 rates | ADDITIVE |
| [FMT-18](#fmt-18-other-file-formats-worth-considering) | AMR, AAC containers, WebM, WMA | ADDITIVE |
| [FMT-19](#fmt-19-no-cross-format-metadata-mapping-or-preservation-policy) | No cross-format metadata mapping or preservation policy | ADDITIVE |
| [FMT-20](#fmt-20-tag-blocks-are-a-decode-correctness-and-robustness-hazard) | Tag blocks are a decode and robustness hazard | ADDITIVE |
| [FMT-21](#fmt-21-what-owning-the-wav-and-aiff-container-layer-requires) | **What owning the WAV/AIFF container layer requires** | ADDITIVE |
| [DSP-1](#dsp-1-anti-aliasing-filter-is-structurally-inadequate) | Anti-aliasing filter inadequate | BEHAVIOURAL |
| [DSP-3](#dsp-3-resampler-block-size-is-a-compile-time-constant) | Resampler block size fixed at compile time | ADDITIVE |
| [DSP-4](#dsp-4-the-library-can-decompose-audio-but-not-compose-it) | **No composition primitives at all** | ADDITIVE |
| [DSP-5](#dsp-5-no-dithering) | No dithering on float to int conversion | ADDITIVE |
| [DSP-6](#dsp-6-downmix-is-unweighted-averaging-only) | Downmix is unweighted averaging only | ADDITIVE |
| [DSP-7](#dsp-7-effects-are-not-composable) | Effects are not composable `Source` wrappers | ADDITIVE |
| [DSP-8](#dsp-8-normalization-buffers-the-entire-stream) | Normalization buffers the entire stream | ADDITIVE |
| [DSP-9](#dsp-9-no-time-stretching-or-pitch-modification) | No time-stretching or pitch modification | ADDITIVE |
| [DSP-10](#dsp-10-no-analysis-primitives) | No analysis primitives (level, silence, F0) | ADDITIVE |
| [OBS-1](#obs-1-no-progress-reporting-user-3) | **No progress reporting; `WithProgress` is a no-op** **[user 3]** | ADDITIVE on API-1/20 |
| [OBS-2](#obs-2-no-processing-statistics) | No processing statistics | ADDITIVE |
| [OBS-3](#obs-3-no-way-to-surface-warnings) | No way to surface non-fatal warnings | ADDITIVE |
| [OBS-5](#obs-5-processing-state-cannot-be-checkpointed-or-restored) | Processing state cannot be checkpointed or restored | ADDITIVE |
| [OBS-6](#obs-6-no-partial-output-or-resume-point-persistence) | No partial-output or resume-point persistence | ADDITIVE |
| [VOIP-1 to VOIP-4](#voip-1-to-voip-4) | Timeline, alignment, packet decode, raw writers | ADDITIVE |
| [TONE-1](#tone-1-no-dtmf-generation-requested) | **No DTMF generation** **[requested]** | ADDITIVE |
| [TONE-2](#tone-2-no-dtmf-detection) | No DTMF detection (Goertzel) | ADDITIVE |
| [TONE-3](#tone-3-no-call-progress-tones-and-no-tone-plan-model) | No call progress tones; no country tone-plan model | ADDITIVE |
| [TONE-4](#tone-4-no-mf--r2-signalling-tones) | No MF / R2 signalling tones | ADDITIVE |
| [TONE-5](#tone-5-no-fax-and-modem-tone-generation-or-detection) | No fax/modem tone generation or detection | ADDITIVE |
| [FILT-1](#filt-1-no-filter-framework) | **No filter framework (biquad/IIR/FIR)** | ADDITIVE |
| [FILT-2](#filt-2-no-telephony-band-pass) | No telephony band-pass (300–3400 Hz) | ADDITIVE |
| [FILT-3](#filt-3-no-dc-offset-removal) | No DC offset removal | ADDITIVE |
| [FILT-4](#filt-4-no-automatic-gain-control) | No automatic gain control | ADDITIVE |
| [FILT-5](#filt-5-no-compressor-or-limiter) | No compressor or limiter | ADDITIVE |
| [FILT-6](#filt-6-no-noise-gate-or-noise-reduction) | No noise gate or noise reduction | ADDITIVE |
| [FILT-7](#filt-7-no-echo-cancellation) | No echo cancellation (likely out of scope) | ADDITIVE |
| [FILT-8](#filt-8-no-voice-activity-detection) | No voice activity detection | ADDITIVE |
| [FILT-9](#filt-9-no-loudness-measurement-or-telephony-level-conventions) | No loudness (LUFS) or dBm0/dBov conventions | ADDITIVE |
| [PBX-1](#pbx-1-no-frame-alignment-or-duration-helpers) | No frame alignment or duration helpers | ADDITIVE |
| [PBX-2](#pbx-2-no-pbx-output-conventions-or-validation) | No PBX output conventions or validation | ADDITIVE |
| [PBX-3](#pbx-3-no-prompt-set-validation) | No prompt-set validation | ADDITIVE |
| [PBX-4](#pbx-4-no-automatic-segmentation) | No automatic segmentation | ADDITIVE |
| [PBX-5](#pbx-5-no-streaming-io-boundary) | No streaming encode to `io.Writer` | ADDITIVE |
| [PBX-6](#pbx-6-no-multi-file-batch-operations) | No multi-file batch operations | ADDITIVE |
| [PBX-7](#pbx-7-no-call-or-prompt-annotation) | No call or prompt annotation (`bext`, sidecars) | ADDITIVE |
| [QA-1](#qa-1-the-test-suite-fails-out-of-the-box) | `go test ./...` fails out of the box | INTERNAL |
| [QA-2](#qa-2-no-ci) | No CI | INTERNAL |
| [QA-3](#qa-3-no-lint-gate-and-ten-unformatted-files) | No lint gate; ten unformatted files | INTERNAL |
| [QA-4](#qa-4-root-package-coverage-is-63) | Root package coverage 63% | INTERNAL |
| [QA-5](#qa-5-no-fuzzing-of-binary-parsers) | No fuzzing of binary parsers | INTERNAL |
| [QA-6](#qa-6-example-modules-are-outside-the-root-test-run) | Example modules outside root test run | INTERNAL |
| [QA-7](#qa-7-no-codec-reference-vectors) | No codec reference vectors | INTERNAL |
| [QA-8](#qa-8-no-pre-release-bug-and-security-audit) | **No pre-release bug and security audit** | INTERNAL |
| [DEP-2](#dep-2-no-dependency-policy) | No dependency policy or licence inventory | INTERNAL |
| [DOC-1 to DOC-4](#doc-1-to-doc-4) | Stability policy, CHANGELOG, CONTRIBUTING, accuracy | INTERNAL |

---

## A. API stability (v1-critical)

### API-1: `Source` cannot report length or duration

**Impact: BREAKING.** This is the single most important item in this document.

`audio.Source` has no `Duration()`, `Length()` or equivalent:

```go
type Source interface {
    SampleRate() int
    Channels() int
    ReadSamples(dst []float32) (n int, err error)
    BufSize() int
    Close() error
}
```

Adding a method to an exported interface after v1 breaks every external
implementer, so it cannot be done in v1.x. And **every underlying decoder
already has the information**:

| Decoder | Available upstream |
|---|---|
| wav | `Decoder.Duration()`, `Decoder.PCMSize` |
| aiff | `Decoder.Duration()`, `Decoder.PCMSize` |
| mp3 | `Decoder.Length()` (decoded byte count) |
| vorbis | `Reader.Length()`, `Reader.Position()` |

So the capability is discarded at the wrapper boundary, and recovering it later
is a v2 change.

This gap blocks [OBS-1](#obs-1-no-progress-reporting-user-3) entirely: progress
cannot be reported as a fraction without knowing the total.

Options to weigh:

- Add to the interface now — simplest for callers, hardest on implementers, and
  awkward for genuinely unbounded sources (a live or synthetic stream).
- A separate optional interface probed with a type assertion, e.g.
  `interface{ Duration() (time.Duration, bool) }`. Keeps `Source` minimal and
  stays open to unbounded sources, at the cost of callers having to probe.
- Both: minimal `Source`, plus a documented optional extension.

The middle option is the most conventional in Go (`io.ReaderAt` and `io.Seeker`
are precedents) and is the recommendation, but the decision must be explicit and
made now. It should be decided together with
[API-18](#api-18-source-cannot-seek), since both are optional-capability
questions of the same shape.

### API-2: No cancellation mechanism

**Impact: BREAKING.** There is no `context.Context` anywhere in the module.

Verified: zero occurrences of `context.Context` in non-test code. Long
operations — `ResampleToMono16` over a 16-minute file, `ProcessChannels` with
concurrency, normalization that buffers a whole stream — cannot be cancelled or
given a deadline. For a library aimed at server-side telephony work, where
requests time out and shutdowns need to drain, this is a notable omission.

Retrofitting means either changing signatures (breaking) or adding parallel
`...Context` variants (workable but ugly; the `database/sql` pattern). Deciding
now is much cheaper.

Related: `ChannelOption` already offers `WithConcurrency`, so
`ProcessChannels` spawns goroutines with no way for the caller to stop them.

### API-3: `ReadSamples` contract is under-specified

**Impact: BREAKING** (tightening a contract changes what implementers must do).

The documented contract is one sentence:

> Returns number of float32 values written (not frames). When `n == 0` with
> `err == io.EOF`, the stream is finished.

It does not say whether `(n > 0, io.EOF)` is permitted. In practice **three of
the four decoders do exactly that** — they return the final partial buffer
together with `io.EOF`. Measured, draining each format to completion:

Reading 4096 values at a time, consuming `n` before checking `err`, the tail of
each stream is:

| Decoder | Last returns | Terminates at |
|---|---|---|
| wav | `(4096, nil) (4096, nil) (1324, io.EOF)` | **the combined return** |
| aiff | `(4096, nil) (4096, nil) (1324, io.EOF)` | **the combined return** |
| ogg vorbis | `(4096, nil) (4096, nil) (940, io.EOF)` | **the combined return** |
| mp3 | `(2304, nil) (2304, nil) (0, io.EOF)` | a separate empty EOF |

An earlier draft of this entry recorded all four as terminating with
`(0, io.EOF)`. That was an artefact of a probe that kept calling past the
combined return; wav, aiff and ogg stop *at* it, and only yield `(0, io.EOF)` if
called again.

Note also that mp3 returns 2304 values when asked for 4096 — one frame of
1152 x 2 channels. Short reads are the normal case for it, not an edge case.

A caller that writes the natural-looking loop

```go
n, err := src.ReadSamples(buf)
if err != nil { break }   // BUG: discards n samples
use(buf[:n])
```

silently loses the tail of the audio for three formats out of four, and works
for MP3. That is a trap the contract should either forbid or mandate, not leave
open.

Separately, `mp3` and `vorbis` both contain a path returning `(0, nil)`:

```go
if n == 0 {
    if err != nil { return 0, err }
    return 0, nil          // neither data nor EOF
}
```

Measured: this never fires today with the current upstream libraries, so it is
latent rather than an active bug. But the contract does not describe it, and a
caller looping until `io.EOF` would spin. `audio.Resampler` already defends
against it explicitly, which is itself evidence that the contract is unclear
enough that in-tree code cannot rely on it.

**Resolved:** `io.Reader` semantics — the combined return is permitted, `(0, nil)`
is forbidden for a non-empty `dst`, and a short read does not imply EOF. See
[roadmap_v1.md](roadmap_v1.md#d5--readsamples-follows-ioreader-semantics),
which also specifies the conformance test that enforces it.

### API-4: `ChannelOption` is a closed extension point

**Impact: BREAKING.**

```go
type ChannelOption func(*channelProcessor)   // channelProcessor is unexported
```

An exported function type over an unexported struct. Callers outside the package
**cannot write their own option**, because they cannot name the parameter type.
This looks like an open extension point and is not one.

Conventional fixes, both breaking:

- `func(*ChannelConfig) error` over an exported config struct.
- An interface with an unexported method — deliberately closed, but honestly so.

The second is a legitimate design if closure is intended. The current form is
the worst of both, implying openness without providing it.

### API-5: No `Encoder` abstraction

**Impact: BREAKING** (introducing it will shape `Decoder`'s symmetry).
**Relates to [user 1].**

There is a `Decoder` interface but **no `Encoder` interface** — verified, the
module contains no encoder abstraction of any kind. The only writer is a single
free function, `wav.WriteWAV16`.

This matters more than "encoders are missing" (that is
[FMT-1](#fmt-1-encoders-exist-for-one-format-in-one-configuration-user-1)).
Without an abstraction:

- `Registry` can register decoders but not encoders, so format-agnostic *output*
  is impossible; every caller hardcodes `wav.WriteWAV16`.
- There is no uniform place to express encoder options — bit depth, channel
  count, quality, bitrate.
- Each new output format adds another bespoke free function.

Designing `Encoder` before v1 also gives a chance to reconsider whether
`Decoder`'s `Decode(io.Reader) (Source, error)` shape is right, since the two
should be symmetric. See also
[API-17](#api-17-decoder-cannot-express-framed-codecs).

### API-6: `WriteWAV16` is permanently mono

**Impact: ADDITIVE to fix, but the wart is permanent.** **Relates to [user 1].**

```go
func WriteWAV16(w io.Writer, sampleRate int, samples []int16) error {
    numChannels := uint16(1)   // a literal, not a parameter
```

The channel count is hardcoded. Stereo output — and therefore two-direction call
recording — is impossible with the current writer.

A new `WriteWAVN(w, sampleRate, channels int, samples []int16)` is additive, and
`WriteWAV16` can delegate to it. `go-audio/wav.NewEncoder` already handles
arbitrary channel counts, bit depths and format tags, which bounds the work —
but it is archived, so it is a reference point rather than a dependency to
adopt. See
[FMT-1](#fmt-1-encoders-exist-for-one-format-in-one-configuration-user-1) and
[DEP-1](#dep-1-five-of-seven-dependencies-are-archived). But if `WriteWAV16` ships in v1 it must be
supported for the life of v1 as a mono-only special case. Worth deciding now
whether to introduce the general form *before* v1 and keep `WriteWAV16` as a
thin deprecated alias, rather than carrying two writers indefinitely.

### API-7: `Source` exposes no format metadata

**Impact: BREAKING.**

A `Source` reports only sample rate and channel count. It cannot report:

- **Bit depth** of the origin — becomes material once
  [FMT-2](#fmt-2-wav-decoding-is-narrow-user-5) lands and sources are no longer
  uniformly 16-bit.
- **Channel layout** — `audio.Channel` exists as a rich bitmask type, but a
  `Source` only says "6 channels", not *which* six. `ProcessChannels` therefore
  requires the caller to pass the layout separately, and `defaultLayout` has to
  guess it from the count.
- **Codec or container identity** — no way to ask what a `Source` came from.

The layout omission is the sharpest: the library has a good channel-layout model
and then cannot attach it to the thing it describes.

### API-8: `BufSize()` is vestigial

**Impact: BREAKING to remove.**

`BufSize()` is in the `Source` interface, so every implementer must provide it,
yet its value is close to meaningless:

- `Resampler.BufSize()` forwards its source's value while reading in blocks of
  its own choosing, so the number it returns describes nothing about its own
  behaviour.
- The wav decoder returns the constant `audio.DefaultBufSize`.
- The aiff decoder returns either a buffer capacity or that same constant,
  depending on which decode path it took.
- It is documented as "a hint, not a limit".

**Correction to an earlier draft of this entry**, which claimed no in-tree code
used it. There are four call sites — but they strengthen the case rather than
weaken it, because all four treat the value as an arbitrary allocation
heuristic:

| Call site | Use |
|---|---|
| `normalize()` | `make([]float32, bufSize)` and `make([]float32, 0, bufSize*100)` |
| `readAllSamplesAsInt16()` | Identical pattern |
| `channel_splitter.go` (×2) | `src.BufSize() / totalCh`, propagated to `ChannelSource.BufSize()`, which nothing outside tests reads |

All four behave identically with a constant, and `ChannelSource` dividing by the
channel count produces values like 682 for a six-channel split — not a useful
buffer size for anything.

**Resolved:** removed from the interface. See
[roadmap_v1.md](roadmap_v1.md#d3--bufsize-is-removed-from-source).

### API-9: `Registry` is string-keyed and its zero value panics

**Impact: BREAKING (keying) / INTERNAL (zero value).**

```go
type Registry struct {
    codecs map[string]Decoder
    mtx    *sync.Mutex          // pointer
}
```

Two problems:

1. **Zero value is unusable.** `var r audio.Registry; r.Register(...)` panics —
   `mtx` is a nil pointer and `codecs` a nil map. Idiomatic Go makes the zero
   value useful, and `sync.Mutex` by value costs nothing. Fixable without
   touching the exported surface, hence INTERNAL, but it should be fixed.
2. **String keys only.** Registration is by arbitrary format string, with
   extension-to-decoder mapping left to the caller — both examples in the tree
   duplicate the same registration block. Once RTP payload types
   (see [VOIP](#e-voip-and-packet-based-audio)) enter, dispatch is by integer
   payload type, which does not fit this map.

Also missing: no way to list registered formats, no unregister, and no
content-sniffing lookup (see
[FMT-9](#fmt-9-no-container-format-detection)).

### API-10: `ChannelSource` value and pointer asymmetry

**Impact: BREAKING.**

`ExtractChannels` and `SplitChannels` return `[]ChannelSource` — a slice of
**values** — while every method is on `*ChannelSource`. Consequences:

- `[]ChannelSource` does not satisfy `[]Source`; callers must take addresses of
  slice elements, and in-tree code does exactly that
  (`processChannel(ch *audio.ChannelSource, ...)`).
- Copying a `ChannelSource` value copies a struct containing a `sync.Mutex`,
  which linters flag in general and which is a latent correctness hazard.
- Inconsistent with `audpbx.SplitToMonoSources`, which returns `[]audio.Source`
  for what is conceptually the same operation.

Returning `[]*ChannelSource` or `[]Source` would be consistent and safe. Both
are breaking changes.

### API-11: Two different `ProcessChannels` functions

**Impact: BREAKING.**

```go
audio.ProcessChannels(src Source, layout Channel, fn ChannelProcessFunc) ([]ProcessedChannel, error)
audpbx.ProcessChannels(src audio.Source, layout audio.Channel, opts ...ChannelOption) (*ProcessingResult, error)
```

Same name, two packages, different signatures, different return types,
overlapping purpose. Legal Go, but for anyone reading docs or search results it
is a genuine trip hazard, and it is not obvious which is intended for external
use. Rename one before v1.

### API-12: Dead exported error values

**Impact: BREAKING to remove.**

Two exported sentinels are never returned by any non-test code:

- `wav.ErrUnsupportedWavChunks`
- `aiff.ErrUnsupportedAiffChunks`

`errors.Is` against them can therefore never match, which is worse than their
absence: they advertise a distinction the library does not make. Either wire
them to real conditions or delete them before v1, because deleting an exported
identifier afterwards requires a v2.

### API-13: Inconsistent error construction

**Impact: BREAKING** (callers' error handling depends on it).

Three styles coexist:

- Sentinels via `errors.New` in `audio`, `wav` and `aiff`.
- Ad-hoc `fmt.Errorf` with no sentinel, so not matchable — for example
  `wav/decoder.go` returns `fmt.Errorf("unsupported audio format: %d ...")`
  rather than wrapping a sentinel, and the root package does this throughout
  (`"source must be mono (1 channel), got %d channels"`, `"buffer size too
  small: %d (minimum 256)"`, `"no output configured: ..."`).
- The same condition modelled inconsistently across packages: `wav` has
  `ErrNegativePosition`, while `aiff` returns a bare
  `fmt.Errorf("negative position")` for the identical case.

A fourth pattern appears eight times in non-test code: `fmt.Errorf("%w", err)`
with no message. This wraps without adding context, so it is equivalent to
returning `err` directly while looking deliberate. It appears in `resample.go`,
`audio/mono_mixer.go`, `audio/resampler.go`, `formats/mp3/decoder.go` and
`formats/wav/pcm_16_writer.go`. Either add context or return the error
unwrapped.

There are also no error *types* carrying structured fields, so a caller cannot
programmatically learn which channel failed, or what bit depth was rejected —
only parse a string. `ProcessingResult` partially compensates for the
multi-channel case.

For v1, callers need a documented, matchable error contract.

### API-14: `ResampleToMono16` signature and memory model

**Impact: BREAKING.**

```go
func ResampleToMono16(src audio.Source, targetRate int, bufferSize int) ([]int16, int, error)
```

- The returned `int` is `targetRate` echoed straight back — the caller already
  has it. It is noise in the signature.
- `bufferSize` is exposed as a tuning knob but, since the resampler reads in
  internal blocks, has almost no effect on throughput. Measured: a 64-sample
  buffer and a 4096-sample buffer are within noise of each other. Exposing a
  parameter that does nothing invites cargo-culting.
- The whole result is accumulated in memory. Fine for prompts, not for the
  16-minute test asset, and there is no streaming variant.

A streaming counterpart — writing to an `io.Writer`, or returning a `Source` —
would be additive. The signature cleanup would not.

### API-15: No documented concurrency guarantees

**Impact: BREAKING (contract).**

Nothing in the API documents whether a `Source` may be read from multiple
goroutines, whether `Close` is safe concurrently with `ReadSamples`, or whether
the `ChannelSource`s returned by a splitter may be read in parallel. The
splitter clearly intends some coordination — it holds a `sync.Mutex` and its doc
says reads are "coordinated" — and `WithConcurrency` implies parallel use is
supported somewhere, but the guarantee is never stated.

Callers will assume something. Whatever is true should be written down before v1
fixes it by accident.

### API-16: `utils` is a grab-bag package

**Impact: BREAKING to rename.**

`utils` currently exports PCM geometry constants (`BitDepth*`,
`BytesPerSample*`, `SampleScale*`, `BitsPerByte`) alongside DSP helpers
(`CubicInterpolate`, `Float32ToInt16`). "Utils" says nothing about what belongs
there, which reliably attracts unrelated additions over time.

A name like `pcm` for the sample-format constants and conversions, with
interpolation living nearer the resampler, would be clearer. Renaming an
exported package after v1 is breaking, so this is now or never — and it is
purely cosmetic, which is exactly the kind of thing that gets deferred past the
point where it can be done.

### API-17: `Decoder` cannot express framed codecs

**Impact: ADDITIVE if designed now.** **Relates to [user 2].**

```go
type Decoder interface {
    Decode(r io.Reader) (Source, error)
}
```

This assumes a self-describing container over a continuous byte stream. It does
not fit codecs that arrive as discrete frames with no container — G.729 (10-byte
frames), Opus (variable frames), or any RTP payload — where frame boundaries
carry meaning, sample rate comes from out-of-band signalling, and lost frames
must be signalled explicitly so predictive decoders can conceal them.

A sibling `PacketDecoder` interface is the proposed answer and is specified in
[from_voip.md](from_voip.md#1-a-frame-oriented-decode-interface). Adding a new
interface is additive; it is listed here because designing it alongside
`Encoder` ([API-5](#api-5-no-encoder-abstraction)) before v1 avoids ending up
with three unrelated codec abstractions.

### API-18: `Source` cannot seek

**Impact: BREAKING.**

There is no way to reposition a `Source`. The only `Seek` implementations in the
tree are the internal `readSeeker` helpers in `wav` and `aiff` used to wrap
non-seekable readers; they are not part of any exported contract.

The underlying capability largely exists — `wav.Decoder` and `mp3.Decoder` both
offer `Seek`, and `oggvorbis.Reader` exposes `Position` — so, as with
[API-1](#api-1-source-cannot-report-length-or-duration), it is discarded at the
wrapper boundary.

What this blocks:

- **Two-pass processing.** Normalization currently buffers the entire stream
  ([DSP-8](#dsp-8-normalization-buffers-the-entire-stream)) precisely because it
  cannot rewind to apply the gain it computed on the first pass.
- **Extracting a time range** from a longer recording without decoding and
  discarding everything before it — the cost falls on every
  [DSP-4](#dsp-4-the-library-can-decompose-audio-but-not-compose-it) trim
  operation.
- **Re-reading a source**, for example to retry after a downstream error.

Same design question as [API-1](#api-1-source-cannot-report-length-or-duration),
and it should be answered at the same time: mandatory interface method, or
optional interface discovered by type assertion? Not every source can seek — a
packet-fed or synthetic stream cannot — which argues for optional. Either way
the decision is a v1 gate.

### API-19: Tag manipulation has no home in the API

**Impact: BREAKING** if it lands on `Source`; ADDITIVE if it is a separate
package. That choice is the gap.

[API-7](#api-7-source-exposes-no-format-metadata) concerns *technical* metadata
needed to interpret samples — bit depth, channel layout. This entry is about
*descriptive* metadata: titles, comments, annotations
([FMT-10](#fmt-10-no-metadata-or-tag-handling)). They are different problems
and probably want different homes.

The decision that must be made before v1 is **whether tags are part of the audio
abstraction at all.** Two arguments against putting them on `Source`:

1. **Reading tags should not require decoding audio.** Listing the titles in a
   prompt directory should be a header read, not a decode of every file.
2. **Writing tags should not require re-encoding audio.** This is the important
   one. If metadata flows only through `Source` and `Encoder`, then *every
   retag becomes a transcode* — lossy for MP3 and Vorbis, and wasteful for
   everything. A lossless retag is the common operation and would be
   structurally impossible.

That argues for a **separate tag package operating on files or
`io.ReadWriteSeeker`s**, independent of the decode path — with the `Encoder`
([API-5](#api-5-no-encoder-abstraction)) additionally able to *accept* a tag set
so metadata can survive a genuine transcode
([FMT-19](#fmt-19-no-cross-format-metadata-mapping-or-preservation-policy)).

Getting this wrong is expensive to undo: if tags are added to `Source` in v1,
removing them later is a v2, and the lossless-retag path stays unavailable for
the life of v1.

### API-21: `Channel` is `uint32`, capping the library at 32 channels

**Impact: BREAKING.**

`audio.Channel` is a `uint32` bitmask, so it can represent at most 32 speaker
positions — 18 named plus 14 unnamed. `defaultLayout`'s fallback loop
terminates when the bit shifts off the top:

```go
for bit := Channel(1); bit != 0 && count < channels; bit <<= 1 {
```

Measured behaviour across channel counts:

```
  18 channels -> ok
  31 channels -> ok
  32 channels -> ok
  33 channels -> ERROR: layout has 32 channels but source has 33
  40 channels -> ERROR: layout has 32 channels but source has 40
  64 channels -> ERROR: layout has 32 channels but source has 64
```

Two problems. The cap itself, and the **failure mode**: the error blames a
layout/source mismatch rather than naming the real cause, so a caller has no
way to tell that 32 is a hard limit rather than a bug in their input.

Whether 32 is enough is a judgement call, and for telephony it is closer to the
boundary than it looks. A single E1 carries 30 voice channels, so a 32-channel
logger sits right at the limit; aggregating multiple E1s, or multi-track
recording of many concurrent calls into one file, exceeds it immediately.

Widening to `uint64` is a breaking change to an exported type, so it is now or
never. If the cap is kept deliberately, the limit should be documented and the
error should say so.

### API-20: `Source` cannot report position

**Impact: BREAKING.**

`Source` cannot say how far through the stream it is. This is distinct from
[API-1](#api-1-source-cannot-report-length-or-duration) (total length) and both
are needed for progress: a fraction is position over total.

A caller driving its own read loop can count the samples it receives, so
position is derivable at the *top* of a pipeline. It is not derivable:

- **from inside a high-level call** — `ResampleToMono16` owns its loop, so a
  caller has no visibility into it at all;
- **for an intermediate stage** — with `decode → resample → mix`, counting
  output samples tells you about the mixer, not about how much of the *input*
  has been consumed, and those differ by the resampling ratio;
- **after a seek** ([API-18](#api-18-source-cannot-seek)), where a counted
  position and the real one diverge.

`oggvorbis.Reader` already exposes `Position()`, and the other decoders can
derive it, so once again the capability exists upstream and is dropped at the
wrapper.

**Decide this together with [API-1](#api-1-source-cannot-report-length-or-duration)
and [API-18](#api-18-source-cannot-seek).** All three are the same question —
optional capabilities on `Source`, mandatory interface method versus optional
interface discovered by type assertion — and answering them separately risks
three inconsistent answers. Not every source can satisfy them: a packet-fed
timeline has a position but no meaningful total until complete, and a synthetic
generator has neither.

---

## B. Format and codec coverage

### FMT-1: Encoders exist for one format in one configuration **[user 1]**

**Impact: ADDITIVE.**

The only encoder in the module is `wav.WriteWAV16` — mono, 16-bit, PCM. Every
other format is decode-only.

Missing, roughly in order of usefulness for telephony work:

| Target | Notes |
|---|---|
| WAV, multi-channel | Blocks two-direction recording. See [API-6](#api-6-writewav16-is-permanently-mono) |
| WAV, 8/24/32-bit and float | Pairs with [FMT-2](#fmt-2-wav-decoding-is-narrow-user-5) |
| WAV, A-law and µ-law payload (tags 6 and 7) | Directly useful for PBX prompts |
| Raw `.ulaw` / `.alaw` / `.sln` | Asterisk native formats; see [FMT-6](#fmt-6-no-headerless-raw-formats) |
| AIFF | Round-trip symmetry with the existing decoder |
| Ogg Vorbis, Opus | Lossy distribution |
| MP3 | Patent position now clear; encoder libraries are large |

The prerequisite is [API-5](#api-5-no-encoder-abstraction) — without an
`Encoder` interface, each of these becomes another one-off function.

**The WAV rows are cheaper than they look.** `go-audio/wav` — already a
dependency — provides `NewEncoder(w io.WriteSeeker, sampleRate, bitDepth,
numChans, audioFormat int)`, which covers arbitrary channel counts, arbitrary
bit depths, an arbitrary format tag (so A-law and µ-law payloads,
[FMT-11](#fmt-11-non-pcm-payloads-inside-the-wav-container)) and metadata
injection ([FMT-10](#fmt-10-no-metadata-or-tag-handling)).

`audpbx` does not use it. `wav.WriteWAV16` is a hand-rolled writer that, in
exchange for avoiding the dependency's encoder, gives up every one of those
capabilities — which is the underlying reason for
[API-6](#api-6-writewav16-is-permanently-mono).

**But do not adopt it.** `go-audio/wav` is archived
([DEP-1](#dep-1-five-of-seven-dependencies-are-archived)), so taking on more of
its surface area is the wrong direction. What the observation is still good for
is scoping: it shows that arbitrary channels, bit depths, format tags and
metadata injection are a well-understood, bounded amount of work — the argument
for owning the WAV writer rather than for importing one.

One caveat that cuts the other way: the upstream encoder requires an
`io.WriteSeeker`, because it patches chunk sizes after writing. That confirms
from the opposite direction that streaming WAV output to a non-seekable
destination needs deliberate design
([PBX-5](#pbx-5-no-streaming-io-boundary)).

### FMT-2: WAV decoding is narrow **[user 5]**

**Impact: ADDITIVE.**

The decoder accepts only format tag 1 and only 16-bit:

```go
if dec.WavAudioFormat != pcmAudioFormat { ... }   // tag 1 only
if int(dec.BitDepth) != utils.BitDepth16 { ... }  // 16 only
```

Rejected as a result:

- **`WAVE_FORMAT_EXTENSIBLE` (tag `0xFFFE`)** — what `ffmpeg` emits for more
  than two channels. Confirmed during earlier testing: a 4-channel WAV produced
  by ffmpeg fails with "unsupported audio format: 65534". Since the library has
  a full `WAVEFORMATEXTENSIBLE` channel-mask model in `audio.Channel`, and the
  extensible header is precisely where that mask lives, this is the format the
  library is best equipped to interpret and currently refuses.
- **8-, 24- and 32-bit PCM**, and **32/64-bit float** (tag 3).
- **A-law (tag 6) and µ-law (tag 7)** payloads.

Note the knock-on: `utils` already defines `BitDepth8/16/24/32` and matching
scales, and the aiff fallback path uses them, so the conversion groundwork
exists. What is missing is the direct-decode paths and the metadata to report
which depth was used ([API-7](#api-7-source-exposes-no-format-metadata)).

Also affected: `ErrOnlyPCM16bitSupported` becomes a misnomer once other depths
are accepted, and it is exported.

### FMT-3: AIFF decoding is 16-bit only

**Impact: ADDITIVE.**

Same restriction as WAV, plus no AIFF-C (`.aifc`) support, so compressed AIFF
variants are rejected. The `PCMBuffer` fallback path already handles 8/24/32-bit
via `utils.SampleScale*`, so widening the direct path is the smaller half of the
work.

### FMT-4: Telco and VoIP codecs absent **[user 2]**

**Impact: ADDITIVE** (given [API-17](#api-17-decoder-cannot-express-framed-codecs)).

No telephony codec is usable through this module. For a package named `audpbx`
targeting Asterisk and FreeSWITCH, this is the largest functional gap.

**What "support" means here.** Per
[the scope note](#scope-what-this-project-is-and-is-not), the gap is *codec
integration*, not codec implementation. Closing it means:

1. the plug-in point — [API-17](#api-17-decoder-cannot-express-framed-codecs)'s
   `PacketDecoder`, which is genuinely this module's work;
2. a thin wrapper per codec, adapting a third-party implementation to it;
3. a survey, per codec, of which implementations exist and under what terms.

Step 3 is a prerequisite task in its own right and has not been done. The
"availability" column below therefore records what is *expected* rather than
what has been verified, and each entry needs confirming before it is planned.
Where no usable Go implementation exists, the honest options are a cgo binding
([FMT-5](#fmt-5-aac-support-user-6) sets that precedent), a separate standalone
project that `audpbx` then imports, or leaving the codec unsupported.

#### Decided routes

A survey has now been done, so this table records a **decision per codec**
rather than a wish list. Availability was checked against pkg.go.dev, crates.io
and each project's own pages; it should be re-checked before work starts, since
it moves.

| Codec | RTP PT | Route | Rationale |
|---|---|---|---|
| **G.711** µ-law / A-law | 0 / 8 | **Adopt `zaf/g711`** | BSD-3-Clause, 32 importers, v1.4.0, explicitly encode and decode. Solved problem; `livekit/media-sdk/g711` (Apache-2.0) and `bluenviron/mediacommon/.../g711` (MIT) are active alternatives, both as subpackages of larger modules. |
| **G.722** | 9 | **Evaluate `livekit/media-sdk/g722`, else own it** | Apache-2.0 and actively maintained, but part of a v0.1.x media SDK. `shenjinti/go722` (BSD-2) has little adoption; `gotranspile/g722` declares **no licence** and is unusable. Writing it is tractable — two QMF filters plus two ADPCM cores. **Carries a clock-rate trap, see below.** |
| **G.726** | dynamic | **Write a standalone package** | The genuine gap. `lkmio/g726`, `general252/g726` and `Distortions81/g726` are all Apache-2.0 but have 0–2 importers and no synopses; nothing looks authoritative. Comparable effort to G.722. **The API must expose the RFC 3551 versus AAL2 bit-packing choice** — picking wrong yields noise, not an error. |
| **G.729** | 18 | **libavcodec, build-tagged** | FFmpeg has a **native G.729 decoder**. No pure-Go implementation exists, and CS-ACELP is weeks of specialist DSP, so this is much the cheapest route. |
| **G.723.1** | 4 | **libavcodec, build-tagged** | Native in FFmpeg (decode and encode). Two excitation schemes (MP-MLQ at 6.3k, ACELP at 5.3k) make it hard to write. |
| **Opus** | dynamic | **`pion/opus`, pending validation** | MIT, pure Go, has both encoder and decoder plus conformance tests. **But its README states no completeness claim, no mode coverage (SILK/CELT/hybrid) and no limitations** — for a codec that absence is itself a finding. Validate against the RFC 6716 test vectors, and read its tracking issue, before committing. `livekit/media-sdk/opus` is an alternative but its pure-Go-versus-cgo status is undocumented. |
| **GSM 06.10** | 3 | **libavcodec if needed** | Native decode in FFmpeg; encode needs libgsm. Needed mainly for Asterisk `.gsm` file compatibility ([FMT-13](#fmt-13-asterisk-and-freeswitch-native-file-formats)). |
| **AMR-NB / AMR-WB** (= G.722.2) | dynamic | **libavcodec if needed** | Native decode in FFmpeg; encode needs libopencore-amrnb / libvo-amrwbenc. `livekit/media-sdk/amrwb` also exists. |
| **L16 / L8** ("slin") | 10, 11, dynamic | **No dependency — plain PCM** | What Asterisk and FreeSWITCH use internally and what `.sln` holds. Easy to overlook. |
| **Speex** | dynamic | **Dropped** | speex.org states it plainly: *"The Speex codec has been obsoleted by Opus… users are encouraged to switch."* Revised BSD, and `libspeexdsp` is a separate library. Legacy interop only. |
| **iLBC** | dynamic | **Dropped** | `TimothyGu/libilbc` is BSD-3-Clause but needs a **C++14 compiler, CMake and Abseil** — heavy for a cgo binding, for a codec superseded by Opus. |
| **G.722.1** (Siren), **EVS**, **Codec2**, **SILK**, **iSAC**, **DVI4**, **LPC** | various | **Not planned** | Legacy or specialist. Revisit on concrete demand; libavcodec covers some. |

**Not codecs, but payload formats that must be handled to decode a stream:**

| Format | RTP PT | Notes |
|---|---|---|
| **Comfort noise** (RFC 3389) | 13 | Not audio — a noise level to synthesise. Needs a decision: synthesise or treat as silence. |
| **`telephone-event`** (RFC 4733) | dynamic | DTMF and tones as *events*, not audio. Occupies timestamp space with no waveform. See [TONE-1](#tone-1-no-dtmf-generation-requested). |
| **RED** (RFC 2198) | dynamic | Redundant coding for loss resilience. Wraps other payloads; must be unwrapped before decoding. |
| **Opus in-band FEC / DTX** | — | Affects frame accounting when reconstructing a timeline. |

#### Why this split, and not one libavcodec binding

Once libavcodec is bound for G.729 and G.723.1, G.722 and G.726 come free — both
are native FFmpeg codecs, decode and encode. Writing standalone packages for
them is therefore deliberate duplication, and worth being explicit about why.

The reason: **G.711, G.722 and G.726 are what appears in day-to-day PBX
traffic; G.723.1 and G.729 are legacy trunk cases.** Keeping the common codecs
pure Go means the **default build needs no cgo at all**, and only callers who
genuinely need a legacy codec pay the toolchain cost. That is a materially
better default than requiring everyone to link libavcodec.

The resulting posture:

- Default build: pure Go, plus one vendored CC0 header for MP3
  ([DEP-1](#dep-1-five-of-seven-dependencies-are-archived))
- cgo only behind build tags, for G.723.1, G.729 and AAC
  ([FMT-5](#fmt-5-aac-support-user-6))
- **No LGPL obligation reaches downstream users unless they opt in** — see
  [DEP-2](#dep-2-no-dependency-policy)

#### Codecs written in-house live outside this module

Where a codec is written rather than adopted — G.726, and possibly G.722 — it
belongs in a **separate Go module that `audpbx` imports**, not inside
`audpbx`. That is what
[the scope note](#scope-what-this-project-is-and-is-not) permits ("a separate
standalone project that `audpbx` then imports"), and it keeps the boundary from
eroding on first contact.

Difficulty across the G.7xx family is very uneven, which is worth knowing before
committing to "own the G.7xx codecs" as a single decision:

| Codec | Effort |
|---|---|
| G.711 | Trivial — companding tables. Adopted anyway. |
| G.722 | Moderate — sub-band ADPCM, well specified, ITU reference exists |
| G.726 | Moderate — ADPCM plus the bit-packing variants |
| G.723.1 | Hard — two excitation schemes |
| G.729 | Hardest — CS-ACELP, adaptive and fixed codebooks, LSP quantisation, post-filter |

#### Prior art worth reading before building tone and timeline features

`livekit/media-sdk` (Apache-2.0, v0.1.1) contains `opus`, `g711`, `g722`,
`amrwb`, **`dtmf`**, **`tones`**, **`jitter`**, `mixer`, `ring`, `rtp`, `srtp`,
`sdp` and `webm`.

That overlaps this document's gap list more than a codec library normally would
— [TONE-1](#tone-1-no-dtmf-generation-requested),
[TONE-3](#tone-3-no-call-progress-tones-and-no-tone-plan-model),
[VOIP-1](#voip-1-to-voip-4) and
[DSP-4](#dsp-4-the-library-can-decompose-audio-but-not-compose-it). Worth
reading as prior art on API shape even if not adopted, since it is people
solving the same problems. Caveats: v0.1.x so expect churn, it is a media SDK
rather than a codec library, and importing a subpackage still brings the module
into the dependency graph.

#### The G.722 clock-rate trap

G.722 samples at **16 kHz** but RFC 3551 assigns it an **RTP clock rate of
8000** — a known historical error in the specification that was never corrected
for compatibility reasons. So for G.722 the timestamp advances at half the
sample rate.

This is the same class of trap as Opus's invariant 48 kHz clock, and it
reinforces the design conclusion already reached in
[from_voip.md](from_voip.md#opus--dynamic-payload-type): **clock rate must be a
per-codec property, never assumed equal to the sample rate.** Any timeline
arithmetic that treats timestamp units as samples will be wrong by 2× for
G.722 and by up to 6× for Opus.

Design work for the packet side is captured in
[from_voip.md](from_voip.md); this entry exists so the gap index is complete.

### FMT-5: AAC support **[user 6]**

**Impact: ADDITIVE.**

No AAC. The stated intent is bindings with static linking, static building or a
dynamic library, selected by build tags.

This is the first dependency that would break the module's native-Go-first
property, so it needs a policy, not just an implementation:

- **Build-tag matrix.** Something like `aac_static`, `aac_dynamic`, absent by
  default. Every combination needs to compile and be tested, or it will rot.
- **Pure-Go fallback behaviour.** Without the tag, does `aac.Decoder{}` exist
  and return an error, or does the package not build at all? The former keeps
  `Registry` registration uniform.
- **CI cost.** Cgo build variants multiply CI jobs, and there is currently no CI
  at all ([QA-2](#qa-2-no-ci)).
- **Licensing.** AAC patent licensing is materially different from the expired
  G.729 situation and should be assessed independently.
- **Precedent.** Whatever is decided here becomes the template for every future
  cgo-backed codec, so it is worth settling deliberately — and per
  [FMT-4](#fmt-4-telco-and-voip-codecs-absent-user-2) the same machinery is
  wanted for G.723.1 and G.729 via libavcodec.

**Resolved:** cgo-backed codecs live in **separate modules**, not behind build
tags in `audpbx` core — build tags control compilation but not the dependency
graph, so tags alone would still put an LGPL binding in every consumer's
`go.sum` and licence audit. Tags operate inside the cgo module, selecting link
mode. See
[D11](roadmap_v1.md#d11--cgo-lives-in-separate-modules-not-behind-build-tags).

#### Trimming libavcodec: size and speed are mostly not a trade-off

Relevant because libavcodec is the planned route for G.723.1 and G.729, and its
full build is large — 19.9 MB with 553 decoders enabled, measured on a typical
distribution package. Three independent levers, which behave differently:

| Lever | Size effect | Speed cost |
|---|---|---|
| `--disable-everything` then `--enable-decoder=...` | **Large** | **None** |
| `--enable-small` | Moderate further gain | Real but modest |
| `--disable-asm` / `--disable-x86asm` | Small | **Significant** |

**Trimming is free.** Removing codecs deletes code that would never execute; the
ones kept run identically. Component selection is per-item — decoders, encoders,
demuxers, muxers, parsers, protocols and bitstream filters are each individually
toggleable (see `./configure --help` for the authoritative flag list).

`--enable-small` genuinely trades throughput for size and is only worth it under
real size pressure. Disabling the hand-written SIMD is where the significant
speed loss lives and should be avoided unless portability demands it.

Recommended shape: trim aggressively, skip `--enable-small`, keep asm enabled,
and build **without `--enable-gpl`** so the result stays LGPL-2.1. Exact figures
need measuring against the chosen configuration.

### FMT-6: No headerless raw formats

**Impact: ADDITIVE.**

No support for raw, headerless audio: `.sln` (Asterisk signed-linear), `.ulaw`,
`.alaw`, `.raw`, `.pcm`. These are exactly what Asterisk and FreeSWITCH consume
natively for prompts, so for the stated use case they matter more than some
container formats.

They are trivial once G.711 exists, but they surface an API question: a
headerless source has **no discoverable sample rate or channel count**, so
`Decoder.Decode(io.Reader) (Source, error)` cannot determine them. Either such
formats get constructors taking explicit parameters, or `Decoder` needs an
options-carrying variant. Another argument for resolving
[API-17](#api-17-decoder-cannot-express-framed-codecs) and
[API-5](#api-5-no-encoder-abstraction) together.

### FMT-7: No FLAC

**Impact: ADDITIVE.**

Common for archival and source material, and mature pure-Go implementations
exist. Lower priority than the telephony codecs but cheap, and it fits the
existing container-decoder shape with no API change.

### FMT-8: Mono MP3 is decoded as stereo

**Impact: BEHAVIOURAL.**

```go
// go-mp3 outputs stereo (2 channels) for most MP3 files
channels: 2,
```

Confirmed accurate against upstream, which documents: *"The stream is always
formatted as 16bit (little endian) 2 channels even if the source is single
channel MP3."* So the value is correct for the library being used, not a bug.

The consequences are still real:

- `Source.Channels()` reports 2 for a genuinely mono file, which is misleading
  metadata for any caller that branches on it.
- A mono MP3 is upmixed by go-mp3, then averaged back down by `MonoMixer` —
  double the decode work and memory for no benefit.

Fixing means reading the MP3 frame header to learn the real channel mode and
downmixing at the wrapper, or changing decoder. Worth an explicit decision
rather than leaving the comment as the only explanation.

### FMT-9: No container format detection

**Impact: ADDITIVE.**

Format selection is entirely the caller's problem. Both examples in the tree
duplicate the same registry-building block and then dispatch on file extension:

```go
ext := filepath.Ext(inPath)
dec, ok := reg.Get(ext[1:])
```

There is no content sniffing (`RIFF`/`WAVE`, `FORM`/`AIFF`, `OggS`, MP3 frame
sync or ID3), and no default registry, so every consumer reimplements both. A
`DetectFormat(io.Reader)` or `OpenAny` helper would remove a recurring chunk of
boilerplate — and extension-based dispatch is simply wrong for RTP payloads and
raw streams, which have no filename.

### FMT-10: No metadata or tag handling

**Impact: ADDITIVE** (but see [API-19](#api-19-tag-manipulation-has-no-home-in-the-api)
for where it lives, which is a v1 decision).

Metadata is neither read nor written for any format. None of **add**, **change**,
**copy** or **remove** is possible, and any transcode discards everything
silently.

This is a conspicuous omission in this repository specifically:
`examples/testdata/METADATA_PRESERVATION.md` documents metadata preservation as
a property of the test assets, maintained by an external `ffmpeg -map_metadata`
script. So the project already depends on metadata the library cannot see.

#### Tag systems per format

Each format has its own system, and they are not variations on one theme:

| Format | System | Notes |
|---|---|---|
| MP3 | **ID3v2.2 / 2.3 / 2.4** at the start | Frame-based and extensible. Version differences are real: v2.4 permits UTF-8 and a footer, v2.3 does not; date handling differs (`TYER`/`TDAT` versus `TDRC`); size fields use different sync-safe encodings. |
| MP3 | **ID3v1 / v1.1** in the last 128 bytes | Fixed 30-byte fields, one-byte genre index, no encoding declaration. v1.1 steals two comment bytes for a track number. |
| MP3 | APEv2, Lyrics3 | Occasionally present; low priority. |
| Ogg Vorbis | **Vorbis comments** | Freeform UTF-8 `KEY=value` in the comment header packet. Field names are convention, not schema. Cover art via base64 `METADATA_BLOCK_PICTURE`. |
| Opus | **OpusTags** | Same comment structure, different header magic. See [FMT-14](#fmt-14-opus-in-ogg-is-not-the-same-as-vorbis-in-ogg). |
| FLAC | `VORBIS_COMMENT`, `PICTURE`, `CUESHEET` blocks | Pairs with [FMT-7](#fmt-7-no-flac). |
| WAV | **`LIST`/`INFO`** chunk | Four-character-code subchunks: `INAM` title, `IART` artist, `ICMT` comment, `ICRD` date, `IPRD` album, `IGNR` genre, `ISFT` software, `ICOP` copyright, `IENG` engineer. |
| WAV | **`bext`** (Broadcast WAV, EBU Tech 3285) | Description, Originator, OriginationDate/Time, **`TimeReference`** (sample-accurate start position), UMID, loudness fields, and **`CodingHistory`**. Directly useful for PBX work — see [PBX-7](#pbx-7-no-call-or-prompt-annotation). |
| WAV | `iXML`, `aXML`, `id3 ` chunk | Production metadata; an ID3 block can legally appear inside RIFF. |
| AIFF | `NAME`, `AUTH`, `(c) `, `ANNO`, `COMT` chunks | Plus `MARK` markers, covered by [FMT-16](#fmt-16-in-file-markers-and-regions-are-discarded). |
| AAC / MP4 | `moov/udta/meta/ilst` atoms | iTunes-style four-character atoms (`©nam`, `©ART`, `trkn`, `covr`). A different model again. Pairs with [FMT-5](#fmt-5-aac-support-user-6). |
| `.sln`, `.ulaw`, `.alaw`, `.gsm`, raw | **none — no capability at all** | Headerless formats cannot carry metadata. See [PBX-7](#pbx-7-no-call-or-prompt-annotation). |

#### For WAV, most of this is already available and discarded

Worth knowing before estimating the work: **`go-audio/wav` already implements
RIFF `INFO` metadata in both directions.** `wav.Decoder` carries a
`Metadata *Metadata` field populated by `ReadMetadata()`, and `wav.Encoder`
accepts a `Metadata` struct and writes the `LIST` chunk itself. The struct
covers Artist, Title, Comments, Copyright, CreationDate, Engineer, Technician,
Genre, Keywords, Medium and more.

`audpbx` calls neither — the same discard-at-the-wrapper pattern as
[API-1](#api-1-source-cannot-report-length-or-duration) and
[API-18](#api-18-source-cannot-seek).

Note though that `go-audio/wav` is archived
([DEP-1](#dep-1-five-of-seven-dependencies-are-archived)), so "just call
`ReadMetadata`" is a short-term expedient rather than the destination. The
lasting value of the observation is that it demonstrates RIFF `INFO` handling
is small and well understood — useful for scoping a replacement, not an
argument for leaning harder on archived code.

`go-audio/aiff` similarly exposes an `AppleInfo` field.

ID3, Vorbis comments and `bext` have no support in the current dependencies.
Consistent with
[the scope note](#scope-what-this-project-is-and-is-not), the first step is a
survey of existing Go tag libraries — several mature ones exist for ID3 and for
multi-format tag reading — rather than writing parsers here. Tag and container
parsing is not codec work, so implementing it in-module is legitimate if no
suitable dependency is found; it is simply not the first option.

#### Operations, and why they are not uniform

The four requested operations have materially different costs per format,
which the API has to acknowledge rather than paper over:

| Operation | Consideration |
|---|---|
| **Add** | Needs somewhere to put the tag. For RIFF and AIFF that means inserting or resizing a chunk, which moves the audio data. |
| **Change** | Must preserve fields it does not understand. A round trip that silently drops unrecognised ID3 frames is a data-loss bug, not a formatting difference. |
| **Copy** | Cross-format copying is lossy and non-bijective — see [FMT-19](#fmt-19-no-cross-format-metadata-mapping-or-preservation-policy). |
| **Remove** | Should distinguish "remove one field", "remove one tag system" (say ID3v1 but not ID3v2) and "strip everything". |

**In-place editing is possible for some formats and not others**, and this should
be explicit rather than discovered:

- **ID3v2** is designed for it — the specification provides padding, so a tag
  can often be rewritten without touching the audio.
- **ID3v1** is a fixed-size trailer, so it is always cheap.
- **Vorbis comments and OpusTags live inside an Ogg header packet.** Changing
  them means rewriting and repaginating the whole logical stream. There is no
  in-place edit.
- **RIFF and AIFF** need chunk resizing, so anything but a same-size overwrite
  rewrites the file.

A "lossless retag" that alters metadata without re-encoding audio is the common
case and should be the fast path where the format allows it.

#### Character encoding

Worth calling out because it is a frequent source of corruption:

- Vorbis comments and OpusTags are **always UTF-8**.
- **ID3v2.4** allows UTF-8; **ID3v2.3** does not — only ISO-8859-1 or UTF-16.
- **ID3v1** has no encoding field at all; contents are conventionally
  ISO-8859-1 and in practice frequently a local code page.
- **RIFF `INFO`** is nominally ASCII/Latin-1 with no declared encoding.

So non-Latin text — Hebrew, Arabic, Cyrillic, CJK — has no well-defined
representation in ID3v1 or RIFF `INFO`, and writing UTF-8 bytes into those
fields produces something that reads back as mojibake in other tools. Any API
here needs a documented position on what it writes and what it does on
transcoding a string it cannot represent.

### FMT-11: Non-PCM payloads inside the WAV container

**Impact: ADDITIVE.**

[FMT-2](#fmt-2-wav-decoding-is-narrow-user-5) covers bit depths and
`WAVE_FORMAT_EXTENSIBLE`. Separately, a WAV container can carry a *compressed*
payload, identified by its format tag, and none of these are handled:

| Tag | Payload | Why it matters for a PBX |
|---|---|---|
| `0x0006` / `0x0007` | A-law / µ-law | **Asterisk writes these.** A `.wav` holding G.711 is common for prompts and recordings. |
| `0x0031` | GSM 06.10 | Asterisk's "WAV-GSM" output. Frequently encountered in voicemail stores. |
| `0x0002` / `0x0011` | MS ADPCM / IMA ADPCM | Older IVR prompt libraries. |
| `0x0003` | IEEE float (32/64-bit) | Common from DAWs when producing prompts. |
| `0x0055` | MP3 inside WAV | Occasionally produced by Windows tooling. |
| G.726 variants | ADPCM | Tag assignments vary by registry; confirm against the source of the files in question. |

Two structural details go with this:

- **The `fact` chunk** is required for non-PCM WAV and carries the true sample
  count. Without reading it, duration and length for compressed payloads cannot
  be determined — which ties directly to
  [API-1](#api-1-source-cannot-report-length-or-duration).
- **Streaming WAV with unknown length.** When writing to a pipe or socket the
  final size is not known in advance, and the convention is to write
  `0xFFFFFFFF` (or a placeholder) for the RIFF and data sizes. `wav.WriteWAV16`
  computes sizes up front from a complete `[]int16`, so it cannot write to a
  non-seekable destination at all. Relevant to any streaming output
  ([API-14](#api-14-resampletomono16-signature-and-memory-model),
  [PBX-5](#pbx-5-no-streaming-io-boundary)).

### FMT-12: AIFF-C compression types

**Impact: ADDITIVE.**

The decoder rejects AIFF-C outright. The variants worth supporting are not
exotic:

| Compression type | Meaning |
|---|---|
| `sowt` | **16-bit little-endian PCM.** Despite AIFF being a big-endian format, this is what Apple tooling commonly produces. The most likely AIFF-C file to be handed to this library. |
| `NONE` | Big-endian PCM in an AIFF-C wrapper; already decodable but currently refused by the container check. |
| `ulaw` / `ULAW`, `alaw` / `ALAW` | G.711 in AIFF-C. |
| `fl32` / `fl64` | Float PCM. |
| `ima4`, `MAC3`, `MAC6` | Legacy Apple compression. |

`sowt` is the notable one: a caller with an ordinary Apple-produced file gets a
flat rejection today, and the fix is a byte-order flag rather than a codec.

### FMT-13: Asterisk and FreeSWITCH native file formats

**Impact: ADDITIVE.** Arguably the highest-value entry in this section for the
stated use case.

A PBX plays prompts from its own on-disk formats, chosen so no transcoding is
needed at call time. None are supported:

| Extension | Content | Notes |
|---|---|---|
| `.sln` | 16-bit signed linear, 8 kHz, mono, headerless | Asterisk's internal "slin". The `.sln12`, `.sln16`, `.sln24`, `.sln32`, `.sln44`, `.sln48` variants carry the rate in the name. |
| `.ulaw` / `.al` / `.alaw` / `.ul` | Raw G.711, 8 kHz | Zero-transcode playback. |
| `.gsm` | Raw GSM 06.10, 33-byte frames | The classic Asterisk voicemail format. |
| `.g722`, `.g729`, `.g726-16/24/32/40` | Raw codec frames | Named by bitrate for G.726. |
| `.siren7`, `.siren14` | G.722.1 / G.722.1C | Conferencing. |
| `.au` / `.snd` | Sun/NeXT, usually 8-bit µ-law at 8 kHz | Magic `.snd`. A telephony classic, still found in older IVR prompt sets and Java-era systems. Has a real header, unlike the raw formats. |

Note the split of responsibility. The **framing, headers and file conventions**
are this module's work: recognising the format, knowing its rate and channel
count, reading and writing frames of the right size, and honouring the naming
convention a PBX uses to pick a file. The **codecs inside** — GSM, G.729,
G.722, G.726 — come from dependencies as described in
[FMT-4](#fmt-4-telco-and-voip-codecs-absent-user-2) and
[the scope note](#scope-what-this-project-is-and-is-not). For `.sln`, `.ulaw`
and `.alaw` there is no codec to obtain at all beyond G.711 companding.

Beyond the codecs, this implies a **format convention layer**: the rate,
channel count and frame alignment each format requires, and the naming
convention that lets a PBX pick a file. Writing a file Asterisk will refuse to
play is an easy mistake — see
[PBX-2](#pbx-2-no-pbx-output-conventions-or-validation).

### FMT-14: Opus in Ogg is not the same as Vorbis in Ogg

**Impact: ADDITIVE.**

`formats/vorbis` decodes Vorbis in an Ogg container. A `.opus` file is **Opus in
an Ogg container** — same container, different codec, different headers
(`OpusHead` / `OpusTags`), and a mandatory pre-skip value that must be honoured
or the start of the audio is offset.

Worth calling out explicitly because the shared `.ogg`-family extension and
container invite the assumption that existing Ogg support covers it. It does
not, and a `.opus` file handed to the Vorbis decoder will simply fail.

Also in this family: **WebM** as an Opus container, which is what browser
`MediaRecorder` produces — the likely input format for anything WebRTC-adjacent.

### FMT-15: No large-file or extended containers

**Impact: ADDITIVE.**

Classic RIFF caps a file at 4 GB because sizes are `uint32`. For 8 kHz mono
that is about 74 hours, so it rarely bites for single calls — but it does for
multi-channel trunk logging or long-running capture. Unsupported:

- **RF64 / BW64** — the EBU and ITU 64-bit RIFF extension, using an `RF64`
  signature and a `ds64` chunk.
- **Wave64 (`.w64`)** — Sony's GUID-based 64-bit variant.
- **CAF** — Apple Core Audio Format, also 64-bit capable.

Related risk worth noting even without these formats: a truncated or
overflowing 32-bit size field in a hostile WAV is exactly the input class that
[QA-5](#qa-5-no-fuzzing-of-binary-parsers) exists to cover.

### FMT-16: In-file markers and regions are discarded

**Impact: ADDITIVE.**

Both supported container families can carry labelled time markers, and neither
is read:

- **WAV**: `cue` chunk with `labl`, `note` and `ltxt` for labelled points and
  regions; `smpl` for loop points.
- **AIFF**: `MARK` chunk for markers, `INST` for instrument/loop data.

This matters more than it sounds for prompt work. A single long recording of an
announcer reading many prompts, with markers placed at the boundaries, is a
normal studio deliverable. Reading the markers turns splitting that file into a
mechanical operation instead of a manual one — and pairs directly with
[DSP-4](#dsp-4-the-library-can-decompose-audio-but-not-compose-it)'s `Trim` and
[PBX-4](#pbx-4-no-automatic-segmentation).

Writing markers is equally useful: annotating a call recording with events
(answer, transfer, DTMF received) makes the output self-describing.

### FMT-17: MP3 container details

**Impact: ADDITIVE.**

Three details the current wrapper ignores:

- **Xing / VBRI / LAME headers.** For variable-bitrate MP3 these carry the true
  frame count, which is the only reliable way to get duration without decoding
  the whole file. Without them, VBR duration estimates are wrong — relevant to
  [API-1](#api-1-source-cannot-report-length-or-duration) and
  [OBS-1](#obs-1-no-progress-reporting-user-3).
- **MPEG-2 and MPEG-2.5 low sample rates** — 8, 11.025, 12, 16, 22.05 and
  24 kHz. These are exactly the rates telephony-adjacent MP3 uses, so they are
  not an edge case here. Whether `go-mp3` handles them should be confirmed
  rather than assumed.
- **ID3v1 / ID3v2 tags** — see
  [FMT-10](#fmt-10-no-metadata-or-tag-handling). A large ID3v2 block at the
  start of a file is also a parser robustness concern.

### FMT-18: Other file formats worth considering

**Impact: ADDITIVE.**

Lower priority, listed for completeness:

| Format | Rationale |
|---|---|
| `.amr` / `.3ga` | AMR storage format with its `#!AMR\n` magic. Arrives from mobile handsets and MMS. |
| `.m4a` / `.mp4` / ADTS / ADIF | AAC containers. [FMT-5](#fmt-5-aac-support-user-6) covers the codec; the container matters separately, and ADTS versus MP4 is a real fork in the road. |
| `.webm` | Opus from browser recording. See [FMT-14](#fmt-14-opus-in-ogg-is-not-the-same-as-vorbis-in-ogg). |
| `.wma` | Occasionally in older Windows-based IVR estates. |

### FMT-19: No cross-format metadata mapping or preservation policy

**Impact: ADDITIVE.**

Copying metadata between formats — the operation `ffmpeg -map_metadata`
performs, and which this repository's own test-asset script relies on — needs a
mapping table, because **the tag systems are not isomorphic**:

- ID3v2 defines on the order of eighty standard frames.
- Vorbis comments have no schema at all, only conventional field names.
- RIFF `INFO` has roughly twenty four-character codes.
- MP4 uses its own atom set.

There is no bijection. `TRACKNUMBER` in a Vorbis comment, `TRCK` in ID3v2 and
`ITRK` in RIFF are *approximately* the same field with different syntax; many
fields exist in only one system; embedded cover art exists in ID3v2, Vorbis
comments and MP4 but not in RIFF `INFO` or AIFF text chunks.

Consequences the design has to face:

- **A copy is lossy, and the caller should be told what was dropped.** Silent
  loss during a transcode is how metadata quietly disappears from an archive.
  This is a natural use for [OBS-3](#obs-3-no-way-to-surface-warnings).
- **A canonical intermediate representation is needed** — a normalised tag set
  that each format maps into and out of — rather than N×N per-format
  conversions.
- **Round-tripping should be stable.** Format A → B → A must not accumulate
  damage, and unrecognised fields should survive where the target can hold
  them.
- **Target formats with no metadata capability** (`.sln`, `.ulaw`, `.gsm`) need
  a defined behaviour: drop with a warning, or write a sidecar
  ([PBX-7](#pbx-7-no-call-or-prompt-annotation)).

This also interacts with [API-5](#api-5-no-encoder-abstraction): if metadata is
to survive a decode/resample/encode pipeline, the `Encoder` interface must be
able to accept it, which is a reason to design the two together.

### FMT-20: Tag blocks are a decode correctness and robustness hazard

**Impact: ADDITIVE,** but this one is a latent-bug entry rather than a feature
request.

Metadata is not only a missing feature — unparsed tags can break decoding:

- **ID3v2 sits before the audio** and can be large; embedded cover art routinely
  runs to megabytes. A decoder that scans for a frame sync without skipping the
  tag can mistake tag bytes for audio.
- **ID3v1 sits in the final 128 bytes.** A decoder that treats the whole file as
  audio can interpret the trailer as a corrupt final frame. This is a
  well-known MP3 pitfall.
- **ID3v2 unsynchronisation** deliberately alters byte patterns that resemble
  frame syncs; a parser unaware of it computes wrong offsets.
- **Sync-safe integer sizes** differ between ID3v2.3 and 2.4. Reading the size
  field with the wrong rule yields a wrong tag length and a wrong audio start.
- **Oversized or lying chunk sizes** in RIFF and AIFF tag chunks are exactly the
  input class [QA-5](#qa-5-no-fuzzing-of-binary-parsers) exists to cover, and
  metadata chunks are the easiest place to put a hostile length.

Whether the current decoders are affected in practice depends on the upstream
libraries rather than on this module's code, so this is recorded as a
verification task as much as an implementation one: confirm behaviour with a
large ID3v2 block, an ID3v1 trailer, and an unsynchronised tag before assuming
it is handled.

### FMT-21: What owning the WAV and AIFF container layer requires

**Impact: ADDITIVE.** This entry exists because
[DEP-1](#dep-1-five-of-seven-dependencies-are-archived) concludes that the
container layer should be owned rather than depended on, and that conclusion is
only actionable with a concrete list of what "owned" means.

**Priority: this is now the first work item**, ahead of other features, because
it is the fix for
[RES-1](#res-1-a-62-byte-file-can-force-gigabytes-of-allocation) in both the WAV
and AIFF paths. The **shared, size-validating chunk walker** is the slice that
closes the security defect; the per-format work follows it. See
[roadmap_v1.md](roadmap_v1.md#d8--archived-dependencies-with-known-security-defects-are-removed-not-mitigated).

Both formats are chunked containers of the same shape, differing in magic and
endianness, so a **single chunk-walker abstraction serves both**. Neither has
changed in decades, which is what makes this bounded work rather than an
open-ended commitment.

#### Shared: the chunk walker

- 4CC identifier, then a 32-bit size, then payload
- **Word alignment**: a chunk with an odd size is followed by a pad byte that is
  **not counted in the declared size**. Getting this wrong desynchronises every
  subsequent chunk. Already covered by an existing test —
  `wav.TestDecoder_OddSizedChunkPadding`.
- Unknown chunks must be skipped, not rejected
- **Every declared size must be validated against the actual remaining input.**
  This is the lesson of
  [RES-1](#res-1-a-62-byte-file-can-force-gigabytes-of-allocation): a size field
  read from the file and used for an allocation, unchecked, is a
  memory-amplification DoS. Validating by construction is a large part of the
  reason to own this layer.
- Endianness is the only structural difference: RIFF is little-endian, AIFF
  big-endian

#### WAV / RIFF specifics

| Element | Requirement |
|---|---|
| Container | `RIFF` + size + `WAVE`; also `RF64`/`BW64` with a `ds64` chunk for >4 GB ([FMT-15](#fmt-15-no-large-file-or-extended-containers)) |
| `fmt ` chunk | `wFormatTag`, `nChannels`, `nSamplesPerSec`, `nAvgBytesPerSec`, `nBlockAlign`, `wBitsPerSample`; plus `cbSize`, `wValidBitsPerSample`, `dwChannelMask` and a SubFormat GUID when extensible |
| Format tags | 1 PCM, 3 IEEE float, 6 A-law, 7 µ-law, 0x11 IMA ADPCM, 0x31 GSM, 0x55 MP3, 0xFFFE extensible ([FMT-2](#fmt-2-wav-decoding-is-narrow-user-5), [FMT-11](#fmt-11-non-pcm-payloads-inside-the-wav-container)) |
| `dwChannelMask` | This is where the `WAVEFORMATEXTENSIBLE` channel mask lives — the thing `audio.Channel` already models and [API-7](#api-7-source-exposes-no-format-metadata) cannot currently attach to a `Source` |
| `fact` chunk | Sample count; **required** for non-PCM payloads, and the only reliable length source for them ([API-1](#api-1-source-cannot-report-length-or-duration)) |
| `data` chunk | Sample payload; note the streaming convention of a placeholder size when the length is not yet known |
| Metadata chunks | `LIST`/`INFO`, `bext` (BWF), `iXML`, `id3 ` ([FMT-10](#fmt-10-no-metadata-or-tag-handling), [PBX-7](#pbx-7-no-call-or-prompt-annotation)) |
| Marker chunks | `cue` with `labl`/`note`/`ltxt`, and `smpl` ([FMT-16](#fmt-16-in-file-markers-and-regions-are-discarded)) |
| Write side | Sizes are patched after the payload is written, so a seekable destination is the easy path and a placeholder-size streaming mode is the harder one ([PBX-5](#pbx-5-no-streaming-io-boundary)) |

#### AIFF specifics

| Element | Requirement |
|---|---|
| Container | `FORM` + size + `AIFF`, or `AIFC` for AIFF-C |
| `COMM` chunk | `numChannels` (int16), `numSampleFrames` (uint32), `sampleSize` (int16), `sampleRate` as an **80-bit IEEE 754 extended float**; plus `compressionType` (4CC) and a Pascal-string `compressionName` for AIFF-C |
| 80-bit extended float | Sign, 15-bit exponent, 64-bit mantissa **with an explicit integer bit**. An encoder for this already exists in-tree as `aiff.encodeFloat80`; the decoder is its inverse. |
| `SSND` chunk | `offset` (uint32), `blockSize` (uint32), then samples. **The `offset` field is the RES-1 attack vector** — it must be validated against remaining input before it drives anything. |
| AIFF-C types | `NONE`, **`sowt`** (little-endian PCM — the common Apple case), `fl32`, `fl64`, `ulaw`/`ULAW`, `alaw`/`ALAW`, `ima4`, `MAC3`/`MAC6` ([FMT-12](#fmt-12-aiff-c-compression-types)) |
| Text and marker chunks | `NAME`, `AUTH`, `(c) `, `ANNO`, `COMT`, `MARK`, `INST` |

#### Groundwork already in the tree

A useful amount of this is already written, mostly as test helpers, which
lowers the starting cost and doubles as a partial specification:

| Existing | Where | Reusable for |
|---|---|---|
| `encodeFloat80` | `formats/aiff/raw_decode_test.go` | The `COMM` sample-rate field; the decoder is the inverse |
| `createAIFFFile` | `formats/aiff/raw_decode_test.go` | A known-good `FORM`/`COMM`/`SSND` writer to test a reader against |
| `createWAVFile` | `formats/wav/decoder_test.go` | The same for RIFF |
| Direct int16 sample decode | `formats/{wav,aiff}/decoder.go` | Already owned — little-endian and big-endian paths both exist |
| `WriteWAV16` | `formats/wav/pcm_16_writer.go` | Already lays out a complete 44-byte RIFF header |
| Odd-size padding test | `formats/wav/decoder_test.go` | Chunk-walker conformance |

#### What this unblocks

Owning the chunk layer is a prerequisite for, or directly delivers, a cluster of
entries that are currently limited by what the archived dependency chose to
expose: [FMT-2](#fmt-2-wav-decoding-is-narrow-user-5),
[FMT-10](#fmt-10-no-metadata-or-tag-handling),
[FMT-11](#fmt-11-non-pcm-payloads-inside-the-wav-container),
[FMT-12](#fmt-12-aiff-c-compression-types),
[FMT-15](#fmt-15-no-large-file-or-extended-containers),
[FMT-16](#fmt-16-in-file-markers-and-regions-are-discarded),
[API-6](#api-6-writewav16-is-permanently-mono),
[API-7](#api-7-source-exposes-no-format-metadata),
[PBX-5](#pbx-5-no-streaming-io-boundary),
[PBX-7](#pbx-7-no-call-or-prompt-annotation) and
[RES-1](#res-1-a-62-byte-file-can-force-gigabytes-of-allocation).

It also removes four archived dependencies at once and deletes the `[]int`
intermediate that the current fallback path still carries.

**Fuzz it.** These are binary parsers reading sizes and offsets from untrusted
input, which is exactly what [QA-5](#qa-5-no-fuzzing-of-binary-parsers) exists
for, and exactly the class of defect that produced RES-1 in the first place.

---

## C. Resampling, DSP and composition

### DSP-1: Anti-aliasing filter is structurally inadequate

**Impact: BEHAVIOURAL.**

The filter applied when downsampling is a one-pole low-pass with a fixed
coefficient that does not track the resampling ratio:

```go
const defaultFilterAlpha float32 = 0.5
```

Measured behaviour, 44.1 kHz to 8 kHz (output Nyquist 4 kHz), against `ffmpeg`:

| Input tone | audpbx output | ffmpeg output |
|---|---|---|
| 3500 Hz | 3500 Hz, −25.8 dB | 3500 Hz, −24.5 dB |
| 5000 Hz | **3000 Hz**, −27.1 dB | silence |
| 6000 Hz | **2000 Hz**, −27.9 dB | silence |
| 7000 Hz | **1000 Hz**, −28.7 dB | silence |
| 9000 Hz | **1000 Hz**, −30.2 dB | silence |

Out-of-band tones survive at 3–6 dB below passband level, folded to the exact
predicted alias frequencies. Measurements match one-pole theory within 0.2 dB,
confirming the filter is wholly responsible.

The key point is that this is **not a tuning problem**: a single pole rolls off
at 6 dB/octave, reaching only −9.5 dB at 22 kHz, so roughly 10 dB of rejection
is its ceiling. Sixteen-bit audio wants 80–90 dB. There is also 1.7 dB of
passband droop at 3.5 kHz, because a filter this gentle cannot be flat below
cutoff and steep above it.

Resolution is [DSP-2](#dsp-2-no-resampler-algorithm-selection-user-4): a
polyphase FIR does band-limiting and rate conversion in one operation and
replaces both the cubic interpolator and this filter.

Because fixing it changes output samples, it needs a stated policy on what
callers may rely on — see
[Decisions required before v1](#decisions-required-before-v1).

### DSP-2: No resampler algorithm selection **[user 4]**

**Impact: ADDITIVE if designed now.**

```go
func NewResampler(src Source, dstRate int) *Resampler
```

The algorithm is hardcoded to cubic Catmull-Rom. The intent is to offer linear,
cubic and polyphase FIR.

Adding a variadic option (`NewResampler(src, rate, opts ...ResamplerOption)`) is
source-compatible with the current signature, so this is lower-risk than most
items here. The real pre-v1 question is the **default**: if polyphase should
eventually become the default, changing it mid-v1 is a behavioural break, so the
policy belongs in v1.

Design notes are recorded in
[roadmap_v1.md](roadmap_v1.md#dsp-2-design-notes-moved-from-the-readme).
Two points worth repeating:
polyphase replaces rather than layers on the cubic path, and common telephony
ratios are exact rationals (44.1 kHz to 8 kHz is 441/80).

### DSP-3: Resampler block size is a compile-time constant

**Impact: ADDITIVE.**

```go
const resamplerBlockFrames = 8192
```

This constant is load-bearing for throughput — reducing it to 1 costs roughly
41× — but it also fixes startup buffering. Measured with 20 ms packet-sized
reads at 8 kHz, the resampler consumed **8194 frames (52 packets, 1024 ms)**
before producing its first output sample.

For file work that is invisible. For anything latency-sensitive it is
disqualifying, and it cannot currently be changed without editing the library.
Making it a per-instance option, alongside
[DSP-2](#dsp-2-no-resampler-algorithm-selection-user-4), would let callers trade
throughput against latency explicitly.

### DSP-4: The library can decompose audio but not compose it

**Impact: ADDITIVE.** This is a structural gap rather than a missing feature,
and it is broader than it first appears.

Every operation in the library runs in one direction — taking one stream apart:

| Direction | Exists |
|---|---|
| Split multi-channel into mono | `SplitToMonoSources`, `SplitChannels`, `ExtractChannels` |
| Collapse multi-channel to mono | `MonoMixer` |
| Combine anything into anything | **nothing** |

Verified: **no exported function in the module accepts more than one `Source`.**
`MonoMixer` takes a single source and collapses *its* channels; it cannot mix
two sources together. The following are all absent:

| Primitive | Purpose | Blocked without it |
|---|---|---|
| **`Concat` / `Sequence`** | play sources back to back | Asterisk-style prompt assembly (`digits/1` + `digits/2` + `"shekels"`) |
| **`Mix`** | sum N sources into one | announcement over hold music; folding two call directions into one mono track |
| **`Trim` / `Slice`** | take a time range | extracting a clip from a longer recording |
| **`Crossfade`** | blend across a join | every concatenation clicks audibly without it |
| **`Fade` in/out, gain envelope** | shape amplitude over time | clean starts and ends on assembled prompts |
| **`Silence`** | generate silence of a given duration | inter-prompt pauses; gap filling for [VOIP-1](#voip-1-to-voip-4) |
| **`Repeat` / `Pad`** | loop or extend to a length | padding channels to equal length when interleaving |
| **Interleave** | N mono sources to one multi-channel | two-direction stereo recording |

Two consequences worth stating separately:

1. **Prompt concatenation is the single most common PBX audio operation** and the
   library cannot do it. That is arguably a larger practical gap than several of
   the missing codecs.
2. **The `Source` decorator pattern is already established** by `Resampler` and
   `MonoMixer`, so these fit the existing design cleanly — each is a `Source`
   wrapping one or more others. The work is mostly mechanical; what is missing
   is that nobody has written it.

`Trim` and any multi-pass use of these primitives are much cheaper with
[API-18](#api-18-source-cannot-seek), so the two are worth designing together.

Note that once functions accept `[]Source` or `...Source`, the rules for
mismatched inputs become part of the API: differing sample rates (resample, or
reject?), differing channel counts, and differing lengths (pad, truncate, or
reject?). Those choices are easier to make now than to change later.

### DSP-5: No dithering

**Impact: ADDITIVE.**

`utils.Float32ToInt16` truncates after scaling. No dither and no noise shaping
is offered, so quantisation error is correlated with the signal — audible as
distortion rather than noise on quiet passages, and most noticeable exactly
where telephony lives: at low bit depths and after gain reduction.

Truncation is also a slightly odd choice versus rounding, since it biases every
sample toward zero by up to one LSB. Worth confirming that is intentional.

### DSP-6: Downmix is unweighted averaging only

**Impact: ADDITIVE.**

`MonoMixer` averages all channels equally:

```
5.1 → mono:  (FL + FR + FC + LFE + BL + BR) / 6
```

This is not how surround downmixing is normally done. Standard practice
attenuates surrounds, handles centre at about −3 dB, and usually discards or
heavily attenuates LFE — a non-directional low-frequency channel whose equal
inclusion here muddies the result. There is no stereo downmix at all
(5.1 → 2.0), only all-the-way-to-mono, and no way to supply custom coefficients.

Averaging is at least clip-safe, which is worth preserving as the default.

### DSP-7: Effects are not composable

**Impact: ADDITIVE.**

Gain and normalization exist only as `ChannelOption`s inside
`audpbx.ProcessChannels`. They are not available as `Source` wrappers, so a
caller building a custom pipeline with `NewResampler` and `NewMonoMixer` — the
documented low-level path — cannot apply gain without writing it themselves.

This is the same structural point as
[DSP-4](#dsp-4-the-library-can-decompose-audio-but-not-compose-it) from the
other side: **the high-level API has capabilities the low-level API cannot
reach**, which inverts the usual relationship. Gain, normalization, DC-offset
removal and silence trimming should all exist as `Source` decorators, with
`ProcessChannels` composing them rather than owning them.

### DSP-8: Normalization buffers the entire stream

**Impact: ADDITIVE.**

Normalization needs a peak, so it reads everything before emitting anything:

```go
buf := make([]float32, 0, bufSize*100)   // grows to hold the whole stream
```

Memory is therefore proportional to input duration — for the 16-minute, 167 MB
test asset, roughly 340 MB of `float32`. The doc comment does note that
normalization reads all audio into memory, so this is disclosed rather than
hidden.

Two-pass normalization over a seekable source, or a streaming limiter as an
alternative, would remove the ceiling. Two-pass requires
[API-18](#api-18-source-cannot-seek).

### DSP-9: No time-stretching or pitch modification

**Impact: ADDITIVE.**

There is no way to change duration without changing pitch, or pitch without
changing duration. The resampler couples them inseparably: resampling to a
different rate and then playing back at the original rate alters both together.

Independent control requires a time-domain or frequency-domain technique —
overlap-add methods such as WSOLA, or pitch-synchronous overlap-add — and, for
the pitch-synchronous variants, pitch-mark detection
([DSP-10](#dsp-10-no-analysis-primitives)).

Lower priority than the rest of this section, but recorded because it is a
capability the current architecture has no place for, and because it is
substantial DSP work rather than plumbing.

### DSP-10: No analysis primitives

**Impact: ADDITIVE.**

The library transforms audio but cannot measure it. Nothing reports:

- **Peak and RMS level**, per channel or over a window. Needed for any
  level-matching, and for [OBS-2](#obs-2-no-processing-statistics).
- **Silence detection** — where speech starts and ends, for trimming or
  segmentation.
- **Fundamental frequency (F0)** — classical estimators (autocorrelation, YIN,
  cepstral) are well documented and modest to implement.
- **Clipping detection** — currently `Float32ToInt16` clamps silently, so
  clipping leaves no trace at all.

Analysis is a prerequisite for several other entries here rather than an end in
itself: normalization already computes a peak internally but does not expose it,
and silence trimming is impossible without detection.

---

## D. Observability: progress, timing, cancellation

### OBS-1: No progress reporting **[user 3]**

**Impact: ADDITIVE, but blocked on [API-1](#api-1-source-cannot-report-length-or-duration).**

#### `WithProgress` exists and does nothing

`ProcessChannels` accepts
`WithProgress(callback func(ch audio.Channel, progress float64))`, and its
godoc states it is *"called periodically with channel and progress (0.0 to
1.0)"*.

**It is never called.** The callback is stored in `channelProcessor.progressCallback`
and there is no invocation site anywhere in the module. No test exercises it.
Meanwhile [README.markdown](README.markdown) lists it among the available
processing options.

So this is not only a missing feature but a **silent no-op in the public API**,
documented as working. A caller wiring a progress bar to it sees nothing and has
no indication why. That makes it the highest-priority item in this section
regardless of what else is built: either implement it or mark it clearly as not
yet functional.

Nothing equivalent exists at all for `ResampleToMono16` or for a hand-built
pipeline.

#### The structural blocker

**Progress as a fraction requires knowing the total**, and `Source` cannot
report length ([API-1](#api-1-source-cannot-report-length-or-duration)) or
current position ([API-20](#api-20-source-cannot-report-position)). Since the
underlying decoders can do both, the information is discarded at the wrapper
boundary.

#### Four separable capabilities

These are routinely conflated, and they have very different costs:

1. **Progress** — fraction complete. Needs total length and position.
2. **Elapsed and estimated time** — wall-clock plus a rate estimate. Derives
   from (1) with no further API support.
3. **Throughput** — samples or realtime-multiple per second. Already computed ad
   hoc by `examples/profile_resampler`, which reports "1637.33x realtime"; that
   logic could be promoted.
4. **Resume from a stopping point** — a different problem entirely, and much
   harder. See [OBS-4](#obs-4-no-task-abstraction),
   [OBS-5](#obs-5-processing-state-cannot-be-checkpointed-or-restored) and
   [OBS-6](#obs-6-no-partial-output-or-resume-point-persistence).

Settle (1) as part of the
[API-1](#api-1-source-cannot-report-length-or-duration) decision and treat (2)
and (3) as additive helpers afterwards. Note that (2) interacts with
[API-2](#api-2-no-cancellation-mechanism): callers who can observe progress
usually want to be able to abort.

#### Two design points worth deciding early

- **Progress should be input-driven.** Fraction of *input consumed*, not output
  produced. Output length is not knowable in advance for variable-rate formats,
  and for a resampling pipeline it is only knowable if the input length is —
  so input-driven is the only measure that generalises.
- **Delivery needs rate limiting.** A callback fired per buffer runs roughly
  2,200 times for a 103-second file at 4096 samples, and around 140,000 times at
  64 samples. A UI cannot absorb that. Either the library rate-limits (say, on
  a minimum interval or a minimum delta), or progress is exposed as state the
  caller polls rather than an event stream. The choice is API surface — see
  [OBS-4](#obs-4-no-task-abstraction) and
  [the v1 decisions](#decisions-required-before-v1).

### OBS-2: No processing statistics

**Impact: ADDITIVE.**

Nothing is reported about what happened during a conversion. Values a telephony
operator would reasonably want:

- **Samples clipped** during gain or format conversion.
  `utils.Float32ToInt16` clamps silently, so audible clipping leaves no trace.
- Peak and RMS level of input and output
  ([DSP-10](#dsp-10-no-analysis-primitives)).
- Detected silence, and its duration.
- For packet input: gaps, concealed frames, duplicates
  ([VOIP-1](#voip-1-to-voip-4)).

Silent clipping is the notable one: a conversion can degrade audibly and still
report success.

### OBS-3: No way to surface warnings

**Impact: ADDITIVE.**

The API is binary — an operation either returns an error or succeeds. There is
no channel for non-fatal observations: "input was clipped", "metadata
discarded", "channel layout guessed from count", "aliasing likely at this
ratio". `ProcessingResult` does carry per-channel errors, which is the closest
existing mechanism and could be generalised.

There is no logging abstraction either, which is arguably correct for a library
— returning structured warnings is usually better than logging from library code
— but the decision should be deliberate rather than incidental.

### OBS-4: No task abstraction

**Impact: BREAKING** if progress or lifecycle is later attached to existing
signatures.

Every operation in the module is a plain function call that runs to completion
and returns:

```go
pcm16, rate, err := audpbx.ResampleToMono16(src, 8000, 4096)
```

There is no handle, no object, nothing representing "a conversion in progress".
That means there is nowhere to hang any of:

- current progress, elapsed time or estimated remaining time
- a cancellation signal ([API-2](#api-2-no-cancellation-mechanism))
- accumulated statistics ([OBS-2](#obs-2-no-processing-statistics)) or warnings
  ([OBS-3](#obs-3-no-way-to-surface-warnings))
- a resume point ([OBS-6](#obs-6-no-partial-output-or-resume-point-persistence))

A progress bar needs *something to ask*. Today the only mechanism available is
a callback passed in up front, which is why
[OBS-1](#obs-1-no-progress-reporting-user-3)'s `WithProgress` is shaped the way
it is — and callbacks make rate limiting, cancellation and post-hoc inspection
all awkward.

The decision is whether long-running operations return a handle whose state can
be polled, or stay as blocking calls with callbacks. It is listed as BREAKING
because retrofitting a handle changes signatures; adding it now is free.

### OBS-5: Processing state cannot be checkpointed or restored

**Impact: ADDITIVE** in mechanism, but it is the substantive blocker for
resuming mid-stream.

To continue a conversion from where it stopped, three things must be
restorable. Only the first is even approachable today:

| | Restorable now? |
|---|---|
| Input position | No — [API-18](#api-18-source-cannot-seek) and [API-20](#api-20-source-cannot-report-position) are both missing, though the underlying decoders can seek |
| Output position | Only for headerless formats; see [OBS-6](#obs-6-no-partial-output-or-resume-point-persistence) |
| **Internal DSP state** | **No, and nothing exposes it** |

The third is the hard one. `Resampler` carries a sliding input window,
one-pole filter state, and three counters (`inBase`, `srcFramesRead`,
`outFrame`) that determine output phase. Every field is unexported, and there
is no `MarshalBinary`, snapshot, or equivalent anywhere in the module — verified
by search. `MonoMixer` holds a scratch buffer; any future filter
([FILT-1](#filt-1-no-filter-framework)) will hold its own state.

Restarting a stream without restoring that state produces a discontinuity at
the join — a click from the filter state resetting, and potentially a
sample-count drift from the resampler's phase counter restarting.

There are three tiers of answer, and they differ enormously in cost:

**Tier 1 — resume at file granularity.** For batch work, record which files
completed and skip them. No state to restore at all. Given that a 103-second
file converts in about 59 ms, this covers almost all of the realistic value,
and it needs nothing from this entry — only
[PBX-6](#pbx-6-no-multi-file-batch-operations) and a completeness marker
([OBS-6](#obs-6-no-partial-output-or-resume-point-persistence)).

**Tier 2 — resume mid-file using pre-roll re-priming.** Seek the input to
*before* the resume point, process a short pre-roll and discard its output,
then continue writing. Filter and resampler state re-converge naturally, so
nothing needs serialising. This is the standard technique and the recommended
route: it needs [API-18](#api-18-source-cannot-seek),
[API-20](#api-20-source-cannot-report-position) and appendable output, all of
which are wanted anyway. The cost is a small amount of repeated work and
possible sub-sample phase difference at the join.

**Tier 3 — serialise the internal state.** Exact, and a versioning trap: the
checkpoint format would encode implementation details like
`resamplerBlockFrames` and the filter topology, so any change to the resampler
— including the planned algorithm selection
([DSP-2](#dsp-2-no-resampler-algorithm-selection-user-4)) — invalidates every
stored checkpoint. Not recommended unless Tier 2 proves insufficient.

The practical recommendation is Tier 1 now, Tier 2 when a real need appears,
and Tier 3 probably never.

### OBS-6: No partial-output or resume-point persistence

**Impact: ADDITIVE.**

Even with input-side resume solved, the output side has no answer:

- **Nothing distinguishes a partial output file from a complete one.** A process
  killed mid-write leaves a file that looks plausible. The conventional fix is
  to write to a temporary name and rename atomically on success, which also
  makes Tier 1 resume trivial — a file either exists and is complete, or does
  not exist. That convention is not established anywhere in the module or its
  examples.
- **WAV output cannot be appended to.** `wav.WriteWAV16` computes RIFF and data
  chunk sizes from a complete `[]int16` up front
  ([API-6](#api-6-writewav16-is-permanently-mono),
  [FMT-11](#fmt-11-non-pcm-payloads-inside-the-wav-container)). Resuming into an
  existing WAV means reopening it, appending samples, then repairing the header
  sizes — none of which the writer supports.
- **Headerless formats are trivially resumable**, which is worth knowing:
  `.sln`, `.ulaw`, `.alaw` and `.gsm`
  ([FMT-13](#fmt-13-asterisk-and-freeswitch-native-file-formats)) have no header
  to repair, so resume is seek-to-end-and-continue. If intra-file resume matters
  for a workload, these formats make it dramatically cheaper.
- **There is nowhere to record the resume point.** A sidecar checkpoint file, or
  deriving it from the output length, are the options; neither exists and the
  choice should be deliberate.

---

## E. VoIP and packet-based audio

Full design analysis is in **[from_voip.md](from_voip.md)**. Summarised here so
the gap index is complete.

### VOIP-1 to VOIP-4

**Impact: ADDITIVE.**

| ID | Gap |
|---|---|
| VOIP-1 | No timeline reconstruction: no way to place timestamped payloads on a timeline, fill gaps, or deduplicate and reorder packets |
| VOIP-2 | No cross-direction alignment: two RTP streams have independent random starting timestamps, so aligning call legs needs wall-clock arrival times or RTCP Sender Reports |
| VOIP-3 | No frame-oriented decode path — see [API-17](#api-17-decoder-cannot-express-framed-codecs) |
| VOIP-4 | No raw companded writers (`.ulaw`, `.alaw`) — see [FMT-6](#fmt-6-no-headerless-raw-formats) |

Three items already in this document are hard prerequisites:
[API-6](#api-6-writewav16-is-permanently-mono) (mono-only WAV writer),
[DSP-4](#dsp-4-the-library-can-decompose-audio-but-not-compose-it) (no
interleaver or silence generator), and
[API-17](#api-17-decoder-cannot-express-framed-codecs). Without them the
headline two-direction stereo recording output is impossible.

---

## F. Testing, CI and hygiene

### QA-1: The test suite fails out of the box

**Impact: INTERNAL,** but it is the first command a contributor runs.

```
ok      github.com/ik5/audpbx
ok      github.com/ik5/audpbx/audio
FAIL    github.com/ik5/audpbx/formats/aiff
FAIL    github.com/ik5/audpbx/formats/mp3
FAIL    github.com/ik5/audpbx/formats/vorbis
ok      github.com/ik5/audpbx/formats/wav
FAIL
```

Cause: `example_test.go` in each of those three packages calls `log.Fatal` when
`testdata/sample.{mp3,aiff,ogg}` is absent, and those fixtures are not in the
repository. The unit tests themselves pass — `go test -run Test ./...` is green
— so the failure is entirely in the examples.

Fix options: ship small fixtures, have the examples skip gracefully, or point
them at `examples/testdata`. Any of the three is preferable to a red suite,
which trains contributors to ignore failures.

### QA-2: No CI

**Impact: INTERNAL.**

No `.github/workflows`, and no other CI configuration anywhere. Everything in
this section is consequently unenforced, and
[QA-1](#qa-1-the-test-suite-fails-out-of-the-box) would have been caught immediately
by any CI at all.

Minimum worth having before v1: `go build`, `go vet`, `go test -race`, and a
`gofmt` check, across the supported Go versions, for every module in the
workspace (see
[QA-6](#qa-6-example-modules-are-outside-the-root-test-run)). If
[FMT-5](#fmt-5-aac-support-user-6) lands with build tags, the matrix must cover
those variants too.

### QA-3: No lint gate, and ten unformatted files

**Impact: INTERNAL.**

`gofmt -l` currently reports ten files — five non-test, five test:

```
audpbx.go            audio/audio.go        audio/channels.go
audio/mono_mixer.go  formats/wav/errors.go
audio/example_test.go   audio/audio_test.go
formats/wav/errors_test.go   formats/wav/decoder_test.go
formats/vorbis/decoder_test.go
```

`audpbx.go` is the largest public API surface in the module. There is also no
`golangci-lint` configuration, so nothing checks for the issues noted elsewhere
here — copying a struct containing a mutex
([API-10](#api-10-channelsource-value-and-pointer-asymmetry)) is a standard
linter finding.

### QA-4: Root package coverage is 63%

**Impact: INTERNAL.**

| Package | Coverage |
|---|---|
| `utils` | 100.0% |
| `audio` | 94.1% |
| `formats/wav` | 93.9% |
| **`audpbx` (root)** | **63.1%** |
| `formats/mp3`, `aiff`, `vorbis` | not measurable — see [QA-1](#qa-1-the-test-suite-fails-out-of-the-box) |

The weakest coverage is on the largest public API: `ProcessChannels` plus
thirteen options, normalization, mixdown and the concurrency paths. That inverts
the desirable distribution, and three packages cannot be measured at all.

### QA-5: No fuzzing of binary parsers

**Impact: INTERNAL.**

No `func Fuzz*` anywhere. The module parses four untrusted binary container
formats, with chunk sizes and offsets read from the input and used for
allocation and slicing. That is textbook fuzzing territory, native fuzzing has
shipped in Go since 1.18, and a crash in a PBX media path is a denial of
service.

This is not hypothetical here. Two concrete defects of exactly this class have
already been found by hand: a slice-bounds panic in the Vorbis wrapper that only
appeared at realistic buffer sizes, and
[RES-1](#res-1-a-62-byte-file-can-force-gigabytes-of-allocation) — a 62-byte
AIFF that drives a ~191 MB allocation and returns no error. Both would have been
found immediately by a fuzzer, and neither was found by the existing tests.

That makes fuzzing the highest-value item in this section rather than a hygiene
nicety, and it is the centrepiece of the pre-release audit in
[QA-8](#qa-8-no-pre-release-bug-and-security-audit) — with a corpus persisted
between releases so coverage accumulates.

### QA-6: Example modules are outside the root test run

**Impact: INTERNAL.**

`go.work` includes `./examples/resampler` and `./examples/profile_resampler` as
separate modules. A root `go test ./...` therefore never builds or tests them,
so they can break silently — and they are the code most likely to be copied by
new users. CI should iterate the workspace modules explicitly.

### QA-7: No codec reference vectors

**Impact: INTERNAL.**

Correctness is currently established by round-tripping through the library's own
code. There are no external reference vectors, so a systematic error — a wrong
companding table, an inverted byte order, a scaling factor off by a factor of
two — would round-trip perfectly and pass.

This becomes important as soon as
[FMT-4](#fmt-4-telco-and-voip-codecs-absent-user-2) starts. Note what is being
validated: not a codec implementation — that belongs to the dependency — but
**this module's wrapper around it**, which is where byte order, frame
alignment, bit packing and scaling errors actually occur. G.711 has fully
enumerable companding tables (all 256 values, both directions), and published
ITU-T vectors exist for several codecs, making wrapper conformance cheap to
check. The precedent for why this matters already exists
in this repository: an AIFF decode path passed its entire test suite with the
byte order inverted, because the tests only exercised a mock rather than real
bytes.

### QA-8: No pre-release bug and security audit

**Impact: INTERNAL,** but it is the process that would have caught several
entries in this document before they reached a tag.

Three tags exist (`v0.0.1`–`v0.0.3`) with no recorded audit, no CHANGELOG
([DOC-2](#doc-1-to-doc-4)), and no CI ([QA-2](#qa-2-no-ci)). Nothing currently
runs before a release that would surface a regression, a new vulnerability, or a
dependency that has gone unmaintained.

The evidence that this matters is in this document: a memory-amplification
defect reachable from a 62-byte file
([RES-1](#res-1-a-62-byte-file-can-force-gigabytes-of-allocation)), five of
seven dependencies archived and unnoticed
([DEP-1](#dep-1-five-of-seven-dependencies-are-archived)), a public API option
that silently does nothing ([OBS-1](#obs-1-no-progress-reporting-user-3)), and
a decode path that passed its whole test suite with the byte order inverted
([QA-7](#qa-7-no-codec-reference-vectors)). Every one of those was found by
looking, not by any automated gate.

#### What the audit should cover

**Security and robustness**

| Check | Tool or method |
|---|---|
| Fuzz every binary parser | `go test -fuzz` (native since Go 1.18), with a **corpus persisted between releases** so coverage accumulates rather than restarting — see [QA-5](#qa-5-no-fuzzing-of-binary-parsers) |
| Known vulnerabilities | `govulncheck` against the Go vulnerability database |
| Resource limits | Deliberate oversized declared sizes, truncated files, and unbounded-read cases — the [RES-1](#res-1-a-62-byte-file-can-force-gigabytes-of-allocation) / [RES-2](#res-2-untrusted-input-is-read-into-memory-without-bound) / [RES-3](#res-3-channel-splitting-buffers-unread-channels-without-bound) class |
| Race conditions | `go test -race ./...` across all workspace modules |
| Static analysis | `go vet` plus a linter; see [QA-3](#qa-3-no-lint-gate-and-ten-unformatted-files) |

**Correctness**

| Check | Method |
|---|---|
| Output regression | Golden-file comparison against the previous release. This technique already proved itself during the resampler work — bit-identical output across nine real files was what made a 118x rewrite safe to land |
| Codec conformance | Reference vectors where they exist, validating **this module's wrappers** rather than the codecs — see [QA-7](#qa-7-no-codec-reference-vectors) |
| Test suite actually passes | Currently it does not on a bare `go test ./...` — see [QA-1](#qa-1-the-test-suite-fails-out-of-the-box) |
| Coverage did not regress | Per-package thresholds; see [QA-4](#qa-4-root-package-coverage-is-63) |

**Dependencies**

| Check | Why |
|---|---|
| Archival and last-release status of every dependency | This is exactly what [DEP-1](#dep-1-five-of-seven-dependencies-are-archived) went unnoticed for; it is a cheap check that has to be periodic because status changes silently |
| Licence inventory unchanged | Per [DEP-2](#dep-2-no-dependency-policy); a transitive dependency changing licence is not otherwise visible |
| `go mod tidy` produces no diff | Catches drift between declared and actual dependencies |

#### The check that matters most given the v1 premise

**Automated API-diff against the previous release.** This document's entire
premise is that v1 freezes the API, and that promise is unenforceable by good
intentions — `golang.org/x/exp/cmd/gorelease` reports whether a change is
backward-compatible and what version number it implies, built on
`golang.org/x/exp/apidiff`.

Running it as a release gate turns "we intend not to break the API" into
something mechanical. Without it, an accidental breaking change in v1.3 is found
by a user, not by the project.

#### Making it real

An audit that lives in a person's memory does not happen. For it to hold it
needs to be a **committed checklist plus CI jobs**, with the release-only steps
(golden comparison against the previous tag, API diff, dependency status) run on
tag and the rest on every push. Continuous fuzzing is worth considering
separately, since fuzzing finds things in hours that it will not find in
minutes.

This is a v1 deliverable in its own right, alongside the stability policy in
[DOC-1](#doc-1-to-doc-4).

---

## G. Documentation

### DOC-1 to DOC-4

**Impact: INTERNAL.**

| ID | Gap |
|---|---|
| DOC-1 | *(resolved — [D12](roadmap_v1.md#d12--api-stability-policy-go-version-range-deprecation))* **No API stability policy.** Nothing states what pre-v1 guarantees exist, what v1 will freeze, or how deprecation will work. Given this document's premise, that policy is a v1 deliverable in its own right. |
| DOC-2 | *(resolved — [D13](roadmap_v1.md#d13--changelog-and-contributing))* **No CHANGELOG.** Tags `v0.0.1`–`v0.0.3` exist with no record of what changed between them. |
| DOC-3 | *(resolved — [D13](roadmap_v1.md#d13--changelog-and-contributing))* **No CONTRIBUTING.** No statement of Go version support, formatting expectations, how to obtain test assets (which matters given [QA-1](#qa-1-the-test-suite-fails-out-of-the-box)), or review process. The README has a short contribution section asking contributors to ensure `go test ./...` passes — which it currently cannot. |
| DOC-4 | *(unresolved — see below)* **Documentation describes behaviour that does not exist.** Two instances. `WithProgress`'s godoc says it is *"called periodically with channel and progress (0.0 to 1.0)"* and the README listed it as an available option, but it is never invoked ([OBS-1](#obs-1-no-progress-reporting-user-3)). `WithBufferSize`'s godoc says it *"returns error if too small or too large"*, but it returns a `ChannelOption`; validation happens later inside `ProcessChannels` ([API-4](#api-4-channeloption-is-a-closed-extension-point)). Separately, "Comprehensive Testing" and "Comprehensive unit testing" sit alongside a suite that fails on a bare `go test ./...`, and the Caveat section lists "Zero-allocation code" as a goal that is now true of the resampler's steady state but not of the module generally. |

---

## H. Telephony signalling and tones

Nothing in the module generates or detects a tone of any kind. For PBX
application work this is a conspicuous hole: tones are how a switch talks to a
human and to other switches.

All entries in this section depend on
[DSP-4](#dsp-4-the-library-can-decompose-audio-but-not-compose-it), because a
tone sequence is structurally a concatenation of generated segments and silence.

### TONE-1: No DTMF generation **[requested]**

**Impact: ADDITIVE.**

There is no way to render a dial string such as `*123#` to audio. This is
probably the single most requested generator in PBX application work — IVR test
fixtures, automated dial-out, DTMF injection into recordings, conformance
testing.

DTMF is a sum of two sinusoids drawn from a row and a column frequency
(ITU-T Q.23):

| | 1209 Hz | 1336 Hz | 1477 Hz | 1633 Hz |
|---|---|---|---|---|
| **697 Hz** | 1 | 2 | 3 | A |
| **770 Hz** | 4 | 5 | 6 | B |
| **852 Hz** | 7 | 8 | 9 | C |
| **941 Hz** | * | 0 | # | D |

Getting it *right* rather than merely audible needs a handful of specifics that
are easy to miss:

- **Level and twist.** The two tones are not equal: the high group is normally
  a little hotter than the low group ("forward twist"), and receivers enforce a
  tolerance on the difference. Two sinusoids at full scale also sum to clipping,
  so each component must be scaled with headroom.
- **Cadence.** Minimum tone duration is specified (Q.24 territory); IVR
  practice is typically around 100 ms on and 100 ms off, and generators
  generally need both configurable.
- **Phase continuity and windowing.** Starting and stopping a sinusoid at a
  non-zero amplitude produces a click and spectral splatter, which can trip
  detectors. Segments need short fades or zero-crossing alignment — which is
  [DSP-4](#dsp-4-the-library-can-decompose-audio-but-not-compose-it)'s `Fade`.
- **A/B/C/D digits** exist (the 1633 Hz column) and are used in some signalling
  contexts, so a complete generator should include them.
- **Pause and wait characters.** Dial strings in the wild contain `,` and `w`;
  worth deciding whether the parser accepts them.

Prior art worth reading first: `livekit/media-sdk` has `dtmf` and `tones`
subpackages (Apache-2.0) — see
[FMT-4](#fmt-4-telco-and-voip-codecs-absent-user-2).

The natural API shape is a `Source` that renders a dial string, so the result
composes with everything else and can be written to any supported format:

```go
// Sketch only.
func NewDTMF(dialString string, sampleRate int, opts ...ToneOption) (Source, error)
```

### TONE-2: No DTMF detection

**Impact: ADDITIVE.**

The counterpart, and equally useful: extracting the digits from a recording,
verifying that generated tones are decodable, or building IVR test harnesses
that assert on what was sent.

The classical method is the **Goertzel algorithm** — one cheap recursive filter
per target frequency, which is why DTMF receivers were practical in the 1970s.
No statistical modelling is involved. A usable detector also needs twist
checking, second-harmonic rejection to reject speech that happens to contain
the frequencies ("talk-off"), and minimum-duration gating.

### TONE-3: No call progress tones, and no tone-plan model

**Impact: ADDITIVE.**

Dial tone, busy, ringback, congestion, special information tones, stutter dial
tone and message-waiting indication are all absent.

The important design point is that **these are country-specific**, which means
this is a data problem as much as a code problem. North America uses dual-tone
combinations (dial 350+440 Hz, busy 480+620 Hz, ringback 440+480 Hz); much of
Europe and the Middle East, including Israel, uses a single tone around
425 Hz distinguished purely by cadence. ITU-T E.180 documents the national
variations.

Asterisk solves this with `indications.conf` — a declarative per-country tone
plan — and that is the model worth following: a tone-plan description plus a
generic renderer, rather than hardcoded frequency constants. Supporting the same
syntax would let existing tone plans be reused directly.

### TONE-4: No MF / R2 signalling tones

**Impact: ADDITIVE.**

MF (R1) and R2 register signalling use a different frequency set from DTMF,
with KP and ST framing digits. Legacy, but still met on TDM interconnects and
in emulation or testing work. Low priority; recorded so the tone story is
complete.

### TONE-5: No fax and modem tone generation or detection

**Impact: ADDITIVE.**

Worth separating from call progress tones because the use case is different —
these are usually *detected* in order to route a call:

| Tone | Signal | Purpose |
|---|---|---|
| **CNG** (calling) | 1100 Hz, 0.5 s on / 3 s off | A fax machine announcing itself |
| **CED** / **ANSam** | 2100 Hz, a few seconds | Called-station answer; ANSam adds phase reversals |
| **V.8 / V.21 preamble** | — | Fax negotiation |
| Modem answer tone | 2100 Hz | Data call |

Detecting CNG or CED is how a PBX decides to hand a call to a fax stack rather
than a voice application. Generating them is needed for testing that path.

Note the interaction with [DSP-1](#dsp-1-anti-aliasing-filter-is-structurally-inadequate):
2100 Hz survives the telephony band comfortably, but any resampling artefact
near these frequencies can cause false detection, so tone detection and
resampling quality are related concerns.

---

## I. Filters, dynamics and enhancement

The module can convert and rearrange audio but cannot *condition* it. For PBX
application work this is the difference between a format converter and a useful
audio toolkit — prompt sets need levelling, recordings need cleaning, and input
from arbitrary sources needs taming.

### FILT-1: No filter framework

**Impact: ADDITIVE.** This is the foundation the rest of the section needs.

There is no biquad, IIR or FIR implementation, and no generic filter type. The
only filter in the module is the resampler's hardcoded one-pole
([DSP-1](#dsp-1-anti-aliasing-filter-is-structurally-inadequate)), which is not
reusable and not exposed.

A minimal foundation would be a biquad section with the standard design
formulae (low-pass, high-pass, band-pass, notch, peaking, shelf), cascadable
for steeper slopes, plus an FIR convolver with window-based design. Everything
below is then a configuration rather than new code — and
[DSP-2](#dsp-2-no-resampler-algorithm-selection-user-4)'s polyphase FIR would
share the same machinery.

### FILT-2: No telephony band-pass

**Impact: ADDITIVE.**

The 300–3400 Hz passband is *the* defining characteristic of telephone audio.
Two distinct uses, both unavailable:

- **Conditioning before encoding.** Removing energy outside the band before
  G.711 or G.729 improves perceived quality and avoids wasting bits.
- **Simulating a phone line.** Auditioning how a prompt will actually sound to
  a caller is something prompt authors need constantly, and doing it requires
  band-limiting, not just resampling to 8 kHz.

### FILT-3: No DC offset removal

**Impact: ADDITIVE.**

A DC offset — common in audio captured from cheap hardware — wastes headroom,
biases peak measurement, and degrades codec performance because the predictive
codecs are modelling a signal that is not centred. A first-order high-pass at a
few hertz fixes it and costs almost nothing. There is currently no way to
measure it ([DSP-10](#dsp-10-no-analysis-primitives)) or remove it.

### FILT-4: No automatic gain control

**Impact: ADDITIVE.**

Prompt sets assembled from multiple sources, and recordings from varied
endpoints, arrive at wildly different levels. `WithNormalize` applies a single
static gain based on peak
([DSP-8](#dsp-8-normalization-buffers-the-entire-stream)), which is not the same
thing: it cannot correct level *variation within* a file, and peak
normalization is a poor proxy for perceived loudness — one transient sets the
gain for the whole recording.

AGC with attack, release and target level, and loudness-based normalization
([FILT-9](#filt-9-no-loudness-measurement-or-telephony-level-conventions)),
address different halves of this.

### FILT-5: No compressor or limiter

**Impact: ADDITIVE.**

Nothing prevents clipping except hard clamping in
`utils.Float32ToInt16`, which is the worst-sounding option and is applied
silently ([OBS-2](#obs-2-no-processing-statistics)). A limiter with lookahead
would let gain be applied aggressively without artefacts — which matters
because prompts are usually wanted as loud as possible without distortion.

Compression also improves intelligibility over a narrowband channel, which is
why broadcast and telephony both use it heavily.

### FILT-6: No noise gate or noise reduction

**Impact: ADDITIVE.**

- **Noise gate** — attenuate below a threshold to clean up room tone between
  phrases in a prompt recording. Straightforward given
  [DSP-10](#dsp-10-no-analysis-primitives).
- **Noise reduction** — spectral subtraction against an estimated noise floor
  is the classic approach and predates any machine learning. Needs an FFT, so
  it also implies a spectral-processing capability the module does not have.

### FILT-7: No echo cancellation

**Impact: ADDITIVE.** Substantial work; listed because its absence is notable
in a PBX context.

Line echo cancellation (ITU-T G.168) and acoustic echo cancellation are
standard PBX concerns. The classical implementation is an adaptive FIR filter
(NLMS) plus a double-talk detector and residual suppression.

Honest assessment: this is weeks of specialist DSP, it needs a reference signal
as well as the echoed one, and it is genuinely hard to get stable. For
*offline* file work it is also rarely what is wanted — echo cancellation
belongs in the media path, which is explicitly out of scope
([Explicitly not gaps](#explicitly-not-gaps)). Recorded for completeness, and
recommended as out of scope unless a concrete need appears.

### FILT-8: No voice activity detection

**Impact: ADDITIVE.**

Needed for silence trimming, automatic segmentation
([PBX-4](#pbx-4-no-automatic-segmentation)), talk-time measurement, and
deciding where to place prompt boundaries. Energy plus zero-crossing rate is
the simple approach; the G.729 Annex B VAD is the telephony-standard one and
would come along with that codec.

### FILT-9: No loudness measurement or telephony level conventions

**Impact: ADDITIVE.**

Two related omissions:

- **Perceptual loudness.** Peak level correlates poorly with how loud something
  sounds. ITU-R BS.1770 loudness (LUFS) is the standard measure and is what
  makes a prompt set sound consistent. Without it, "normalize" means peak
  normalization, which does not deliver consistency.
- **Telephony level units.** Telephony works in **dBm0** and **dBov**, with
  established conventions — test tones at a defined level, speech at a nominal
  level below full scale. A library that only speaks dBFS forces every caller
  to do the conversion, and makes it easy to produce prompts that are correct
  in a file and wrong on a trunk.

Supporting both, and being explicit about which unit any API uses, is a small
amount of code with a large effect on whether the output is right.

**Resolved:** distinct named types — `Gain`, `DBFS`, `DBov`, `LUFS` — with
linear amplitude canonical, a typed `WithGain`, and normalisation split into
explicit peak and loudness variants. The *types* are a v1 requirement; the
measurements and dynamics processors remain additive. See
[roadmap_v1.md](roadmap_v1.md#d6--level-and-loudness-units-are-distinct-named-types).

---

## J. PBX application support

Capabilities that are less about audio theory and more about what an
application developer actually has to do, repeatedly, when building on a PBX.

### PBX-1: No frame alignment or duration helpers

**Impact: ADDITIVE.**

PBX media is frame-based, and codec frame sizes are fixed: G.711 by convention
20 ms, GSM 20 ms in 33-byte frames, G.729 10 ms in 10 bytes, G.723.1 30 ms,
Opus a choice of 2.5–60 ms. Output that is not a whole number of frames is
either padded by the platform, truncated, or rejected.

Nothing in the module helps with this. There is no way to pad to a frame
boundary, to ask how many samples a given duration is at a given rate, or to
express a length in milliseconds rather than samples. Every caller converts by
hand, and off-by-one-frame errors are easy and annoying to debug.

Related: the whole API speaks in samples, never in time. A `Duration`-based
vocabulary for trimming, padding and generating would remove a large class of
arithmetic errors, and connects to
[API-1](#api-1-source-cannot-report-length-or-duration).

### PBX-2: No PBX output conventions or validation

**Impact: ADDITIVE.**

Producing a file a PBX will actually play requires getting several things right
simultaneously: the correct sample rate for the format, mono, the right codec,
frame-aligned length, and a filename the platform recognises
([FMT-13](#fmt-13-asterisk-and-freeswitch-native-file-formats)). Getting any
one wrong produces either silence, a refusal, or noise — and the failure
usually appears at call time rather than at conversion time.

A conventions layer that knows each platform's requirements, and refuses to
write something invalid, would prevent a whole category of deployment-time
surprises.

### PBX-3: No prompt-set validation

**Impact: ADDITIVE.**

A prompt library is typically hundreds of files that must be *consistent*: same
rate, same channel count, comparable loudness, no clipping, no DC offset, no
excessive leading or trailing silence, no empty files.

There is no way to check any of this. A validator — effectively a linter for a
prompt directory, reporting per-file and aggregate statistics — is
straightforward once [DSP-10](#dsp-10-no-analysis-primitives) and
[FILT-9](#filt-9-no-loudness-measurement-or-telephony-level-conventions) exist,
and is the kind of tool that pays for itself immediately. Inconsistent prompt
loudness is one of the most common and most noticeable defects in deployed IVR
systems.

### PBX-4: No automatic segmentation

**Impact: ADDITIVE.**

Splitting one long recording into many prompts is a routine task with no
support. Two mechanisms, both absent:

- **Split on silence** — detect gaps above a duration threshold and cut there.
  Needs [FILT-8](#filt-8-no-voice-activity-detection) and
  [DSP-4](#dsp-4-the-library-can-decompose-audio-but-not-compose-it).
- **Split at markers** — use embedded cue points
  ([FMT-16](#fmt-16-in-file-markers-and-regions-are-discarded)), which is more
  reliable because a human placed them.

Both also apply in reverse for call recordings: segmenting on silence is how
talk spurts are identified.

### PBX-5: No streaming I/O boundary

**Impact: ADDITIVE.**

The module reads from `io.Reader` and, for output, requires a complete
`[]int16` in memory. There is no encoder that writes progressively to an
`io.Writer` — `wav.WriteWAV16` needs the whole buffer up front to compute
header sizes ([FMT-11](#fmt-11-non-pcm-payloads-inside-the-wav-container)).

For a PBX service this shows up quickly: transcoding on the fly to an HTTP
response, feeding a media server over a socket, or handling a recording longer
than available memory. The decode side streams properly; the encode side does
not, and the asymmetry is a direct consequence of
[API-5](#api-5-no-encoder-abstraction).

### PBX-6: No multi-file batch operations

**Impact: ADDITIVE.**

Bulk conversion of a prompt directory — with consistent settings, parallelism,
per-file error reporting and progress — is the most common thing anyone does
with a library like this, and every consumer will write it themselves.
`ProcessChannels` already demonstrates the pattern (concurrency, per-item
errors, progress callbacks) but only across the channels of one file, not
across many files.

This is a thin convenience layer rather than new capability, which is exactly
why it is worth providing once rather than having every caller reinvent it. It
depends on [OBS-1](#obs-1-no-progress-reporting-user-3) and
[API-2](#api-2-no-cancellation-mechanism) to be properly useful.

### PBX-7: No call or prompt annotation

**Impact: ADDITIVE.** The PBX-specific half of
[FMT-10](#fmt-10-no-metadata-or-tag-handling).

Generic tag support (title, artist, album) is only part of what telephony
needs. Two concrete uses, neither possible today:

**Call recordings** want provenance attached to the audio: calling and called
number, call or session identifier, direction, agent or extension, start time,
and which transcoding steps were applied. Two `bext` fields are made for this:

- **`TimeReference`** is a sample-accurate offset from midnight, which is
  exactly what correlating a recording against CDR or signalling logs requires
  — far better than relying on filesystem timestamps.
- **`CodingHistory`** is a free-text audit trail of format conversions. For a
  recording that has passed through G.711, been resampled and re-encoded, this
  is the difference between an auditable artefact and an anonymous file. For
  compliance-retained recordings that distinction can matter.

**Prompt libraries** want the opposite direction: the script text the prompt
speaks, language and locale, voice talent, revision, and approval state.
Carrying the script *inside* the audio file makes a prompt set
self-documenting, and makes [PBX-3](#pbx-3-no-prompt-set-validation) able to
check that a file matches its intended text.

**The hard part is that the formats a PBX actually plays cannot hold any of
this.** `.sln`, `.ulaw`, `.alaw` and `.gsm` are headerless
([FMT-13](#fmt-13-asterisk-and-freeswitch-native-file-formats)). So annotation
needs a defined answer for them:

- **Sidecar files** — a `.json` or `.xml` next to the audio, with a documented
  naming convention. Simple and format-independent, but easy to separate from
  the audio it describes.
- **A metadata-capable container** where the platform permits one: WAV with
  `bext` plus an A-law or µ-law payload
  ([FMT-11](#fmt-11-non-pcm-payloads-inside-the-wav-container)) gives both
  zero-transcode playback *and* a place for metadata. That combination is
  probably the best answer, and it is a good argument for prioritising
  companded payloads inside WAV.

Either way this needs a policy rather than an ad-hoc choice, because the
decision determines whether metadata survives deployment to a PBX at all.

---

## K. Robustness and resource limits

A library that a PBX points at caller-supplied audio is processing untrusted
input. Nothing in the module currently bounds what that input can cost, and the
three entries below were each confirmed by measurement rather than inspection.

### RES-1: A 62-byte file can force gigabytes of allocation

**Impact: ADDITIVE** to fix. Recorded first because it is the most serious
finding in this document.

AIFF's SSND chunk begins with a 32-bit *offset* field. `go-audio/aiff` allocates
a buffer of exactly that size in order to skip it:

```go
if offset > 0 {
    d.PCMSize -= offset
    buf := make([]byte, offset)   // offset comes straight from the file
```

`audpbx`'s `aiff.Decode` calls `FwdToPCM()`, so this is reached on every AIFF
decode. Measured, with a hand-built 62-byte AIFF declaring a 100 MB offset:

```
file=62 bytes  declared offset=0            allocated=    0.0 MB  err=<nil>
file=62 bytes  declared offset=100000000    allocated=  190.8 MB  err=<nil>
```

Two things to note. The amplification is roughly **3,000,000×**, and because the
field is a full `uint32` the ceiling is about **4 GB from a 62-byte input**. And
`Decode` returns **no error** — the allocation simply succeeds, so a service has
no signal that anything unusual happened.

**The same defect class is present and reachable in the WAV path.**
`riff.Chunk.DecodeWavHeader` performs `extra := make([]byte, ch.Size-16)` where
`ch.Size` is the `fmt ` chunk size from the file, reached via `readHeaders`
which `NewDecoder`, `IsValidFile` and `FwdToPCM` all trigger. Measured:

```
file=48 bytes  declared fmt size=16           allocated=    0.0 MB  err=<nil>
file=48 bytes  declared fmt size=100000000    allocated=  190.8 MB  err=...EOF
```

About 4,000,000x amplification. WAV differs from AIFF only in returning an error
— after the allocation has already happened. Three further instances sit in
`cue_chunk.go`, `list_chunk.go` and `smpl_chunk.go`, behind `ReadMetadata()`
which `audpbx` never calls; latent, but they would become live if
[FMT-10](#fmt-10-no-metadata-or-tag-handling) were built against this
dependency.

The defect is in the dependency, but the exposure is this module's: `audpbx` is
what a service hands untrusted bytes to, and it applies no guard.

**Resolved:** fixed by removing the dependency, not by mitigation — a validated
chunk walker plus owned WAV and AIFF header parsing, treated as a security fix
and taking precedence over other feature work. See
[roadmap_v1.md](roadmap_v1.md#d8--archived-dependencies-with-known-security-defects-are-removed-not-mitigated).

This also reframes [QA-5](#qa-5-no-fuzzing-of-binary-parsers): fuzzing is not a
hygiene item here, it is the thing that would have found this.

### RES-2: Untrusted input is read into memory without bound

**Impact: ADDITIVE.**

Both the WAV and AIFF decoders fall back to reading the entire input when the
reader is not an `io.ReadSeeker`:

```go
rs, ok := r.(io.ReadSeeker)
if !ok {
    data, err := io.ReadAll(r)      // unbounded
```

That path is taken by exactly the callers most likely to be handling untrusted
data — an HTTP request body, a socket, a pipe — since those are not seekable.
There is no size limit and no way for a caller to impose one short of wrapping
the reader in an `io.LimitReader` themselves, which the documentation does not
mention.

Note the interaction with [API-18](#api-18-source-cannot-seek): if seeking
becomes a first-class capability, this fallback becomes more prominent rather
than less, because more code paths will want a seekable source.

### RES-3: Channel splitting buffers unread channels without bound

**Impact: ADDITIVE,** and it affects the documented usage pattern.

`ChannelSource` holds a per-channel ring buffer that doubles whenever it fills:

```go
if cs.count == len(cs.ring) {
    newRing := make([]float32, len(cs.ring)*2)
```

There is no cap. Because all channels share one underlying source, reading one
channel forces the samples for every *other* channel to be buffered until
someone reads them. So memory grows with **stream length × unread channels**.

The trap is that this is the natural way to use the API. `SplitToMonoSources`
documents that "each returned Source can be read independently", and the
obvious pattern — finish channel 0, then start channel 1 — is the pathological
case. Measured on a 60-second 44.1 kHz stereo source, reading channel 0 to
completion first:

```
input:            2646000 frames stereo (60.0 s)
heap growth:      24.1 MB while channel 1 sat unread
```

Extrapolating: the 16-minute test asset would buffer roughly 170 MB per unread
channel, and a 5.1 source read one channel at a time would hold five of them.

Three possible answers, and the choice is API-visible: cap the buffer and block
(requires the caller to read channels concurrently, which contradicts the
current documentation); cap it and return an error; or require a seekable
source and re-read per channel
([API-18](#api-18-source-cannot-seek)). Whichever is chosen, the concurrency
expectation needs stating explicitly —
[API-15](#api-15-no-documented-concurrency-guarantees) is the entry for that.

---

## L. Dependency health

### DEP-1: Five of seven dependencies are archived

**Impact: BREAKING in practice** — not because an API changes, but because v1
promises stability for the whole v1 series and most of what it rests on is
read-only.

Checked against GitHub:

| Dependency | Role in audpbx | Status |
|---|---|---|
| `go-audio/wav` | WAV container and chunk navigation | **Archived 2026-02-21** |
| `go-audio/aiff` | AIFF container and chunk navigation | **Archived 2026-02-21** |
| `go-audio/audio` | `IntBuffer`, `Format` types | **Archived 2026-02-21** |
| `go-audio/riff` | RIFF chunk walking (indirect) | **Archived 2026-02-21** |
| `hajimehoshi/go-mp3` | MP3 decoder | **Archived 2023-04-02**, "no longer maintained" |
| `jfreymuth/oggvorbis` | Ogg container | Active |
| `jfreymuth/vorbis` | Vorbis codec (indirect) | Active |

The whole `go-audio` organisation went read-only on a single day. `go-mp3` has
been unmaintained since April 2023.

Direct consequences:

- **[RES-1](#res-1-a-62-byte-file-can-force-gigabytes-of-allocation) will never
  be fixed upstream.** The allocation-amplification defect is in
  `go-audio/aiff`; reporting it is pointless.
- **The MP3 performance gap is permanent** unless the dependency changes.
  Roughly 2043 ms versus ffmpeg's 198 ms, and 552,000 allocations, all inside
  `go-mp3`.
- **No security fixes** for four binary parsers handling untrusted input, which
  raises the stakes on both
  [QA-5](#qa-5-no-fuzzing-of-binary-parsers) and
  [RES-2](#res-2-untrusted-input-is-read-into-memory-without-bound).

#### Two earlier recommendations in this document are withdrawn

[FMT-1](#fmt-1-encoders-exist-for-one-format-in-one-configuration-user-1) and
[FMT-10](#fmt-10-no-metadata-or-tag-handling) both suggested adopting
`go-audio/wav`'s `Encoder` and `Metadata`, on the grounds that they already
support arbitrary channel counts, bit depths, format tags and RIFF `INFO`. That
was written before the archival was known.

**Closing gaps by taking on more surface area from an archived dependency is
the wrong direction.** The capability observation still holds — it shows the
work is well understood and bounded — but it is no longer an argument for
adopting that code.

#### What audpbx actually uses go-audio for

This is the useful part, and it is a smaller surface than it looks. After the
earlier performance work, `audpbx` decodes WAV and AIFF samples itself: the
16-bit paths read the PCM and SSND chunks directly. What remains is:

| Package | What audpbx still calls |
|---|---|
| `go-audio/wav` | `NewDecoder`, `IsValidFile`, `WavAudioFormat`, `BitDepth`, `FwdToPCM`, `Format`, `PCMChunk.R` |
| `go-audio/aiff` | the same shape, plus `PCMBuffer` only as a fallback for bit depths other than 16 |
| `go-audio/audio` | `IntBuffer` and `Format`, used only on that fallback path |
| `go-audio/riff` | reached only through `go-audio/wav` |

In other words: **container parsing and chunk navigation.** Nothing else.

#### Three routes, and they differ per format

**WAV and AIFF — own the container layer.** This is the recommended route, and
it is not a scope expansion: container and header parsing sits squarely in the
"audpbx does" column of
[the scope note](#scope-what-this-project-is-and-is-not). RIFF and AIFF are
stable, thoroughly specified formats that have not changed in decades, and
audpbx already owns the sample decoding. Owning the chunk layer as well would:

- remove four archived dependencies at once;
- fix [RES-1](#res-1-a-62-byte-file-can-force-gigabytes-of-allocation) by
  construction, since declared sizes can be validated against actual input;
- eliminate the `[]int` intermediate that the fallback path still carries;
- unblock a cluster of gaps that are currently limited by what the dependency
  chooses to expose —
  [FMT-2](#fmt-2-wav-decoding-is-narrow-user-5),
  [FMT-11](#fmt-11-non-pcm-payloads-inside-the-wav-container),
  [FMT-12](#fmt-12-aiff-c-compression-types),
  [FMT-16](#fmt-16-in-file-markers-and-regions-are-discarded),
  [FMT-10](#fmt-10-no-metadata-or-tag-handling),
  [API-6](#api-6-writewav16-is-permanently-mono) and
  [PBX-5](#pbx-5-no-streaming-io-boundary).

It is real work — a RIFF chunk walker, `fmt`/`data`/`LIST`/`bext` handling, and
AIFF's `FORM`/`COMM`/`SSND` plus the 80-bit extended float sample rate — but it
is bounded, testable, and the kind of code that stays written once fuzzed.

**MP3 — the survey has been done, and there is no replacement.** An MP3
decoder *is* a codec, so
[the scope note](#scope-what-this-project-is-and-is-not) rules out writing one
here. But looking for a maintained alternative turns up a dead end:

| Candidate | Finding |
|---|---|
| `hajimehoshi/go-mp3` | Archived, and still **the de facto pure-Go MP3 decoder** |
| `hajimehoshi/ebiten/v2/audio/mp3` | Actively maintained (released within days of this writing) but **imports `go-mp3`** — a consumer, not a replacement |
| `gopxl/beep/v2/mp3` | Maintained fork of the abandoned `faiface/beep`; wraps the same decoder. Worth confirming, but almost certainly the same situation |
| `tosone/minimp3` | **cgo** binding to the `lieff/minimp3` C library |
| `tcolgate/mp3` | Frame parser only, not a full decoder; last published 2017 |

So **there is no maintained pure-Go MP3 decoder.** Everything pure-Go wraps the
archived one. That leaves four real options:

1. **Keep `go-mp3` as-is.** MP3 is a frozen format — no new variants are
   coming — so an unmaintained decoder keeps working indefinitely. The residual
   risk is unfixed security defects in a binary parser fed untrusted input,
   which is [QA-5](#qa-5-no-fuzzing-of-binary-parsers) and
   [RES-2](#res-2-untrusted-input-is-read-into-memory-without-bound) territory
   rather than a functional problem. Lowest effort.
2. **Fork `go-mp3`.** Qualitatively easier than forking `go-audio`: one
   self-contained repository rather than four interdependent ones, Apache-2.0,
   and a frozen format that will never need feature work. A fork is still an
   external dependency under
   [the scope note](#scope-what-this-project-is-and-is-not), which explicitly
   allows "a separate standalone project that `audpbx` then imports". It would
   also make the known performance defects fixable — the per-call allocation in
   `imdct.Win` and the `math.Pow` calls in requantisation account for most of
   the 552,000 allocations and much of the 2043 ms.
3. **cgo to minimp3.** Fast, but breaks the native-Go-first property and needs
   the build-tag precedent from [FMT-5](#fmt-5-aac-support-user-6) settled
   first.
4. **Drop MP3 support.**

Options 1 and 2 differ only in whether you want the performance and the ability
to patch. Neither requires implementing a codec.

**Decided: vendor `minimp3`** (option 3, narrowed to decode only). It is
CC0-1.0, single-header, actively maintained, and covers Layer I, II and III
across MPEG-1, MPEG-2 and MPEG-2.5 with a stated conformance result
("Conformance test passed on all vectors (PSNR > 96db)"). Its `minimp3_ex` API
also provides `mp3dec_ex_seek` with a sample-accurate mode and a total sample
count — which is precisely what `go-mp3` fails to expose and what
[API-1](#api-1-source-cannot-report-length-or-duration),
[API-18](#api-18-source-cannot-seek) and
[API-20](#api-20-source-cannot-report-position) need.

Two properties make this cheaper than a normal cgo dependency: the header is
**vendored into the repository**, so there is no system library, no pkg-config
and no presence check; and because it is decode-only, **no build-tag matrix is
required**. MP3 *encoding* is the thing that would force a real binding (LAME,
LGPL), and that decision can wait until encoding is actually wanted.

**Forking `go-audio` — the weakest option.** It means four interdependent repos
(`wav` needs `riff` needs `audio`), and it inherits a design audpbx has already
spent effort working around: `PCMBuffer`'s per-call allocations and the `[]int`
intermediate four times wider than the samples it carries. Worth considering
only as a stopgap.

**Resolved:** own the WAV and AIFF container layer (as a security fix — see
[D8](roadmap_v1.md#d8--archived-dependencies-with-known-security-defects-are-removed-not-mitigated)),
replace `go-mp3` with vendored `minimp3` within v1, keep the Ogg Vorbis pair.
Per-codec routes in
[D9](roadmap_v1.md#d9--per-dependency-strategy).

### DEP-2: No dependency policy

**Impact: INTERNAL,** but it is what let DEP-1 go unnoticed.

Nothing records why each dependency was chosen, what would trigger replacing
one, or whether maintenance status is checked. There is no license inventory
either, which matters because the project is EPL-2.0 and
[FMT-5](#fmt-5-aac-support-user-6) contemplates cgo bindings to libraries whose
terms vary considerably — some codec libraries are LGPL or GPL, which
constrains distribution in ways a permissive dependency does not.

For v1 this wants: an inventory with licence and maintenance status per
dependency, a stated position on archived dependencies, and a note in
[CONTRIBUTING](#doc-1-to-doc-4) on how new dependencies are evaluated.

#### The posture the FMT-4 decisions produce

Worth recording as an explicit goal, because it is easy to erode one dependency
at a time:

- **Default build: permissive licences only, and no cgo.** `zaf/g711`
  (BSD-3-Clause), `pion/opus` (MIT), own G.722/G.726, plus `minimp3` (CC0)
  vendored as source rather than linked as a library.
- **cgo and LGPL only behind build tags**, for G.723.1, G.729 and AAC via
  libavcodec (LGPL-2.1 when built without `--enable-gpl`).
- **No LGPL obligation reaches downstream users unless they opt in.**

That last point is the one that matters most for a library: obligations attach
when a *binary* is distributed, and the party doing that is the consumer of
`audpbx`, not `audpbx` itself. So a default-permissive posture is a decision
made on their behalf.

Licence notes gathered while surveying, worth keeping in the inventory:

| Library | Licence | Note |
|---|---|---|
| `minimp3` | **CC0-1.0** | Single-header; vendorable, so no system dependency and no build-tag matrix for MP3 decode |
| `dr_libs` (`dr_mp3`) | Public domain | Equivalent alternative to minimp3; also ships `dr_wav` and `dr_flac` |
| LAME | **LGPL** | The quality reference for MP3 *encode*; only needed if encoding is added. Static linking triggers relinking obligations, so dynamic should be its default mode |
| libmpg123 | LGPL-2.1 | Mature decoder; not needed given minimp3 covers Layer I/II/III across MPEG-1/2/2.5 |
| **libmad** | **GPL-2.0+** | **Avoid.** Frequently recommended for MP3 decode, but GPL is viral for downstream users, and it has been unmaintained since roughly 2004 |
| FFmpeg libavcodec | LGPL-2.1+ | Only if built **without** `--enable-gpl` |
| libspeex | Revised BSD | Dropped; obsoleted by Opus per its own project page |
| `libilbc` | BSD-3-Clause | Dropped; needs a C++14 compiler, CMake and Abseil |

**Resolved:** a committed `DEPENDENCIES.md` rather than a table in an analysis
document, `google/go-licenses` in CI, licence review scoped to direct
dependencies, and archival status re-checked each release. See
[D10](roadmap_v1.md#d10--dependency-governance-is-a-committed-artifact-not-prose).

One further note for any future revisit of Rust implementations: **Rust does not
avoid cgo.** Rust has no stable ABI, so a Go caller needs an
`extern "C"` surface compiled to a staticlib or cdylib and then invoked through
cgo — meaning cgo *plus* a Rust toolchain, which is strictly more build surface
than binding a C library.

---

## Decisions required before v1

These are the choices that cannot be deferred, because deferring them means a
v2. Roughly in dependency order.

**Resolution status is tracked in [roadmap_v1.md](roadmap_v1.md), not here.**
That document is the decision log; this list is the inventory of what needs
deciding. Four are resolved so far.

1. **Does `Source` gain length, position and seeking — and how?**
   ([API-1](#api-1-source-cannot-report-length-or-duration),
   [API-18](#api-18-source-cannot-seek),
   [API-20](#api-20-source-cannot-report-position)) All three are the same
   question and should get one answer: directly on the interface, or optional
   interfaces discovered by type assertion? This gates
   [OBS-1](#obs-1-no-progress-reporting-user-3), resume
   ([OBS-5](#obs-5-processing-state-cannot-be-checkpointed-or-restored)), the
   two-pass form of
   [DSP-8](#dsp-8-normalization-buffers-the-entire-stream), and efficient
   trimming in
   [DSP-4](#dsp-4-the-library-can-decompose-audio-but-not-compose-it).
   *Recommendation: optional interfaces, keeping `Source` minimal.*

2. **How is progress delivered, and is there a task handle?**
   ([OBS-1](#obs-1-no-progress-reporting-user-3),
   [OBS-4](#obs-4-no-task-abstraction)) A callback supplied up front, or a
   handle whose state the caller polls? This determines whether long-running
   operations keep their current blocking signatures, so it cannot be deferred.
   It also settles where cancellation, statistics and warnings attach.
   *Recommendation: a handle — it makes rate limiting, cancellation and
   post-hoc inspection natural, where a callback makes all three awkward.*

3. **How is cancellation expressed?**
   ([API-2](#api-2-no-cancellation-mechanism)) `context.Context` as a first
   parameter, parallel `...Context` functions, or an explicit `Cancel` method?

4. **What exactly does `ReadSamples` promise?**
   ([API-3](#api-3-readsamples-contract-is-under-specified)) Is `(n>0, io.EOF)`
   legal? Is `(0, nil)` legal? Then make all four decoders agree and add a
   conformance test.

5. **Does `BufSize()` stay in the interface?**
   ([API-8](#api-8-bufsize-is-vestigial)) Give it a real contract or remove it
   now.

6. **What are the codec abstractions?**
   ([API-5](#api-5-no-encoder-abstraction),
   [API-17](#api-17-decoder-cannot-express-framed-codecs)) `Encoder` and
   `PacketDecoder` should be designed together with `Decoder`, not bolted on
   separately.

7. **Is `ChannelOption` open or closed?**
   ([API-4](#api-4-channeloption-is-a-closed-extension-point)) If third parties
   should be able to add options, the type must change.

8. **What are the rules when combining mismatched sources?**
   ([DSP-4](#dsp-4-the-library-can-decompose-audio-but-not-compose-it)) Once any
   function accepts multiple `Source`s, the handling of differing sample rates,
   channel counts and lengths becomes part of the API. Resample implicitly or
   reject? Pad, truncate or reject?

9. **What may callers rely on regarding output samples?**
   ([DSP-1](#dsp-1-anti-aliasing-filter-is-structurally-inadequate),
   [DSP-2](#dsp-2-no-resampler-algorithm-selection-user-4)) Improving the
   anti-aliasing filter changes output. Either state that exact samples are not
   part of the API contract, or version the algorithms so existing behaviour
   remains reachable. Without this, every DSP improvement is a breaking change.
   *Recommendation: state explicitly that sample-exact output is not guaranteed,
   and make the algorithm selectable so old behaviour stays available.*

10. **Which of the naming and shape cleanups happen?**
   ([API-10](#api-10-channelsource-value-and-pointer-asymmetry),
   [API-11](#api-11-two-different-processchannels-functions),
   [API-12](#api-12-dead-exported-error-values),
   [API-13](#api-13-inconsistent-error-construction),
   [API-14](#api-14-resampletomono16-signature-and-memory-model),
   [API-16](#api-16-utils-is-a-grab-bag-package)) Individually cosmetic,
   collectively the difference between a clean v1 and a decade of warts. They
   are also the easiest items to defer past the point of no return.

11. **Does the API speak in samples, or in time?**
    ([PBX-1](#pbx-1-no-frame-alignment-or-duration-helpers)) Every current
    signature counts samples. Generators, trimming and padding are far more
    naturally expressed as `time.Duration`, and tone cadences are specified in
    milliseconds. Introducing a duration-based vocabulary later means either
    parallel functions or a break. Decide the convention before the
    [H](#h-telephony-signalling-and-tones) and
    [I](#i-filters-dynamics-and-enhancement) sections are built on top of it.

12. **Which level and loudness units does the API use?**
    ([FILT-9](#filt-9-no-loudness-measurement-or-telephony-level-conventions))
    dBFS, dBov, dBm0 or LUFS. Gain and normalization signatures will encode
    this choice, and telephony convention differs from the audio-software
    default. Being explicit costs nothing now and is awkward to change later.

13. **Where does metadata live, and does it flow through the pipeline?**
    ([API-19](#api-19-tag-manipulation-has-no-home-in-the-api),
    [FMT-19](#fmt-19-no-cross-format-metadata-mapping-or-preservation-policy))
    A separate tag package, or methods on `Source`? The latter makes a lossless
    retag structurally impossible, because every metadata change would have to
    pass through a decode and re-encode. Also decide whether `Encoder` accepts a
    tag set, and what happens when a target format cannot represent a field.
    *Recommendation: separate tag package for read/write, plus optional tag
    acceptance on `Encoder` for genuine transcodes.*

14. **What is the supported Go version range, and what is the deprecation
    policy?** ([DOC-1](#doc-1-to-doc-4))

15. **Does the library enforce resource limits, or document that callers must?**
    ([RES-1](#res-1-a-62-byte-file-can-force-gigabytes-of-allocation),
    [RES-2](#res-2-untrusted-input-is-read-into-memory-without-bound),
    [RES-3](#res-3-channel-splitting-buffers-unread-channels-without-bound)) If
    limits become options or constructor parameters, that is API surface and
    belongs in v1. If the position is instead "callers must bound their own
    input", that needs saying explicitly, because the current silence reads as
    "this is safe to point at an upload".

16. **What is the dependency strategy?**
    ([DEP-1](#dep-1-five-of-seven-dependencies-are-archived),
    [DEP-2](#dep-2-no-dependency-policy)) Five of seven dependencies are
    archived, so this gates v1 regardless of API shape. The routes differ by
    format: own the container layer for WAV and AIFF, survey for an MP3
    replacement, fork only as a stopgap. Deciding this early matters because
    owning the container layer changes what
    [FMT-2](#fmt-2-wav-decoding-is-narrow-user-5),
    [FMT-10](#fmt-10-no-metadata-or-tag-handling),
    [FMT-11](#fmt-11-non-pcm-payloads-inside-the-wav-container) and
    [API-6](#api-6-writewav16-is-permanently-mono) cost.

17. **Is cgo acceptable, and under what conditions?**
    ([FMT-5](#fmt-5-aac-support-user-6)) This sets the precedent for every
    future binding — and several
    [FMT-4](#fmt-4-telco-and-voip-codecs-absent-user-2) codecs (AMR, EVS,
    G.722.1) are most realistically obtained that way.

---

## Explicitly not gaps

Recorded so they are not re-litigated:

- **Codec implementation is out of scope entirely.** See
  [the scope note](#scope-what-this-project-is-and-is-not). This project
  integrates codecs; it does not implement them. Any entry in this document
  that mentions a codec is asking for a wrapper and a dependency decision.
- **Codec performance is not this project's to fix.** The MP3 and Ogg
  throughput gap is inside `hajimehoshi/go-mp3` and `jfreymuth/vorbis`. The
  remedy is a different dependency, not local optimisation.
- **MP3 encoding is absent by design so far.** Encoder libraries are large and
  the decode path covers the stated use cases. Listed under
  [FMT-1](#fmt-1-encoders-exist-for-one-format-in-one-configuration-user-1) as
  optional.
- **Real-time RTP transport is out of scope.** Covered in
  [from_voip.md](from_voip.md#out-of-scope); the pull-based `Source` contract is
  a poor fit for lossy real-time streams, and the resampler's 1024 ms startup
  buffering makes it unusable for live audio regardless.
- **`MonoMixer` averaging rather than summing** is deliberate and clip-safe.
  [DSP-6](#dsp-6-downmix-is-unweighted-averaging-only) asks for *additional*
  weighted modes, not a change of default.
- **`Resampler` is not reusable across streams.** It carries end-of-stream state
  by design; construct one per stream. Documented in `audio/doc.go`.

---

## How these findings were verified

So that anything contested can be re-checked rather than argued about.

| Finding | Method |
|---|---|
| Exported API surface | `go doc -all` over every package |
| No G.711/Opus/G.729/AAC/RTP support | `grep` across non-test sources |
| No `Encoder` abstraction | `grep` for `type Encoder` and `Encode(` |
| `WriteWAV16` mono-only | Read source: `numChannels := uint16(1)` |
| No composition primitives | Searched for every plausible name (`Concat`, `Mix`, `Trim`, `Crossfade`, `Fade`, `Silence`, `Repeat`, `Pad`, …); then confirmed no exported function anywhere accepts `[]Source` or `...Source` |
| Dead exported errors | Counted non-test, non-declaration references per sentinel |
| `Registry` zero value panics | Read source: `mtx *sync.Mutex`, nil `codecs` map |
| `ChannelOption` closed | Read source: parameter type `*channelProcessor` is unexported |
| EOF contract divergence | Drained all four decoders, counting `(n>0, io.EOF)` and `(0, nil)` occurrences |
| Upstream length and seek availability | Inspected wav, aiff, go-mp3 and oggvorbis in the module cache |
| Mono MP3 upmixed to stereo | Upstream doc comment in `go-mp3/decode.go` |
| Anti-aliasing measurements | Ten tones through both audpbx and `ffmpeg`; FFT peak and RMS per output; compared against one-pole theory |
| Resampler startup buffering | Instrumented source counting frames consumed before first output: 8194 frames / 52 packets / 1024 ms |
| Block size is load-bearing | Set `resamplerBlockFrames = 1`: WAV pipeline 59 ms to 2409 ms |
| Normalization buffers all | Read source: growing `[]float32` accumulator |
| Test suite state | `go test ./...` compared against `go test -run Test ./...` |
| Coverage | `go test -cover ./...` |
| Formatting | `gofmt -l` over all sources |
| No CI, no fuzzing | Directory listing; `grep` for `func Fuzz` |
| `WithProgress` is a no-op | `grep` for `progressCallback`: two hits, a declaration and an assignment, no invocation site; no test references `WithProgress` |
| 62-byte AIFF forces ~191 MB allocation | Built a minimal AIFF declaring a 100 MB SSND offset; measured `TotalAlloc` across `Decode`; confirmed no error returned |
| Unbounded `io.ReadAll` on non-seekable input | Read `formats/wav/decoder.go` and `formats/aiff/decoder.go` fallback paths |
| Splitter buffers unread channels | Read channel 0 of a 60 s stereo source to completion, measured `HeapAlloc` growth: 24.1 MB |
| 32-channel cap and its error message | Called `SplitToMonoSources` with sources of 2, 6, 18, 31, 32, 33, 40 and 64 channels |
| Go codec availability per codec | pkg.go.dev searches for `g711`, `g722`, `g726`; importer counts, versions, dates and licences read from the result pages |
| Rust MP3 landscape | crates.io API search; Symphonia repo page (decode-only, MPL-2.0); `rusty_mp3` crate API (first published 2026-07-29, 0.8.0, ten releases) |
| C/C++ library licences | Each project's own page: minimp3 (CC0-1.0), dr_libs (public domain), LAME (LGPL, v4.0 July 2026), mpg123 (LGPL-2.1, v1.33.7), speex.org (revised BSD, obsoleted by Opus), TimothyGu/libilbc (BSD-3-Clause, C++14 + Abseil) |
| FFmpeg native codec coverage | FFmpeg general.html audio codec table — native decode for G.723.1, G.726, G.729, GSM, AMR-NB/WB, MP3, AAC, Opus, G.722 |
| libavcodec full-build size | `ls -lL` on the installed `libavcodec.so`: 19.9 MB; `ffmpeg -decoders` count: 553 |
| minimp3 layer and API coverage | minimp3 README: Layers I/II/III across MPEG-1/2/2.5, "Conformance test passed on all vectors (PSNR > 96db)", `mp3dec_ex_seek` and `dec.samples` |
| pion/opus documents no completeness claim | Read the repository page and raw README; encoder and decoder both present, but no statement of mode coverage or limitations |
| In-tree container groundwork | `grep` for `encodeFloat80`, `createAIFFFile`, `createWAVFile`, `TestDecoder_OddSizedChunkPadding` |
| Five of seven dependencies archived | Fetched each repository page and read the archival banner: `go-audio/{wav,aiff,audio,riff}` archived 2026-02-21; `hajimehoshi/go-mp3` archived 2023-04-02 with a "no longer maintained" notice; `jfreymuth/{oggvorbis,vorbis}` show no banner |
| What audpbx still calls from go-audio | Enumerated call sites in `formats/wav/decoder.go` and `formats/aiff/decoder.go` |
| Eight pointless error wraps | `grep` for `fmt.Errorf("%w", err)` across non-test sources |
| No state export or position reporting | `grep` for `MarshalBinary`, `GobEncode`, `Snapshot`, `Checkpoint`, `Restore`, `Position()`, `Tell()` across non-test sources |
| No metadata handling in `audpbx` | `grep` for ID3, Vorbis comment, `LIST`/`INFO`, `bext`, `OpusTags` across non-test sources |
| WAV metadata and N-channel encoding available upstream | Read `go-audio/wav` `decoder.go` (`Metadata`, `ReadMetadata`), `metadata.go` (struct fields) and `encoder.go` (`NewEncoder`, `writeMetadata`) |

---

## Related documents

- [README.markdown](README.markdown) — current capabilities, limitations, TODO
- [from_voip.md](from_voip.md) — RTP payload reconstruction design
- [examples/profile_resampler/PROFILING_GUIDE.md](examples/profile_resampler/PROFILING_GUIDE.md)
  — resampler performance history, measurement methodology, regression guards
