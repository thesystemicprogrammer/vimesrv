#!/bin/bash
#
# reset-stuck-jobs.sh - Reset jobs and transcodes stuck in running/processing state
#
# This script resets:
#   - Jobs stuck in 'running' status -> 'queued'
#   - Transcodes stuck in 'processing' status -> 'pending'
#
# Usage:
#   ./scripts/reset-stuck-jobs.sh              # Dry-run (preview only)
#   ./scripts/reset-stuck-jobs.sh --apply      # Apply changes with confirmation
#   ./scripts/reset-stuck-jobs.sh --apply --yes # Apply without confirmation
#   ./scripts/reset-stuck-jobs.sh --db /path/to/db --apply
#

set -e

# Default values
DB_PATH="./bin/data/vimesrv.db"
DRY_RUN=true
SKIP_CONFIRM=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --db)
            DB_PATH="$2"
            shift 2
            ;;
        --apply)
            DRY_RUN=false
            shift
            ;;
        --yes|-y)
            SKIP_CONFIRM=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Reset jobs and transcodes stuck in running/processing state."
            echo ""
            echo "Options:"
            echo "  --db PATH    Path to database (default: ./bin/data/vimesrv.db)"
            echo "  --apply      Apply changes (default is dry-run)"
            echo "  --yes, -y    Skip confirmation prompt"
            echo "  --help, -h   Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                    # Preview changes (dry-run)"
            echo "  $0 --apply            # Apply with confirmation"
            echo "  $0 --apply --yes      # Apply without confirmation"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Check if database exists
if [[ ! -f "$DB_PATH" ]]; then
    echo -e "${RED}Error: Database not found at: $DB_PATH${NC}"
    echo "Use --db to specify the database path"
    exit 1
fi

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Stuck Jobs/Transcodes Reset Script${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "Database: ${YELLOW}$DB_PATH${NC}"
if $DRY_RUN; then
    echo -e "Mode:     ${YELLOW}DRY-RUN (no changes will be made)${NC}"
else
    echo -e "Mode:     ${RED}APPLY (changes will be made)${NC}"
fi
echo ""

# Function to run SQL and display results
run_sql() {
    sqlite3 -header -column "$DB_PATH" "$1"
}

# Function to run SQL and get count
get_count() {
    sqlite3 "$DB_PATH" "$1"
}

# Show current state
echo -e "${BLUE}--- Current State ---${NC}"
echo ""

# Count stuck jobs
STUCK_JOBS=$(get_count "SELECT COUNT(*) FROM jobs WHERE status = 'running';")
echo -e "Stuck jobs (running):        ${YELLOW}$STUCK_JOBS${NC}"

# Count stuck transcodes
STUCK_TRANSCODES=$(get_count "SELECT COUNT(*) FROM transcodes WHERE status = 'processing';")
echo -e "Stuck transcodes (processing): ${YELLOW}$STUCK_TRANSCODES${NC}"
echo ""

# If nothing to do, exit
if [[ "$STUCK_JOBS" -eq 0 && "$STUCK_TRANSCODES" -eq 0 ]]; then
    echo -e "${GREEN}No stuck jobs or transcodes found. Nothing to do.${NC}"
    exit 0
fi

# Show stuck jobs details
if [[ "$STUCK_JOBS" -gt 0 ]]; then
    echo -e "${BLUE}Stuck Jobs:${NC}"
    run_sql "
        SELECT 
            id,
            type,
            status,
            worker_id,
            datetime(started_at) as started_at
        FROM jobs 
        WHERE status = 'running'
        ORDER BY id;
    "
    echo ""
fi

# Show stuck transcodes details
if [[ "$STUCK_TRANSCODES" -gt 0 ]]; then
    echo -e "${BLUE}Stuck Transcodes:${NC}"
    run_sql "
        SELECT 
            id,
            media_id,
            track_type,
            quality,
            status
        FROM transcodes 
        WHERE status = 'processing'
        ORDER BY id;
    "
    echo ""
fi

# If dry-run, show what would happen and exit
if $DRY_RUN; then
    echo -e "${BLUE}--- Changes that would be applied ---${NC}"
    echo ""
    echo "1. Reset $STUCK_JOBS jobs: running -> queued"
    echo "   - Clear worker_id"
    echo "   - Clear started_at"
    echo "   - Update updated_at"
    echo ""
    echo "2. Reset $STUCK_TRANSCODES transcodes: processing -> pending"
    echo "   - Update updated_at"
    echo ""
    echo -e "${YELLOW}This is a dry-run. No changes were made.${NC}"
    echo -e "Run with ${GREEN}--apply${NC} to apply these changes."
    exit 0
fi

# Confirm before applying
if ! $SKIP_CONFIRM; then
    echo -e "${YELLOW}This will reset $STUCK_JOBS jobs and $STUCK_TRANSCODES transcodes.${NC}"
    read -p "Are you sure you want to proceed? [y/N] " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Aborted.${NC}"
        exit 0
    fi
fi

echo ""
echo -e "${BLUE}--- Applying Changes ---${NC}"
echo ""

# Reset stuck jobs
if [[ "$STUCK_JOBS" -gt 0 ]]; then
    echo -n "Resetting $STUCK_JOBS stuck jobs... "
    sqlite3 "$DB_PATH" "
        UPDATE jobs 
        SET status = 'queued', 
            worker_id = NULL, 
            started_at = NULL, 
            updated_at = CURRENT_TIMESTAMP 
        WHERE status = 'running';
    "
    echo -e "${GREEN}Done${NC}"
fi

# Reset stuck transcodes
if [[ "$STUCK_TRANSCODES" -gt 0 ]]; then
    echo -n "Resetting $STUCK_TRANSCODES stuck transcodes... "
    sqlite3 "$DB_PATH" "
        UPDATE transcodes 
        SET status = 'pending', 
            updated_at = CURRENT_TIMESTAMP 
        WHERE status = 'processing';
    "
    echo -e "${GREEN}Done${NC}"
fi

echo ""
echo -e "${BLUE}--- After State ---${NC}"
echo ""

# Verify changes
NEW_STUCK_JOBS=$(get_count "SELECT COUNT(*) FROM jobs WHERE status = 'running';")
NEW_STUCK_TRANSCODES=$(get_count "SELECT COUNT(*) FROM transcodes WHERE status = 'processing';")

echo -e "Stuck jobs (running):          ${GREEN}$NEW_STUCK_JOBS${NC} (was: $STUCK_JOBS)"
echo -e "Stuck transcodes (processing): ${GREEN}$NEW_STUCK_TRANSCODES${NC} (was: $STUCK_TRANSCODES)"
echo ""

# Show job status distribution
echo -e "${BLUE}Job Status Distribution:${NC}"
run_sql "SELECT status, COUNT(*) as count FROM jobs GROUP BY status ORDER BY status;"
echo ""

# Show transcode status distribution
echo -e "${BLUE}Transcode Status Distribution:${NC}"
run_sql "SELECT status, COUNT(*) as count FROM transcodes GROUP BY status ORDER BY status;"
echo ""

echo -e "${GREEN}Reset complete!${NC}"
echo "The affected jobs will be re-queued and processed by workers."
