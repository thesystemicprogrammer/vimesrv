package media

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// saveSegmentTimingsForTest is a test helper that saves segment timing data to JSON
func saveSegmentTimingsForTest(outputPath string, segments []ports.SegmentInfo) error {
	data := struct {
		Segments []ports.SegmentInfo `json:"segments"`
	}{
		Segments: segments,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	timingFilePath := filepath.Join(outputPath, "segments.json")
	return os.WriteFile(timingFilePath, jsonData, 0644)
}

// TestNewFFmpegTranscoder tests the constructor
func TestNewFFmpegTranscoder(t *testing.T) {
	transcoder := NewFFmpegTranscoder(30)
	if transcoder == nil {
		t.Fatal("Expected transcoder to be non-nil")
	}

	ffmpeg, ok := transcoder.(*FFmpegTranscoder)
	if !ok {
		t.Fatal("Expected transcoder to be *FFmpegTranscoder")
	}

	expectedTimeout := int64(30000000000) // 30 seconds in nanoseconds
	if int64(ffmpeg.timeout) != expectedTimeout {
		t.Errorf("Expected timeout to be %d, got %d", expectedTimeout, ffmpeg.timeout)
	}
}

// TestNewFFmpegTranscoder_DefaultTimeout tests default timeout
func TestNewFFmpegTranscoder_DefaultTimeout(t *testing.T) {
	transcoder := NewFFmpegTranscoder(0)
	ffmpeg := transcoder.(*FFmpegTranscoder)

	expectedTimeout := int64(2 * time.Hour)
	if int64(ffmpeg.timeout) != expectedTimeout {
		t.Errorf("Expected default timeout to be %d, got %d", expectedTimeout, ffmpeg.timeout)
	}
}

// TestFFmpegTranscoder_IsAvailable tests if FFmpeg is available
func TestFFmpegTranscoder_IsAvailable(t *testing.T) {
	// Check if ffmpeg is in PATH
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed on this system, skipping test")
	}

	transcoder := NewFFmpegTranscoder(30)
	err := transcoder.IsAvailable()
	if err != nil {
		t.Errorf("Expected ffmpeg to be available, got error: %v", err)
	}
}

// TestFFmpegTranscoder_IsAvailable_NotInstalled tests behavior when FFmpeg is not available
func TestFFmpegTranscoder_IsAvailable_NotInstalled(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set PATH to empty to simulate ffmpeg not being available
	os.Setenv("PATH", "")

	transcoder := NewFFmpegTranscoder(30)
	err := transcoder.IsAvailable()

	if err == nil {
		t.Error("Expected error when ffmpeg is not available, got nil")
	}
}

// TestFFmpegTranscoder_TranscodeVideo_InvalidOptions tests validation errors
func TestFFmpegTranscoder_TranscodeVideo_InvalidOptions(t *testing.T) {
	transcoder := NewFFmpegTranscoder(30)
	ctx := context.Background()

	testCases := []struct {
		name string
		opts ports.TranscodeOptions
	}{
		{
			name: "missing input path",
			opts: ports.TranscodeOptions{
				OutputPath: "/tmp/output",
				Width:      1280,
				Height:     720,
				CRF:        23,
			},
		},
		{
			name: "missing output path",
			opts: ports.TranscodeOptions{
				InputPath: "/tmp/input.mp4",
				Width:     1280,
				Height:    720,
				CRF:       23,
			},
		},
		{
			name: "invalid dimensions",
			opts: ports.TranscodeOptions{
				InputPath:  "/tmp/input.mp4",
				OutputPath: "/tmp/output",
				Width:      0,
				Height:     720,
				CRF:        23,
			},
		},
		{
			name: "missing CRF and bitrate",
			opts: ports.TranscodeOptions{
				InputPath:  "/tmp/input.mp4",
				OutputPath: "/tmp/output",
				Width:      1280,
				Height:     720,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := transcoder.TranscodeVideo(ctx, tc.opts, nil)
			if err == nil {
				t.Error("Expected validation error, got nil")
			}
		})
	}
}

// TestFFmpegTranscoder_TranscodeAudio_InvalidOptions tests audio validation errors
func TestFFmpegTranscoder_TranscodeAudio_InvalidOptions(t *testing.T) {
	transcoder := NewFFmpegTranscoder(30)
	ctx := context.Background()

	testCases := []struct {
		name string
		opts ports.TranscodeOptions
	}{
		{
			name: "missing input path",
			opts: ports.TranscodeOptions{
				OutputPath:        "/tmp/output",
				SourceStreamIndex: 1,
				AudioBitrate:      128,
			},
		},
		{
			name: "missing stream index",
			opts: ports.TranscodeOptions{
				InputPath:         "/tmp/input.mp4",
				OutputPath:        "/tmp/output",
				SourceStreamIndex: -1,
				AudioBitrate:      128,
			},
		},
		{
			name: "missing audio bitrate",
			opts: ports.TranscodeOptions{
				InputPath:         "/tmp/input.mp4",
				OutputPath:        "/tmp/output",
				SourceStreamIndex: 1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := transcoder.TranscodeAudio(ctx, tc.opts, nil)
			if err == nil {
				t.Error("Expected validation error, got nil")
			}
		})
	}
}

