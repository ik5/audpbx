package audio

import "errors"

var (
	ErrInvalidDstSize = errors.New("dst size must be multiple of channels")
)
