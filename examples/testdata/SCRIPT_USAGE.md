# Test Data Management Script Usage

The `manage_testdata.sh` script helps you manage test audio files easily.

## Quick Start

```bash
cd examples/testdata

# Interactive mode (menu-driven)
./manage_testdata.sh

# Direct mode (provide file as argument)
./manage_testdata.sh /path/to/your/audio/file.wav
```

## Features

### 1. Add New File (Option 1)

**What it does:**
- Copies or moves your audio file to testdata/
- Converts to all formats (MP3, OGG, WAV, AIFF)
- Preserves metadata using ffmpeg
- Generates LICENSE entry automatically
- Shows file information (duration, size, etc.)

**Example workflow:**
```bash
$ ./manage_testdata.sh

Select option: 1

Enter the path to the audio file:
> /downloads/mytrack.mp3

File Information:
  Duration: 03:45
  Size: 8.5 MB
  Sample Rate: 44100 Hz
  Channels: Stereo

Keep original filename? (y/n)
> y

Converting to Other Formats...
✓ Created: mytrack.ogg
✓ Created: mytrack.wav
✓ Created: mytrack.aiff

Enter the source information:

Artist name:
> John Doe

Track title:
> My Amazing Track

Source URL:
> https://freesound.org/people/johndoe/sounds/123456/

License (default: CC-BY 4.0):
> [Enter]

Short description:
> Ambient soundscape for testing

Add this entry to LICENSE file? (y/n)
> y

✓ LICENSE file updated
✓ Done! All formats created and LICENSE updated.
```

### 2. Convert Existing File (Option 2)

**What it does:**
- Lists all audio files in testdata/
- Converts selected file to missing formats
- Preserves metadata
- Asks before overwriting existing files

**Use case:** You added a WAV file manually and want MP3/OGG/AIFF versions.

```bash
Select option: 2

Available audio files:
  1) my_file.wav (156.3 MB)
  2) another_track.mp3 (3.2 MB)

Enter filename to convert:
> my_file.wav

Converting my_file.wav
✓ Created: my_file.mp3
✓ Created: my_file.ogg
✓ Created: my_file.aiff
```

### 3. List All Test Files (Option 3)

**What it does:**
- Shows all audio files with details
- Duration, size, sample rate, channels

```bash
Select option: 3

Test Audio Files

📄 Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg
   Duration: 01:30 | Size: 1.4 MB | 44100 Hz | Stereo

📄 844152__kevp888__020a_100111_0243_exp02_bells_drone.wav
   Duration: 16:29 | Size: 166.5 MB | 44100 Hz | Stereo
```

### 4. Verify Metadata (Option 4)

**What it does:**
- Checks if metadata was preserved across all formats
- Shows tags for each format

```bash
Select option: 4

Enter base filename (without extension):
> my_track

=== my_track.mp3 ===
TAG:artist=John Doe
TAG:title=My Amazing Track
TAG:encoder=Lavc61.19.101 libmp3lame

=== my_track.ogg ===
TAG:ARTIST=John Doe
TAG:TITLE=My Amazing Track
TAG:encoder=Lavc61.19.101 libvorbis
```

### 5. Check Dependencies (Option 5)

**What it does:**
- Verifies ffmpeg and ffprobe are installed
- Shows installation instructions if missing

## Command-Line Mode

For automation or quick adds:

```bash
# Add a file directly
./manage_testdata.sh ~/downloads/newtrack.wav

# The script will guide you through:
# 1. File info display
# 2. Keep/rename filename
# 3. Convert to all formats
# 4. Update LICENSE
```

## Requirements

- **ffmpeg** - For audio conversion
- **ffprobe** - For metadata extraction (comes with ffmpeg)
- **bc** - For calculations (usually pre-installed)

### Installation:

```bash
# Ubuntu/Debian
sudo apt install ffmpeg

# macOS
brew install ffmpeg

# Arch Linux
sudo pacman -S ffmpeg
```

## What the Script Does Automatically

### Metadata Preservation

All conversions use `-map_metadata 0` to preserve:
- Artist information
- Title
- License tags
- Comments
- Encoder information

### Quality Settings

- **MP3:** VBR quality 2 (~190 kbps, high quality)
- **OGG:** Quality 6 (~192 kbps, high quality)
- **WAV:** PCM 16-bit (lossless)
- **AIFF:** PCM 16-bit (lossless)

### LICENSE Entry Generation