// TestFFmpegTranscoder_ExtractSubtitle_InvalidOptions tests subtitle validation errors
func TestFFmpegTranscoder_ExtractSubtitle_InvalidOptions(t *testing.T) {
	transcoder := NewFFmpegTranscoder(30)
	ctx := context.Background()

	testCases := []struct {
		name string
		opts ports.TranscodeOptions
	}{
		{
			name: "missing input path",
			opts: ports.TranscodeOptions{
				OutputPath:        "/tmp/output.vtt",
				SourceStreamIndex: 0,
			},
		},
		{
			name: "missing stream index",
			opts: ports.TranscodeOptions{
				InputPath:         "/tmp/input.mp4",
				OutputPath:        "/tmp/output.vtt",
				SourceStreamIndex: -1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := transcoder.ExtractSubtitle(ctx, tc.opts)
			if err == nil {
				t.Error("Expected validation error, got nil")
			}
		})
	}
}

// TestFFmpegTranscoder_TranscodeVideo_Success tests successful video transcoding
func TestFFmpegTranscoder_TranscodeVideo_Success(t *testing.T) {
	// Check if ffmpeg is installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed on this system, skipping test")
	}

	// Create a test video file
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "input.mp4")
	outputPath := filepath.Join(tempDir, "output")

	// Create a simple test video (1 second, 640x360)
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "color=c=blue:s=640x360:d=1",
		"-c:v", "libx264",
		"-t", "1",
		"-pix_fmt", "yuv420p",
		"-y",
		inputPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to create test video: %v", err)
	}

	transcoder := NewFFmpegTranscoder(60)
	ctx := context.Background()

	opts := ports.TranscodeOptions{
		InputPath:   inputPath,
		OutputPath:  outputPath,
		Width:       640,
		Height:      360,
		VideoCodec:  "libx264",
		CRF:         23,
		MaxBitrate:  900,
		Preset:      "ultrafast",
		SegmentTime: 4,
	}

	// Track progress
	progressCalled := false
	callback := func(progress ports.TranscodeProgress) {
		progressCalled = true
	}

	err := transcoder.TranscodeVideo(ctx, opts, callback)
	if err != nil {
		t.Fatalf("Expected successful transcoding, got error: %v", err)
	}

	// Verify output files were created
	initSegment := filepath.Join(outputPath, "init.mp4")
	if _, err := os.Stat(initSegment); err != nil {
		t.Errorf("Expected init.mp4 to exist, but got error: %v", err)
	}

	// Check for at least one media segment
	entries, err := os.ReadDir(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}

	hasSegments := false
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".m4s" {
			hasSegments = true
			break
		}
	}

	if !hasSegments {
		t.Error("Expected at least one .m4s segment file")
	}

	if !progressCalled {
		t.Error("Expected progress callback to be called")
	}
}

// TestFFmpegTranscoder_TranscodeAudio_Success tests successful audio transcoding
func TestFFmpegTranscoder_TranscodeAudio_Success(t *testing.T) {
	// Check if ffmpeg is installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed on this system, skipping test")
	}

	// Create a test video file with audio
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "input.mp4")
	outputPath := filepath.Join(tempDir, "audio_output")

	// Create a test video with audio (1 second)
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "color=c=blue:s=640x360:d=1",
		"-f", "lavfi", "-i", "anullsrc=r=48000:cl=stereo",
		"-c:v", "libx264",
		"-c:a", "aac",
		"-t", "1",
		"-pix_fmt", "yuv420p",
		"-y",
		inputPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to create test video with audio: %v", err)
	}

	transcoder := NewFFmpegTranscoder(60)
	ctx := context.Background()

	opts := ports.TranscodeOptions{
		InputPath:         inputPath,
		OutputPath:        outputPath,
		SourceStreamIndex: 1, // Audio stream index
		AudioCodec:        "aac",
		AudioBitrate:      128,
		AudioChannels:     2,
		SegmentTime:       4,
	}

	err := transcoder.TranscodeAudio(ctx, opts, nil)
	if err != nil {
		t.Fatalf("Expected successful audio transcoding, got error: %v", err)
	}

	// Verify output files were created
	initSegment := filepath.Join(outputPath, "init.mp4")
	if _, err := os.Stat(initSegment); err != nil {
		t.Errorf("Expected init.mp4 to exist, but got error: %v", err)
	}

	// Check for at least one media segment
	entries, err := os.ReadDir(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}

	hasSegments := false
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".m4s" {
			hasSegments = true
			break
		}
	}

	if !hasSegments {
		t.Error("Expected at least one .m4s audio segment file")
	}
}

