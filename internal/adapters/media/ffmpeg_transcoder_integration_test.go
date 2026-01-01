package media

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// TestFFmpegTranscoder_RealVideo_ExtractMetadata tests extracting metadata from real sample video
func TestFFmpegTranscoder_RealVideo_ExtractMetadata(t *testing.T) {
	// Check if ffprobe is installed
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	// Path to sample video
	sampleVideo := filepath.Join("..", "..", "..", "test", "fixtures", "sample_video.mp4")
	if _, err := os.Stat(sampleVideo); err != nil {
		t.Skipf("Sample video not found at %s, skipping test", sampleVideo)
	}

	ffprobe := NewFFProbeAdapter(30)

	// Extract metadata
	metadata, err := ffprobe.ExtractMetadata(sampleVideo)
	if err != nil {
		t.Fatalf("Failed to extract metadata: %v", err)
	}

	// Verify metadata
	if metadata.VideoCodec == "" {
		t.Error("Expected video codec to be non-empty")
	}
	if metadata.Width <= 0 || metadata.Height <= 0 {
		t.Errorf("Expected valid dimensions, got %dx%d", metadata.Width, metadata.Height)
	}
	if metadata.Duration <= 0 {
		t.Errorf("Expected positive duration, got %d", metadata.Duration)
	}
	if metadata.FileSize <= 0 {
		t.Errorf("Expected positive file size, got %d", metadata.FileSize)
	}

	t.Logf("Sample video metadata: %dx%d, %s, %d seconds, %d bytes",
		metadata.Width, metadata.Height, metadata.VideoCodec, metadata.Duration, metadata.FileSize)
}

// TestFFmpegTranscoder_RealVideo_GetAudioStreams tests extracting audio streams from real video
func TestFFmpegTranscoder_RealVideo_GetAudioStreams(t *testing.T) {
	// Check if ffprobe is installed
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	sampleVideo := filepath.Join("..", "..", "..", "test", "fixtures", "sample_video.mp4")
	if _, err := os.Stat(sampleVideo); err != nil {
		t.Skipf("Sample video not found at %s, skipping test", sampleVideo)
	}

	ffprobe := NewFFProbeAdapter(30)

	// Get audio streams
	audioStreams, err := ffprobe.GetAudioStreams(sampleVideo)
	if err != nil {
		t.Fatalf("Failed to get audio streams: %v", err)
	}

	t.Logf("Found %d audio stream(s)", len(audioStreams))

	for i, stream := range audioStreams {
		t.Logf("Audio stream %d: codec=%s, channels=%d, sample_rate=%d, language=%s",
			i, stream.Codec, stream.Channels, stream.SampleRate, stream.Language)

		if stream.StreamIndex < 0 {
			t.Errorf("Expected valid stream index, got %d", stream.StreamIndex)
		}
		if stream.Codec == "" {
			t.Error("Expected codec to be non-empty")
		}
	}
}

// TestFFmpegTranscoder_RealVideo_GetSubtitleStreams tests extracting subtitle streams
func TestFFmpegTranscoder_RealVideo_GetSubtitleStreams(t *testing.T) {
	// Check if ffprobe is installed
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	sampleVideo := filepath.Join("..", "..", "..", "test", "fixtures", "sample_video.mp4")
	if _, err := os.Stat(sampleVideo); err != nil {
		t.Skipf("Sample video not found at %s, skipping test", sampleVideo)
	}

	ffprobe := NewFFProbeAdapter(30)

	// Get subtitle streams
	subtitleStreams, err := ffprobe.GetSubtitleStreams(sampleVideo)
	if err != nil {
		t.Fatalf("Failed to get subtitle streams: %v", err)
	}

	t.Logf("Found %d subtitle stream(s)", len(subtitleStreams))

	for i, stream := range subtitleStreams {
		t.Logf("Subtitle stream %d: codec=%s, language=%s",
			i, stream.Codec, stream.Language)

		if stream.StreamIndex < 0 {
			t.Errorf("Expected valid stream index, got %d", stream.StreamIndex)
		}
		if stream.Codec == "" {
			t.Error("Expected codec to be non-empty")
		}
	}
}

// TestFFmpegTranscoder_RealVideo_TranscodeVideo tests transcoding real sample video
func TestFFmpegTranscoder_RealVideo_TranscodeVideo(t *testing.T) {
	// Check if ffmpeg is installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed on this system, skipping test")
	}

	sampleVideo := filepath.Join("..", "..", "..", "test", "fixtures", "sample_video.mp4")
	if _, err := os.Stat(sampleVideo); err != nil {
		t.Skipf("Sample video not found at %s, skipping test", sampleVideo)
	}

	// Create output directory
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "360p_video")

	transcoder := NewFFmpegTranscoder(120) // 2 minute timeout
	ctx := context.Background()

	opts := ports.TranscodeOptions{
		InputPath:   sampleVideo,
		OutputPath:  outputPath,
		Width:       640,
		Height:      360,
		VideoCodec:  "libx264",
		CRF:         25,
		MaxBitrate:  900,
		Preset:      "ultrafast", // Use fastest preset for testing
		SegmentTime: 4,
	}

	// Track progress
	var lastProgress ports.TranscodeProgress
	callback := func(progress ports.TranscodeProgress) {
		lastProgress = progress
		t.Logf("Progress: frame=%d, fps=%.2f, time=%s, speed=%s",
			progress.Frame, progress.FPS, progress.Time, progress.Speed)
	}

	err := transcoder.TranscodeVideo(ctx, opts, callback)
	if err != nil {
		t.Fatalf("Failed to transcode video: %v", err)
	}

	// Verify output files
	initSegment := filepath.Join(outputPath, "init.mp4")
	if _, err := os.Stat(initSegment); err != nil {
		t.Errorf("Expected init.mp4 to exist: %v", err)
	}

	// Count media segments
	entries, err := os.ReadDir(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}

	segmentCount := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".m4s" {
			segmentCount++
		}
	}

	if segmentCount == 0 {
		t.Error("Expected at least one .m4s segment file")
	}

	t.Logf("Created %d segment(s)", segmentCount)

	// Verify final progress shows 100%
	if lastProgress.Percentage != 100.0 {
		t.Logf("Warning: Final progress was %.1f%%, expected 100%%", lastProgress.Percentage)
	}
}

