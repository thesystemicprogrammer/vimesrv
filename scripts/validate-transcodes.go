//go:build ignore

// Comprehensive validation and correction tool for transcoded media files.
//
// This script validates the integrity and consistency of transcoded media,
// checking for issues like missing files, corrupted segments, audio/video
// sync drift, and database/filesystem mismatches.
//
// Usage:
//   go run scripts/validate-transcodes.go [options] <database-path> <media-root-path>
//
// Options:
//   --dry-run              Report issues without making changes
//   --fix-segments         Regenerate corrupted/missing segments.json files
//   --fix-orphaned-dirs    Delete transcode directories with no database record
//   --fix-orphaned-records Delete database records with no files on disk
//   --create-transcode-jobs Create transcode jobs for media needing re-transcode
//   --drift-threshold <ms> Audio/video drift threshold in ms (default: 200)
//   --no-probe             Trust existing segments.json instead of probing with ffprobe
//   --verbose              Show detailed output for each media file
//   --json                 Output results as JSON for programmatic use
//
// Examples:
//   go run scripts/validate-transcodes.go ./data/vimesrv.db /mnt/media
//   go run scripts/validate-transcodes.go --fix-segments ./data/vimesrv.db /mnt/media
//   go run scripts/validate-transcodes.go --drift-threshold 500 --verbose ./data/vimesrv.db /mnt/media

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const (
	defaultDriftThreshold = 200 // milliseconds
	probeTimeout          = 30 * time.Minute
)

// SegmentInfo contains timing information for a media segment
type SegmentInfo struct {
	Number   int   `json:"Number"`
	Duration int64 `json:"Duration"`
}

// SegmentsFile represents the structure of segments.json
type SegmentsFile struct {
	Segments []SegmentInfo `json:"segments"`
}

// TranscodeRecord represents a transcode record from the database
type TranscodeRecord struct {
	ID         string
	MediaID    string
	Quality    string
	TrackType  string
	TrackIndex int
	Status     string
	OutputPath string
}

// MediaFile represents a media file from the database
type MediaFile struct {
	ID       string
	FilePath string
	Filename string
	Duration int64 // in milliseconds
}

