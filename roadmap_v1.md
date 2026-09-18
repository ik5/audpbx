<!-- SPDX-License-Identifier: EPL-2.0 -->

# Roadmap to v1

A log of the decisions that must be settled before v1 freezes the API, recorded
as they are taken.

## What this document is, and how it relates to `current_gaps.md`

[current_gaps.md](current_gaps.md) is the **compass**: a full inventory of 94
gaps across the project — missing formats and codecs, DSP and composition
primitives, tones, filters, observability, robustness, dependency health,
testing and documentation.

**It is not a v1 plan.** Most of what it contains lands well after v1, and some
of it may never land. It exists so that nothing is lost and so that every
decision can be made with the whole picture in view. Read it as "everything this
project could be", classified by how urgently each item has to be decided.

What makes it usable for planning is that every entry carries an **API impact**
class:

| Class | v1 relationship |
|---|---|
| **BREAKING** | Must be *resolved* before v1 — resolved meaning "the shape is decided", not "the feature is built" |
| **BEHAVIOURAL** | A policy must be *stated* before v1 |
| **ADDITIVE** | Does not need to be in v1 at all |
| **INTERNAL** | Gates the ability to release safely, not the API |

**This document is the v1 subset**: the decisions drawn from that inventory that
cannot be deferred, plus the reasoning behind each one. It is a decision log, not
a feature list.

## How v1 is scoped

**v1 is an API-freeze milestone, not a feature milestone.**

v1 could ship with close to today's feature set and a settled API. That is not a
weak v1 — it is precisely what allows v1.1 through v1.9 to add codecs, tones,
filters, metadata and composition without breaking anyone. Trying to make v1
feature-rich is the usual route to needing a v2.

Three tracks run in parallel:

1. **Decision track** — the items in this document. Thinking, not coding. Some
   decisions gate others.