// TestFFmpegTranscoder_RealVideo_TranscodeAudio tests audio transcoding from real video
func TestFFmpegTranscoder_RealVideo_TranscodeAudio(t *testing.T) {
	// Check if ffmpeg and ffprobe are installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed on this system, skipping test")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	sampleVideo := filepath.Join("..", "..", "..", "test", "fixtures", "sample_video.mp4")
	if _, err := os.Stat(sampleVideo); err != nil {
		t.Skipf("Sample video not found at %s, skipping test", sampleVideo)
	}

	// First, get audio streams to find the stream index
	ffprobe := NewFFProbeAdapter(30)
	audioStreams, err := ffprobe.GetAudioStreams(sampleVideo)
	if err != nil {
		t.Fatalf("Failed to get audio streams: %v", err)
	}

	if len(audioStreams) == 0 {
		t.Skip("Sample video has no audio streams, skipping test")
	}

	// Create output directory
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "audio_0")

	transcoder := NewFFmpegTranscoder(120)
	ctx := context.Background()

	opts := ports.TranscodeOptions{
		InputPath:         sampleVideo,
		OutputPath:        outputPath,
		SourceStreamIndex: audioStreams[0].StreamIndex,
		AudioCodec:        "aac",
		AudioBitrate:      128,
		AudioChannels:     2,
		SegmentTime:       4,
	}

	err = transcoder.TranscodeAudio(ctx, opts, nil)
	if err != nil {
		t.Fatalf("Failed to transcode audio: %v", err)
	}

	// Verify output files
	initSegment := filepath.Join(outputPath, "init.mp4")
	if _, err := os.Stat(initSegment); err != nil {
		t.Errorf("Expected init.mp4 to exist: %v", err)
	}

	// Count media segments
	entries, err := os.ReadDir(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}

	segmentCount := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".m4s" {
			segmentCount++
		}
	}

	if segmentCount == 0 {
		t.Error("Expected at least one .m4s audio segment file")
	}

	t.Logf("Created %d audio segment(s)", segmentCount)
}

// TestFFmpegTranscoder_RealVideo_ProbeSegments tests segment probing on real video
func TestFFmpegTranscoder_RealVideo_ProbeSegments(t *testing.T) {
	// Check if ffmpeg and ffprobe are installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed on this system, skipping test")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	sampleVideo := filepath.Join("..", "..", "..", "test", "fixtures", "sample_video.mp4")
	if _, err := os.Stat(sampleVideo); err != nil {
		t.Skipf("Sample video not found at %s, skipping test", sampleVideo)
	}

	// Create output directory
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "360p_video")

	transcoder := NewFFmpegTranscoder(120)
	ctx := context.Background()

	// Transcode first
	opts := ports.TranscodeOptions{
		InputPath:   sampleVideo,
		OutputPath:  outputPath,
		Width:       640,
		Height:      360,
		VideoCodec:  "libx264",
		CRF:         25,
		MaxBitrate:  900,
		Preset:      "ultrafast",
		SegmentTime: 4,
	}

	if err := transcoder.TranscodeVideo(ctx, opts, nil); err != nil {
		t.Fatalf("Failed to transcode video: %v", err)
	}

	// Probe segment durations
	segments, err := transcoder.ProbeSegmentDurations(ctx, outputPath)
	if err != nil {
		t.Fatalf("Failed to probe segment durations: %v", err)
	}

	if len(segments) == 0 {
		t.Error("Expected at least one segment")
	}

	// Log segment information
	totalDuration := int64(0)
	for _, seg := range segments {
		t.Logf("Segment %d: %d ms", seg.Number, seg.Duration)
		totalDuration += seg.Duration

		if seg.Duration <= 0 {
			t.Errorf("Expected positive duration for segment %d, got %d", seg.Number, seg.Duration)
		}
	}

	t.Logf("Total duration: %d ms (%d segments)", totalDuration, len(segments))

	// Save segment timings
	err = SaveSegmentTimings(outputPath, segments)
	if err != nil {
		t.Fatalf("Failed to save segment timings: %v", err)
	}

	// Verify segments.json was created
	timingFile := filepath.Join(outputPath, "segments.json")
	if _, err := os.Stat(timingFile); err != nil {
		t.Errorf("Expected segments.json to exist: %v", err)
	}
}
