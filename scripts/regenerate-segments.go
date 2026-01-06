//go:build ignore

// Standalone script to regenerate all segments.json files with correct sorting
//
// This script fixes corrupted segments.json files that were generated with
// alphabetical sorting instead of numeric sorting. The bug caused segments
// like chunk-1000.m4s to be sorted between chunk-099.m4s and chunk-100.m4s,
// resulting in incorrect segment durations (negative values, impossibly large values).
//
// Usage:
//   go run scripts/regenerate-segments.go [--dry-run] <transcode-root-path>
//
// Example:
//   go run scripts/regenerate-segments.go /mnt/nasber02/video/vimesrv/library/media
//   go run scripts/regenerate-segments.go --dry-run /mnt/nasber02/video/vimesrv/library/media

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
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

func main() {
	dryRun := false
	args := os.Args[1:]

	if len(args) == 0 {
		fmt.Println("Usage: go run scripts/regenerate-segments.go [--dry-run] <transcode-root-path>")
		fmt.Println("Example: go run scripts/regenerate-segments.go /mnt/nasber02/video/vimesrv/library/media")
		os.Exit(1)
	}

	if args[0] == "--dry-run" {
		dryRun = true
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Println("Error: missing transcode root path")
		os.Exit(1)
	}

	rootPath := args[0]

	// Verify ffprobe is available
	if _, err := exec.LookPath("ffprobe"); err != nil {
		fmt.Println("Error: ffprobe not found in PATH")
		os.Exit(1)
	}

	fmt.Printf("Scanning for segments.json files in: %s\n", rootPath)
	if dryRun {
		fmt.Println("DRY RUN MODE - no files will be modified")
	}
	fmt.Println()

	var segmentFiles []string

	// Find all segments.json files
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip inaccessible directories
		}
		if info.Name() == "segments.json" && !info.IsDir() {
			segmentFiles = append(segmentFiles, path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d segments.json files\n\n", len(segmentFiles))

	var fixed, skipped, failed int

	for _, segmentFile := range segmentFiles {
		outputPath := filepath.Dir(segmentFile)
		relPath, _ := filepath.Rel(rootPath, outputPath)

		// Check if this file needs fixing (has negative or very large durations)
		needsFix, err := checkNeedsFix(segmentFile)
		if err != nil {
			fmt.Printf("[ERROR] %s: failed to read: %v\n", relPath, err)
			failed++
			continue
		}

		if !needsFix {
			fmt.Printf("[OK]    %s: durations look correct\n", relPath)
			skipped++
			continue
		}

		fmt.Printf("[FIX]   %s: regenerating...", relPath)

		if dryRun {
			fmt.Println(" (dry run)")
			fixed++
			continue
		}

		// Regenerate the segments.json
		segments, err := probeSegmentDurations(outputPath)
		if err != nil {
			fmt.Printf(" FAILED: %v\n", err)
			failed++
			continue
		}

		// Write the new segments.json
		if err := writeSegmentsJSON(segmentFile, segments); err != nil {
			fmt.Printf(" FAILED to write: %v\n", err)
			failed++
			continue
		}

		fmt.Printf(" done (%d segments)\n", len(segments))
		fixed++
	}

	fmt.Println()
	fmt.Printf("Summary: %d fixed, %d skipped (OK), %d failed\n", fixed, skipped, failed)
}

// checkNeedsFix checks if a segments.json file has corrupted durations
func checkNeedsFix(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	var sf SegmentsFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return false, err
	}

	for _, seg := range sf.Segments {
		// Negative duration or duration > 30 seconds indicates corruption
		if seg.Duration < 0 || seg.Duration > 30000 {
			return true, nil
		}
	}

	return false, nil
}

// probeSegmentDurations probes all segment files and returns their exact durations
func probeSegmentDurations(outputPath string) ([]SegmentInfo, error) {
	// Find all .m4s segment files
	entries, err := os.ReadDir(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output directory: %w", err)
	}

	var segmentFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".m4s") {
			segmentFiles = append(segmentFiles, entry.Name())
		}
	}

	if len(segmentFiles) == 0 {
		return nil, fmt.Errorf("no segment files found in %s", outputPath)
	}

	// Sort segment files numerically by segment number
	sort.Slice(segmentFiles, func(i, j int) bool {
		return extractSegmentNumber(segmentFiles[i]) < extractSegmentNumber(segmentFiles[j])
	})

	// Probe each segment to get cumulative durations
	var cumulativeDurations []int64
	initSegmentPath := filepath.Join(outputPath, "init.mp4")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	for _, segmentFile := range segmentFiles {
		segmentPath := filepath.Join(outputPath, segmentFile)

		cumulativeDuration, err := probeSegmentDuration(ctx, segmentPath, initSegmentPath)
		if err != nil {
			continue // Skip segments that fail to probe
		}

		cumulativeDurations = append(cumulativeDurations, cumulativeDuration)
	}

	if len(cumulativeDurations) == 0 {
		return nil, fmt.Errorf("failed to probe any segment durations")
	}

	// Convert cumulative durations to individual segment durations
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

	return segments, nil
}

// probeSegmentDuration probes a single segment file and returns its cumulative duration in milliseconds
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

		// Probe the concatenated file
		args[len(args)-1] = tmpPath
		cmd = exec.CommandContext(ctx, "ffprobe", args...)
		output, err = cmd.Output()
		if err != nil {
			return 0, fmt.Errorf("ffprobe failed: %w", err)
		}
	}

	// Parse duration
	durationStr := strings.TrimSpace(string(output))
	durationSec, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration '%s': %w", durationStr, err)
	}

	// Convert to milliseconds
	durationMs := int64(durationSec*1000 + 0.5)

	return durationMs, nil
}

// extractSegmentNumber extracts the numeric segment number from a filename like "chunk-000.m4s"
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

// writeSegmentsJSON writes segment timing data to segments.json
func writeSegmentsJSON(path string, segments []SegmentInfo) error {
	data := SegmentsFile{
		Segments: segments,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal segments: %w", err)
	}

	return os.WriteFile(path, jsonData, 0644)
}
