#!/bin/bash
#
# VimeSRV Media Player Script
#
# Description:
#   Interactive script to select and play media files from VimeSRV using mpv.
#   Allows choosing between HLS and DASH streaming protocols.
#
# Usage:
#   ./scripts/play-media.sh

set -e
set -u
set -o pipefail

# Constants
DB_PATH="$HOME/dev/git/vimesrv/bin/data/vimesrv.db"
SERVER_URL="http://127.0.0.1:8080"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to format duration in seconds to human-readable format
format_duration() {
    local total_seconds=$1
    local hours=$((total_seconds / 3600))
    local minutes=$(((total_seconds % 3600) / 60))
    local seconds=$((total_seconds % 60))
    
    if [ $hours -gt 0 ]; then
        printf "%dh %02dm %02ds" $hours $minutes $seconds
    else
        printf "%dm %02ds" $minutes $seconds
    fi
}

# Function to print error and exit
error_exit() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

# Check dependencies
command -v sqlite3 >/dev/null 2>&1 || error_exit "sqlite3 is not installed"
command -v mpv >/dev/null 2>&1 || error_exit "mpv is not installed"

# Check database exists
[ -f "$DB_PATH" ] || error_exit "Database not found at $DB_PATH"

echo -e "${BLUE}VimeSRV Media Player${NC}"
echo "=========================="
echo ""

# Query media files
echo "Loading media files..."
media_data=$(sqlite3 "$DB_PATH" "SELECT id, filename, duration FROM media_files WHERE status = 'ready' ORDER BY created_at DESC" 2>/dev/null) || error_exit "Failed to query database"

# Check if any media files exist
if [ -z "$media_data" ]; then
    error_exit "No media files found in database"
fi

# Parse and display media files
echo -e "${GREEN}Available Media Files:${NC}"
echo ""

declare -a media_ids
declare -a media_filenames
index=1

while IFS='|' read -r id filename duration; do
    media_ids+=("$id")
    media_filenames+=("$filename")
    formatted_duration=$(format_duration "$duration")
    echo -e "  ${YELLOW}$index.${NC} $filename ${BLUE}($formatted_duration)${NC}"
    ((index++))
done <<< "$media_data"

total_media=$((index - 1))
echo ""

# Prompt for media selection
while true; do
    read -p "Select media file (1-$total_media): " media_selection
    
    # Validate input is a number
    if ! [[ "$media_selection" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}Invalid input. Please enter a number.${NC}"
        continue
    fi
    
    # Validate input is in range
    if [ "$media_selection" -lt 1 ] || [ "$media_selection" -gt "$total_media" ]; then
        echo -e "${RED}Invalid selection. Please enter a number between 1 and $total_media.${NC}"
        continue
    fi
    
    break
done

# Get selected media ID
selected_index=$((media_selection - 1))
selected_id="${media_ids[$selected_index]}"
selected_filename="${media_filenames[$selected_index]}"

echo ""
echo -e "${GREEN}Selected:${NC} $selected_filename"
echo ""

# Prompt for protocol selection
echo -e "${GREEN}Choose streaming protocol:${NC}"
echo -e "  ${YELLOW}1.${NC} HLS"
echo -e "  ${YELLOW}2.${NC} DASH"
echo ""

while true; do
    read -p "Select protocol (1-2): " protocol_selection
    
    # Validate input is a number
    if ! [[ "$protocol_selection" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}Invalid input. Please enter 1 or 2.${NC}"
        continue
    fi
    
    # Validate input is 1 or 2
    if [ "$protocol_selection" -ne 1 ] && [ "$protocol_selection" -ne 2 ]; then
        echo -e "${RED}Invalid selection. Please enter 1 or 2.${NC}"
        continue
    fi
    
    break
done

# Build streaming URL
if [ "$protocol_selection" -eq 1 ]; then
    protocol="HLS"
    streaming_url="${SERVER_URL}/stream/hls/${selected_id}/master.m3u8"
else
    protocol="DASH"
    streaming_url="${SERVER_URL}/stream/dash/${selected_id}/manifest.mpd"
fi

echo ""
echo -e "${GREEN}Protocol:${NC} $protocol"
echo -e "${GREEN}URL:${NC} $streaming_url"
echo ""

# Fetch and display manifest
echo -e "${BLUE}Fetching manifest...${NC}"
echo "=========================="
command -v curl >/dev/null 2>&1 || error_exit "curl is not installed"
manifest_content=$(curl -s "$streaming_url") || error_exit "Failed to fetch manifest from server"
echo "$manifest_content"
echo "=========================="
echo ""

echo "Starting mpv..."
echo ""

# Launch mpv
mpv "$streaming_url"

# Capture mpv exit code
exit_code=$?

if [ $exit_code -eq 0 ]; then
    echo ""
    echo -e "${GREEN}Playback completed successfully${NC}"
else
    echo ""
    echo -e "${RED}mpv exited with code $exit_code${NC}"
fi

exit $exit_code
