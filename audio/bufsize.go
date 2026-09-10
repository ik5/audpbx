// SPDX-License-Identifier: EPL-2.0

package audio

// DefaultBufSize is the sample count a Source reports from BufSize when it has
// no better figure to offer, and a sensible default for callers sizing their
// own read buffers.
//
// It is a hint, not a limit: every Source accepts any buffer length, and
// Resampler pulls from its source in blocks of its own choosing regardless of
// what BufSize reports.
const DefaultBufSize = 4096
