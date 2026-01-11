package audio

import (
	"errors"
	"testing"
)

func TestErrInvalidDstSize(t *testing.T) {
	t.Parallel()

	if ErrInvalidDstSize == nil {
		t.Fatal("ErrInvalidDstSize is nil")
	}

	expectedMsg := "dst size must be multiple of channels"
	if ErrInvalidDstSize.Error() != expectedMsg {
		t.Errorf("ErrInvalidDstSize.Error() = %q, want %q", ErrInvalidDstSize.Error(), expectedMsg)
	}
}

func TestErrInvalidDstSize_IsError(t *testing.T) {
	t.Parallel()

	// Verify it implements error interface
	var err error = ErrInvalidDstSize
	if err == nil {
		t.Error("ErrInvalidDstSize does not implement error interface")
	}
}

func TestErrInvalidDstSize_Comparison(t *testing.T) {
	t.Parallel()

	// Test errors.Is compatibility
	err := ErrInvalidDstSize
	if !errors.Is(err, ErrInvalidDstSize) {
		t.Error("errors.Is() failed for ErrInvalidDstSize")
	}

	// Test with a different error
	otherErr := errors.New("some other error")
	if errors.Is(otherErr, ErrInvalidDstSize) {
		t.Error("errors.Is() should return false for different error")
	}
}

func TestErrInvalidDstSize_Wrapping(t *testing.T) {
	t.Parallel()

	// Test that wrapped error can be unwrapped
	wrappedErr := errors.Join(ErrInvalidDstSize, errors.New("additional context"))
	if !errors.Is(wrappedErr, ErrInvalidDstSize) {
		t.Error("errors.Is() failed for wrapped ErrInvalidDstSize")
	}
}
