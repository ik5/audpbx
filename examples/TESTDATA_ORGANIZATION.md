# Test Data Organization Guide

This guide explains how test audio files are organized in the `audpbx` project.

## Current Structure

Test audio files are organized in a shared `testdata/` directory following Go conventions:

```
examples/
├── testdata/                    # Shared test files
│   ├── LICENSE                  # Complete license information
│   ├── README.md                # Usage guide
│   ├── METADATA_PRESERVATION.md # Metadata documentation
│   ├── SCRIPT_USAGE.md          # Script documentation
│   ├── manage_testdata.sh       # Management script
│   │
│   # Track 1 - Daniel Bautista (original names preserved)
│   ├── Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg    # Original (1.4 MB)
│   ├── Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).mp3    # Converted w/ metadata
│   ├── Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).wav    # Converted w/ metadata
│   ├── Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).aiff   # Converted w/ metadata
│   │
│   # Track 2 - GloryToTheMachine (original names preserved)
│   ├── 843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.mp3   # Original (2.1 MB)
│   ├── 843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.wav   # Converted w/ metadata
│   ├── 843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.ogg   # Converted w/ metadata
│   ├── 843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.aiff  # Converted w/ metadata
│   │
│   # Track 3 - kevp888 (original names preserved)
│   ├── 844152__kevp888__020a_100111_0243_exp02_bells_drone.wav    # Original (167 MB, 16:29)
│   ├── 844152__kevp888__020a_100111_0243_exp02_bells_drone.mp3    # Converted w/ metadata
│   ├── 844152__kevp888__020a_100111_0243_exp02_bells_drone.ogg    # Converted w/ metadata
│   └── 844152__kevp888__020a_100111_0243_exp02_bells_drone.aiff   # Converted w/ metadata
│
├── resampler/                   # Basic example
│   └── main.go
│
├── profile_resampler/           # Profiling tools
│   └── main.go
│
└── TESTDATA_ORGANIZATION.md     # This file
```

## Why This Structure?

### 1. **Follows Go Conventions**
- `testdata` is a special directory name in Go
- Recognized by `go test` and tooling
- Standard location for test fixtures

### 2. **Shared Across Examples**
- All examples can access files via `../testdata/file.ext`
- No duplication of large audio files
- Consistent test data across the project

### 3. **Attribution Preserved**
- Original filenames credit artists and sources
- Names match embedded metadata
- Easy to trace back to source (Jamendo, Freesound)

### 4. **Metadata Included**
- All conversions preserve metadata using ffmpeg
- ID3 tags (MP3), Vorbis comments (OGG) maintained
- Artist, license, source info embedded in files

### 5. **Real-World Testing**
- Filenames with spaces and special characters
- Various file sizes (1.4 MB to 167 MB)
- Different durations (1 min to 16 min)
- Multiple formats and codecs

### 6. **Professional Organization**
- Proper licensing documentation
- Complete attribution
- Clear usage terms

## File Characteristics

| Track | Original Format | Duration | Size | Best For |
|-------|----------------|----------|------|----------|
| Capricerie | OGG | 1:30 | 1.4 MB | Quick tests, format validation |
| Sneakers | MP3 | 1:00 | 2.1 MB | Standard tests, rhythm analysis |
| Bells Drone | WAV | 16:29 | 167 MB | **Performance testing, large files** |

## Managing Test Files

### Method 1: Automated Script (Recommended) ⭐

Use the provided `manage_testdata.sh` script:

```bash
cd examples/testdata

# Interactive mode (menu)
./manage_testdata.sh

# Direct mode (add specific file)
./manage_testdata.sh /path/to/newfile.wav
```

**The script handles:**
- ✅ File validation and info display
- ✅ Conversion to all formats (MP3, OGG, WAV, AIFF)
- ✅ Metadata preservation
- ✅ LICENSE file updates (auto-generated entries)
- ✅ File listing and verification
- ✅ Dependency checking

**See:** [SCRIPT_USAGE.md](testdata/SCRIPT_USAGE.md) for complete guide.

### Method 2: Manual with FFmpeg

If you prefer manual control:

