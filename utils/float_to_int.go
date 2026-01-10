package utils

func Float32ToInt16(x float32) int16 {
	const maxInt16 float32 = 32768.0 // 2^15 -> +32767

	// Clamp and scale
	if x > 1 {
		x = 1
	}

	if x < -1 {
		x = -1
	}

	// Use 32767 for positive max to avoid overflow
	return int16(x * maxInt16)
}
