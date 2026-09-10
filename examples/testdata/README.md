# Test Audio Files

This directory contains audio files used for testing and examples across the project.

## Files

All files are kept with their original names to provide proper attribution to the artists 
and sources.

### Track 1: Capricerie No. 5 (Bach, Paganini) - Daniel Bautista

**Originals:**
- `Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg` - Source from Jamendo

**Converted formats (with preserved metadata):**
- `Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).mp3`
- `Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).wav`
- `Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).aiff`

All conversions were done with ffmpeg using metadata preservation:
```bash
ffmpeg -i input.ogg -map_metadata 0 output.mp3
```

**Metadata includes:**
- Artist: Daniel Bautista
- License: CC-BY 3.0/4.0
- Jamendo Track ID
- Genre: Progressive Metal, Neoclassical
- Source URL and comments

**Characteristics:**
- Duration: 1:43 (103 seconds)
- Size: ~1.4 MB (OGG), ~18 MB (WAV)
- Content: Complex instrumental (good for quality testing)

### Track 2: SNEAKERS 128BPM - GloryToTheMachine

**Originals:**
- `843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.mp3` - Source from Freesound

**Converted formats (with preserved metadata):**
- `843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.wav`
- `843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.ogg`
- `843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.aiff`

All conversions preserve metadata including encoder information and source details.

**Characteristics:**
- Duration: 1:30 (90 seconds)
- Size: ~2.1 MB (MP3), ~16 MB (WAV)
- Content: Electronic/beats (good for rhythm testing)

### Track 3: Bells Drone - kevp888 (Kevin Luce)

**Originals:**
- `844152__kevp888__020a_100111_0243_exp02_bells_drone.wav` - Source from Freesound

**Converted formats (with preserved metadata):**
- `844152__kevp888__020a_100111_0243_exp02_bells_drone.mp3`
- `844152__kevp888__020a_100111_0243_exp02_bells_drone.ogg`
- `844152__kevp888__020a_100111_0243_exp02_bells_drone.aiff`

**Special characteristics:**
- Duration: 16:29 (very long - **excellent for performance testing!**)
- Size: ~167 MB (WAV) - **tests large file handling**
- Sample Rate: 44.1 kHz, Stereo, 16-bit
- Content: Meditative drone - **excellent for stress testing continuous audio processing**

This file is particularly valuable for:
- Performance profiling with large files
- Memory usage testing
- Long-running stream processing
- Stability testing

## Why Keep Original Names?

1. **Attribution** - Filenames credit the artists and sources
2. **Traceability** - Easy to identify which track from which source
3. **Metadata consistency** - Names match embedded metadata
4. **Professional** - Shows respect for Creative Commons licensing
5. **Test value** - Real-world filenames often have spaces and special characters

## Usage in Examples

### From `examples/resampler/`:
```go
inPath := "../testdata/Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg"
```

### From `examples/profile_resampler/`:
```go
inPath := "../testdata/843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.mp3"

// For large file performance testing:
inPath := "../testdata/844152__kevp888__020a_100111_0243_exp02_bells_drone.wav"
```

### In Shell (use quotes for special characters):
```bash
go run main.go "../testdata/Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg" output.wav

# Test with large file (16+ minutes, 167 MB)
go run main.go "../testdata/844152__kevp888__020a_100111_0243_exp02_bells_drone.wav" output.wav
```

## Format Coverage

These files provide comprehensive format testing:

| Format | Codec | Container | Metadata | Source | Size Range |
|--------|-------|-----------|----------|--------|------------|
| OGG | Vorbis | Ogg | ✅ Vorbis comments | Capricerie (original) | Small |
| MP3 | MP3 | ID3v2 | ✅ ID3 tags | Sneakers (original) | Medium |
| WAV | PCM | RIFF | ⚠️ Limited | All (converted + Bells original) | Large |
| AIFF | PCM | IFF | ⚠️ Limited | All (converted) | Large |

## Test File Characteristics

| Track | Duration | Size (Original) | Best For |
|-------|----------|-----------------|----------|
| Capricerie | 1:43 | 1.4 MB | Quick tests, format validation |
| Sneakers | 1:30 | 2.1 MB | Rhythm analysis, standard tests |
| Bells Drone | 16:29 | **167 MB** | **Performance, memory, large files** |

## Metadata Testing

These files are valuable for testing because they contain:

- **Artist information** - Tests metadata parsing
- **License information** - Embedded CC-BY licenses
- **Comments and URLs** - Source tracking
- **Genre tags** - Tag handling
- **Encoder information** - Format compliance

This ensures the audio library handles real-world files correctly, not just synthetic test data.

## Regenerating Conversions

If you need to regenerate converted formats while preserving metadata:

```bash
# Convert OGG to other formats (preserve metadata)
ffmpeg -i "Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg" \
       -map_metadata 0 -id3v2_version 3 \
       "Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).mp3"

ffmpeg -i "Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg" \
       -map_metadata 0 \
       "Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).wav"

# Convert MP3 to other formats (preserve metadata)
ffmpeg -i "843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.mp3" \
       -map_metadata 0 -c:a libvorbis \
       "843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.ogg"

# Convert WAV to other formats (preserve metadata)
ffmpeg -i "844152__kevp888__020a_100111_0243_exp02_bells_drone.wav" \
       -map_metadata 0 -b:a 192k \
       "844152__kevp888__020a_100111_0243_exp02_bells_drone.mp3"

ffmpeg -i "844152__kevp888__020a_100111_0243_exp02_bells_drone.wav" \
       -map_metadata 0 -c:a libvorbis -q:a 6 \
       "844152__kevp888__020a_100111_0243_exp02_bells_drone.ogg"
```

Note: WAV and AIFF have limited metadata support compared to OGG/MP3.

## License

See [LICENSE](LICENSE) file for complete attribution and usage terms.

## File Sizes Summary

**Original files:**
- Capricerie OGG: ~1.4 MB
- Sneakers MP3: ~2.1 MB
- Bells Drone WAV: ~167 MB

**Total testdata size:** ~500-600 MB (all formats)

## Testing Strategy

**Use smaller files (Capricerie, Sneakers) for:**
- Quick format validation tests
- Decoder correctness
- Integration tests in CI/CD
- Development/debugging

**Use large file (Bells Drone) for:**
- Performance profiling (this is what you're doing!)
- Memory usage analysis
- Large file handling
- Stability testing
- Real-world performance scenarios

## Verification

Check that metadata is preserved in conversions:

```bash
# View metadata
ffprobe -v quiet -show_format "Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).mp3"

# Check file info
ffprobe -v quiet -show_format "844152__kevp888__020a_100111_0243_exp02_bells_drone.wav"
```

## Performance Note

The 16-minute Bells Drone file is the best asset here for profiling:
- Real-world large file scenario
- Tests sustained processing performance
- Exposes memory leaks or inefficiencies
- Shows true throughput rates

Note that its WAV and AIFF renderings are excluded from version control because
of their size; use `manage_testdata.sh` to regenerate them.

The shorter Capricerie file is what
[`internal/perfbench`](../../internal/perfbench) benchmarks against, in all four
formats, since it exists in every container and runs quickly:

```bash
go test ./internal/perfbench/ -bench . -benchtime 5x -benchmem
```

For context on what these files were originally used to investigate — a
pipeline that took ~7 s to convert 103 seconds of audio, since fixed and now
~59 ms — see
[../profile_resampler/PROFILING_GUIDE.md](../profile_resampler/PROFILING_GUIDE.md).
