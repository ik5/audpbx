// SPDX-License-Identifier: EPL-2.0

package utils

// CubicInterpolate performs cubic interpolation
// x is the fractional position between y1 and y2 (0 <= x <= 1)
// y0, y1, y2, y3 are four consecutive samples
func CubicInterpolate(y0, y1, y2, y3, x float32) float32 {
	// Catmull-Rom spline interpolation
	a0 := -0.5*y0 + 1.5*y1 - 1.5*y2 + 0.5*y3
	a1 := y0 - 2.5*y1 + 2*y2 - 0.5*y3
	a2 := -0.5*y0 + 0.5*y2
	a3 := y1

	return a0*x*x*x + a1*x*x + a2*x + a3
}
