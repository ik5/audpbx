#!/bin/bash
# SPDX-License-Identifier: EPL-2.0
#
# Test Audio File Management Script
#
# This script helps manage test audio files:
# - Converts files to all needed formats (MP3, OGG, WAV, AIFF)
# - Preserves metadata during conversion
# - Updates LICENSE file with new entries
# - Validates file integrity

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
FORMATS=("mp3" "ogg" "wav" "aiff")
REQUIRED_TOOLS=("ffmpeg" "ffprobe")

#############################################
# Helper Functions
#############################################

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

check_dependencies() {
    print_header "Checking Dependencies"

    local missing=0
    for tool in "${REQUIRED_TOOLS[@]}"; do
        if command -v "$tool" &> /dev/null; then
            print_success "$tool is installed"
        else
            print_error "$tool is not installed"
            missing=1
        fi
    done

    echo ""

    if [ $missing -eq 1 ]; then
        print_error "Missing required tools. Please install:"
        echo "  Ubuntu/Debian: sudo apt install ffmpeg"
        echo "  macOS: brew install ffmpeg"
        echo "  Arch: sudo pacman -S ffmpeg"
        exit 1
    fi
}

get_file_info() {
    local file="$1"

    # Get duration
    local duration=$(ffprobe -v quiet -show_format "$file" | grep "^duration=" | cut -d= -f2)
    local duration_formatted=$(printf "%02d:%02d" $((${duration%.*}/60)) $((${duration%.*}%60)))

    # Get size
    local size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
    local size_mb=$(echo "scale=1; $size / 1048576" | bc)

    # Get audio info
    local sample_rate=$(ffprobe -v quiet -select_streams a:0 -show_entries stream=sample_rate -of default=noprint_wrappers=1:nokey=1 "$file")
    local channels=$(ffprobe -v quiet -select_streams a:0 -show_entries stream=channels -of default=noprint_wrappers=1:nokey=1 "$file")
    local channel_layout=$([ "$channels" -eq 1 ] && echo "Mono" || echo "Stereo")

    echo "$duration_formatted|$size_mb|$sample_rate|$channel_layout"
}

get_metadata() {
    local file="$1"

    # Try to get artist
    local artist=$(ffprobe -v quiet -show_entries format_tags=artist -of default=noprint_wrappers=1:nokey=1 "$file" 2>/dev/null || echo "")

    # Try to get title
    local title=$(ffprobe -v quiet -show_entries format_tags=title -of default=noprint_wrappers=1:nokey=1 "$file" 2>/dev/null || echo "")

    echo "$artist|$title"
}

get_extension() {
    local filename="$1"
    echo "${filename##*.}"
}

get_basename() {
    local filename="$1"
    echo "${filename%.*}"
}

convert_file() {
    local input="$1"
    local output="$2"
    local output_ext=$(get_extension "$output")

    print_info "Converting to $output_ext..."

    case "$output_ext" in
        mp3)
            ffmpeg -i "$input" -map_metadata 0 -codec:a libmp3lame -q:a 2 -id3v2_version 3 "$output" -y 2>&1 | grep -v "^frame="
            ;;
        ogg)
            ffmpeg -i "$input" -map_metadata 0 -codec:a libvorbis -q:a 6 "$output" -y 2>&1 | grep -v "^frame="
            ;;
        wav)
            ffmpeg -i "$input" -map_metadata 0 -codec:a pcm_s16le "$output" -y 2>&1 | grep -v "^frame="
            ;;
        aiff)
            ffmpeg -i "$input" -map_metadata 0 -codec:a pcm_s16be "$output" -y 2>&1 | grep -v "^frame="
            ;;
        *)
            print_error "Unsupported format: $output_ext"
            return 1
            ;;
    esac

    if [ $? -eq 0 ]; then
        print_success "Created: $output"
        return 0
    else
        print_error "Failed to create: $output"
        return 1
    fi
}

#############################################
# Main Functions
#############################################

