package aiff

import "errors"

var (
	// ErrNotAiffFile indicates the file is not a valid AIFF file
	ErrNotAiffFile = errors.New("not an AIFF file")

	// ErrOnlyPCM16bitSupported indicates only 16-bit PCM is supported
	ErrOnlyPCM16bitSupported = errors.New("only 16-bit PCM AIFF is supported")

	// ErrUnsupportedAiffLayout indicates an unsupported AIFF layout
	ErrUnsupportedAiffLayout = errors.New("unsupported AIFF layout")
)