```bash
cd examples/testdata

# Convert with metadata preservation
ffmpeg -i "input.ogg" -map_metadata 0 -codec:a libmp3lame -q:a 2 "output.mp3"
ffmpeg -i "input.ogg" -map_metadata 0 -codec:a libvorbis -q:a 6 "output.ogg"
ffmpeg -i "input.ogg" -map_metadata 0 -codec:a pcm_s16le "output.wav"
ffmpeg -i "input.ogg" -map_metadata 0 -codec:a pcm_s16be "output.aiff"
```

**Important:** Always use `-map_metadata 0` to preserve tags!

**Then manually update LICENSE file** with proper attribution.

### Method 3: Migration Script

If moving files from `resampler/` to `testdata/`:

```bash
cd examples
./move_to_testdata.sh
```

This will:
- Copy files to testdata/ (keeps originals as backup)
- Preserve original filenames
- Show file information
- Guide you through verification

## Adding New Test Files

### Quick Steps:

1. **Get the file** from Freesound, Jamendo, or other CC-BY source
2. **Run the script:**
   ```bash
   cd examples/testdata
   ./manage_testdata.sh /path/to/newfile.wav
   ```
3. **Follow prompts:**
   - Keep or rename filename (recommend: keep)
   - Enter artist, title, source URL
   - Enter license (usually CC-BY 4.0)
   - Enter description
4. **Done!** Script converts to all formats and updates LICENSE

### Example Workflow:

```bash
# Download from Freesound
cd ~/Downloads
wget "https://freesound.org/.../mytrack.wav"

# Add to testdata
cd ~/projects/audpbx/examples/testdata
./manage_testdata.sh ~/Downloads/mytrack.wav

# Follow prompts
# ✓ Converts to MP3, OGG, WAV, AIFF
# ✓ Updates LICENSE automatically
# ✓ Shows file sizes and info
```

## Usage in Code

### From Examples:

```go
// examples/resampler/main.go
inPath := "../testdata/Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg"

// examples/profile_resampler/main.go
inPath := "../testdata/843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.mp3"

// For performance testing with large file:
inPath := "../testdata/844152__kevp888__020a_100111_0243_exp02_bells_drone.wav"
```

### In Shell:

```bash
# Use quotes for filenames with special characters
go run main.go "../testdata/Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg" output.wav

# Large file performance test
go run main.go "../testdata/844152__kevp888__020a_100111_0243_exp02_bells_drone.wav" output.wav
```

### In Tests:

```go
func TestDecoder(t *testing.T) {
    // Relative to test file location
    testFile := "../testdata/Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg"
    
    f, err := os.Open(testFile)
    // ... test decoder
}
```

## File Naming Convention

### Original Names Are Kept

**Why?**
1. **Attribution** - Credits artists in the filename
2. **Traceability** - Easy to find source on Freesound/Jamendo
3. **Metadata match** - Names match embedded tags
4. **Professional** - Shows respect for CC-BY licensing
5. **Real-world** - Tests handling of complex filenames

### Freesound Convention:

```
[ID]__[artist]__[title].ext
844152__kevp888__020a_100111_0243_exp02_bells_drone.wav
```

### Jamendo Convention:

```
[Artist]_-_[Title]_([Details]).ext
Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg
```

**Keep these!** They provide valuable context and attribution.

## License and Attribution

### LICENSE File

The `testdata/LICENSE` file contains:
- Complete track information
- Artist names and titles
- Source URLs (Jamendo, Freesound)
- License types (all CC-BY 4.0)
- File lists per track
- Attribution requirements
- Usage terms

### Auto-Generated Entries

When using `manage_testdata.sh`, LICENSE entries are auto-generated with:
- Track number (auto-incremented)
- All required fields
- Proper formatting
- Freesound attribution (when applicable)

### Manual Updates

If adding files manually, follow the format in LICENSE:

```markdown
## Track N: [Title]

- **Artist:** [Name]
- **Title:** [Full Title]
- **Source:** [URL]
- **License:** CC-BY 4.0 (https://creativecommons.org/licenses/by/4.0/)
- **Description:** [Description]. [Duration] duration, [Channels], [Sample Rate].
- **Files:**
  - `[basename].format` (original)
  - `[basename].format`
  - ...

**Usage:** Testing purposes only under CC-BY 4.0 license.

**Attribution:** [If Freesound, add attribution line]
```

## Metadata Preservation

All converted files preserve metadata from originals:

### What Gets Preserved:

