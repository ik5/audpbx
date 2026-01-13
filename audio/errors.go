// SPDX-License-Identifier: EPL-2.0

package audio

import "errors"

var (
	ErrInvalidDstSize = errors.New("dst size must be multiple of channels")
)
