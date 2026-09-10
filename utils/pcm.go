// SPDX-License-Identifier: EPL-2.0

package utils

// BitsPerByte is the number of bits in a byte, used to derive sample widths
// from bit depths.
const BitsPerByte = 8

// Supported PCM bit depths.
//
// These name the widths that decoders inspect when they normalize raw integer
// samples into float32.
const (
	// BitDepth8 is 8-bit PCM.
	BitDepth8 = 8
	// BitDepth16 is 16-bit PCM, the depth every decoder in this module accepts.
	BitDepth16 = 16
	// BitDepth24 is 24-bit PCM.
	BitDepth24 = 24
	// BitDepth32 is 32-bit PCM.
	BitDepth32 = 32
)

// Width in bytes of a single PCM sample at each supported bit depth.
const (
	// BytesPerSample8 is the width of one 8-bit PCM sample.
	BytesPerSample8 = BitDepth8 / BitsPerByte
	// BytesPerSample16 is the width of one 16-bit PCM sample.
	BytesPerSample16 = BitDepth16 / BitsPerByte
	// BytesPerSample24 is the width of one 24-bit PCM sample.
	BytesPerSample24 = BitDepth24 / BitsPerByte
	// BytesPerSample32 is the width of one 32-bit PCM sample.
	BytesPerSample32 = BitDepth32 / BitsPerByte
)

// Scaling factors that map a normalized float sample in [-1, 1] onto the
// integer range of each bit depth.
//
// Each is 2^(bits-1): the magnitude of the most negative representable sample.
// Dividing a raw integer sample by it yields a float in [-1, 1).
const (
	// SampleScale8 is 2^7, the scale for 8-bit PCM.
	SampleScale8 float32 = 128.0
	// SampleScale16 is 2^15, the scale for 16-bit PCM.
	SampleScale16 float32 = 32768.0
	// SampleScale24 is 2^23, the scale for 24-bit PCM.
	SampleScale24 float32 = 8388608.0
	// SampleScale32 is 2^31, the scale for 32-bit PCM.
	SampleScale32 float32 = 2147483648.0
)