// ValidationResult holds the result of validating a single media file
type ValidationResult struct {
	MediaID       string   `json:"media_id"`
	Filename      string   `json:"filename"`
	Status        string   `json:"status"` // PASS, WARN, FAIL
	VideoDuration int64    `json:"video_duration_ms,omitempty"`
	AudioDuration int64    `json:"audio_duration_ms,omitempty"`
	DriftMs       int64    `json:"drift_ms,omitempty"`
	Issues        []string `json:"issues,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
	Actions       []string `json:"actions,omitempty"`
}

// SummaryResult holds the overall validation summary
type SummaryResult struct {
	TotalMedia              int                `json:"total_media"`
	Passed                  int                `json:"passed"`
	Warnings                int                `json:"warnings"`
	Failures                int                `json:"failures"`
	OrphanedDirs            int                `json:"orphaned_dirs"`
	OrphanedRecords         int                `json:"orphaned_records"`
	DriftIssues             int                `json:"drift_issues"`
	MissingSegments         int                `json:"missing_segments"`
	CorruptedSegments       int                `json:"corrupted_segments"`
	Results                 []ValidationResult `json:"results,omitempty"`
	MediaNeedingRetranscode []string           `json:"media_needing_retranscode,omitempty"`
}

// Config holds the script configuration
type Config struct {
	DatabasePath        string
	MediaRootPath       string
	DryRun              bool
	FixSegments         bool
	FixOrphanedDirs     bool
	FixOrphanedRecords  bool
	CreateTranscodeJobs bool
	DriftThreshold      int64
	NoProbe             bool
	Verbose             bool
	JSONOutput          bool
}

func main() {
	config := parseArgs()

	if config.DatabasePath == "" || config.MediaRootPath == "" {
		printUsage()
		os.Exit(1)
	}

	// Verify ffprobe is available (unless --no-probe)
	if !config.NoProbe {
		if _, err := exec.LookPath("ffprobe"); err != nil {
			fmt.Println("Error: ffprobe not found in PATH (use --no-probe to skip probing)")
			os.Exit(1)
		}
	}

	// Open database
	db, err := sql.Open("sqlite", config.DatabasePath+"?_foreign_keys=on")
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if !config.JSONOutput {
		fmt.Println("Validating transcodes...")
		fmt.Printf("Database: %s\n", config.DatabasePath)
		fmt.Printf("Media root: %s\n", config.MediaRootPath)
		fmt.Printf("Drift threshold: %dms\n", config.DriftThreshold)
		if config.NoProbe {
			fmt.Println("Mode: Trusting existing segments.json (--no-probe)")
		} else {
			fmt.Println("Mode: Probing segments with ffprobe")
		}
		if config.DryRun {
			fmt.Println("DRY RUN MODE - no changes will be made")
		}
		fmt.Println()
	}

	summary := &SummaryResult{}

	// Step 1: Check for orphaned database records (DB entries with no files)
	orphanedRecords := checkOrphanedRecords(db, config, summary)

	// Step 2: Check for orphaned directories (files with no DB entries)
	orphanedDirs := checkOrphanedDirectories(db, config, summary)

	// Step 3: Validate each media file's transcodes
	validateMediaTranscodes(db, config, summary)

	// Step 4: Fix orphaned records if requested
	if config.FixOrphanedRecords && len(orphanedRecords) > 0 {
		fixOrphanedRecords(db, config, orphanedRecords)
	}

	// Step 5: Fix orphaned directories if requested
	if config.FixOrphanedDirs && len(orphanedDirs) > 0 {
		fixOrphanedDirectories(config, orphanedDirs)
	}

	// Step 6: Create transcode jobs if requested
	if config.CreateTranscodeJobs && len(summary.MediaNeedingRetranscode) > 0 {
		createTranscodeJobs(db, config, summary.MediaNeedingRetranscode)
	}

	// Output results
	if config.JSONOutput {
		jsonData, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Println(string(jsonData))
	} else {
		printSummary(summary, config)
	}

	if summary.Failures > 0 {
		os.Exit(1)
	}
}

func parseArgs() Config {
	config := Config{
		DriftThreshold: defaultDriftThreshold,
	}

	args := os.Args[1:]
	positional := []string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			config.DryRun = true
		case "--fix-segments":
			config.FixSegments = true
		case "--fix-orphaned-dirs":
			config.FixOrphanedDirs = true
		case "--fix-orphaned-records":
			config.FixOrphanedRecords = true
		case "--create-transcode-jobs":
			config.CreateTranscodeJobs = true
		case "--drift-threshold":
			if i+1 < len(args) {
				i++
				if val, err := strconv.ParseInt(args[i], 10, 64); err == nil {
					config.DriftThreshold = val
				}
			}
		case "--no-probe":
			config.NoProbe = true
		case "--verbose":
			config.Verbose = true
		case "--json":
			config.JSONOutput = true
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		default:
			if !strings.HasPrefix(args[i], "-") {
				positional = append(positional, args[i])
			}
		}
	}

	if len(positional) >= 2 {
		config.DatabasePath = positional[0]
		config.MediaRootPath = positional[1]
	}

	return config
}

func printUsage() {
	fmt.Println(`Usage: go run scripts/validate-transcodes.go [options] <database-path> <media-root-path>

Validates transcoded media files for integrity, consistency, and audio/video sync.

Options:
  --dry-run              Report issues without making changes
  --fix-segments         Regenerate corrupted/missing segments.json files
  --fix-orphaned-dirs    Delete transcode directories with no database record
  --fix-orphaned-records Delete database records with no files on disk
  --create-transcode-jobs Create transcode jobs for media needing audio re-transcode
  --drift-threshold <ms> Audio/video drift threshold in ms (default: 200)
  --no-probe             Trust existing segments.json instead of probing with ffprobe
  --verbose              Show detailed output for each media file
  --json                 Output results as JSON for programmatic use
  -h, --help             Show this help message

Examples:
  go run scripts/validate-transcodes.go ./data/vimesrv.db /mnt/media
  go run scripts/validate-transcodes.go --fix-segments ./data/vimesrv.db /mnt/media
  go run scripts/validate-transcodes.go --drift-threshold 500 --verbose ./data/vimesrv.db /mnt/media
  go run scripts/validate-transcodes.go --create-transcode-jobs ./data/vimesrv.db /mnt/media`)
}

func checkOrphanedRecords(db *sql.DB, config Config, summary *SummaryResult) []TranscodeRecord {
	var orphaned []TranscodeRecord

	rows, err := db.Query(`
		SELECT t.id, t.media_id, t.quality, t.track_type, t.track_index, t.status, t.output_path
		FROM transcodes t
		WHERE t.status = 'completed' AND t.output_path IS NOT NULL AND t.output_path != ''
	`)
	if err != nil {
		if !config.JSONOutput {
			fmt.Printf("Error querying transcodes: %v\n", err)
		}
		return orphaned
	}
	defer rows.Close()

	for rows.Next() {
		var rec TranscodeRecord
		if err := rows.Scan(&rec.ID, &rec.MediaID, &rec.Quality, &rec.TrackType, &rec.TrackIndex, &rec.Status, &rec.OutputPath); err != nil {
			continue
		}

		// Check if output path exists
		exists := false
		if rec.TrackType == "subtitle" {
			// Subtitles are single files
			checkPath := rec.OutputPath
			if !strings.HasSuffix(checkPath, ".vtt") {
				checkPath = checkPath + ".vtt"
			}
			if _, err := os.Stat(checkPath); err == nil {
				exists = true
			}
		} else {
			// Video/audio have init.mp4 in their directory
			initPath := filepath.Join(rec.OutputPath, "init.mp4")
			if _, err := os.Stat(initPath); err == nil {
				exists = true
			}
		}

		if !exists {
			orphaned = append(orphaned, rec)
		}
	}

	summary.OrphanedRecords = len(orphaned)

	if !config.JSONOutput && len(orphaned) > 0 {
		fmt.Printf("Found %d orphaned database records (no files on disk)\n", len(orphaned))
		if config.Verbose {
			for _, rec := range orphaned {
				fmt.Printf("  - %s (%s/%s)\n", rec.ID, rec.TrackType, rec.Quality)
			}
		}
		fmt.Println()
	}

	return orphaned
}

func checkOrphanedDirectories(db *sql.DB, config Config, summary *SummaryResult) []string {
	var orphaned []string

	// Get all media IDs from database
	mediaIDs := make(map[string]bool)
	rows, err := db.Query("SELECT id FROM media_files")
	if err != nil {
		return orphaned
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			mediaIDs[id] = true
		}
	}

	// Walk media root looking for transcode directories without DB entries
	entries, err := os.ReadDir(config.MediaRootPath)
	if err != nil {
		return orphaned
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		mediaID := entry.Name()
		transcodePath := filepath.Join(config.MediaRootPath, mediaID, "transcoded")

		// Check if transcoded directory exists
		if _, err := os.Stat(transcodePath); err != nil {
			continue
		}

		// Check if media ID exists in database
		if !mediaIDs[mediaID] {
			orphaned = append(orphaned, transcodePath)
		}
	}

	summary.OrphanedDirs = len(orphaned)

	if !config.JSONOutput && len(orphaned) > 0 {
		fmt.Printf("Found %d orphaned transcode directories (no database record)\n", len(orphaned))
		if config.Verbose {
			for _, dir := range orphaned {
				fmt.Printf("  - %s\n", dir)
			}
		}
		fmt.Println()
	}

	return orphaned
}

func validateMediaTranscodes(db *sql.DB, config Config, summary *SummaryResult) {
	// Get all media files with completed transcodes
	rows, err := db.Query(`
		SELECT DISTINCT m.id, m.file_path, m.filename, m.duration
		FROM media_files m
		INNER JOIN transcodes t ON t.media_id = m.id
		WHERE t.status = 'completed'
		ORDER BY m.filename
	`)
	if err != nil {
		if !config.JSONOutput {
			fmt.Printf("Error querying media files: %v\n", err)
		}
		return
	}
	defer rows.Close()

	var mediaFiles []MediaFile
	for rows.Next() {
		var m MediaFile
		var duration sql.NullInt64
		if err := rows.Scan(&m.ID, &m.FilePath, &m.Filename, &duration); err != nil {
			continue
		}
		if duration.Valid {
			m.Duration = duration.Int64
		}
		mediaFiles = append(mediaFiles, m)
	}

	summary.TotalMedia = len(mediaFiles)

	for _, media := range mediaFiles {
		result := validateSingleMedia(db, config, media)
		summary.Results = append(summary.Results, result)

		switch result.Status {
		case "PASS":
			summary.Passed++
		case "WARN":
			summary.Warnings++
		case "FAIL":
			summary.Failures++
		}

		// Track specific issue counts
		for _, issue := range result.Issues {
			if strings.Contains(issue, "drift") {
				summary.DriftIssues++
				summary.MediaNeedingRetranscode = append(summary.MediaNeedingRetranscode, media.ID)
			}
			if strings.Contains(issue, "Missing segments.json") {
				summary.MissingSegments++
			}
			if strings.Contains(issue, "Corrupted segments.json") {
				summary.CorruptedSegments++
			}
		}
		for _, warn := range result.Warnings {
			if strings.Contains(warn, "Missing segments.json") {
				summary.MissingSegments++
			}
			if strings.Contains(warn, "Corrupted segments.json") {
				summary.CorruptedSegments++
			}
		}

		if !config.JSONOutput {
			printValidationResult(result, config)
		}
	}
}

func validateSingleMedia(db *sql.DB, config Config, media MediaFile) ValidationResult {
	result := ValidationResult{
		MediaID:  media.ID,
		Filename: media.Filename,
		Status:   "PASS",
	}

	// Get transcodes for this media
	rows, err := db.Query(`
		SELECT id, quality, track_type, track_index, output_path
		FROM transcodes
		WHERE media_id = ? AND status = 'completed'
		ORDER BY track_type, track_index
	`, media.ID)
	if err != nil {
		result.Status = "FAIL"
		result.Issues = append(result.Issues, fmt.Sprintf("Database query error: %v", err))
		return result
	}
	defer rows.Close()

	var videoTranscodes []TranscodeRecord
	var audioTranscodes []TranscodeRecord
	var subtitleTranscodes []TranscodeRecord

	for rows.Next() {
		var rec TranscodeRecord
		rec.MediaID = media.ID
		if err := rows.Scan(&rec.ID, &rec.Quality, &rec.TrackType, &rec.TrackIndex, &rec.OutputPath); err != nil {
			continue
		}

		switch rec.TrackType {
		case "video":
			videoTranscodes = append(videoTranscodes, rec)
		case "audio":
			audioTranscodes = append(audioTranscodes, rec)
		case "subtitle":
			subtitleTranscodes = append(subtitleTranscodes, rec)
		}
	}

	// Validate video transcodes
	var videoDuration int64
	for _, vt := range videoTranscodes {
		dur, issues, warnings := validateTranscodeDirectory(config, vt, "video")
		if dur > 0 {
			videoDuration = dur
			result.VideoDuration = dur
		}
		result.Issues = append(result.Issues, issues...)
		result.Warnings = append(result.Warnings, warnings...)
	}

	// Validate audio transcodes
	var audioDuration int64
	for _, at := range audioTranscodes {
		dur, issues, warnings := validateTranscodeDirectory(config, at, "audio")
		if dur > 0 && (audioDuration == 0 || dur > audioDuration) {
			audioDuration = dur
			result.AudioDuration = dur
		}
		result.Issues = append(result.Issues, issues...)
		result.Warnings = append(result.Warnings, warnings...)
	}

	// Validate subtitle transcodes
	for _, st := range subtitleTranscodes {
		_, issues, warnings := validateSubtitleFile(st)
		result.Issues = append(result.Issues, issues...)
		result.Warnings = append(result.Warnings, warnings...)
	}

	// Check audio/video sync drift
	if videoDuration > 0 && audioDuration > 0 {
		drift := audioDuration - videoDuration
		result.DriftMs = drift

		absDrift := drift
		if absDrift < 0 {
			absDrift = -absDrift
		}

		if absDrift > config.DriftThreshold {
			result.Issues = append(result.Issues,
				fmt.Sprintf("Audio/video drift exceeds threshold (%dms > %dms)", absDrift, config.DriftThreshold))
			result.Actions = append(result.Actions, "Re-transcode audio tracks")
		}
	}

	// Determine overall status
	if len(result.Issues) > 0 {
		result.Status = "FAIL"
	} else if len(result.Warnings) > 0 {
		result.Status = "WARN"
	}

	return result
}

func validateTranscodeDirectory(config Config, rec TranscodeRecord, trackType string) (int64, []string, []string) {
	var issues, warnings []string
	var totalDuration int64

	outputPath := rec.OutputPath

	// Check init.mp4 exists
	initPath := filepath.Join(outputPath, "init.mp4")
	if _, err := os.Stat(initPath); err != nil {
		issues = append(issues, fmt.Sprintf("Missing init.mp4 for %s/%s", trackType, rec.Quality))
		return 0, issues, warnings
	}

	// Check segment files exist
	entries, err := os.ReadDir(outputPath)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Cannot read directory for %s/%s: %v", trackType, rec.Quality, err))
		return 0, issues, warnings
	}

	var segmentFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".m4s") {
			segmentFiles = append(segmentFiles, entry.Name())
		}
	}

	if len(segmentFiles) == 0 {
		issues = append(issues, fmt.Sprintf("No segment files for %s/%s", trackType, rec.Quality))
		return 0, issues, warnings
	}

	// Sort segments numerically
	sort.Slice(segmentFiles, func(i, j int) bool {
		return extractSegmentNumber(segmentFiles[i]) < extractSegmentNumber(segmentFiles[j])
	})

	// Check segments.json
	segmentsJSONPath := filepath.Join(outputPath, "segments.json")
	segmentsJSON, err := loadSegmentsJSON(segmentsJSONPath)

	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Missing segments.json for %s/%s", trackType, rec.Quality))

		// Optionally regenerate
		if config.FixSegments && !config.DryRun {
			if regenerated := regenerateSegmentsJSON(config, outputPath, segmentsJSONPath); regenerated != nil {
				segmentsJSON = regenerated
				warnings[len(warnings)-1] = fmt.Sprintf("Regenerated segments.json for %s/%s", trackType, rec.Quality)
			}
		}
	} else {
		// Validate segments.json content
		corrupted := false
		for _, seg := range segmentsJSON.Segments {
			if seg.Duration < 0 || seg.Duration > 30000 {
				corrupted = true
				break
			}
		}

		if corrupted {
			warnings = append(warnings, fmt.Sprintf("Corrupted segments.json for %s/%s (invalid durations)", trackType, rec.Quality))

			if config.FixSegments && !config.DryRun {
				if regenerated := regenerateSegmentsJSON(config, outputPath, segmentsJSONPath); regenerated != nil {
					segmentsJSON = regenerated
					warnings[len(warnings)-1] = fmt.Sprintf("Regenerated segments.json for %s/%s", trackType, rec.Quality)
				}
			}
		}

		// Check segment count matches
		if len(segmentsJSON.Segments) != len(segmentFiles) {
			warnings = append(warnings, fmt.Sprintf("Segment count mismatch for %s/%s: %d in JSON vs %d files",
				trackType, rec.Quality, len(segmentsJSON.Segments), len(segmentFiles)))

			if config.FixSegments && !config.DryRun {
				if regenerated := regenerateSegmentsJSON(config, outputPath, segmentsJSONPath); regenerated != nil {
					segmentsJSON = regenerated
				}
			}
		}
	}

	// Calculate total duration
	if segmentsJSON != nil && !config.NoProbe {
		// Probe actual durations
		probed := probeSegmentDurations(outputPath, segmentFiles)
		if probed != nil {
			for _, seg := range probed.Segments {
				totalDuration += seg.Duration
			}
		}
	} else if segmentsJSON != nil {
		// Trust segments.json
		for _, seg := range segmentsJSON.Segments {
			totalDuration += seg.Duration
		}
	}

	return totalDuration, issues, warnings
}

func validateSubtitleFile(rec TranscodeRecord) (int64, []string, []string) {
	var issues, warnings []string

	checkPath := rec.OutputPath
	if !strings.HasSuffix(checkPath, ".vtt") {
		checkPath = checkPath + ".vtt"
	}

	info, err := os.Stat(checkPath)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Missing subtitle file: %s", filepath.Base(checkPath)))
		return 0, issues, warnings
	}

	if info.Size() == 0 {
		warnings = append(warnings, fmt.Sprintf("Empty subtitle file: %s", filepath.Base(checkPath)))
	}

	return 0, issues, warnings
}

func loadSegmentsJSON(path string) (*SegmentsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sf SegmentsFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}

	return &sf, nil
}

func regenerateSegmentsJSON(config Config, outputPath, segmentsJSONPath string) *SegmentsFile {
	if config.NoProbe {
		return nil
	}

	entries, err := os.ReadDir(outputPath)
	if err != nil {
		return nil
	}

	var segmentFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".m4s") {
			segmentFiles = append(segmentFiles, entry.Name())
		}
	}

	if len(segmentFiles) == 0 {
		return nil
	}

	probed := probeSegmentDurations(outputPath, segmentFiles)
	if probed == nil {
		return nil
	}

	// Write to file
	jsonData, err := json.MarshalIndent(probed, "", "  ")
	if err != nil {
		return nil
	}

	if err := os.WriteFile(segmentsJSONPath, jsonData, 0644); err != nil {
		return nil
	}

	return probed
}

func probeSegmentDurations(outputPath string, segmentFiles []string) *SegmentsFile {
	// Sort segment files numerically
	sort.Slice(segmentFiles, func(i, j int) bool {
		return extractSegmentNumber(segmentFiles[i]) < extractSegmentNumber(segmentFiles[j])
	})

	initSegmentPath := filepath.Join(outputPath, "init.mp4")

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	var cumulativeDurations []int64

	for _, segmentFile := range segmentFiles {
		segmentPath := filepath.Join(outputPath, segmentFile)
		duration, err := probeSegmentDuration(ctx, segmentPath, initSegmentPath)
		if err != nil {
			continue
		}
		cumulativeDurations = append(cumulativeDurations, duration)
	}

	if len(cumulativeDurations) == 0 {
		return nil
	}

	var segments []SegmentInfo
	for i := range cumulativeDurations {
		var segmentDuration int64
		if i == 0 {
			segmentDuration = cumulativeDurations[i]
		} else {
			segmentDuration = cumulativeDurations[i] - cumulativeDurations[i-1]
		}

		segments = append(segments, SegmentInfo{
			Number:   i,
			Duration: segmentDuration,
		})
	}

	return &SegmentsFile{Segments: segments}
}

func probeSegmentDuration(ctx context.Context, segmentPath, initSegmentPath string) (int64, error) {
	args := []string{
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		segmentPath,
	}

	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	output, err := cmd.Output()

	if err != nil || len(output) == 0 {
		// Try concatenating with init segment
		tmpFile, err := os.CreateTemp("", "probe-*.mp4")
		if err != nil {
			return 0, fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		initData, err := os.ReadFile(initSegmentPath)
		if err != nil {
			tmpFile.Close()
			return 0, fmt.Errorf("failed to read init segment: %w", err)
		}

		segmentData, err := os.ReadFile(segmentPath)
		if err != nil {
			tmpFile.Close()
			return 0, fmt.Errorf("failed to read segment: %w", err)
		}

		if _, err := tmpFile.Write(initData); err != nil {
			tmpFile.Close()
			return 0, fmt.Errorf("failed to write init data: %w", err)
		}

		if _, err := tmpFile.Write(segmentData); err != nil {
			tmpFile.Close()
			return 0, fmt.Errorf("failed to write segment data: %w", err)
		}

		tmpFile.Close()

		args[len(args)-1] = tmpPath
		cmd = exec.CommandContext(ctx, "ffprobe", args...)
		output, err = cmd.Output()
		if err != nil {
			return 0, fmt.Errorf("ffprobe failed: %w", err)
		}
	}

	durationStr := strings.TrimSpace(string(output))
	durationSec, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration '%s': %w", durationStr, err)
	}

	durationMs := int64(durationSec*1000 + 0.5)
	return durationMs, nil
}

func extractSegmentNumber(filename string) int {
	name := strings.TrimSuffix(filename, ".m4s")
	lastDash := strings.LastIndex(name, "-")
	if lastDash == -1 || lastDash >= len(name)-1 {
		return 0
	}
	numStr := name[lastDash+1:]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0
	}
	return num
}

func fixOrphanedRecords(db *sql.DB, config Config, records []TranscodeRecord) {
	if config.DryRun {
		if !config.JSONOutput {
			fmt.Printf("Would delete %d orphaned database records (dry run)\n", len(records))
		}
		return
	}

	deleted := 0
	for _, rec := range records {
		_, err := db.Exec("DELETE FROM transcodes WHERE id = ?", rec.ID)
		if err == nil {
			deleted++
		}
	}

	if !config.JSONOutput {
		fmt.Printf("Deleted %d orphaned database records\n", deleted)
	}
}

func fixOrphanedDirectories(config Config, dirs []string) {
	if config.DryRun {
		if !config.JSONOutput {
			fmt.Printf("Would delete %d orphaned directories (dry run)\n", len(dirs))
		}
		return
	}

	deleted := 0
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err == nil {
			deleted++
		}
	}

	if !config.JSONOutput {
		fmt.Printf("Deleted %d orphaned directories\n", deleted)
	}
}

func createTranscodeJobs(db *sql.DB, config Config, mediaIDs []string) {
	if config.DryRun {
		if !config.JSONOutput {
			fmt.Printf("Would create transcode jobs for %d media files (dry run)\n", len(mediaIDs))
		}
		return
	}

	created := 0
	for _, mediaID := range mediaIDs {
		// Get audio transcodes for this media
		rows, err := db.Query(`
			SELECT id, track_index, output_path FROM transcodes
			WHERE media_id = ? AND track_type = 'audio' AND status = 'completed'
		`, mediaID)
		if err != nil {
			continue
		}

		var audioTracks []struct {
			ID         string
			TrackIndex int
			OutputPath string
		}

		for rows.Next() {
			var t struct {
				ID         string
				TrackIndex int
				OutputPath string
			}
			if err := rows.Scan(&t.ID, &t.TrackIndex, &t.OutputPath); err == nil {
				audioTracks = append(audioTracks, t)
			}
		}
		rows.Close()

		// Delete existing audio transcodes and create new pending ones
		for _, track := range audioTracks {
			// Delete old transcode record
			db.Exec("DELETE FROM transcodes WHERE id = ?", track.ID)

			// Delete old files
			os.RemoveAll(track.OutputPath)

			// Create new pending transcode
			newID := fmt.Sprintf("%s-audio-%d", mediaID, track.TrackIndex)
			_, err := db.Exec(`
				INSERT INTO transcodes (id, media_id, quality, track_type, track_index, status, output_path)
				VALUES (?, ?, '', 'audio', ?, 'pending', ?)
			`, newID, mediaID, track.TrackIndex, track.OutputPath)
			if err == nil {
				created++
			}
		}

		// Create a job to process the transcode
		jobID := uuid.New().String()
		payload := fmt.Sprintf(`{"media_id":"%s"}`, mediaID)
		db.Exec(`
			INSERT INTO jobs (id, type, status, payload, priority, created_at, updated_at)
			VALUES (?, 'transcode', 'pending', ?, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, jobID, payload)
	}

	if !config.JSONOutput {
		fmt.Printf("Created %d audio transcode jobs\n", created)
	}
}

