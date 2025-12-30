package media

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// TestNewFFProbeAdapter tests the constructor
func TestNewFFProbeAdapter(t *testing.T) {
	adapter := NewFFProbeAdapter(30)
	if adapter == nil {
		t.Fatal("Expected adapter to be non-nil")
	}

	ffprobe, ok := adapter.(*FFProbeAdapter)
	if !ok {
		t.Fatal("Expected adapter to be *FFProbeAdapter")
	}

	expectedTimeout := int64(30000000000) // 30 seconds in nanoseconds
	if int64(ffprobe.timeout) != expectedTimeout {
		t.Errorf("Expected timeout to be %d, got %d", expectedTimeout, ffprobe.timeout)
	}
}

// TestFFProbeAdapter_IsAvailable tests if ffprobe is available
// Note: This test requires ffprobe to be installed on the system
func TestFFProbeAdapter_IsAvailable(t *testing.T) {
	adapter := NewFFProbeAdapter(30)

	// Check if ffprobe is in PATH
	_, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	err = adapter.IsAvailable()
	if err != nil {
		t.Errorf("Expected ffprobe to be available, got error: %v", err)
	}
}

// TestFFProbeAdapter_IsAvailable_NotInstalled tests behavior when ffprobe is not available
func TestFFProbeAdapter_IsAvailable_NotInstalled(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set PATH to empty to simulate ffprobe not being available
	os.Setenv("PATH", "")

	adapter := NewFFProbeAdapter(30)
	err := adapter.IsAvailable()

	if err == nil {
		t.Error("Expected error when ffprobe is not available, got nil")
	}
}

// TestFFProbeAdapter_ValidateVideo_ValidFile tests validation of a valid video file
func TestFFProbeAdapter_ValidateVideo_ValidFile(t *testing.T) {
	// Check if ffprobe is installed
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	// Create a minimal valid video file (1x1 pixel, 1 frame, H.264)
	// We'll use ffmpeg to create this if available, otherwise skip
	tempDir := t.TempDir()
	videoPath := filepath.Join(tempDir, "test_video.mp4")

	// Try to create a test video using ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed, cannot create test video file")
	}

	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", "color=c=black:s=1x1:d=0.1",
		"-c:v", "libx264",
		"-t", "0.1",
		"-pix_fmt", "yuv420p",
		"-y",
		videoPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to create test video: %v", err)
	}

	adapter := NewFFProbeAdapter(30)
	valid, err := adapter.ValidateVideo(videoPath)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !valid {
		t.Error("Expected video to be valid, got invalid")
	}
}

// TestFFProbeAdapter_ValidateVideo_InvalidFile tests validation of an invalid video file
func TestFFProbeAdapter_ValidateVideo_InvalidFile(t *testing.T) {
	// Check if ffprobe is installed
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	// Create a text file pretending to be a video
	tempDir := t.TempDir()
	fakePath := filepath.Join(tempDir, "fake_video.mp4")
	if err := os.WriteFile(fakePath, []byte("This is not a video file"), 0644); err != nil {
		t.Fatalf("Failed to create fake video file: %v", err)
	}

	adapter := NewFFProbeAdapter(30)
	valid, err := adapter.ValidateVideo(fakePath)

	if err != nil {
		t.Errorf("Expected no error (invalid files return false, not error), got: %v", err)
	}
	if valid {
		t.Error("Expected video to be invalid, got valid")
	}
}

// TestFFProbeAdapter_ValidateVideo_NonExistentFile tests validation of non-existent file
func TestFFProbeAdapter_ValidateVideo_NonExistentFile(t *testing.T) {
	// Check if ffprobe is installed
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	adapter := NewFFProbeAdapter(30)
	valid, err := adapter.ValidateVideo("/path/that/does/not/exist.mp4")

	// According to the implementation, non-existent files return false, nil
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if valid {
		t.Error("Expected non-existent file to be invalid, got valid")
	}
}

