package media

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

const (
	// defaultFPS is the baseline framerate used for GOP size calculations
	defaultFPS = 24

	// bufSizeMultiplier is the multiplier applied to max bitrate for buffer size
	bufSizeMultiplier = 2
)

// FFmpegTranscoder implements video transcoding using FFmpeg
type FFmpegTranscoder struct {
	ffmpegPath  string
	ffprobePath string
	timeout     time.Duration
}

// NewFFmpegTranscoder creates a new FFmpegTranscoder with the specified timeout in seconds
// If timeoutSeconds is 0, defaults to 2 hours
func NewFFmpegTranscoder(timeoutSeconds int) ports.Transcoder {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 2 * time.Hour // Default timeout for transcoding
	}

	return &FFmpegTranscoder{
		ffmpegPath:  "ffmpeg",  // Use PATH
		ffprobePath: "ffprobe", // Use PATH
		timeout:     timeout,
	}
}

// IsAvailable checks if FFmpeg is available and executable
func (t *FFmpegTranscoder) IsAvailable() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.ffmpegPath, "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg is not available or not executable: %w", err)
	}

	return nil
}

// TranscodeVideo transcodes a video stream to the specified quality with CMAF segmentation
func (t *FFmpegTranscoder) TranscodeVideo(ctx context.Context, opts ports.TranscodeOptions, callback ports.ProgressCallback) error {
	// Validate options
	if err := t.validateVideoOptions(opts); err != nil {
		return fmt.Errorf("invalid transcode options: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(opts.OutputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build FFmpeg arguments for video-only transcoding
	args := t.buildVideoArgs(opts)

	// Execute transcoding
	return t.executeTranscode(ctx, args, opts, callback)
}

// TranscodeAudio transcodes an audio stream to AAC with CMAF segmentation
func (t *FFmpegTranscoder) TranscodeAudio(ctx context.Context, opts ports.TranscodeOptions, callback ports.ProgressCallback) error {
	// Validate options
	if err := t.validateAudioOptions(opts); err != nil {
		return fmt.Errorf("invalid transcode options: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(opts.OutputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build FFmpeg arguments for audio-only transcoding
	args := t.buildAudioArgs(opts)

	// Execute transcoding
	return t.executeTranscode(ctx, args, opts, callback)
}

// ExtractSubtitle extracts a subtitle stream and converts it to WebVTT format
func (t *FFmpegTranscoder) ExtractSubtitle(ctx context.Context, opts ports.TranscodeOptions) error {
	// Validate options
	if err := t.validateSubtitleOptions(opts); err != nil {
		return fmt.Errorf("invalid transcode options: %w", err)
	}

	// Create output directory
	outputDir := filepath.Dir(opts.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build FFmpeg arguments for subtitle extraction
	args := t.buildSubtitleArgs(opts)

	// Execute subtitle extraction (no progress callback for subtitles)
	return t.executeTranscode(ctx, args, opts, nil)
}

// ProbeSegmentDurations probes all segment files and returns their exact durations
func (t *FFmpegTranscoder) ProbeSegmentDurations(ctx context.Context, outputPath string) ([]ports.SegmentInfo, error) {
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

	// Sort segment files by name
	sort.Strings(segmentFiles)

	// Probe each segment to get cumulative durations
	var cumulativeDurations []int64
	initSegmentPath := filepath.Join(outputPath, "init.mp4")

	for _, segmentFile := range segmentFiles {
		segmentPath := filepath.Join(outputPath, segmentFile)

		cumulativeDuration, err := t.probeSegmentDuration(ctx, segmentPath, initSegmentPath)
		if err != nil {
			continue // Skip segments that fail to probe
		}

		cumulativeDurations = append(cumulativeDurations, cumulativeDuration)
	}

	if len(cumulativeDurations) == 0 {
		return nil, fmt.Errorf("failed to probe any segment durations")
	}

	// Convert cumulative durations to individual segment durations
	var segments []ports.SegmentInfo
	for i := range cumulativeDurations {
		var segmentDuration int64
		if i == 0 {
			segmentDuration = cumulativeDurations[i]
		} else {
			segmentDuration = cumulativeDurations[i] - cumulativeDurations[i-1]
		}

		segments = append(segments, ports.SegmentInfo{
			Number:   i,
			Duration: segmentDuration,
		})
	}

	return segments, nil
}

// validateCommonOptions validates options common to all transcode types
func (t *FFmpegTranscoder) validateCommonOptions(opts ports.TranscodeOptions) error {
	if opts.InputPath == "" {
		return fmt.Errorf("input path is required")
	}
	if opts.OutputPath == "" {
		return fmt.Errorf("output path is required")
	}
	if _, err := os.Stat(opts.InputPath); err != nil {
		return fmt.Errorf("input file does not exist: %w", err)
	}
	return nil
}

// validateVideoOptions validates video transcoding options
func (t *FFmpegTranscoder) validateVideoOptions(opts ports.TranscodeOptions) error {
	if err := t.validateCommonOptions(opts); err != nil {
		return err
	}
	if opts.Width <= 0 || opts.Height <= 0 {
		return fmt.Errorf("invalid dimensions: %dx%d", opts.Width, opts.Height)
	}
	if opts.CRF <= 0 && opts.VideoBitrate <= 0 {
		return fmt.Errorf("either CRF or video bitrate must be specified")
	}
	return nil
}

// validateAudioOptions validates audio transcoding options
func (t *FFmpegTranscoder) validateAudioOptions(opts ports.TranscodeOptions) error {
	if err := t.validateCommonOptions(opts); err != nil {
		return err
	}
	if opts.SourceStreamIndex < 0 {
		return fmt.Errorf("source stream index required for audio track")
	}
	if opts.AudioBitrate <= 0 {
		return fmt.Errorf("invalid audio bitrate: %d", opts.AudioBitrate)
	}
	return nil
}

// validateSubtitleOptions validates subtitle extraction options
func (t *FFmpegTranscoder) validateSubtitleOptions(opts ports.TranscodeOptions) error {
	if err := t.validateCommonOptions(opts); err != nil {
		return err
	}
	if opts.SourceStreamIndex < 0 {
		return fmt.Errorf("source stream index required for subtitle track")
	}
	return nil
}

// buildVideoArgs builds FFmpeg arguments for video-only transcoding
func (t *FFmpegTranscoder) buildVideoArgs(opts ports.TranscodeOptions) []string {
	args := []string{
		"-i", opts.InputPath,
		"-y",                  // Overwrite output
		"-progress", "pipe:2", // Progress to stderr
	}

	// Map only video stream
	args = append(args, "-map", "0:v:0")

	// Calculate GOP size based on segment duration
	segmentTime := opts.SegmentTime
	if segmentTime == 0 {
		segmentTime = 4
	}
	gopSize := segmentTime * defaultFPS

	// Video codec settings
	videoCodec := opts.VideoCodec
	if videoCodec == "" {
		videoCodec = "libx264"
	}

	preset := opts.Preset
	if preset == "" {
		preset = "medium"
	}

	args = append(args,
		"-c:v", videoCodec,
		"-profile:v", "main",
		"-level", "4.0",
		"-preset", preset,
		"-pix_fmt", "yuv420p",
	)

	// CRF mode (quality-based) or bitrate mode
	if opts.CRF > 0 {
		args = append(args,
			"-crf", fmt.Sprintf("%d", opts.CRF),
			"-maxrate", fmt.Sprintf("%dk", opts.MaxBitrate),
			"-bufsize", fmt.Sprintf("%dk", opts.MaxBitrate*bufSizeMultiplier),
		)
	} else {
		args = append(args,
			"-b:v", fmt.Sprintf("%dk", opts.VideoBitrate),
			"-maxrate", fmt.Sprintf("%dk", opts.MaxBitrate),
			"-bufsize", fmt.Sprintf("%dk", opts.MaxBitrate*bufSizeMultiplier),
		)
	}

	// Scaling and GOP settings
	args = append(args,
		"-vf", fmt.Sprintf("scale=%d:%d", opts.Width, opts.Height),
		"-g", fmt.Sprintf("%d", gopSize),
		"-keyint_min", fmt.Sprintf("%d", gopSize),
		"-sc_threshold", "0",
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", segmentTime),
		"-an", // No audio
		"-sn", // No subtitles
	)

	// Add CMAF segmentation
	return t.addCMAFSegmentingArgs(args, opts)
}

// buildAudioArgs builds FFmpeg arguments for audio-only transcoding
func (t *FFmpegTranscoder) buildAudioArgs(opts ports.TranscodeOptions) []string {
	args := []string{
		"-i", opts.InputPath,
		"-y",                  // Overwrite output
		"-progress", "pipe:2", // Progress to stderr
	}

	// Map specific audio stream by index
	args = append(args, "-map", fmt.Sprintf("0:%d", opts.SourceStreamIndex))

	// Audio codec settings
	audioCodec := opts.AudioCodec
	if audioCodec == "" {
		audioCodec = "aac"
	}

	args = append(args,
		"-c:a", audioCodec,
		"-b:a", fmt.Sprintf("%dk", opts.AudioBitrate),
		"-ar", "48000", // 48kHz sample rate
	)

	// Channel configuration
	if opts.AudioChannels > 0 {
		args = append(args, "-ac", fmt.Sprintf("%d", opts.AudioChannels))
	} else {
		args = append(args, "-ac", "2") // Default to stereo
	}

	args = append(args,
		"-vn", // No video
		"-sn", // No subtitles
	)

	// Add CMAF segmentation
	return t.addCMAFSegmentingArgs(args, opts)
}

// buildSubtitleArgs builds FFmpeg arguments for subtitle extraction
func (t *FFmpegTranscoder) buildSubtitleArgs(opts ports.TranscodeOptions) []string {
	args := []string{
		"-i", opts.InputPath,
		"-y", // Overwrite output
	}

	// Map specific subtitle stream by index
	args = append(args, "-map", fmt.Sprintf("0:%d", opts.SourceStreamIndex))

	// Convert to WebVTT
	args = append(args, "-c:s", "webvtt")

	// Ensure output has .vtt extension
	outputPath := opts.OutputPath
	if !strings.HasSuffix(outputPath, ".vtt") {
		outputPath = outputPath + ".vtt"
	}

	args = append(args, outputPath)

	return args
}

// addCMAFSegmentingArgs adds CMAF segmentation arguments
func (t *FFmpegTranscoder) addCMAFSegmentingArgs(args []string, opts ports.TranscodeOptions) []string {
	segmentTime := opts.SegmentTime
	if segmentTime == 0 {
		segmentTime = 4
	}

	segmentPattern := opts.SegmentPattern
	if segmentPattern == "" {
		segmentPattern = "chunk-%03d.m4s"
	}

	segmentFilePath := filepath.Join(opts.OutputPath, segmentPattern)
	initSegmentPath := filepath.Join(opts.OutputPath, "init.mp4")

	args = append(args,
		"-f", "segment",
		"-segment_time", fmt.Sprintf("%d", segmentTime),
		"-segment_format", "mp4",
		"-segment_format_options", "movflags=frag_keyframe+empty_moov+default_base_moof+dash",
		"-segment_header_filename", initSegmentPath,
		segmentFilePath,
	)

	return args
}

// executeTranscode executes the FFmpeg transcoding command
func (t *FFmpegTranscoder) executeTranscode(ctx context.Context, args []string, opts ports.TranscodeOptions, callback ports.ProgressCallback) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, t.ffmpegPath, args...)

	// Get stderr pipe for progress tracking
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Buffer to capture stderr for error reporting
	var stderrBuf strings.Builder
	stderrReader := io.TeeReader(stderr, &stderrBuf)

	// Track progress (consumes stderr to prevent blocking)
	done := make(chan struct{})
	go func() {
		defer close(done)
		t.trackProgress(stderrReader, callback)
	}()

	// Wait for command to finish
	err = cmd.Wait()

	// Ensure stderr consumption is complete
	<-done

	// Check for errors
	if err != nil {
		stderrOutput := stderrBuf.String()

		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("transcode timeout after %s: %w", t.timeout, err)
		}

		return fmt.Errorf("transcode failed: %w (stderr: %s)", err, stderrOutput)
	}

	// Verify output was created
	return t.verifyOutput(opts)
}

// verifyOutput verifies that transcoding produced the expected output files
func (t *FFmpegTranscoder) verifyOutput(opts ports.TranscodeOptions) error {
	// Check if this is subtitle extraction
	if strings.HasSuffix(opts.OutputPath, ".vtt") || strings.Contains(opts.TrackType, "subtitle") {
		outputPath := opts.OutputPath
		if !strings.HasSuffix(outputPath, ".vtt") {
			outputPath = outputPath + ".vtt"
		}
		if _, err := os.Stat(outputPath); err != nil {
			return fmt.Errorf("subtitle file not created: %w", err)
		}
		return nil
	}

	// For video/audio, verify CMAF segments
	initSegmentPath := filepath.Join(opts.OutputPath, "init.mp4")
	if _, err := os.Stat(initSegmentPath); err != nil {
		return fmt.Errorf("init segment (init.mp4) not created: %w", err)
	}

	// Verify media segments were created
	entries, err := os.ReadDir(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to read output directory: %w", err)
	}

	hasSegments := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".m4s") {
			hasSegments = true
			break
		}
	}

	if !hasSegments {
		return fmt.Errorf("no CMAF media segments created in output directory")
	}

	return nil
}

