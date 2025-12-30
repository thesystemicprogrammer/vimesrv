package media

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// FFProbeAdapter implements video validation and metadata extraction using ffprobe
type FFProbeAdapter struct {
	timeout time.Duration
}

// NewFFProbeAdapter creates a new FFProbeAdapter with the specified timeout in seconds
func NewFFProbeAdapter(timeoutSeconds int) ports.FFProbeService {
	return &FFProbeAdapter{
		timeout: time.Duration(timeoutSeconds) * time.Second,
	}
}

// IsAvailable checks if ffprobe is available and executable
func (f *FFProbeAdapter) IsAvailable() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffprobe", "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffprobe is not available or not executable: %w", err)
	}

	return nil
}

// ValidateVideo checks if the file is a valid video file
func (f *FFProbeAdapter) ValidateVideo(filePath string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()

	// Use ffprobe to check if file has video streams
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_type",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return false, nil // File is not a valid video or ffprobe failed
	}

	// Check if output contains "video"
	return strings.TrimSpace(string(output)) == "video", nil
}

// ExtractMetadata extracts metadata from a video file
func (f *FFProbeAdapter) ExtractMetadata(filePath string) (*ports.VideoMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), f.timeout)
	defer cancel()

	// Use ffprobe to extract detailed metadata as JSON
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_format",
		"-show_streams",
		"-of", "json",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute ffprobe: %w", err)
	}

	// Parse JSON output
	var probeData ffprobeOutput
	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// Extract metadata
	metadata := &ports.VideoMetadata{
		AudioCodecs:       []string{},
		SubtitleLanguages: []string{},
	}

	// Process format information
	if probeData.Format != nil {
		metadata.Format = probeData.Format.FormatName

		if size, err := strconv.ParseInt(probeData.Format.Size, 10, 64); err == nil {
			metadata.FileSize = size
		}

		if duration, err := strconv.ParseFloat(probeData.Format.Duration, 64); err == nil {
			metadata.Duration = int(duration)
		}

		if bitrate, err := strconv.ParseInt(probeData.Format.BitRate, 10, 64); err == nil {
			metadata.Bitrate = int(bitrate)
		}
	}

	// Process streams
	audioCodecsMap := make(map[string]bool)
	subtitleLangsMap := make(map[string]bool)

	for _, stream := range probeData.Streams {
		switch stream.CodecType {
		case "video":
			if metadata.VideoCodec == "" { // Take first video stream
				metadata.VideoCodec = stream.CodecName
				metadata.Width = stream.Width
				metadata.Height = stream.Height
				if metadata.Width > 0 && metadata.Height > 0 {
					metadata.Resolution = fmt.Sprintf("%dx%d", metadata.Width, metadata.Height)
				}
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

	return metadata, nil
}

// ffprobeOutput represents the JSON structure returned by ffprobe
type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  *ffprobeFormat  `json:"format"`
}

type ffprobeStream struct {
	CodecType string           `json:"codec_type"`
	CodecName string           `json:"codec_name"`
	Width     int              `json:"width"`
	Height    int              `json:"height"`
	Tags      ffprobeStreamTag `json:"tags"`
}

type ffprobeStreamTag struct {
	Language string `json:"language"`
}

type ffprobeFormat struct {
	FormatName string `json:"format_name"`
	Duration   string `json:"duration"`
	Size       string `json:"size"`
	BitRate    string `json:"bit_rate"`
}
