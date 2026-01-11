package audio

import (
	"errors"
	"io"
	"testing"
)

// mockDecoder is a test decoder implementation
type mockDecoder struct {
	name string
}

func (d *mockDecoder) Decode(r io.Reader) (Source, error) {
	return newSilentSource(44100, 2, 100), nil
}

// failingDecoder always returns an error
type failingDecoder struct{}

func (d *failingDecoder) Decode(r io.Reader) (Source, error) {
	return nil, errors.New("decode failed")
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	decoder := &mockDecoder{name: "wav"}

	registry.Register("wav", decoder)

	got, ok := registry.Get("wav")
	if !ok {
		t.Fatal("Registry.Get() failed to retrieve registered decoder")
	}

	if got != decoder {
		t.Error("Registry.Get() returned different decoder instance")
	}
}

func TestRegistry_GetNonExistent(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Registry.Get() returned ok=true for non-existent format")
	}
}

func TestRegistry_MultipleFormats(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	wavDecoder := &mockDecoder{name: "wav"}
	mp3Decoder := &mockDecoder{name: "mp3"}
	oggDecoder := &mockDecoder{name: "ogg"}

	registry.Register("wav", wavDecoder)
	registry.Register("mp3", mp3Decoder)
	registry.Register("ogg", oggDecoder)

	tests := []struct {
		format  string
		want    Decoder
		wantOK  bool
	}{
		{"wav", wavDecoder, true},
		{"mp3", mp3Decoder, true},
		{"ogg", oggDecoder, true},
		{"flac", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got, ok := registry.Get(tt.format)
			if ok != tt.wantOK {
				t.Errorf("Registry.Get(%q) ok = %v, want %v", tt.format, ok, tt.wantOK)
			}
			if tt.wantOK && got != tt.want {
				t.Errorf("Registry.Get(%q) returned wrong decoder", tt.format)
			}
		})
	}
}

func TestRegistry_Overwrite(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	decoder1 := &mockDecoder{name: "first"}
	decoder2 := &mockDecoder{name: "second"}

	registry.Register("wav", decoder1)
	registry.Register("wav", decoder2)

	got, ok := registry.Get("wav")
	if !ok {
		t.Fatal("Registry.Get() failed after overwrite")
	}

	if got != decoder2 {
		t.Error("Registry.Get() did not return the overwritten decoder")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	decoder := &mockDecoder{name: "test"}

	// Register concurrently
	done := make(chan bool)
	for i := range 10 {
		go func(id int) {
			registry.Register("format", decoder)
			done <- true
		}(i)
	}

	// Get concurrently
	for i := range 10 {
		go func(id int) {
			_, _ = registry.Get("format")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range 20 {
		<-done
	}

	// Verify the decoder is registered
	got, ok := registry.Get("format")
	if !ok {
		t.Error("Registry.Get() failed after concurrent operations")
	}
	if got != decoder {
		t.Error("Registry returned wrong decoder after concurrent operations")
	}
}

func TestRegistry_EmptyFormatName(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	decoder := &mockDecoder{name: "test"}

	// Empty string as format name should work (no validation in current impl)
	registry.Register("", decoder)

	got, ok := registry.Get("")
	if !ok {
		t.Error("Registry.Get(\"\") failed for empty format name")
	}
	if got != decoder {
		t.Error("Registry.Get(\"\") returned wrong decoder")
	}
}

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if registry.codecs == nil {
		t.Error("NewRegistry() did not initialize codecs map")
	}

	if registry.mtx == nil {
		t.Error("NewRegistry() did not initialize mutex")
	}
}

// BenchmarkRegistry_Register benchmarks registering decoders
func BenchmarkRegistry_Register(b *testing.B) {
	registry := NewRegistry()
	decoder := &mockDecoder{}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		registry.Register("wav", decoder)
	}
}

// BenchmarkRegistry_Get benchmarks retrieving decoders
func BenchmarkRegistry_Get(b *testing.B) {
	registry := NewRegistry()
	decoder := &mockDecoder{}
	registry.Register("wav", decoder)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = registry.Get("wav")
	}
}

// BenchmarkRegistry_GetMiss benchmarks cache misses
func BenchmarkRegistry_GetMiss(b *testing.B) {
	registry := NewRegistry()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = registry.Get("nonexistent")
	}
}

// BenchmarkRegistry_ConcurrentRegisterGet benchmarks concurrent operations
func BenchmarkRegistry_ConcurrentRegisterGet(b *testing.B) {
	registry := NewRegistry()
	decoder := &mockDecoder{}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				registry.Register("wav", decoder)
			} else {
				_, _ = registry.Get("wav")
			}
			i++
		}
	})
}