- ✅ Artist information
- ✅ Title
- ✅ License tags
- ✅ Comments and descriptions
- ✅ Source URLs
- ✅ Genre tags
- ✅ Encoder information
- ✅ Date/year

### Format Support:

| Format | Metadata System | Support |
|--------|----------------|---------|
| OGG | Vorbis Comments | ✅ Full |
| MP3 | ID3v2 | ✅ Full |
| WAV | RIFF INFO | ⚠️ Limited |
| AIFF | ID3/COMM | ⚠️ Limited |

### Verification:

```bash
cd examples/testdata

# View metadata
ffprobe -v quiet -show_format -show_entries format_tags "file.mp3"

# Or use the script
./manage_testdata.sh
# Select option 4: Verify metadata
```

**See:** [METADATA_PRESERVATION.md](testdata/METADATA_PRESERVATION.md) for details.

## Testing Strategy

### Quick Tests (CI/CD):
Use smaller files:
- Capricerie (1.4 MB, 1:30)
- Sneakers (2.1 MB, 1:00)

### Performance Tests:
Use large file:
- Bells Drone (167 MB, 16:29)

### Format Tests:
Use variety:
- OGG (Vorbis codec)
- MP3 (MPEG codec)
- WAV (PCM, uncompressed)
- AIFF (PCM, uncompressed)

### Metadata Tests:
All files have embedded metadata for testing parsers.

## .gitignore Recommendations

### Option 1: Keep Only Originals

```gitignore
# Keep originals, ignore converted formats
examples/testdata/*.wav
examples/testdata/*.aiff
examples/testdata/Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).mp3
examples/testdata/843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.{wav,ogg,aiff}

# Keep the original large file
!examples/testdata/844152__kevp888__020a_100111_0243_exp02_bells_drone.wav

# Always keep documentation
!examples/testdata/LICENSE
!examples/testdata/README.md
!examples/testdata/*.md
!examples/testdata/*.sh
```

### Option 2: Keep All (If Space Allows)

```gitignore
# Keep everything (if repo size allows)
# Having all formats in git ensures consistent testing

# Just ignore generated outputs
examples/testdata/*output*.wav
examples/testdata/profiles/
```

### Recommendation:

**Keep all files in git** because:
- Total size ~500-600 MB (manageable)
- Ensures consistent testing across environments
- No need to regenerate formats
- Metadata preserved and verified

## Troubleshooting

### "File not found" errors

Check path from your example:
```bash
cd examples/resampler
ls ../testdata/  # Should see all test files
```

### Conversion failures

Check ffmpeg:
```bash
cd examples/testdata
./manage_testdata.sh
# Select option 5: Check dependencies
```

### Metadata not preserved

Verify conversion used `-map_metadata 0`:
```bash
ffprobe -show_format yourfile.mp3 | grep TAG
```

### LICENSE not updated

Use the script - it auto-generates correct entries:
```bash
./manage_testdata.sh
# Select option 1: Add new file
```

## Documentation Files

| File | Purpose |
|------|---------|
| `LICENSE` | Complete license information and attribution |
| `README.md` | Overview, file list, usage examples |
| `METADATA_PRESERVATION.md` | Details on metadata handling |
| `SCRIPT_USAGE.md` | Complete script documentation |
| `manage_testdata.sh` | Management script |
| `TESTDATA_ORGANIZATION.md` | This file - organizational guide |

## Quick Reference

### Add New File:
```bash
cd examples/testdata
./manage_testdata.sh newfile.wav
```

### Convert Existing:
```bash
./manage_testdata.sh
# Option 2
```

### List Files:
```bash
./manage_testdata.sh
# Option 3
```

### Verify Metadata:
```bash
./manage_testdata.sh
# Option 4
```

### Use in Code:
```go
inPath := "../testdata/filename.ext"
```

## Summary

✅ **Structure:** Shared `testdata/` directory  
✅ **Files:** Original names preserved for attribution  
✅ **Formats:** MP3, OGG, WAV, AIFF (all with metadata)  
✅ **Licensing:** Complete LICENSE file with all attribution  
✅ **Management:** Automated script for easy maintenance  
✅ **Documentation:** Complete guides and usage examples  
✅ **Testing:** Files for quick tests, performance tests, and format validation  

This organization makes test data management professional, maintainable, and compliant with CC-BY licensing requirements.