2. **Enablement track** — the `INTERNAL` items from the inventory
   ([QA-1](current_gaps.md#qa-1-the-test-suite-fails-out-of-the-box) green test
   suite, [QA-2](current_gaps.md#qa-2-no-ci) CI,
   [QA-3](current_gaps.md#qa-3-no-lint-gate-and-ten-unformatted-files) lint,
   [QA-8](current_gaps.md#qa-8-no-pre-release-bug-and-security-audit) audit).
   No API impact, but without them the freeze cannot be *enforced* — `gorelease`
   in CI is what turns "we intend not to break the API" into something
   mechanical.
3. **Proving track** — a small number of `ADDITIVE` features that need to land
   *before* v1, not because they are wanted but because **an interface nothing
   has used cannot safely be frozen.** An `Encoder` interface with no encoder
   written against it is a guess.

## How v0.0.x slices work

Each patch release should:

- resolve **one bounded cluster** of decisions, or land **one capability**;
- leave the tree green;
- where possible **prove an API shape** rather than merely declaring it;
- respect the dependency order between decisions.

Ordering is not free. Dependencies discovered so far are recorded with each
decision below.

---

## Decisions resolved

### D1 — `Source` capability surface: length, position, seeking

**Resolves:** [API-1](current_gaps.md#api-1-source-cannot-report-length-or-duration),
[API-18](current_gaps.md#api-18-source-cannot-seek),
[API-20](current_gaps.md#api-20-source-cannot-report-position)

**Impact:** BREAKING — adds a method to an exported interface, so it must land
before v1.

#### Decision

`Position` is **mandatory** on `Source`. Length and seeking are **optional
interfaces**, discovered by type assertion, with a composite provided as a
convenience.

```go
// Mandatory: frames emitted so far.
type Source interface {
    // ... existing methods ...
    Position() int64
}

// Optional: exact, and able to report that the length is not known.
type Lengther interface {
    Length() (frames int64, known bool)
}

// Optional.
type FrameSeeker interface {
    SeekFrames(n int64) error
}

// Convenience for the common file-backed case. NOT a requirement —
// callers needing one capability should assert for that one alone.
type SeekableSource interface {
    Source
    Lengther
    FrameSeeker
}
```

#### Why optional rather than mandatory

The capability is **transport-dependent, not container-dependent**. Both
upstream decoders say so themselves:

> `oggvorbis`: *"A return value of zero means the length is unknown, probably
> because the underlying reader is not seekable."*

> `go-mp3`: *"Length returns -1 when the total size is not available e.g. when
> the given source is not io.Seeker."*

Both internally perform an optional `in.(io.Seeker)` assertion and degrade when
it fails. Additional cases: a WAV written to a pipe carries a placeholder `data`
size; a VBR MP3 without a Xing header requires a full file scan.

So the *same* WAV decoder can report length from a file and cannot from a pipe.
Interface satisfaction is a property of the type, but this capability varies per
**value** — which rules out encoding it in a single composite interface as a
requirement.

Decisive detail: for a non-seekable Ogg stream, `oggvorbis` assigns
`length = position` as it reads, so length is **progressively discovered**, not
merely absent. "Does this source know its length" can change *during* the
stream, which cannot be expressed in the type system at all. Hence the `known`
boolean in the signature rather than a sentinel such as `0` or `-1` — the trap
both upstream libraries fell into and had to document their way out of.

#### Why `Position` is mandatory anyway

Position is **always implementable** — a source counts what it emits, and no
container or transport can prevent that. It is also the only way to know how
much *input* a multi-stage pipeline has consumed: with `decode → resample →
mix`, counting mixer output says nothing about input progress, because the two
differ by the resampling ratio.

#### Why a composite as well

`SeekableSource` captures the ergonomic case — a file-backed decoder, where all
three hold — with one assertion instead of three, without forcing all-or-nothing
on sources where the capabilities diverge. It is derived from the small
interfaces, so nothing is duplicated.

---

### D2 — Unit vocabulary: frames are canonical, duration is derived

**Resolves:** [PBX-1](current_gaps.md#pbx-1-no-frame-alignment-or-duration-helpers)

**Impact:** BREAKING — determines the signatures in D1 and every later API that
expresses a position, length or span.

#### Decision

**Frames are the canonical unit.** Duration is available as a derived view,
converted once at the boundary rather than by each implementer.

```go
// Derived convenience — not on the interface, so the lossy conversion
// happens in exactly one place and is visible.
func Duration(s Source) (time.Duration, bool)
func FramesForDuration(d time.Duration, rate int) int64
```

#### Why frames, not samples

"Position 1000" in a stereo stream is ambiguous as a sample count — frame 500 or
frame 1000? Frames are the unit of time position, and the library already thinks
this way implicitly: `ErrInvalidDstSize` requires `dst` to be a whole number of
frames.

`ReadSamples` continues to count *values*, because it is about buffer sizing,
which is a different concern. That distinction needs documenting rather than
eliminating.

#### Why not `time.Duration` as canonical

`time.Duration` is integer nanoseconds, and one frame is not a whole number of
nanoseconds at most sample rates. Measured:

| Rate | ns per frame | Round-trips exactly? |
|---|---|---|
| 8000, 16000, 32000 | integer | **yes** |
| 22050, 44100, 48000 | fractional | **no** |

Drift when `time.Duration` is canonical, over one hour of audio:

- 44.1 kHz — **5,160 frames lost (0.12 s)**
- 48 kHz — **2,765 frames lost (0.06 s)**

Telephony rates happen to be exact, which is convenient for this library's
primary use. Source material is not: 44.1 and 48 kHz both drift. Operations that
must be exact — splicing two recordings, trimming at an embedded marker,
aligning to a 20 ms codec frame boundary — cannot tolerate that, and the failure
appears much later as drift rather than immediately as an error.

#### Why duration is still offered

Tone cadences, DTMF timing, RTP frame sizes and PBX frame alignment are all
specified in milliseconds. "Trim 20 ms" is rate-independent; "trim 882 frames"
is only 20 ms at one specific rate. Duration-accepting convenience variants
belong wherever that reads better — generators, trim, fade, pad.

Keeping duration off the interface matters: as a method, every implementer would
perform its own lossy conversion and they would disagree. As a free function
over `Lengther` and `SampleRate()`, there is exactly one conversion.

---

### D3 — `BufSize()` is removed from `Source`

**Resolves:** [API-8](current_gaps.md#api-8-bufsize-is-vestigial)

**Impact:** BREAKING — removes a method from an exported interface.

#### Decision

`BufSize()` leaves the `Source` interface. `audio.DefaultBufSize` remains as a
package-level constant for callers sizing their own buffers.

#### Why

There are four call sites, and all four use the value as an **arbitrary
allocation heuristic** rather than acting on a contract:

| Call site | Use |
|---|---|
| `normalize()` | `make([]float32, bufSize)` and `make([]float32, 0, bufSize*100)` |
| `readAllSamplesAsInt16()` | Identical pattern |
| `channel_splitter.go` (×2) | `bufSize: src.BufSize() / totalCh`, propagated to `ChannelSource.BufSize()`, which nothing outside tests reads |

All four behave identically with a constant. A method that every implementer is
obliged to write exists to serve something that does not need a method.

The implementations also disagree about what it means, which is the clearest
sign there is no contract to preserve:

- `mp3` returns `cap(s.buf) / 2`, which **changes with the last read size** — so
  it is not even stable for a given source
- `ChannelSource` returns `src.BufSize() / totalCh`; splitting a 4096-sample
  source across six channels yields 682, which is not a useful buffer size for
  anything
- `wav`, `bufferSource` and the test mock return a constant

#### The idea worth keeping, elsewhere

`BufSize` gestures at something real — **natural frame granularity.** "This
source produces 20 ms frames" genuinely matters for frame alignment
([PBX-1](current_gaps.md#pbx-1-no-frame-alignment-or-duration-helpers)) and for
writing output a PBX will accept.

But that is a property of a *codec*, not a buffer hint. It belongs on the
frame-oriented interface
([API-17](current_gaps.md#api-17-decoder-cannot-express-framed-codecs)) under an
honest name such as `FrameSize()`. Retaining `BufSize()` because it vaguely
points at that would preserve the wrong abstraction.

#### Work implied

- Remove the method from all ten implementations
- Switch `normalize()` and `readAllSamplesAsInt16()` to `audio.DefaultBufSize`
- Stop propagating `bufSize` through `ChannelSource`
- Delete the tests that assert on it: `audio/channel_splitter_test.go`,
  `audio/mono_mixer_test.go`, and the `BufSize` assertions in all four format
  packages

---

### D4 — `ChannelOption` is sealed, and applying an option can fail

**Resolves:** [API-4](current_gaps.md#api-4-channeloption-is-a-closed-extension-point)

**Impact:** BREAKING — changes the type of every `With*` return value.

#### Decision

```go
type ChannelOption interface {
    apply(*channelProcessor) error
}
```

Sealed by the unexported method, and option application can return an error.

#### Why sealed rather than opened

**`WithProcessor` is already the extension hook:**

```go
func WithProcessor(procFunc func(ch audio.Channel, src audio.Source) (audio.Source, error)) ChannelOption
```

A caller can inject arbitrary behaviour by returning a wrapped `Source`. Third
parties therefore never needed to write their own `ChannelOption` — they already
have a documented way in. **Closing the option set costs no capability.**

That settles the trade-off. Opening it via `func(*ChannelConfig) error` would
make the config struct's fields part of the frozen API, and `channelProcessor`
still has to grow during v1.x — cancellation
([API-2](current_gaps.md#api-2-no-cancellation-mechanism)), a task handle
([OBS-4](current_gaps.md#obs-4-no-task-abstraction)) and progress delivery
([OBS-1](current_gaps.md#obs-1-no-progress-reporting-user-3)) all land in there.
Exporting it now would trade that freedom away for extensibility that already
exists.

The current form is the worst of both: closed in fact, open in appearance. An
interface with an unexported method is the standard Go idiom and is *visibly*
sealed.

Callers are unaffected — they write `audpbx.WithGain(...)` and pass it along as
before. The only breakage is code that declared a `ChannelOption` variable and
assigned a raw function literal.

A struct with an unexported field seals it equally well; the interface is chosen
for being idiomatic and for leaving room for options that apply differently.

#### Why `apply` returns an error

It makes a documented behaviour true that currently is not. `WithBufferSize`
says:

> *"Must be at least 256 samples. Returns error if too small or too large."*

It returns `ChannelOption`, not an error. Validation actually happens later
inside `ProcessChannels`. This is the same class of defect as
[OBS-1](current_gaps.md#obs-1-no-progress-reporting-user-3)'s no-op
`WithProgress` — documentation describing an API that does not exist.

With an error return, validation happens at the offending option and the message
can name it. `ProcessChannels` should keep its central validation as a backstop
for combinations that only become invalid together.

#### Work implied

- Convert the thirteen `With*` functions to return the sealed type
- Move per-option validation into the options themselves
- Correct `WithBufferSize`'s doc comment, or make it accurate by validating there
- Keep central validation in `ProcessChannels` for cross-option constraints

---

### D5 — `ReadSamples` follows `io.Reader` semantics

**Resolves:** [API-3](current_gaps.md#api-3-readsamples-contract-is-under-specified)

**Impact:** BREAKING — tightens a contract, which changes what implementers must
guarantee.

#### Decision

The contract mirrors `io.Reader`, deliberately and verbatim where possible, so
that callers already know the rules.

1. **`n` counts float32 values written into `dst`** — not frames, not bytes.
   This must be the first sentence of the doc comment, not a parenthetical.
2. **`(n > 0, io.EOF)` is permitted.** Callers must always consume the `n` values
   before considering `err`.
3. **`(0, nil)` is forbidden** except when `len(dst) == 0`.
4. **A short read is not EOF.** `n < len(dst)` with a `nil` error means "that is
   what was available"; it carries no implication about the end of the stream.
5. **`ReadSamples` keeps its name.** Renaming a core method for clarity alone is
   churn, and `ReadSamples` counting values while `Position` counts frames is
   defensible — they measure different things. The asymmetry is documented rather
   than removed.

#### Why A rather than forbidding the combined return

Measured tail of each decoder, reading 4096 values at a time with
consume-then-check:

```
wav   total=9102636   ... (4096, nil) (4096, nil) (1324, io.EOF)
aiff  total=9102636   ... (4096, nil) (4096, nil) (1324, io.EOF)
ogg   total=9102636   ... (4096, nil) (4096, nil) ( 940, io.EOF)
mp3   total=9107712   ... (2304, nil) (2304, nil) (   0, io.EOF)
```

The naive loop — `if err != nil { break }` before using `n` — silently drops
1324 values on WAV and AIFF, 940 on Ogg, and **nothing on MP3**. A bug that is
both silent and format-dependent is the worst kind to leave available.

Choosing to permit the combined return has a useful property: **it requires no
change to any decoder.** Three already do it, and MP3's stricter behaviour is
*legal* under this rule — permitting the combined form does not require it.

Forbidding it would mean changing three decoders to hold back the EOF, carrying
a "pending EOF" flag, adding a round-trip per stream, and diverging from the
convention every Go programmer already knows from `io.Reader`. The cost is all
on the side of forbidding.

Note also that MP3 returns 2304 values when asked for 4096 — one frame of
1152 × 2 channels. Short reads are the normal case, not an edge case, which is
the other half of why the naive loop is wrong.

#### Correction to the inventory

[API-3](current_gaps.md#api-3-readsamples-contract-is-under-specified) recorded
that all four decoders "terminate with `(0, io.EOF)`". That was an artefact of
the probe used, which kept calling past the combined return. In fact wav, aiff
and ogg terminate *at* `(n > 0, io.EOF)`; they only produce `(0, io.EOF)` if
called again.

#### Work implied

- State the contract on the `Source` interface, mirroring `io.Reader`'s wording
  including the "callers should always process the n > 0 values returned before
  considering the error" clause
- **Fix the `(0, nil)` paths in `formats/mp3` and `formats/vorbis`** — a bounded
  retry, then `io.ErrNoProgress`, which exists in the standard library for
  exactly this situation
- Remove the defensive `(0, nil)` guard in `audio.Resampler`; a conformance test
  should catch violators instead of every consumer defending against them
- Correct `ReadSamples`'s doc comment so the unit appears in the first sentence

#### The deliverable that makes this real

A **`Source` conformance test** that every implementation must pass — the four
decoders, `Resampler`, `MonoMixer`, `ChannelSource`, and the test mocks. A
contract with no enforcement is a comment.

It should assert, at minimum:

- draining with consume-then-check yields the full stream
- `(0, nil)` never occurs for a non-empty `dst`
- a short read does not imply EOF
- `Position()` is monotonic and, at EOF, equals the total frames emitted (D1)
- where `Lengther` reports `known`, the drained total matches it (D1)

Bundling the D1 assertions here is deliberate: both decisions constrain the same
method set, and one test suite covering both is cheaper to write and harder to
let rot.

---

### D6 — Level and loudness units are distinct named types

**Resolves:** [FILT-9](current_gaps.md#filt-9-no-loudness-measurement-or-telephony-level-conventions)

**Impact:** BREAKING — introduces types used in existing signatures, and changes
`WithGain`.

#### Decision

Linear amplitude is canonical; decibel forms are **distinct named types**, not
bare floats.

```go
type Gain float64   // linear multiplier; 1.0 = unity
type DBFS float64   // peak-referenced; 0 = full-scale peak
type DBov float64   // RMS-referenced; 0 = RMS of a full-scale square wave
type LUFS float64   // ITU-R BS.1770, K-weighted and gated
```

These live in `audio`, since they are audio-domain values rather than PCM
geometry — independent of how
[API-16](current_gaps.md#api-16-utils-is-a-grab-bag-package) resolves.

`float64` rather than `float32`: these are control and measurement scalars, not
sample data, and logarithmic arithmetic benefits from the headroom. `Gain`
converts when multiplied into `float32` samples, which is free.

`WithGain` becomes typed:

```go
func WithGain(func(ch audio.Channel) Gain) ChannelOption
func GainFromDB(db float64) Gain
```

And normalisation splits, with the unit in the function name:

```go
func WithPeakNormalize(func(ch audio.Channel) (DBFS, bool)) ChannelOption
// v1.x, purely additive once LUFS measurement exists:
func WithLoudnessNormalize(func(ch audio.Channel) (LUFS, bool)) ChannelOption
```

`WithNormalize` is deprecated in favour of the explicit pair.

#### Why typed rather than `float64`

The same signal reports differently under each convention. Measured:

| Signal | peak dBFS | RMS dBFS / dBov |
|---|---|---|
| full-scale sine | 0.00 | **−3.01** |
| full-scale square | 0.00 | 0.00 |
| speech, typical | −6.02 | −26.02 |

**A full-scale sine is 0 dBFS peak but −3.01 dB RMS.** So `level = 0` and
`level = -3.01` can describe the same signal, and a bare `float64` carries none
of the distinction. Four reasons the types earn their keep:

1. The audience splits. Audio practitioners read a bare float as dBFS;
   telephony practitioners read it as dBm0 or dBov. Both are right half the
   time.
2. Measured levels are **returned** — by
   [DSP-10](current_gaps.md#dsp-10-no-analysis-primitives) analysis,
   [OBS-2](current_gaps.md#obs-2-no-processing-statistics) statistics and
   [PBX-3](current_gaps.md#pbx-3-no-prompt-set-validation) validation — and an
   untyped return leaves the caller guessing.
3. It is cheap: a named `float64` with `String()` and explicit conversions.
4. **It lets the library refuse a conversion it cannot make.**

#### The fourth reason, which decided it: dBm0

| Unit | Reference | Derivable from samples? |
|---|---|---|
| `Gain` | 1.0 = unity | n/a — a control, not a measurement |
| `DBFS` | full-scale peak | yes |
| `DBov` | full-scale square RMS | yes |
| `LUFS` | BS.1770, gated | yes, given a K-weighting filter |
| dBm0 | network 0 TLP | **no** |

dBm0 requires an external reference, and the G.711 overload point differs
between A-law and µ-law, so the library cannot choose one. Typed units let that
be expressed as a requirement on the caller:

```go
func (d DBov) ToDBm0(overload DBm0) DBm0
```

An untyped API has no way to state that refusal — it would have to pick a
convention silently and be wrong on half the networks.

A `DBm0` type is **not** part of v1. Adding an exported type later is purely
additive, so it lands when the conversion is actually implemented.

#### Why this decision is cheap

v1 needs the **types defined and used in signatures** — that is the breaking
part, and it is naming rather than signal processing. It does not need any of
the implementations:

- LUFS measurement needs the K-weighting biquads and gating, which is
  [FILT-1](current_gaps.md#filt-1-no-filter-framework) plus
  [DSP-10](current_gaps.md#dsp-10-no-analysis-primitives) work — ADDITIVE
- AGC ([FILT-4](current_gaps.md#filt-4-no-automatic-gain-control)), limiter
  ([FILT-5](current_gaps.md#filt-5-no-compressor-or-limiter)), gate
  ([FILT-6](current_gaps.md#filt-6-no-noise-gate-or-noise-reduction)) — ADDITIVE
- `WithLoudnessNormalize` — ADDITIVE, and the signature is already reserved

#### Work implied

- Define the four types with `String()` methods
- Conversion functions in both directions where they are defined:
  `GainFromDB`, `Gain.DB()`, `DBFS` and `DBov` to and from linear
- Document explicitly **which conversions exist and which deliberately do not**
- Change `WithGain` to return `Gain`
- Add `WithPeakNormalize`; deprecate `WithNormalize`

#### What it unblocks

Gives a settled vocabulary to
[FILT-4](current_gaps.md#filt-4-no-automatic-gain-control) through
[FILT-6](current_gaps.md#filt-6-no-noise-gate-or-noise-reduction),
[DSP-6](current_gaps.md#dsp-6-downmix-is-unweighted-averaging-only) downmix
coefficients, [DSP-10](current_gaps.md#dsp-10-no-analysis-primitives),
[OBS-2](current_gaps.md#obs-2-no-processing-statistics) and
[PBX-3](current_gaps.md#pbx-3-no-prompt-set-validation) — and makes
[DSP-8](current_gaps.md#dsp-8-normalization-buffers-the-entire-stream)'s
peak-versus-loudness distinction expressible rather than implicit.

---

### D7 — Resource limits are enforced by default, and configurable

**Resolves:** [RES-1](current_gaps.md#res-1-a-62-byte-file-can-force-gigabytes-of-allocation),
[RES-2](current_gaps.md#res-2-untrusted-input-is-read-into-memory-without-bound),
[RES-3](current_gaps.md#res-3-channel-splitting-buffers-unread-channels-without-bound)

**Impact: ADDITIVE at the interface level** — unusually for this list. All four
decoders are empty structs, so limits become fields and the `Decoder` interface
is untouched.

#### Decision

Limits are **enforced by default** with safe defaults, and **configurable** per
decoder:

```go
type Decoder struct {
    MaxInputBytes    int64 // 0 = library default
    MaxDeclaredChunk int64 // reject headers claiming more than the input holds
}
```

`wav.Decoder{}` keeps working. Registry users set limits at registration.

Exceeding a limit returns a **distinct sentinel error**, so callers can
distinguish hostile or malformed input from an I/O failure. The exact error
shape is deferred to
[API-13](current_gaps.md#api-13-inconsistent-error-construction), which is still
open.

For [RES-3](current_gaps.md#res-3-channel-splitting-buffers-unread-channels-without-bound),
channel splitting gets two modes:

- **Seekable source** — re-read the stream per channel. **Zero buffering.**
- **Non-seekable** — bounded buffer, distinct error when exceeded, documented as
  "read channels concurrently, or supply a seekable source."

That resolution was not available before D1; it uses `FrameSeeker` directly.

#### Why enforce rather than document

1. **The caller cannot know they are exposed.**
   [RES-1](current_gaps.md#res-1-a-62-byte-file-can-force-gigabytes-of-allocation)
   returns *no error at all* — a 62-byte file drives a 191 MB allocation and
   `Decode` reports success.
2. "Callers must bound their own input" means every caller writes the same
   wrapper, and most will not.
3. Defaults should be safe. An escape hatch serves the minority who genuinely
   need unbounded reads.

Configurable rather than fixed, because a library guessing at a constant will be
wrong — this repository's own 167 MB test asset would trip a conservative
default.

#### Sequencing: two of the three are symptoms of not owning the parser

This matters for planning, because "fix RES-1" sounds smaller than it is.

**RES-1** — the allocation happens *inside* `go-audio/aiff`'s `FwdToPCM`, which
`aiff.Decode` calls on every decode. It cannot be guarded from outside:
`make([]byte, offset)` executes **before any read**, so an `io.LimitReader` does
not help — that bounds reads, not allocations.

The only external mitigation is to **pre-walk the chunk headers and validate
declared sizes against the actual input before delegating.** That is 50–100
lines, and it is the first component of
[FMT-21](current_gaps.md#fmt-21-what-owning-the-wav-and-aiff-container-layer-requires)
— so the interim fix is also progress toward the permanent one.

**RES-2** — `io.ReadAll` exists *only* because `go-audio` requires an
`io.ReadSeeker`. Once the parser is owned, a forward-only stream decodes
natively and simply does not offer `Lengther` or `FrameSeeker` — the D1 pattern
working as designed. The fallback disappears rather than being capped. Interim:
bound it.

**RES-3** — independent of the dependency, and fixable immediately.

#### A gap in the audit this exposes

`govulncheck` will **not** catch
[RES-1](current_gaps.md#res-1-a-62-byte-file-can-force-gigabytes-of-allocation).
It is not in the Go vulnerability database, so the
[QA-8](current_gaps.md#qa-8-no-pre-release-bug-and-security-audit) audit would
pass clean while the defect sits on the hot path.

The step that would have caught it is **fuzzing**, which is why
[QA-5](current_gaps.md#qa-5-no-fuzzing-of-binary-parsers) is the highest-value
item in that section rather than a hygiene nicety. Vulnerability scanning finds
*known* problems; fuzzing finds yours.

#### Disposition of the upstream defect

Decided in [D8](#d8--archived-dependencies-with-known-security-defects-are-removed-not-mitigated):
the defect is addressed by **removing the dependency**, not by reporting it or
working around it. Reporting to the Go vulnerability database remains available
afterwards and is deliberately deferred, not dismissed.

---

### D8 — Archived dependencies with known security defects are removed, not mitigated

**Resolves:** the disposition of
[RES-1](current_gaps.md#res-1-a-62-byte-file-can-force-gigabytes-of-allocation),
and sets a standing rule for
[DEP-1](current_gaps.md#dep-1-five-of-seven-dependencies-are-archived) /
[DEP-2](current_gaps.md#dep-2-no-dependency-policy).

**Impact:** policy, plus a re-prioritisation of
[FMT-21](current_gaps.md#fmt-21-what-owning-the-wav-and-aiff-container-layer-requires).

#### Decision

**A validated chunk walker plus owned WAV and AIFF header parsing takes
precedence over all other feature work**, and is treated as a security fix
rather than a refactor.

The rule this sets, recorded in
[README.markdown](README.markdown#dependency-policy-native-go-first):

> An archived dependency with a known security defect is not acceptable.
> Removing it takes precedence over feature work. Mitigating around it, or
> documenting it as a caveat, is not sufficient — depending on unmaintained code
> with a known unfixable flaw and telling users about it is worse advice than
> not depending on it.

#### Why removal rather than reporting or mitigating

Three reasons, in order of weight:

1. **The upstream cannot be fixed.** `go-audio/aiff` and `go-audio/wav` were
   archived on 2026-02-21. No report, patch or advisory changes that. Mitigation
   would be permanent, not temporary.
2. **Mitigating from outside is barely possible.** The allocation executes
   *before* any read — `make([]byte, offset)` — so an `io.LimitReader` cannot
   help. The only external guard is to pre-walk the chunk headers and validate
   declared sizes before delegating, which is already the first component of
   FMT-21. The interim fix and the permanent fix are the same code.
3. **A library should not recommend what it would not do.** The project's own
   dependency policy makes maintenance status part of the decision. Continuing
   to depend on an archived package with a known unfixable defect, while
   documenting that fact, would contradict the policy it publishes.

#### Scope: both formats, not AIFF alone

Initially scoped to AIFF, then widened on evidence. **The same defect class is
present and reachable in the WAV path.**

`riff.Chunk.DecodeWavHeader` performs `extra := make([]byte, ch.Size-16)`, where
`ch.Size` is the `fmt ` chunk size read from the file. It is reached via
`readHeaders`, which `NewDecoder`, `IsValidFile` and `FwdToPCM` all trigger.
Measured:

```
file=48 bytes  declared fmt size=16           allocated=    0.0 MB  err=<nil>
file=48 bytes  declared fmt size=100000000    allocated=  190.8 MB  err=...EOF
```

Roughly 4,000,000x amplification, and a `uint32` field so the ceiling is about
4 GB. WAV differs from AIFF only in returning an error — after the allocation has
already happened.

Three further instances exist in `cue_chunk.go`, `list_chunk.go` and
`smpl_chunk.go`. Those sit behind `ReadMetadata()`, which `audpbx` never calls,
so they are latent — but they would become live the moment
[FMT-10](current_gaps.md#fmt-10-no-metadata-or-tag-handling) metadata support was
wired against the existing dependency. That is an additional reason not to build
new capability on this code.

**Consequently the first slice is the shared, size-validating chunk walker**,
not the AIFF container specifically. Both formats are chunked containers
differing only in magic and endianness, so one walker that validates every
declared size against remaining input closes both instances at once — and it is
the foundation for the rest of FMT-21.

#### What this decision does not do

Removing the dependency protects `audpbx` users. It does **not** help anyone
else using `go-audio/wav` or `go-audio/aiff`, and those packages had real
adoption. Reporting to the Go vulnerability database is the only mechanism that
would warn them, and it remains available once `audpbx` is clean — at which
point the report no longer flags this project's own builds.

Recorded as deliberately deferred rather than declined.

---

### D9 — Per-dependency strategy

**Resolves:** [DEP-1](current_gaps.md#dep-1-five-of-seven-dependencies-are-archived)

**Impact:** ADDITIVE at the interface level; the work is substantial but changes
no exported signature.

#### Decision

| Dependency | Route |
|---|---|
| `go-audio/wav`, `go-audio/aiff`, `go-audio/audio`, `go-audio/riff` | **Removed.** Container layer owned — [D8](#d8--archived-dependencies-with-known-security-defects-are-removed-not-mitigated) |
| `hajimehoshi/go-mp3` | **Replaced by vendored `minimp3` (CC0), within v1** |
| `jfreymuth/oggvorbis`, `jfreymuth/vorbis` | **Kept.** Both active; re-checked each release |
| G.711 | Adopt `zaf/g711` (BSD-3-Clause) |
| G.722 | Evaluate `livekit/media-sdk/g722`, else write a standalone module |
| G.726 | Write a standalone module — nothing suitable exists |
| G.723.1, G.729 | libavcodec, in a separate module — [D11](#d11--cgo-lives-in-separate-modules-not-behind-build-tags) |
| Speex, iLBC | Dropped |

Per [the scope note](current_gaps.md#scope-what-this-project-is-and-is-not),
codecs written rather than adopted live in their own modules that `audpbx`
imports.

#### Why `go-mp3` goes inside v1 rather than later

[D8](#d8--archived-dependencies-with-known-security-defects-are-removed-not-mitigated)
does **not** compel this. Its rule triggers on *archived **and** a known
security defect*; `go-mp3` is archived (April 2023) but no defect has been found
in it, so only the softer "maintenance status is part of the decision" rule
applies.

Replacing it anyway, for two reasons:

1. It is a **binary parser fed untrusted input that will never receive a
   security fix.** The absence of a known defect reflects the absence of
   anyone looking.
2. **[QA-5](current_gaps.md#qa-5-no-fuzzing-of-binary-parsers) fuzzing will
   drive it.** Fuzzing the MP3 wrapper exercises `go-mp3`'s frame parser, and if
   it finds anything then D8 triggers and forces the removal on a worse
   schedule — after a release rather than before one.

Better to know before v1 than after. The replacement is also cheap: `minimp3` is
vendored CC0 source, so it needs no system library, no build tags, and no
licence obligation, and it supplies the seek and total-sample support that
`go-mp3` withholds ([D1](#d1--source-capability-surface-length-position-seeking)).

---

### D10 — Dependency governance is a committed artifact, not prose

**Resolves:** [DEP-2](current_gaps.md#dep-2-no-dependency-policy)

**Impact:** INTERNAL.

#### Decision

- A committed **`DEPENDENCIES.md`** records, per direct dependency: purpose,
  licence, maintenance status and last release. Not a table buried in an
  analysis document, which rots.
- **`google/go-licenses` runs in CI** to check the licence half automatically.
- **Licence review covers direct dependencies only**, reviewed at each release
  as part of [QA-8](current_gaps.md#qa-8-no-pre-release-bug-and-security-audit).
  Transitive dependencies are pinned by `go.sum` but not licence-tracked; this
  is a stated limit rather than an oversight.
- Archival and last-release status is re-checked at each release. This is the
  check whose absence let
  [DEP-1](current_gaps.md#dep-1-five-of-seven-dependencies-are-archived) go
  unnoticed — five of seven dependencies archived, one of them for over three
  years.

The rules that govern *adoption* live in
[README.markdown](README.markdown#dependency-policy-native-go-first), where
users can see them. This decision covers the artifacts that keep them honest.

---

### D11 — cgo lives in separate modules, not behind build tags

**Resolves:** [FMT-5](current_gaps.md#fmt-5-aac-support-user-6)

**Impact:** architectural; determines how every future cgo-backed codec is
delivered.

#### Decision

cgo-backed codecs live in **separate Go modules** that `audpbx` does not import.
Build tags operate *inside* those modules, selecting static, dynamic or
system linking.

```go
// The user opts in by importing the module at all.
reg.Register("g729", audpbxlibav.G729Decoder{})
```

`audpbx` core needs no build tags, and the existing `Registry` is already the
integration point.

#### Why a module boundary rather than build tags

Build tags were the obvious answer and they do not work for this purpose.
Measured: a file behind `//go:build aac_dynamic` importing an external package
still puts that package in `go.mod` after `go mod tidy`, and in
`go list -m all`:

```
optional.go is behind //go:build aac_dynamic
default build compiles fine, the code is excluded
BUT github.com/google/uuid appears in go.mod and in `go list -m all`
```

**Build tags control compilation, not the dependency graph.** `go mod tidy`
considers all build configurations, so with tags alone:

- the dependency is pinned in every consumer's `go.sum`
- `go mod download` fetches it whether or not anyone enables the tag
- **a licence scanner flags it** — so an LGPL binding would appear in every
  downstream licence audit even when unused

That last consequence defeats the stated posture that no LGPL obligation reaches
downstream users unless they opt in
([README](README.markdown#dependency-policy-native-go-first),
[DEP-2](current_gaps.md#dep-2-no-dependency-policy)). A module boundary delivers
it; a build tag only appears to.

#### A question this dissolves

An earlier draft asked whether, without the build tag, `aac.Decoder{}` should
exist and return an error or fail to compile — and argued for the former to keep
registry registration uniform.

With module separation the question is moot: the type does not exist unless the
module is imported. Cleaner than either option, and it removes a category of
"configured but non-functional" state.

#### Where the three AAC build modes live

Inside the cgo module, not in `audpbx`. Static linking, static building and
dynamic library selection are properties of how that module binds to its C
library, and they are invisible to anyone who does not import it.

---

### D12 — API stability policy, Go version range, deprecation

**Resolves:** the stability half of
[DOC-1](current_gaps.md#doc-1-to-doc-4)

**Impact:** this decision **defines what BREAKING means**, and so retroactively
underwrites the impact classification used throughout
[current_gaps.md](current_gaps.md#how-to-read-this) and every decision in this
document.

#### The v1 promise

No breaking change to the exported API within v1.x. Breaking means:

- adding a method to an exported interface
- removing or renaming an exported identifier
- changing a signature
- tightening a documented contract

#### What the promise explicitly excludes

This list carries more weight than the promise, because an unstated exclusion
becomes an implied guarantee:

- **`internal/` packages** — `internal/audiotest`, `internal/perfbench`
- **performance characteristics** — throughput, allocation counts, latency
- **unexported behaviour**
- **error strings** — only sentinel *identity* is stable, and the exact form of
  that awaits [API-13](current_gaps.md#api-13-inconsistent-error-construction)
- **sample-exact output** — *pending*
  [DSP-1/DSP-2](current_gaps.md#dsp-2-no-resampler-algorithm-selection-user-4).
  Recorded here as an open exclusion rather than decided as a side effect,
  because it deserves its own decision: without it every DSP improvement is a
  breaking change, and with it callers cannot rely on byte-identical output
  across versions.

#### Pre-v1

**No guarantees at all.** Stated plainly rather than implied by the version
number, since a reader may otherwise assume `v0.0.3` carries some stability.

#### Deprecation

A `// Deprecated:` marker, which keeps working for the life of the major and is
removed only in the next major. So `WithNormalize` (deprecated by
[D6](#d6--level-and-loudness-units-are-distinct-named-types)) and `WriteWAV16`
(if superseded per
[API-6](current_gaps.md#api-6-writewav16-is-permanently-mono)) both survive the
whole of v1.

#### Go version range

**`go 1.24`**, and the **support policy is the two most recent Go majors**,
matching Go's own security-support window.

Two findings behind the change from the current declaration:

1. The real floor is **1.24**, not 1.25. The newest feature actually used is
   `b.Loop()` (12 files); next is range-over-int at 1.22 (30 files); nothing
   from 1.23 or 1.25 appears at all.
2. **All four module files declare `go 1.25.5` — a patch version.** That
   excludes anyone on 1.25.0 through 1.25.4 for no benefit. It reads as
   unintended rather than deliberate.

`b.Loop()` is the only thing holding the floor at 1.24, and it is test-only.
Dropping it in favour of `for i := 0; i < b.N; i++` would allow 1.22. Kept
deliberately: it prevents the compiler eliding loop bodies and handles timer
management, both of which matter given how much benchmark trouble this project
has already had ([QA-8](current_gaps.md#qa-8-no-pre-release-bug-and-security-audit)
records a benchmark that reported 36 ns/op for a 100,000-frame resample).

#### Location

The **README**, alongside [Scope](README.markdown#scope) and the
[dependency policy](README.markdown#dependency-policy-native-go-first), so all
policy is in one discoverable place rather than split across a separate file
that is easier to link and easier to miss.

#### Work implied

- Change `go 1.25.5` to `go 1.24` in `go.mod`, both example module files, and
  `go.work`
- Write the README stability section
- Add `// Deprecated:` markers where D6 requires them
- **Enforce it**: `gorelease` in CI per
  [QA-8](current_gaps.md#qa-8-no-pre-release-bug-and-security-audit). A
  stability promise without an automated check is an intention.

---

### D13 — CHANGELOG and CONTRIBUTING

**Resolves:** [DOC-2 and DOC-3](current_gaps.md#doc-1-to-doc-4)

**Impact:** INTERNAL.

#### CHANGELOG.md

Keep a Changelog conventions — Added / Changed / Deprecated / Removed / Fixed /
**Security** — hand-written rather than generated, because the value is
explaining *why* and a commit log cannot.

The **Security** section is retained deliberately: it is where a fix like
[D8](#d8--archived-dependencies-with-known-security-defects-are-removed-not-mitigated)'s
dependency removal gets announced, and
[RES-1](current_gaps.md#res-1-a-62-byte-file-can-force-gigabytes-of-allocation)
is exactly the kind of change a consumer needs to notice.

**Starts fresh.** One summary line covers pre-history; proper entries begin at
the next release, with an `Unreleased` section maintained as work lands.
Reconstructing `v0.0.1`–`v0.0.3` from git archaeology is effort with little
payoff.

#### CONTRIBUTING.md

Must state:

| Topic | Why it belongs there |
|---|---|
| Supported Go versions | From [D12](#d12--api-stability-policy-go-version-range-deprecation) |
| gofmt required; linter configured | Depends on [QA-3](current_gaps.md#qa-3-no-lint-gate-and-ten-unformatted-files) landing first, or the gate fails immediately on ten existing files |
| **How to obtain test assets** | `go test ./...` currently fails ([QA-1](current_gaps.md#qa-1-the-test-suite-fails-out-of-the-box)) and the large files are gitignored behind `manage_testdata.sh`. A contributor's first command failing is the worst possible introduction. |
| What a PR needs | Tests; and for any new `Source`, passing the conformance suite from [D5](#d5--readsamples-follows-ioreader-semantics) |
| **The scope boundary** | Codecs are integrated, not implemented. Someone could reasonably write a G.729 implementation and submit it, and it would be declined on scope — far better they know before writing it. |
| New dependencies need justification | Against the [README dependency policy](README.markdown#dependency-policy-native-go-first) |

No CLA or DCO. For a project this size that is friction without much return.

---

## What D1 and D2 unblock

Now decided, without further decisions:

| Inventory entry | Effect |
|---|---|
| [OBS-1](current_gaps.md#obs-1-no-progress-reporting-user-3) | Progress becomes implementable — `Position` over `Lengther`, with a documented fallback when length is unknown |
| [OBS-5](current_gaps.md#obs-5-processing-state-cannot-be-checkpointed-or-restored) | Tier 2 resume (pre-roll re-priming) becomes possible: it needs exactly seek plus position |
| [DSP-8](current_gaps.md#dsp-8-normalization-buffers-the-entire-stream) | Two-pass normalisation becomes possible via `FrameSeeker`, removing the unbounded buffer |
| [DSP-4](current_gaps.md#dsp-4-the-library-can-decompose-audio-but-not-compose-it) | `Trim` can seek rather than decode-and-discard |
| [TONE-*](current_gaps.md#h-telephony-signalling-and-tones), [FILT-*](current_gaps.md#i-filters-dynamics-and-enhancement) | Have a settled vocabulary to be designed against |
| [FMT-21](current_gaps.md#fmt-21-what-owning-the-wav-and-aiff-container-layer-requires) | The container layer must surface the `fact` chunk, since it is the only length source for non-PCM payloads |

### Work these imply

Bounded and mechanical, suitable for an early v0.0.x slice:

- Add `Position()` to every `Source` implementation: the four decoders,
  `Resampler`, `MonoMixer`, `ChannelSource`, and the `audiotest` mock
- Implement `Lengther` where the transport allows, returning `known == false`
  where it does not
- Implement `FrameSeeker` where the underlying reader is seekable
- Add `Duration` and `FramesForDuration` helpers
- Add a conformance test asserting the contract across every implementation

---

## Decisions outstanding

Eleven of the inventory's seventeen are resolved above, plus one standing
policy decision (D8) and the documentation artifacts (D13). The rest are listed in
rough dependency order; the ordering is provisional and has already been
corrected once — see the note below.

**No dependencies — can be taken in any order:**

| Decision | Inventory |
|---|---|

**Gated by something above:**

| Decision | Gated on | Inventory |
|---|---|---|
| How is cancellation expressed? | — | [API-2](current_gaps.md#api-2-no-cancellation-mechanism) |
| How is progress delivered; is there a task handle? | cancellation | [OBS-1](current_gaps.md#obs-1-no-progress-reporting-user-3), [OBS-4](current_gaps.md#obs-4-no-task-abstraction) |
| What are the codec abstractions? | cgo policy | [API-5](current_gaps.md#api-5-no-encoder-abstraction), [API-17](current_gaps.md#api-17-decoder-cannot-express-framed-codecs) |
| Where does metadata live? | codec abstractions | [API-19](current_gaps.md#api-19-tag-manipulation-has-no-home-in-the-api) |
| What may callers rely on regarding output samples? | — | [DSP-1](current_gaps.md#dsp-1-anti-aliasing-filter-is-structurally-inadequate), [DSP-2](current_gaps.md#dsp-2-no-resampler-algorithm-selection-user-4) |
| Rules when combining mismatched sources | unit vocabulary (**resolved**) | [DSP-4](current_gaps.md#dsp-4-the-library-can-decompose-audio-but-not-compose-it) |
| Which naming and shape cleanups happen? | — | [API-9](current_gaps.md#api-9-registry-is-string-keyed-and-its-zero-value-panics)–[API-16](current_gaps.md#api-16-utils-is-a-grab-bag-package), [API-21](current_gaps.md#api-21-channel-is-uint32-capping-the-library-at-32-channels) |

### A note on ordering

The dependency order above is provisional, and D2 is the evidence. It was
originally placed *downstream* of D1, on the assumption that the unit vocabulary
was a surface concern. In fact D1's signatures could not be written without it —
`Position() int64` is meaningless until "of what?" is answered.

Expect more of this. Each decision should be re-checked against the remaining
order once taken, rather than following a plan fixed in advance.

---

## Process

Decisions reference the inventory by **gap ID** (`API-1`, `PBX-1`, …), never by
the position of an item in that document's decision list. Gap IDs are stable;
list positions shift whenever an entry is inserted, and an earlier draft of this
file carried five stale citations as a result.

Decisions are taken one at a time, in discussion, and appended here once
settled. Each entry records the decision, the reasoning, and the evidence — so
that a decision can be revisited on its merits later rather than re-argued from
memory.

This document does not list what to *implement* in v1. It records what must be
*decided* before the API can be frozen. Implementation planning follows from the
decisions, not the reverse.

---

## Former README TODO

The README no longer carries a TODO list. Those items were already in
[current_gaps.md](current_gaps.md). Mapping:

| Former README item | Inventory / decision |
|---|---|
| Opus | [FMT-4](current_gaps.md#fmt-4-telco-and-voip-codecs-absent-user-2), [FMT-14](current_gaps.md#fmt-14-opus-in-ogg-is-not-the-same-as-vorbis-in-ogg), [D11](#d11--cgo-lives-in-separate-modules-not-behind-build-tags) |
| AAC in a separate module, registered | [FMT-5](current_gaps.md#fmt-5-aac-support-user-6), [D11](#d11--cgo-lives-in-separate-modules-not-behind-build-tags) |
| Additional audio test files | [QA-7](current_gaps.md#qa-7-no-codec-reference-vectors) |
| Selectable resampling algorithm | [DSP-2](current_gaps.md#dsp-2-no-resampler-algorithm-selection-user-4) — design notes below |
| `WAVE_FORMAT_EXTENSIBLE` and bit depths other than 16 | [FMT-2](current_gaps.md#fmt-2-wav-decoding-is-narrow-user-5) |
| Replace `go-mp3` with vendored minimp3 | [DEP-1](current_gaps.md#dep-1-five-of-seven-direct-dependencies-are-archived), [D9](#d9--per-dependency-strategy) |
| Faster Ogg decoding | [DEP-1](current_gaps.md#dep-1-five-of-seven-direct-dependencies-are-archived) (Vorbis kept) |

### DSP-2 design notes (moved from the README)

Allow the caller to choose the resampling method rather than hard-coding cubic
interpolation, trading throughput against fidelity per use case:

| Method | Characteristics |
|--------|-----------------|
| Linear | Cheapest; adequate when the source is already band-limited or when latency dominates |
| Cubic (Catmull-Rom) | Current behaviour; good general-purpose default |
| Polyphase FIR | Highest fidelity; proper band-limiting for large decimation ratios |

- Cubic should remain the default so existing callers are unaffected. A
  functional option on `NewResampler` (for example `WithInterpolator`) keeps
  the current signature valid.
- The polyphase option also resolves the anti-aliasing shortfall in
  [DSP-1](current_gaps.md#dsp-1-anti-aliasing-filter-is-structurally-inadequate)
  and the README [Limitations](README.markdown#limitations). A polyphase FIR
  performs band-limiting and rate conversion in a single operation, so it
  replaces both the cubic interpolator and the one-pole filter rather than
  being layered on top of them.
- Common telephony conversions are exact rational ratios (44.1 kHz to 8 kHz
  is 441/80), so a polyphase implementation can precompute one coefficient
  set per phase and evaluate only the output samples actually required.
- Selecting a method changes output samples. The bit-exact comparisons and
  the quality measurements in
  [examples/profile_resampler/PROFILING_GUIDE.md](examples/profile_resampler/PROFILING_GUIDE.md)
  should be extended to cover each method independently.

This is a design placeholder; implementation is not implied by recording it
here.