// trackProgress parses FFmpeg stderr output and calls the progress callback
func (t *FFmpegTranscoder) trackProgress(stderr io.Reader, callback ports.ProgressCallback) {
	if callback == nil {
		// Still consume stderr to prevent blocking
		io.Copy(io.Discard, stderr)
		return
	}

	scanner := bufio.NewScanner(stderr)

	// Increase buffer size for long lines
	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	// Regular expressions for parsing progress
	frameRe := regexp.MustCompile(`frame=\s*(\d+)`)
	fpsRe := regexp.MustCompile(`fps=\s*([\d.]+)`)
	bitrateRe := regexp.MustCompile(`bitrate=\s*([\d.]+\w+)`)
	timeRe := regexp.MustCompile(`time=(\d{2}:\d{2}:\d{2}\.\d{2})`)
	speedRe := regexp.MustCompile(`speed=\s*([\d.]+x)`)

	progress := ports.TranscodeProgress{}
	lastUpdate := time.Now()

	for scanner.Scan() {
		line := scanner.Text()

		// Parse progress fields
		if matches := frameRe.FindStringSubmatch(line); len(matches) > 1 {
			if frame, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
				progress.Frame = frame
			}
		}
		if matches := fpsRe.FindStringSubmatch(line); len(matches) > 1 {
			if fps, err := strconv.ParseFloat(matches[1], 64); err == nil {
				progress.FPS = fps
			}
		}
		if matches := bitrateRe.FindStringSubmatch(line); len(matches) > 1 {
			progress.Bitrate = matches[1]
		}
		if matches := timeRe.FindStringSubmatch(line); len(matches) > 1 {
			progress.Time = matches[1]
		}
		if matches := speedRe.FindStringSubmatch(line); len(matches) > 1 {
			progress.Speed = matches[1]
		}

		// Call callback periodically (every second)
		if time.Since(lastUpdate) >= 1*time.Second {
			callback(progress)
			lastUpdate = time.Now()
		}
	}

	// Final callback with 100% progress
	progress.Percentage = 100.0
	callback(progress)
}

// probeSegmentDuration probes a single segment file and returns its duration in milliseconds
func (t *FFmpegTranscoder) probeSegmentDuration(ctx context.Context, segmentPath, initSegmentPath string) (int64, error) {
	args := []string{
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		segmentPath,
	}

	cmd := exec.CommandContext(ctx, t.ffprobePath, args...)
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
		cmd = exec.CommandContext(ctx, t.ffprobePath, args...)
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
