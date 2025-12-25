package wav

import "errors"

var (
	ErrNotWavFile = errors.New("not a WAV file")
	ErrUnsupportedWavLayout = errors.New("unsupported WAV layout")
	ErrOnlyPCM16bitSupported = errors.New("only PCM 16-bit supported")
	ErrUnsupportedWavChunks =  errors.New("unsupported WAV chunks")
)