Automatically creates properly formatted LICENSE entries with:
- Track number (auto-incremented)
- Artist and title
- Source URL
- License type
- File list
- Attribution requirements (for Freesound)

## Examples

### Example 1: Adding a Freesound File

```bash
# Download from Freesound
wget https://freesound.org/data/previews/123/123456_1234567-hq.mp3 -O mytrack.mp3

# Run script
./manage_testdata.sh mytrack.mp3

# Follow prompts, script will:
# - Show file info (duration, size)
# - Convert to OGG, WAV, AIFF
# - Generate LICENSE entry with Freesound attribution
# - Update LICENSE file
```

### Example 2: Converting Existing File

You manually added `bigfile.wav` and want other formats:

```bash
./manage_testdata.sh
# Select option 2
# Choose bigfile.wav
# Script creates bigfile.mp3, bigfile.ogg, bigfile.aiff
```

### Example 3: Checking Metadata

Verify your conversions preserved metadata:

```bash
./manage_testdata.sh
# Select option 4
# Enter base filename: mytrack
# See metadata for all formats
```

## Tips

### 1. Keep Original Names

When asked "Keep original filename?", say **yes** for:
- Attribution (credits artist)
- Traceability (easy to find source)
- Consistency (matches embedded metadata)

### 2. File Naming Convention

Freesound files come with descriptive names like:
```
844152__kevp888__020a_100111_0243_exp02_bells_drone.wav
[ID]__[artist]__[title].wav
```

Keep these! They're perfect for test files.

### 3. License Information

The script prompts for:
- **Artist name** - Use real name or username
- **Source URL** - Full Freesound/Jamendo URL
- **License** - Default CC-BY 4.0 (most common)
- **Description** - Brief description of the audio

### 4. Large Files

For large files (>100 MB):
- WAV/AIFF will be huge (uncompressed)
- MP3/OGG will be much smaller
- Conversion takes longer
- Watch disk space!

### 5. Overwriting

The script asks before overwriting existing files. This protects:
- Manual edits
- Custom conversions
- Originals from being replaced

## Troubleshooting

### "ffmpeg: command not found"

Install ffmpeg:
```bash
# Ubuntu/Debian
sudo apt install ffmpeg

# macOS
brew install ffmpeg
```

### "bc: command not found"

Install bc (basic calculator):
```bash
# Ubuntu/Debian
sudo apt install bc

# macOS (usually pre-installed)
brew install bc
```

### Conversion Fails

Check:
1. Input file is valid: `ffprobe yourfile.mp3`
2. Disk space available: `df -h .`
3. File permissions: `ls -l yourfile.mp3`

### Metadata Not Preserved

Some formats have limited metadata support:
- WAV: Basic tags only
- AIFF: Limited support
- OGG/MP3: Full support

This is normal! The script uses best practices.

## Advanced Usage

### Batch Conversion

Convert multiple files:

```bash
# Convert all MP3s to other formats
for file in *.mp3; do
    ./manage_testdata.sh "$file"
done
```

### Custom Conversions

Edit the script's quality settings around line 122-140:

```bash
# Higher quality MP3
-q:a 0  # Instead of -q:a 2

# Higher quality OGG
-q:a 8  # Instead of -q:a 6
```

### Verify All Files

Check metadata for all base names:

```bash
for base in Daniel_Bautista 843594__glorytothemachine 844152__kevp888; do
    echo "=== $base ==="
    ./manage_testdata.sh
    # Option 4, enter base name
done
```

## What Gets Updated

### Files Created:
- `[basename].mp3`
- `[basename].ogg`
- `[basename].wav`
- `[basename].aiff`

### Files Modified:
- `LICENSE` - New track entry added
- (Optionally) `README.md` - Manual update recommended

### Files Preserved:
- Original file (unless you choose "move")
- Existing LICENSE entries
- Other unrelated files

## See Also

- [README.md](README.md) - Overview of test files
- [LICENSE](LICENSE) - Complete attribution
- [METADATA_PRESERVATION.md](METADATA_PRESERVATION.md) - Metadata details

## Quick Reference

| Task | Command |
|------|---------|
| Add new file | `./manage_testdata.sh newfile.wav` |
| Interactive menu | `./manage_testdata.sh` |
| List files | Option 3 in menu |
| Convert existing | Option 2 in menu |
| Check metadata | Option 4 in menu |
| Verify setup | Option 5 in menu |