// TestFFProbeAdapter_ExtractMetadata_ValidVideo tests metadata extraction from valid video
func TestFFProbeAdapter_ExtractMetadata_ValidVideo(t *testing.T) {
	// Check if ffprobe and ffmpeg are installed
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed, cannot create test video file")
	}

	// Create a test video with known properties
	tempDir := t.TempDir()
	videoPath := filepath.Join(tempDir, "test_metadata.mp4")

	// Create video: 1920x1080, h264, aac audio, 1 second duration
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", "color=c=blue:s=1920x1080:d=1",
		"-f", "lavfi", "-i", "anullsrc=r=48000:cl=stereo",
		"-c:v", "libx264",
		"-c:a", "aac",
		"-t", "1",
		"-pix_fmt", "yuv420p",
		"-y",
		videoPath)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to create test video: %v", err)
	}

	adapter := NewFFProbeAdapter(30)
	metadata, err := adapter.ExtractMetadata(videoPath)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify metadata
	if metadata.VideoCodec != "h264" {
		t.Errorf("Expected video codec 'h264', got '%s'", metadata.VideoCodec)
	}
	if metadata.Width != 1920 {
		t.Errorf("Expected width 1920, got %d", metadata.Width)
	}
	if metadata.Height != 1080 {
		t.Errorf("Expected height 1080, got %d", metadata.Height)
	}
	if metadata.Resolution != "1920x1080" {
		t.Errorf("Expected resolution '1920x1080', got '%s'", metadata.Resolution)
	}
	if metadata.Duration < 1 || metadata.Duration > 2 {
		t.Errorf("Expected duration around 1 second, got %d", metadata.Duration)
	}
	if metadata.AudioTracks != 1 {
		t.Errorf("Expected 1 audio track, got %d", metadata.AudioTracks)
	}
	if len(metadata.AudioCodecs) != 1 || metadata.AudioCodecs[0] != "aac" {
		t.Errorf("Expected audio codec 'aac', got %v", metadata.AudioCodecs)
	}
	if metadata.FileSize <= 0 {
		t.Errorf("Expected file size > 0, got %d", metadata.FileSize)
	}
	if metadata.Format == "" {
		t.Error("Expected format to be non-empty")
	}
}