// TestFFmpegTranscoder_ProbeSegmentDurations tests segment duration probing
func TestFFmpegTranscoder_ProbeSegmentDurations(t *testing.T) {
	// Check if ffmpeg and ffprobe are installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed on this system, skipping test")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	// Create a test video and transcode it first
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "input.mp4")
	outputPath := filepath.Join(tempDir, "output")

	// Create a test video (2 seconds to get multiple segments)
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "color=c=blue:s=640x360:d=2",
		"-c:v", "libx264",
		"-t", "2",
		"-pix_fmt", "yuv420p",
		"-y",
		inputPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to create test video: %v", err)
	}

	// Transcode with 1-second segments
	transcoder := NewFFmpegTranscoder(60)
	ctx := context.Background()

	opts := ports.TranscodeOptions{
		InputPath:   inputPath,
		OutputPath:  outputPath,
		Width:       640,
		Height:      360,
		VideoCodec:  "libx264",
		CRF:         23,
		MaxBitrate:  900,
		Preset:      "ultrafast",
		SegmentTime: 1,
	}

	if err := transcoder.TranscodeVideo(ctx, opts, nil); err != nil {
		t.Fatalf("Failed to transcode video: %v", err)
	}

	// Probe segment durations
	segments, err := transcoder.ProbeSegmentDurations(ctx, outputPath)
	if err != nil {
		t.Fatalf("Expected successful segment probing, got error: %v", err)
	}

	if len(segments) == 0 {
		t.Error("Expected at least one segment")
	}

	// Verify segment info structure
	for i, seg := range segments {
		if seg.Number != i {
			t.Errorf("Expected segment number %d, got %d", i, seg.Number)
		}
		if seg.Duration <= 0 {
			t.Errorf("Expected positive duration for segment %d, got %d", i, seg.Duration)
		}
	}
}

// TestFFmpegTranscoder_ProbeSegmentDurations_NoSegments tests error when no segments exist
func TestFFmpegTranscoder_ProbeSegmentDurations_NoSegments(t *testing.T) {
	tempDir := t.TempDir()

	transcoder := NewFFmpegTranscoder(30)
	ctx := context.Background()

	_, err := transcoder.ProbeSegmentDurations(ctx, tempDir)
	if err == nil {
		t.Error("Expected error when no segments exist, got nil")
	}
}

// TestSaveSegmentTimings tests saving segment timing data
func TestSaveSegmentTimings(t *testing.T) {
	tempDir := t.TempDir()

	segments := []ports.SegmentInfo{
		{Number: 0, Duration: 4000},
		{Number: 1, Duration: 4000},
		{Number: 2, Duration: 3500},
	}

	err := saveSegmentTimingsForTest(tempDir, segments)
	if err != nil {
		t.Fatalf("Expected successful save, got error: %v", err)
	}

	// Verify file was created
	timingFile := filepath.Join(tempDir, "segments.json")
	if _, err := os.Stat(timingFile); err != nil {
		t.Errorf("Expected segments.json to exist, but got error: %v", err)
	}

	// Verify file content
	data, err := os.ReadFile(timingFile)
	if err != nil {
		t.Fatalf("Failed to read timing file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty timing file")
	}
}

// TestFFmpegTranscoder_ContextCancellation tests that context cancellation stops transcoding
func TestFFmpegTranscoder_ContextCancellation(t *testing.T) {
	// Check if ffmpeg is installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed on this system, skipping test")
	}

	// Create a test video file
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "input.mp4")
	outputPath := filepath.Join(tempDir, "output")

	// Create a longer video (10 seconds) to ensure we have time to cancel
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "color=c=blue:s=1920x1080:d=10",
		"-c:v", "libx264",
		"-t", "10",
		"-pix_fmt", "yuv420p",
		"-y",
		inputPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to create test video: %v", err)
	}

	transcoder := NewFFmpegTranscoder(60)

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	opts := ports.TranscodeOptions{
		InputPath:   inputPath,
		OutputPath:  outputPath,
		Width:       1920,
		Height:      1080,
		VideoCodec:  "libx264",
		CRF:         23,
		MaxBitrate:  5000,
		Preset:      "slow", // Use slow preset to make transcoding take longer
		SegmentTime: 4,
	}

	// Start transcoding in a goroutine
	errChan := make(chan error, 1)
	go func() {
		err := transcoder.TranscodeVideo(ctx, opts, nil)
		errChan <- err
	}()

	// Cancel after a short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for transcoding to finish
	err := <-errChan

	// Should get an error due to cancellation
	if err == nil {
		t.Error("Expected error due to context cancellation, got nil")
	}
}