func printValidationResult(result ValidationResult, config Config) {
	statusIcon := map[string]string{
		"PASS": "[PASS]",
		"WARN": "[WARN]",
		"FAIL": "[FAIL]",
	}

	fmt.Printf("%s  %s\n", statusIcon[result.Status], result.Filename)

	if config.Verbose || result.Status != "PASS" {
		if result.VideoDuration > 0 || result.AudioDuration > 0 {
			fmt.Printf("        Video: %dms, Audio: %dms, Drift: %dms\n",
				result.VideoDuration, result.AudioDuration, result.DriftMs)
		}

		for _, issue := range result.Issues {
			fmt.Printf("        * %s\n", issue)
		}
		for _, warn := range result.Warnings {
			fmt.Printf("        ! %s\n", warn)
		}
		for _, action := range result.Actions {
			fmt.Printf("        -> %s\n", action)
		}
	}

	fmt.Println()
}

func printSummary(summary *SummaryResult, config Config) {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Summary:")
	fmt.Printf("  Total media files: %d\n", summary.TotalMedia)
	fmt.Printf("  Passed: %d\n", summary.Passed)
	fmt.Printf("  Warnings: %d\n", summary.Warnings)
	fmt.Printf("  Failures: %d\n", summary.Failures)
	fmt.Println()

	if summary.OrphanedRecords > 0 || summary.OrphanedDirs > 0 ||
		summary.DriftIssues > 0 || summary.MissingSegments > 0 || summary.CorruptedSegments > 0 {
		fmt.Println("Issues found:")
		if summary.OrphanedRecords > 0 {
			fmt.Printf("  - %d orphaned database record(s)\n", summary.OrphanedRecords)
		}
		if summary.OrphanedDirs > 0 {
			fmt.Printf("  - %d orphaned transcode directory(ies)\n", summary.OrphanedDirs)
		}
		if summary.DriftIssues > 0 {
			fmt.Printf("  - %d media file(s) with audio sync drift > %dms\n", summary.DriftIssues, config.DriftThreshold)
		}
		if summary.MissingSegments > 0 {
			fmt.Printf("  - %d transcode(s) missing segments.json\n", summary.MissingSegments)
		}
		if summary.CorruptedSegments > 0 {
			fmt.Printf("  - %d transcode(s) with corrupted segments.json\n", summary.CorruptedSegments)
		}
		fmt.Println()
	}

	if len(summary.MediaNeedingRetranscode) > 0 {
		fmt.Println("Media requiring audio re-transcode:")
		for _, mediaID := range summary.MediaNeedingRetranscode {
			// Find filename
			for _, r := range summary.Results {
				if r.MediaID == mediaID {
					fmt.Printf("  - %s (%s)\n", r.Filename, mediaID)
					break
				}
			}
		}
		fmt.Println()

		if !config.CreateTranscodeJobs {
			fmt.Println("Run with --create-transcode-jobs to queue audio re-transcoding")
		}
	}
}