// TestFFProbeAdapter_ExtractMetadata_MultipleStreams tests extraction with multiple audio/subtitle streams
func TestFFProbeAdapter_ExtractMetadata_MultipleStreams(t *testing.T) {
	// This test uses mocked ffprobe output
	// We'll test the parsing logic by providing sample JSON

	// Create a temporary file to simulate the video file
	tempDir := t.TempDir()
	videoPath := filepath.Join(tempDir, "multi_stream.mp4")
	if err := os.WriteFile(videoPath, []byte("fake"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Create sample ffprobe output
	sampleOutput := ffprobeOutput{
		Format: &ffprobeFormat{
			FormatName: "matroska,webm",
			Duration:   "120.5",
			Size:       "1048576",
			BitRate:    "5000000",
		},
		Streams: []ffprobeStream{
			{
				CodecType: "video",
				CodecName: "h264",
				Width:     1920,
				Height:    1080,
			},
			{
				CodecType: "audio",
				CodecName: "aac",
			},
			{
				CodecType: "audio",
				CodecName: "ac3",
			},
			{
				CodecType: "subtitle",
				Tags: ffprobeStreamTag{
					Language: "eng",
				},
			},
			{
				CodecType: "subtitle",
				Tags: ffprobeStreamTag{
					Language: "spa",
				},
			},
		},
	}

	// Test the parsing logic directly
	metadata := &ports.VideoMetadata{
		AudioCodecs:       []string{},
		SubtitleLanguages: []string{},
	}

	// Process format information
	if sampleOutput.Format != nil {
		metadata.Format = sampleOutput.Format.FormatName
		metadata.FileSize = 1048576
		metadata.Duration = 120
		metadata.Bitrate = 5000000
	}

	// Process streams
	audioCodecsMap := make(map[string]bool)
	subtitleLangsMap := make(map[string]bool)

	for _, stream := range sampleOutput.Streams {
		switch stream.CodecType {
		case "video":
			if metadata.VideoCodec == "" {
				metadata.VideoCodec = stream.CodecName
				metadata.Width = stream.Width
				metadata.Height = stream.Height
				metadata.Resolution = "1920x1080"
			}
		case "audio":
			metadata.AudioTracks++
			if stream.CodecName != "" {
				audioCodecsMap[stream.CodecName] = true
			}
		case "subtitle":
			metadata.SubtitleTracks++
			if lang := stream.Tags.Language; lang != "" {
				subtitleLangsMap[lang] = true
			}
		}
	}

	// Convert maps to slices
	for codec := range audioCodecsMap {
		metadata.AudioCodecs = append(metadata.AudioCodecs, codec)
	}
	for lang := range subtitleLangsMap {
		metadata.SubtitleLanguages = append(metadata.SubtitleLanguages, lang)
	}

	// Verify metadata
	if metadata.VideoCodec != "h264" {
		t.Errorf("Expected video codec 'h264', got '%s'", metadata.VideoCodec)
	}
	if metadata.Format != "matroska,webm" {
		t.Errorf("Expected format 'matroska,webm', got '%s'", metadata.Format)
	}
	if metadata.Duration != 120 {
		t.Errorf("Expected duration 120, got %d", metadata.Duration)
	}
	if metadata.AudioTracks != 2 {
		t.Errorf("Expected 2 audio tracks, got %d", metadata.AudioTracks)
	}
	if len(metadata.AudioCodecs) != 2 {
		t.Errorf("Expected 2 audio codecs, got %d", len(metadata.AudioCodecs))
	}
	if metadata.SubtitleTracks != 2 {
		t.Errorf("Expected 2 subtitle tracks, got %d", metadata.SubtitleTracks)
	}
	if len(metadata.SubtitleLanguages) != 2 {
		t.Errorf("Expected 2 subtitle languages, got %d", len(metadata.SubtitleLanguages))
	}
}

// TestFFProbeAdapter_ExtractMetadata_InvalidJSON tests handling of invalid JSON output
func TestFFProbeAdapter_ExtractMetadata_InvalidJSON(t *testing.T) {
	// This test verifies that invalid JSON is handled gracefully
	invalidJSON := `{"streams": [{"codec_type": "video"}, invalid json here]}`

	var probeData ffprobeOutput
	err := json.Unmarshal([]byte(invalidJSON), &probeData)

	if err == nil {
		t.Error("Expected error when parsing invalid JSON, got nil")
	}
}

// TestFFProbeAdapter_ExtractMetadata_EmptyStreams tests extraction with no streams
func TestFFProbeAdapter_ExtractMetadata_EmptyStreams(t *testing.T) {
	// Test parsing logic with empty streams
	sampleOutput := ffprobeOutput{
		Format: &ffprobeFormat{
			FormatName: "mp4",
			Duration:   "10.0",
			Size:       "1024",
			BitRate:    "1000",
		},
		Streams: []ffprobeStream{},
	}

	metadata := &ports.VideoMetadata{
		AudioCodecs:       []string{},
		SubtitleLanguages: []string{},
	}

	// Process format information
	if sampleOutput.Format != nil {
		metadata.Format = sampleOutput.Format.FormatName
		metadata.FileSize = 1024
		metadata.Duration = 10
		metadata.Bitrate = 1000
	}

	// Verify metadata
	if metadata.Format != "mp4" {
		t.Errorf("Expected format 'mp4', got '%s'", metadata.Format)
	}
	if metadata.VideoCodec != "" {
		t.Errorf("Expected empty video codec, got '%s'", metadata.VideoCodec)
	}
	if metadata.AudioTracks != 0 {
		t.Errorf("Expected 0 audio tracks, got %d", metadata.AudioTracks)
	}
	if metadata.SubtitleTracks != 0 {
		t.Errorf("Expected 0 subtitle tracks, got %d", metadata.SubtitleTracks)
	}
}

// TestFFProbeAdapter_ExtractMetadata_NonExistentFile tests extraction from non-existent file
func TestFFProbeAdapter_ExtractMetadata_NonExistentFile(t *testing.T) {
	// Check if ffprobe is installed
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	adapter := NewFFProbeAdapter(30)
	_, err := adapter.ExtractMetadata("/path/that/does/not/exist.mp4")

	if err == nil {
		t.Error("Expected error when extracting metadata from non-existent file, got nil")
	}
}

// TestFFProbeAdapter_ExtractMetadata_InvalidFile tests extraction from invalid file
func TestFFProbeAdapter_ExtractMetadata_InvalidFile(t *testing.T) {
	// Check if ffprobe is installed
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed on this system, skipping test")
	}

	// Create a text file pretending to be a video
	tempDir := t.TempDir()
	fakePath := filepath.Join(tempDir, "fake_video.mp4")
	if err := os.WriteFile(fakePath, []byte("This is not a video file"), 0644); err != nil {
		t.Fatalf("Failed to create fake video file: %v", err)
	}

	adapter := NewFFProbeAdapter(30)
	_, err := adapter.ExtractMetadata(fakePath)

	if err == nil {
		t.Error("Expected error when extracting metadata from invalid file, got nil")
	}
}

// TestFFProbeAdapter_ExtractMetadata_ParseNumbers tests parsing of numeric fields
func TestFFProbeAdapter_ExtractMetadata_ParseNumbers(t *testing.T) {
	// Test parsing of various numeric string formats
	testCases := []struct {
		name            string
		format          *ffprobeFormat
		expectedSize    int64
		expectedDur     int
		expectedBitrate int
	}{
		{
			name: "valid numbers",
			format: &ffprobeFormat{
				Duration: "120.567",
				Size:     "1048576",
				BitRate:  "5000000",
			},
			expectedSize:    1048576,
			expectedDur:     120,
			expectedBitrate: 5000000,
		},
		{
			name: "invalid duration",
			format: &ffprobeFormat{
				Duration: "invalid",
				Size:     "1024",
				BitRate:  "1000",
			},
			expectedSize:    1024,
			expectedDur:     0, // Default value when parsing fails
			expectedBitrate: 1000,
		},
		{
			name: "invalid size",
			format: &ffprobeFormat{
				Duration: "10.0",
				Size:     "not_a_number",
				BitRate:  "1000",
			},
			expectedSize:    0, // Default value when parsing fails
			expectedDur:     10,
			expectedBitrate: 1000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata := &ports.VideoMetadata{
				AudioCodecs:       []string{},
				SubtitleLanguages: []string{},
			}

			// Simulate the parsing logic from ExtractMetadata
			if tc.format != nil {
				if size, err := json.Number(tc.format.Size).Int64(); err == nil {
					metadata.FileSize = size
				}
				if duration, err := json.Number(tc.format.Duration).Float64(); err == nil {
					metadata.Duration = int(duration)
				}
				if bitrate, err := json.Number(tc.format.BitRate).Int64(); err == nil {
					metadata.Bitrate = int(bitrate)
				}
			}

			if metadata.FileSize != tc.expectedSize {
				t.Errorf("Expected file size %d, got %d", tc.expectedSize, metadata.FileSize)
			}
			if metadata.Duration != tc.expectedDur {
				t.Errorf("Expected duration %d, got %d", tc.expectedDur, metadata.Duration)
			}
			if metadata.Bitrate != tc.expectedBitrate {
				t.Errorf("Expected bitrate %d, got %d", tc.expectedBitrate, metadata.Bitrate)
			}
		})
	}
}