add_new_file() {
    print_header "Add New Test File"

    # Get input file
    if [ -z "$1" ]; then
        echo "Enter the path to the audio file:"
        read -r input_file
    else
        input_file="$1"
    fi

    # Validate input file
    if [ ! -f "$input_file" ]; then
        print_error "File not found: $input_file"
        exit 1
    fi

    print_success "Input file: $input_file"
    echo ""

    # Get file info
    local file_info=$(get_file_info "$input_file")
    IFS='|' read -r duration size_mb sample_rate channels <<< "$file_info"

    print_info "File Information:"
    echo "  Duration: $duration"
    echo "  Size: ${size_mb} MB"
    echo "  Sample Rate: ${sample_rate} Hz"
    echo "  Channels: $channels"
    echo ""

    # Get metadata
    local metadata=$(get_metadata "$input_file")
    IFS='|' read -r artist title <<< "$metadata"

    if [ -n "$artist" ]; then
        print_info "Embedded Metadata:"
        echo "  Artist: $artist"
        echo "  Title: $title"
        echo ""
    fi

    # Ask for filename
    local input_basename=$(basename "$input_file")
    local input_ext=$(get_extension "$input_basename")

    echo "Keep original filename? (y/n)"
    echo "  Original: $input_basename"
    read -r keep_name

    if [[ "$keep_name" =~ ^[Yy]$ ]]; then
        target_basename="$input_basename"
    else
        echo "Enter target filename (without extension):"
        read -r custom_name
        target_basename="${custom_name}.${input_ext}"
    fi

    local target_file="$target_basename"
    local base_name=$(get_basename "$target_basename")

    # Copy/move original file
    if [ "$input_file" != "$target_file" ]; then
        echo ""
        echo "Copy or move the original file? (c/m)"
        read -r copy_or_move

        if [[ "$copy_or_move" =~ ^[Mm]$ ]]; then
            mv "$input_file" "$target_file"
            print_success "Moved: $target_file"
        else
            cp "$input_file" "$target_file"
            print_success "Copied: $target_file"
        fi
    fi

    echo ""
    print_header "Converting to Other Formats"

    # Convert to all other formats
    for format in "${FORMATS[@]}"; do
        output_file="${base_name}.${format}"

        if [ -f "$output_file" ] && [ "$output_file" != "$target_file" ]; then
            print_warning "$output_file already exists, skipping..."
        elif [ "$output_file" == "$target_file" ]; then
            print_info "$output_file is the source file, skipping..."
        else
            convert_file "$target_file" "$output_file"
        fi
    done

    echo ""
    print_header "Update LICENSE File"

    # Get source information
    echo "Enter the source information:"
    echo ""
    echo "Artist name:"
    read -r artist_name

    echo "Track title:"
    read -r track_title

    echo "Source URL (e.g., Freesound, Jamendo):"
    read -r source_url

    echo "License (default: CC-BY 4.0):"
    read -r license
    license=${license:-"CC-BY 4.0"}

    echo "Short description:"
    read -r description

    # Generate LICENSE entry
    echo ""
    print_info "Generated LICENSE entry:"
    echo ""
    echo "---"
    echo "## Track X: $track_title"
    echo ""
    echo "- **Artist:** $artist_name"
    echo "- **Title:** $track_title"
    echo "- **Source:** $source_url"
    echo "- **License:** $license (https://creativecommons.org/licenses/by/4.0/)"
    echo "- **Description:** $description. ${duration} duration, ${channels}, ${sample_rate} Hz."
    echo "- **Files:**"
    for format in "${FORMATS[@]}"; do
        echo "  - \`${base_name}.${format}\`$([ "$format" == "$input_ext" ] && echo " (original)")"
    done
    echo ""
    echo "**Usage:** Testing purposes only under $license license."
    echo ""
    if [[ "$source_url" == *"freesound"* ]]; then
        echo "**Attribution:** Credit to \"$artist_name\" from www.freesound.org"
        echo ""
    fi
    echo "---"
    echo ""

    echo "Add this entry to LICENSE file? (y/n)"
    read -r add_license

    if [[ "$add_license" =~ ^[Yy]$ ]]; then
        # Find the last track number
        local last_track=$(grep -o "## Track [0-9]*:" LICENSE 2>/dev/null | grep -o "[0-9]*" | tail -1)
        local next_track=$((last_track + 1))

        # Create temporary entry
        cat > /tmp/new_track_entry.txt << EOF

## Track $next_track: $track_title

- **Artist:** $artist_name
- **Title:** $track_title
- **Source:** $source_url
- **License:** $license (https://creativecommons.org/licenses/by/4.0/)
- **Description:** $description. ${duration} duration, ${channels}, ${sample_rate} Hz.
- **Files:**
EOF

        for format in "${FORMATS[@]}"; do
            echo "  - \`${base_name}.${format}\`$([ "$format" == "$input_ext" ] && echo " (original)")" >> /tmp/new_track_entry.txt
        done

        cat >> /tmp/new_track_entry.txt << EOF

**Usage:** Testing purposes only under $license license.

EOF

        if [[ "$source_url" == *"freesound"* ]]; then
            echo "**Attribution:** Credit to \"$artist_name\" from www.freesound.org" >> /tmp/new_track_entry.txt
            echo "" >> /tmp/new_track_entry.txt
        fi

        # Insert before the "---" separator
        if [ -f LICENSE ]; then
            # Find line number of last "---"
            local separator_line=$(grep -n "^---$" LICENSE | tail -1 | cut -d: -f1)

            if [ -n "$separator_line" ]; then
                head -n $((separator_line - 1)) LICENSE > LICENSE.tmp
                cat /tmp/new_track_entry.txt >> LICENSE.tmp
                tail -n +$separator_line LICENSE >> LICENSE.tmp
                mv LICENSE.tmp LICENSE
                print_success "LICENSE file updated"
            else
                cat /tmp/new_track_entry.txt >> LICENSE
                print_success "LICENSE entry appended"
            fi
        else
            print_error "LICENSE file not found"
        fi

        rm /tmp/new_track_entry.txt
    fi

    echo ""
    print_success "Done! All formats created and LICENSE updated."
    echo ""
    print_info "Files created:"
    for format in "${FORMATS[@]}"; do
        local file="${base_name}.${format}"
        if [ -f "$file" ]; then
            local file_size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
            local size_mb=$(echo "scale=1; $file_size / 1048576" | bc)
            echo "  - $file (${size_mb} MB)"
        fi
    done
}

convert_existing() {
    print_header "Convert Existing File"

    # List existing audio files
    echo "Available audio files:"
    echo ""

    local files=(*.mp3 *.ogg *.wav *.aiff 2>/dev/null)
    local i=1

    for file in "${files[@]}"; do
        if [ -f "$file" ]; then
            local size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
            local size_mb=$(echo "scale=1; $size / 1048576" | bc)
            echo "  $i) $file (${size_mb} MB)"
            ((i++))
        fi
    done

    echo ""
    echo "Enter filename to convert:"
    read -r input_file

    if [ ! -f "$input_file" ]; then
        print_error "File not found: $input_file"
        exit 1
    fi

    local base_name=$(get_basename "$input_file")

    echo ""
    print_header "Converting $input_file"

    for format in "${FORMATS[@]}"; do
        output_file="${base_name}.${format}"

        if [ -f "$output_file" ]; then
            print_warning "$output_file exists. Overwrite? (y/n)"
            read -r overwrite
            if [[ ! "$overwrite" =~ ^[Yy]$ ]]; then
                continue
            fi
        fi

        if [ "$output_file" == "$input_file" ]; then
            print_info "Skipping source format: $format"
            continue
        fi

        convert_file "$input_file" "$output_file"
    done

    print_success "Conversion complete!"
}

list_files() {
    print_header "Test Audio Files"

    for file in *.{mp3,ogg,wav,aiff}; do
        if [ -f "$file" ]; then
            local info=$(get_file_info "$file")
            IFS='|' read -r duration size_mb sample_rate channels <<< "$info"

            echo "📄 $file"
            echo "   Duration: $duration | Size: ${size_mb} MB | ${sample_rate} Hz | $channels"
            echo ""
        fi
    done
}

verify_metadata() {
    print_header "Verify Metadata Preservation"

    echo "Enter base filename (without extension):"
    read -r base_name

    echo ""

    for format in "${FORMATS[@]}"; do
        local file="${base_name}.${format}"

        if [ ! -f "$file" ]; then
            print_warning "$file not found, skipping..."
            continue
        fi

        echo "=== $file ==="
        ffprobe -v quiet -show_entries format_tags -of default=noprint_wrappers=1 "$file" 2>/dev/null | grep "^TAG:" | head -10
        echo ""
    done
}

show_menu() {
    clear
    print_header "Test Audio File Manager"

    echo "1) Add new file (convert to all formats)"
    echo "2) Convert existing file to other formats"
    echo "3) List all test files"
    echo "4) Verify metadata preservation"
    echo "5) Check dependencies"
    echo "6) Exit"
    echo ""
    echo -n "Select option: "
}

#############################################
# Main Script
#############################################

# Change to script directory
cd "$(dirname "$0")"

# Check if file was provided as argument
if [ $# -eq 1 ]; then
    check_dependencies
    add_new_file "$1"
    exit 0
fi

# Interactive mode
while true; do
    show_menu
    read -r choice

    case $choice in
        1)
            check_dependencies
            add_new_file
            echo ""
            read -p "Press Enter to continue..."
            ;;
        2)
            check_dependencies
            convert_existing
            echo ""
            read -p "Press Enter to continue..."
            ;;
        3)
            list_files
            echo ""
            read -p "Press Enter to continue..."
            ;;
        4)
            verify_metadata
            echo ""
            read -p "Press Enter to continue..."
            ;;
        5)
            check_dependencies
            echo ""
            read -p "Press Enter to continue..."
            ;;
        6)
            print_success "Goodbye!"
            exit 0
            ;;
        *)
            print_error "Invalid option"
            sleep 2
            ;;
    esac
done
