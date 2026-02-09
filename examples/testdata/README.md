# Test Audio Files

This directory contains audio files used for testing and examples across the project.

## Files

### Original Files (Keep These)

- `capricerie.ogg` - Daniel Bautista - Capricerie No. 5 (CC-BY 4.0)
- `sneakers.mp3` - GloryToTheMachine - SNEAKERS 128BPM (CC-BY 4.0)

### Converted Formats (Can be regenerated)

- `capricerie.{mp3,wav,aiff}` - Converted from OGG
- `sneakers.{wav,ogg,aiff}` - Converted from MP3

## Usage in Examples

All examples should reference these files using relative paths:

```go
// From examples/resampler/
inFile := "../testdata/capricerie.ogg"

// From examples/profile_resampler/
inFile := "../testdata/sneakers.mp3"
```

## License

See [LICENSE](LICENSE) file for complete attribution and usage terms.

## Regenerating Converted Formats

If you need to regenerate the converted formats from the originals, use the
conversion tools in `examples/convert/` (if available) or external tools like:

```bash
# Using ffmpeg
ffmpeg -i capricerie.ogg capricerie.mp3
ffmpeg -i capricerie.ogg capricerie.wav
ffmpeg -i capricerie.ogg capricerie.aiff

ffmpeg -i sneakers.mp3 sneakers.wav
ffmpeg -i sneakers.mp3 sneakers.ogg
ffmpeg -i sneakers.mp3 sneakers.aiff
```
