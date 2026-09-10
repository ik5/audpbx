// SPDX-License-Identifier: EPL-2.0

package utils

import "math"

// Float32ToInt16 converts a normalized float32 sample in [-1, 1] to 16-bit PCM.
// Values outside that range are clamped.
func Float32ToInt16(x float32) int16 {
	// Scale first, then clamp in the scaled domain.
	//
	// Clamping the input to 1.0 first is not enough: 1.0 * SampleScale16 is
	// 32768, which does not fit in an int16. The conversion then wraps to
	// -32768, so a full-scale positive sample comes out full-scale *negative* —
	// a polarity flip that clicks audibly on every clipped sample.
	v := x * SampleScale16

	if v >= math.MaxInt16 {
		return math.MaxInt16
	}

	if v <= math.MinInt16 {
		return math.MinInt16
	}

	return int16(v)
}
