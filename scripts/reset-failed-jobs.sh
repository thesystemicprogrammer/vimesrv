#!/bin/bash
#
# reset-failed-jobs.sh - Reset failed jobs and transcodes
#
# This script resets:
#   - Jobs with 'dead' status -> 'queued' (with attempt = 0)
#   - Transcodes with 'failed' status -> 'pending'
#
# Usage:
#   ./scripts/reset-failed-jobs.sh              # Dry-run (preview only)
#   ./scripts/reset-failed-jobs.sh --apply      # Apply changes with confirmation
#   ./scripts/reset-failed-jobs.sh --apply --yes # Apply without confirmation
#   ./scripts/reset-failed-jobs.sh --db /path/to/db --apply
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
            echo "Reset failed jobs and transcodes to allow them to be retried."
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

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}  Failed Jobs/Transcodes Reset Script${NC}"
echo -e "${BLUE}======================================${NC}"
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

# Count failed jobs
FAILED_JOBS=$(get_count "SELECT COUNT(*) FROM jobs WHERE status = 'dead';")
echo -e "Failed jobs (dead):         ${YELLOW}$FAILED_JOBS${NC}"

# Count failed transcodes
FAILED_TRANSCODES=$(get_count "SELECT COUNT(*) FROM transcodes WHERE status = 'failed';")
echo -e "Failed transcodes (failed): ${YELLOW}$FAILED_TRANSCODES${NC}"
echo ""

# If nothing to do, exit
if [[ "$FAILED_JOBS" -eq 0 && "$FAILED_TRANSCODES" -eq 0 ]]; then
    echo -e "${GREEN}No failed jobs or transcodes found. Nothing to do.${NC}"
    exit 0
fi

# Show failed jobs details
if [[ "$FAILED_JOBS" -gt 0 ]]; then
    echo -e "${BLUE}Failed Jobs:${NC}"
    run_sql "
        SELECT 
            id,
            type,
            status,
            attempt,
            max_attempts,
            substr(last_error, 1, 50) as last_error,
            datetime(finished_at) as finished_at
        FROM jobs 
        WHERE status = 'dead'
        ORDER BY id;
    "
    echo ""
fi

# Show failed transcodes details
if [[ "$FAILED_TRANSCODES" -gt 0 ]]; then
    echo -e "${BLUE}Failed Transcodes:${NC}"
    run_sql "
        SELECT 
            id,
            media_id,
            track_type,
            quality,
            status
        FROM transcodes 
        WHERE status = 'failed'
        ORDER BY id;
    "
    echo ""
fi

# If dry-run, show what would happen and exit
if $DRY_RUN; then
    echo -e "${BLUE}--- Changes that would be applied ---${NC}"
    echo ""
    echo "1. Reset $FAILED_JOBS jobs: dead -> queued"
    echo "   - Clear worker_id"
    echo "   - Clear started_at"
    echo "   - Clear finished_at"
    echo "   - Clear last_error"
    echo "   - Reset attempt to 0"
    echo "   - Update updated_at"
    echo ""
    echo "2. Reset $FAILED_TRANSCODES transcodes: failed -> pending"
    echo "   - Update updated_at"
    echo ""
    echo -e "${YELLOW}This is a dry-run. No changes were made.${NC}"
    echo -e "Run with ${GREEN}--apply${NC} to apply these changes."
    exit 0
fi

# Confirm before applying
if ! $SKIP_CONFIRM; then
    echo -e "${YELLOW}This will reset $FAILED_JOBS jobs and $FAILED_TRANSCODES transcodes.${NC}"
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

# Reset failed jobs
if [[ "$FAILED_JOBS" -gt 0 ]]; then
    echo -n "Resetting $FAILED_JOBS failed jobs... "
    sqlite3 "$DB_PATH" "
        UPDATE jobs 
        SET status = 'queued', 
            worker_id = NULL, 
            started_at = NULL, 
            finished_at = NULL,
            last_error = NULL,
            attempt = 0,
            updated_at = CURRENT_TIMESTAMP 
        WHERE status = 'dead';
    "
    echo -e "${GREEN}Done${NC}"
fi

# Reset failed transcodes
if [[ "$FAILED_TRANSCODES" -gt 0 ]]; then
    echo -n "Resetting $FAILED_TRANSCODES failed transcodes... "
    sqlite3 "$DB_PATH" "
        UPDATE transcodes 
        SET status = 'pending', 
            updated_at = CURRENT_TIMESTAMP 
        WHERE status = 'failed';
    "
    echo -e "${GREEN}Done${NC}"
fi

echo ""
echo -e "${BLUE}--- After State ---${NC}"
echo ""

# Verify changes
NEW_FAILED_JOBS=$(get_count "SELECT COUNT(*) FROM jobs WHERE status = 'dead';")
NEW_FAILED_TRANSCODES=$(get_count "SELECT COUNT(*) FROM transcodes WHERE status = 'failed';")

echo -e "Failed jobs (dead):         ${GREEN}$NEW_FAILED_JOBS${NC} (was: $FAILED_JOBS)"
echo -e "Failed transcodes (failed): ${GREEN}$NEW_FAILED_TRANSCODES${NC} (was: $FAILED_TRANSCODES)"
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
