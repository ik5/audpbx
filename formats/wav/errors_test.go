package wav

import (
	"errors"
	"testing"
)

func TestErrNotWavFile(t *testing.T) {
	t.Parallel()

	if ErrNotWavFile == nil {
		t.Fatal("ErrNotWavFile is nil")
	}

	expectedMsg := "not a WAV file"
	if ErrNotWavFile.Error() != expectedMsg {
		t.Errorf("ErrNotWavFile.Error() = %q, want %q", ErrNotWavFile.Error(), expectedMsg)
	}
}

func TestErrUnsupportedWavLayout(t *testing.T) {
	t.Parallel()

	if ErrUnsupportedWavLayout == nil {
		t.Fatal("ErrUnsupportedWavLayout is nil")
	}

	expectedMsg := "unsupported WAV layout"
	if ErrUnsupportedWavLayout.Error() != expectedMsg {
		t.Errorf("ErrUnsupportedWavLayout.Error() = %q, want %q", ErrUnsupportedWavLayout.Error(), expectedMsg)
	}
}

func TestErrOnlyPCM16bitSupported(t *testing.T) {
	t.Parallel()

	if ErrOnlyPCM16bitSupported == nil {
		t.Fatal("ErrOnlyPCM16bitSupported is nil")
	}

	expectedMsg := "only PCM 16-bit supported"
	if ErrOnlyPCM16bitSupported.Error() != expectedMsg {
		t.Errorf("ErrOnlyPCM16bitSupported.Error() = %q, want %q", ErrOnlyPCM16bitSupported.Error(), expectedMsg)
	}
}

func TestErrUnsupportedWavChunks(t *testing.T) {
	t.Parallel()

	if ErrUnsupportedWavChunks == nil {
		t.Fatal("ErrUnsupportedWavChunks is nil")
	}

	expectedMsg := "unsupported WAV chunks"
	if ErrUnsupportedWavChunks.Error() != expectedMsg {
		t.Errorf("ErrUnsupportedWavChunks.Error() = %q, want %q", ErrUnsupportedWavChunks.Error(), expectedMsg)
	}
}

func TestErrors_AreErrors(t *testing.T) {
	t.Parallel()

	// Verify all errors implement error interface
	var err error

	err = ErrNotWavFile
	if err == nil {
		t.Error("ErrNotWavFile does not implement error interface")
	}

	err = ErrUnsupportedWavLayout
	if err == nil {
		t.Error("ErrUnsupportedWavLayout does not implement error interface")
	}

	err = ErrOnlyPCM16bitSupported
	if err == nil {
		t.Error("ErrOnlyPCM16bitSupported does not implement error interface")
	}

	err = ErrUnsupportedWavChunks
	if err == nil {
		t.Error("ErrUnsupportedWavChunks does not implement error interface")
	}
}

func TestErrors_IsComparison(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{"ErrNotWavFile", ErrNotWavFile},
		{"ErrUnsupportedWavLayout", ErrUnsupportedWavLayout},
		{"ErrOnlyPCM16bitSupported", ErrOnlyPCM16bitSupported},
		{"ErrUnsupportedWavChunks", ErrUnsupportedWavChunks},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test errors.Is with same error
			if !errors.Is(tt.err, tt.err) {
				t.Errorf("errors.Is(%s, %s) = false, want true", tt.name, tt.name)
			}

			// Test errors.Is with different error
			otherErr := errors.New("some other error")
			if errors.Is(otherErr, tt.err) {
				t.Errorf("errors.Is(otherErr, %s) = true, want false", tt.name)
			}
		})
	}
}

func TestErrors_Wrapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{"ErrNotWavFile", ErrNotWavFile},
		{"ErrUnsupportedWavLayout", ErrUnsupportedWavLayout},
		{"ErrOnlyPCM16bitSupported", ErrOnlyPCM16bitSupported},
		{"ErrUnsupportedWavChunks", ErrUnsupportedWavChunks},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test that wrapped errors can be unwrapped
			wrappedErr := errors.Join(tt.err, errors.New("additional context"))
			if !errors.Is(wrappedErr, tt.err) {
				t.Errorf("errors.Is(wrappedErr, %s) = false, want true", tt.name)
			}
		})
	}
}

func TestErrors_Uniqueness(t *testing.T) {
	t.Parallel()

	// Ensure all error variables are distinct
	allErrors := []error{
		ErrNotWavFile,
		ErrUnsupportedWavLayout,
		ErrOnlyPCM16bitSupported,
		ErrUnsupportedWavChunks,
	}

	for i := range allErrors {
		for j := range allErrors {
			if i != j && allErrors[i] == allErrors[j] {
				t.Errorf("errors[%d] and errors[%d] are the same instance", i, j)
			}
		}
	}
}

func TestErrors_Messages(t *testing.T) {
	t.Parallel()

	// Ensure all errors have unique messages
	messages := make(map[string]error)
	allErrors := map[string]error{
		"ErrNotWavFile":            ErrNotWavFile,
		"ErrUnsupportedWavLayout":  ErrUnsupportedWavLayout,
		"ErrOnlyPCM16bitSupported": ErrOnlyPCM16bitSupported,
		"ErrUnsupportedWavChunks":  ErrUnsupportedWavChunks,
	}

	for name, err := range allErrors {
		msg := err.Error()
		if existing, found := messages[msg]; found {
			t.Errorf("%s has same message as %s: %q", name, existing, msg)
		}
		messages[msg] = err
	}
}
