#!/bin/bash
#
# VimeSRV Build Script for Raspberry Pi Test Setup
#
# Usage:
#   ./scripts/build-pi.sh
#
# Description:
#   This script builds all components for production and deploys the server
#   binary to the Raspberry Pi. It performs the following steps:
#   1. Build the PWA
#   2. Build production server (vimesrv)
#   3. Build production worker (vimesrv-worker)
#   4. Build Pi server (vimesrv-linux-arm64)
#   5. Copy worker binary to ./bin/worker/
#   6. Deploy vimesrv-linux-arm64 to Raspberry Pi
#

set -e
set -u
set -o pipefail

# Configuration
PI_HOST="ras01.home"
PI_USER="thomas"
PI_DEST="/home/thomas/vimesrv"

# Determine project root (parent of scripts directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "=========================================="
echo "VimeSRV Build Script for Pi Test Setup"
echo "=========================================="
echo ""

# Step 1: Build PWA
echo "[1/6] Building PWA..."
make build-prod-pwa
echo ""

# Step 2: Build production server
echo "[2/6] Building production server..."
make build-prod-server
echo ""

# Step 3: Build production worker
echo "[3/6] Building production worker..."
make build-prod-worker
echo ""

# Step 4: Build Pi server
echo "[4/6] Building Pi server (linux/arm64)..."
make build-prod-pi
echo ""

# Step 5: Copy worker to worker directory
echo "[5/6] Copying worker binary to ./bin/worker/..."
cp "$PROJECT_ROOT/bin/vimesrv-worker" "$PROJECT_ROOT/bin/worker/"
echo "Worker copied to ./bin/worker/vimesrv-worker"
echo ""

# Step 6: Deploy to Raspberry Pi
echo "[6/6] Deploying to Raspberry Pi ($PI_USER@$PI_HOST)..."
echo "Creating remote directory if needed..."
ssh "$PI_USER@$PI_HOST" "mkdir -p $PI_DEST"
echo "Copying vimesrv-linux-arm64 to Pi..."
scp "$PROJECT_ROOT/bin/vimesrv-linux-arm64" "$PI_USER@$PI_HOST:$PI_DEST/"
echo "Deployment complete."
echo ""

echo "=========================================="
echo "Build and deployment finished successfully!"
echo "=========================================="
echo ""
echo "Local binaries:"
echo "  - ./bin/vimesrv"
echo "  - ./bin/vimesrv-worker"
echo "  - ./bin/worker/vimesrv-worker"
echo "  - ./bin/vimesrv-linux-arm64"
echo ""
echo "Deployed to Pi:"
echo "  - $PI_USER@$PI_HOST:$PI_DEST/vimesrv-linux-arm64"
