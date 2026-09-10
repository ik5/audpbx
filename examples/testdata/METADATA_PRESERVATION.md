# Metadata Preservation in Test Files

All converted audio files in this directory preserve metadata from their source files using ffmpeg's `-map_metadata` option.

## Why Metadata Matters for Testing

1. **Real-world compliance** - Production files have metadata
2. **Format correctness** - Tests that decoders handle tags properly
3. **License tracking** - Embedded CC-BY license information
4. **Attribution** - Artist and source information preserved
5. **Regression testing** - Ensures library doesn't corrupt metadata

## Metadata Included

### Capricerie Files (OGG/MP3/WAV/AIFF)

**Vorbis Comments (in OGG) / ID3 Tags (in MP3):**
- Artist: Daniel Bautista
- Title: Capricerie No. 5 (Bach, Paganini)
- License: CC-BY 3.0/4.0
- Source: Jamendo (https://www.jamendo.com)
- Genre: Progressive Metal, Neoclassical
- Jamendo Track ID: 2283005
- Copyright: Creative Commons License URL
- Encoder information

### Sneakers Files (MP3/WAV/OGG/AIFF)

**ID3 Tags (in MP3) / Vorbis Comments (in OGG):**
- Encoded by: LAME in FL Studio 2024
- Date: 2026
- Source: Freesound (https://freesound.org)

## Conversion Commands Used

All conversions preserve metadata:

### From OGG to other formats:
```bash
# To MP3 (preserves Vorbis comments as ID3 tags)
ffmpeg -i input.ogg -map_metadata 0 -id3v2_version 3 output.mp3

# To WAV (preserves what WAV supports)
ffmpeg -i input.ogg -map_metadata 0 output.wav

# To AIFF (preserves what AIFF supports)
ffmpeg -i input.ogg -map_metadata 0 output.aiff
```

### From MP3 to other formats:
```bash
# To OGG (preserves ID3 tags as Vorbis comments)
ffmpeg -i input.mp3 -map_metadata 0 -c:a libvorbis output.ogg

# To WAV
ffmpeg -i input.mp3 -map_metadata 0 output.wav

# To AIFF
ffmpeg -i input.mp3 -map_metadata 0 output.aiff
```

## Metadata Format Support

| Format | Metadata System | Support Level | Notes |
|--------|----------------|---------------|-------|
| OGG | Vorbis Comments | ✅ Full | Best metadata support |
| MP3 | ID3v2 | ✅ Full | Industry standard |
| WAV | RIFF INFO chunks | ⚠️ Limited | Basic tags only |
| AIFF | ID3 or COMM chunks | ⚠️ Limited | Less common |

## Verifying Metadata

Check metadata in files:

```bash
# View all metadata
ffprobe -v quiet -show_format -show_entries format_tags "Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg"

# Compare metadata between formats
ffprobe -v quiet -show_format "Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg" | grep TAG
ffprobe -v quiet -show_format "Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).mp3" | grep TAG
```

## Testing Impact

These metadata-rich files help test that the audio library:

1. **Doesn't corrupt tags** - Metadata survives decode/encode cycles
2. **Handles tag formats** - Works with ID3v2, Vorbis comments, etc.
3. **Parses correctly** - Can read embedded information
4. **Preserves on conversion** - Maintains tags when converting formats
5. **Works with real files** - Not just synthetic test data

## Regenerating Files

If you need to recreate conversions while preserving metadata:

```bash
# Ensure ffmpeg is installed
ffmpeg -version

# Convert with metadata preservation
./convert_with_metadata.sh  # (if available)

# Or manually with the commands above
```

## License Information in Metadata

Both tracks have CC-BY license information embedded:

- **Capricerie**: LICENSE tag contains CC-BY URL
- **Sneakers**: License requires attribution (CC-BY 4.0)

This metadata embedding demonstrates compliance with attribution requirements even in the audio files themselves.

## Audio Characteristics

### Capricerie (Daniel Bautista)
- Duration: ~90 seconds
- Channels: Stereo
- Sample Rate: 44.1 kHz (typical for music)
- Format: Originally Vorbis (OGG)
- Style: Complex instrumental (good for quality testing)

### Sneakers (GloryToTheMachine)  
- Duration: ~60 seconds
- Channels: Stereo
- Sample Rate: 44.1 kHz
- Format: Originally MP3
- Style: Electronic/beats (good for rhythm testing)

## Best Practices

1. **Keep originals** - OGG and MP3 as downloaded from sources
2. **Document conversions** - Note the ffmpeg commands used
3. **Verify metadata** - Check tags after conversion
4. **Test with metadata** - Ensure decoders handle tags correctly
5. **Respect licenses** - Metadata helps track attribution requirements
