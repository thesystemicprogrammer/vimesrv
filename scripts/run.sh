#!/bin/bash
#
# VimeSRV Runner Script
#
# Usage:
#   ./scripts/run.sh         # Normal run: build and start vimesrv
#   ./scripts/run.sh --clean # Clean run: reset bin directory, copy config and test video, build and start
#
# Description:
#   This script builds and runs vimesrv from the bin directory. In clean mode, it sets up
#   a fresh environment with the configuration file and test video.

set -e
set -u
set -o pipefail

# Set and export TMDB configuration
export TMDB_ENABLED=true
export TMDB_API_KEY=89e0c9aacea8b7d6d6cb590a9fd564e1

# Determine project root (parent of scripts directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Define paths
BIN_DIR="$PROJECT_ROOT/bin"
CONFIGS_SRC="$PROJECT_ROOT/configs"
FIXTURES_DIR="$PROJECT_ROOT/test/fixtures"

# Parse arguments
CLEAN_MODE=false
if [[ "${1:-}" == "--clean" ]]; then
    CLEAN_MODE=true
fi

# Clean mode operations
if [ "$CLEAN_MODE" = true ]; then
    echo "Running in clean mode..."
    
    # Remove bin contents
    echo "Cleaning bin directory..."
    rm -rf "$BIN_DIR"/*
    
    # Create directory structure
    echo "Creating directory structure..."
    mkdir -p "$BIN_DIR/configs"
    mkdir -p "$BIN_DIR/library/staging"
    mkdir -p "$BIN_DIR/data"
    
    # Copy config
    echo "Copying configuration..."
    cp "$CONFIGS_SRC/default.yaml" "$BIN_DIR/configs/"
    
    # Copy all video files from test/fixtures if any exist
    echo "Copying video files from test/fixtures..."
    VIDEO_COUNT=0
    for ext in mkv mp4 avi mov wmv flv webm m4v mpeg mpg ts vob 3gp 3g2 mts m2ts; do
        if ls "$FIXTURES_DIR"/*.$ext 1> /dev/null 2>&1; then
            cp "$FIXTURES_DIR"/*.$ext "$BIN_DIR/library/staging/"
            COUNT=$(ls "$FIXTURES_DIR"/*.$ext 2>/dev/null | wc -l)
            VIDEO_COUNT=$((VIDEO_COUNT + COUNT))
        fi
    done
    if [ "$VIDEO_COUNT" -gt 0 ]; then
        echo "Copied $VIDEO_COUNT video file(s)"
    else
        echo "No video files found in $FIXTURES_DIR"
    fi
fi

# Build
echo "Building vimesrv..."
cd "$PROJECT_ROOT"

if ! make build-prod-pwa; then
    echo "ERROR: Build failed!"
    exit 1
fi

# Run from bin directory
echo "Starting vimesrv from bin directory..."
cd "$BIN_DIR"
./vimesrv &

# Store the PID
VIMESRV_PID=$!

# Wait for server to start
echo "Waiting 3 seconds for server to start..."
sleep 3

# Wait for the server process
wait $VIMESRV_PID
