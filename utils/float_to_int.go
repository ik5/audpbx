package utils

func Float32ToInt16(x float32) int16 {
	// Clamp and scale
	if x > 1 {
		x = 1
	} else if x < -1 {
		x = -1
	}

	// Use 32767 for positive max to avoid overflow
	return int16(x * 32767.0)
}
