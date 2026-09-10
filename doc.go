// SPDX-License-Identifier: EPL-2.0

// Package audpbx provides high-level audio processing utilities for Go applications.
//
// This package offers convenient functions for common audio processing tasks such as
// resampling, format conversion, and decoding various audio formats. It's designed to
// be simple to use while maintaining good performance.
//
// # Supported Formats
//
// The package supports decoding the following audio formats:
//   - WAV (PCM 16-bit) via formats/wav
//   - MP3 via formats/mp3
//   - Ogg Vorbis via formats/vorbis
//   - AIFF (PCM 16-bit) via formats/aiff
//
// # Quick Start
//
// The simplest way to process audio is using ResampleToMono16:
//
//	// Decode an audio file
//	decoder := wav.Decoder{}
//	file, _ := os.Open("audio.wav")
//	src, _ := decoder.Decode(file)
//
//	// Resample to 8kHz mono, 16-bit PCM
//	samples, rate, _ := audpbx.ResampleToMono16(src, 8000, 4096)
//
//	// samples is now []int16 at 8kHz mono
//
// # High-Level Channel Processing
//
// For multi-channel audio (stereo, 5.1 surround, etc.), use the high-level API:
//
//	// Split stereo into separate mono files
//	result, _ := audpbx.ProcessChannels(src, audio.LayoutStereo,
//	    audpbx.WithSaveToFile(func(ch audio.Channel) string {
//	        return fmt.Sprintf("%s.wav", ch)
//	    }))
//
//	// Process with resampling and gain adjustment
//	result, _ := audpbx.ProcessChannels(src, audio.Layout5Point1,
//	    audpbx.WithChannels(audio.ChannelFrontLeft, audio.ChannelFrontRight),
//	    audpbx.WithResample(func(ch audio.Channel) int { return 48000 }),
//	    audpbx.WithGain(func(ch audio.Channel) float32 { return 1.5 }),
//	    audpbx.WithSaveToFile(func(ch audio.Channel) string {
//	        return fmt.Sprintf("output_%s_48k.wav", ch)
//	    }))
//
// Common audio processing options:
//   - Resample: Change sample rate (e.g., 44.1 kHz to 16 kHz)
//   - Gain: Adjust volume (1.0 = no change, 2.0 = double, 0.5 = half)
//   - Normalize: Automatically maximize volume without distortion
//   - Mixdown: Combine multiple channels into mono
//
// # Audio Processing Pipeline
//
// For more control, you can build custom audio processing pipelines using the
// audio subpackage:
//
//	// Create a resampler
//	resampler := audio.NewResampler(source, 16000)
//
//	// Convert to mono
//	mono := audio.NewMonoMixer(resampler)
//
//	// Read samples
//	buf := make([]float32, 4096)
//	n, err := mono.ReadSamples(buf)
//
// # Format Decoders
//
// Each format has its own decoder:
//
//	// WAV
//	wavDecoder := wav.Decoder{}
//	src, _ := wavDecoder.Decode(reader)
//
//	// MP3
//	mp3Decoder := mp3.Decoder{}
//	src, _ := mp3Decoder.Decode(reader)
//
//	// Vorbis
//	vorbisDecoder := vorbis.Decoder{}
//	src, _ := vorbisDecoder.Decode(reader)
//
//	// AIFF
//	aiffDecoder := aiff.Decoder{}
//	src, _ := aiffDecoder.Decode(reader)
//
// All decoders return an audio.Source interface which can be used with
// the audio processing functions.
//
// # Writing WAV Files
//
// The package can write PCM WAV files:
//
//	samples := []int16{100, -100, 200, -200}
//	file, _ := os.Create("output.wav")
//	wav.WriteWAV16(file, 8000, samples)
//
// # Audio Terms Glossary
//
// For those new to audio processing, here are key terms used in this package:
//
// Sample Rate (Hz): How many times per second audio is measured. Higher rates
// capture more detail but use more storage. Common rates:
//   - 8000 Hz: Telephone quality (minimum for speech)
//   - 16000 Hz: Wideband speech (better quality calls)
//   - 44100 Hz: CD quality (music standard)
//   - 48000 Hz: Professional audio/video
//
// Channels: Independent audio streams in multi-channel audio:
//   - Mono (1): Single channel (one speaker)
//   - Stereo (2): Two channels (left and right speakers)
//   - 5.1 Surround (6): Front-left, front-right, center, subwoofer, back-left, back-right
//
// Resampling: Changing the sample rate without affecting pitch or speed.
// Used to convert between different audio standards or reduce bandwidth.
//
// Gain: Volume multiplier (1.0 = original, 2.0 = double, 0.5 = half).
// Applied by multiplying each sample by the gain value.
//
// Normalization: Automatically adjusting volume to use the full available
// dynamic range without clipping. Finds the loudest point and scales everything
// proportionally so that point reaches (but doesn't exceed) the maximum.
//
// Mixdown/Downmixing: Combining multiple channels into fewer channels (usually mono)
// by averaging. For example, stereo to mono: output = (left + right) / 2.
//
// Clipping: Distortion that occurs when audio becomes too loud and exceeds
// the maximum representable value. Sounds harsh and should be avoided.
//
// Interpolation: Creating new sample values between existing ones when
// upsampling. This package uses cubic interpolation for smooth, high-quality results.
//
// # Performance
//
// Converting a 103 second 44.1 kHz stereo recording to 8 kHz mono takes about
// 59 ms from WAV or AIFF on a mid-range x86-64 laptop, which is roughly 2.5x
// faster than ffmpeg doing the same job, and allocates about 70 times for the
// whole file. MP3 sources take about 2 s and Ogg Vorbis about 470 ms; that time
// is spent inside the third-party decoders rather than in resampling.
//
// What makes this fast:
//   - Resampler reads its source in large blocks, so decoder call overhead and
//     read(2) traffic are amortised instead of paid per audio frame
//   - The WAV and AIFF decoders decode samples directly, avoiding a per-call
//     allocation and an intermediate []int four times wider than the samples
//   - Resampler.ReadSamples allocates nothing once streaming has started
//   - Batch conversions reduce per-sample overhead
//
// Buffer size has very little effect on throughput: the resampler reads its
// source in fixed internal blocks regardless of how much a caller requests per
// call. audio.DefaultBufSize is a reasonable default.
//
// # Limitations
//
// Resampling quality: the anti-aliasing filter used when downsampling is a
// one-pole low-pass with a fixed coefficient that does not track the resampling
// ratio, so content above the output Nyquist frequency is not fully removed and
// will alias. For a 6 kHz tone resampled from 44.1 kHz to 8 kHz, this package
// leaves it around -28 dB where ffmpeg reaches -74 dB.
//
// Format coverage: WAV and AIFF support 16-bit PCM only, and WAV rejects
// WAVE_FORMAT_EXTENSIBLE files. Only WAV can be written.
//
// See the individual subpackages for more detailed documentation.
package audpbx
