# Test Data Organization Guide

## Current Situation

All test audio files are currently in `examples/resampler/` with long filenames:
- `Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).{ogg,mp3,wav,aiff}`
- `843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.{mp3,wav,ogg,aiff}`

## Recommended Structure

Create a shared `testdata/` directory following Go conventions:

```
examples/
├── testdata/              # ← NEW: Shared test files
│   ├── LICENSE           # License information
│   ├── README.md         # Usage instructions
│   │
│   # Track 1 - Daniel Bautista (original names preserved)
│   ├── Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg    # Original
│   ├── Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).mp3    # Converted w/ metadata
│   ├── Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).wav    # Converted w/ metadata
│   ├── Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).aiff   # Converted w/ metadata
│   │
│   # Track 2 - GloryToTheMachine (original names preserved)
│   ├── 843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.mp3   # Original
│   ├── 843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.wav   # Converted w/ metadata
│   ├── 843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.ogg   # Converted w/ metadata
│   └── 843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.aiff  # Converted w/ metadata
│
├── resampler/            # Example code
│   └── main.go
│
└── profile_resampler/    # Profiling tools
    └── main.go
```

**Note:** Original filenames are kept to:
- Properly attribute the artists
- Maintain traceability to source
- Test with real-world filenames (spaces, special chars)
- Match embedded metadata

## Benefits

1. **Shared across all examples** - Any example can use `../testdata/file.ext`
2. **Go convention** - `testdata` directories are recognized by Go tools and git
3. **Attribution preserved** - Original filenames credit artists and sources
4. **Metadata included** - All conversions preserve metadata (ID3, Vorbis comments)
5. **Real-world testing** - Filenames with spaces and special characters
6. **Better organization** - Test files separate from example code

## Implementation

### Option 1: Automated (Recommended)

Run the provided script:

```bash
cd examples
./move_to_testdata.sh
```

This will:
- Create `testdata/` directory
- Copy files (keeping original names)
- Preserve metadata in all formats
- Create LICENSE and README
- Show you the new structure
- Keep backups in resampler/ until verified

### Option 2: Manual

```bash
cd examples
mkdir -p testdata

# Copy all Capricerie formats (keeping original names)
for ext in ogg mp3 wav aiff; do
    cp "resampler/Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).$ext" \
       "testdata/Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).$ext"
done

# Copy all Sneakers formats (keeping original names)
for ext in mp3 wav ogg aiff; do
    cp "resampler/843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.$ext" \
       "testdata/843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.$ext"
done

# Copy license file
cp resampler/music_license testdata/LICENSE.txt
```

## Usage in Code

After reorganization, update your examples:

### Before:
```go
inPath := "Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg"
```

### After:
```go
// From examples/resampler/
inPath := "../testdata/Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg"

// From examples/profile_resampler/
inPath := "../testdata/843594__glorytothemachine__sneakers-128bpm-melodic_data_stream-glorytothemachine.mp3"
```

**In shell (use quotes):**
```bash
go run main.go "../testdata/Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg" output.wav
```

## .gitignore Recommendations

Add to `.gitignore` to keep only originals in git:

```gitignore
# Keep only original test files
examples/testdata/*.wav
examples/testdata/*.aiff
examples/testdata/capricerie.mp3
examples/testdata/sneakers.ogg

# Or keep all test files (if they're not too large)
# examples/testdata/*
# !examples/testdata/LICENSE
# !examples/testdata/README.md
# !examples/testdata/capricerie.ogg
# !examples/testdata/sneakers.mp3
```

## License File

Your license information is good! The improvements made:

1. ✅ **Better formatting** - Markdown with clear sections
2. ✅ **Complete attribution** - Artist, source, license per file
3. ✅ **Usage terms** - Clear testing-only purpose
4. ✅ **File list** - Shows which files belong to which track

The new `testdata/LICENSE` file includes:
- Proper markdown formatting
- Clear sections for each track
- Direct links to sources
- License information (CC-BY 4.0 for both)
- Usage restrictions
- Purpose documentation

## Migration Checklist

- [ ] Run `./organize_testdata.sh` or manually move files
- [ ] Verify testdata structure: `ls -lh testdata/`
- [ ] Update examples to use new paths
- [ ] Update profile_resampler to use new paths
- [ ] Update .gitignore (optional)
- [ ] Test that examples still work
- [ ] Remove old output files from resampler/ directory

## File Naming Convention

**Short, memorable names:**
- `capricerie.{ogg,mp3,wav,aiff}` - Classical music piece
- `sneakers.{mp3,wav,ogg,aiff}` - Electronic music

**Why these names?**
- Easy to type and remember
- No special characters or spaces
- Short enough for command line use
- Descriptive enough to identify

## Testing After Migration

```bash
# Test from resampler example
cd examples/resampler
go run main.go ../testdata/capricerie.ogg output.wav

# Test from profiler
cd examples/profile_resampler
go run main.go ../testdata/sneakers.mp3 output.wav
```

## Questions?

This structure:
1. ✅ Follows Go conventions (`testdata` is special in Go)
2. ✅ Allows sharing across all examples
3. ✅ Makes licensing clear and central
4. ✅ Simplifies file names
5. ✅ Separates originals from converted formats
