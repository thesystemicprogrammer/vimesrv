package transcoding

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

// FFmpegConfig holds configuration for the FFmpeg transcoder
type FFmpegConfig struct {
	FFmpegPath     string        // Path to ffmpeg binary (default: "ffmpeg")
	FFprobePath    string        // Path to ffprobe binary (default: "ffprobe")
	TimeoutSeconds int           // Timeout for transcoding operations (default: 7200)
	Timeout        time.Duration // Alternative: specify timeout directly
}

// NewFFmpegTranscoder creates a new FFmpegTranscoder with the specified configuration
func NewFFmpegTranscoder(cfg FFmpegConfig) *FFmpegTranscoder {
	ffmpegPath := cfg.FFmpegPath
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	ffprobePath := cfg.FFprobePath
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}

	timeout := cfg.Timeout
	if timeout == 0 && cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	if timeout == 0 {
		timeout = 2 * time.Hour // Default timeout for transcoding
	}

	return &FFmpegTranscoder{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		timeout:     timeout,
	}
}

// NewFFmpegTranscoderWithTimeout creates a new FFmpegTranscoder with the specified timeout in seconds
// If timeoutSeconds is 0, defaults to 2 hours
func NewFFmpegTranscoderWithTimeout(timeoutSeconds int) *FFmpegTranscoder {
	return NewFFmpegTranscoder(FFmpegConfig{
		TimeoutSeconds: timeoutSeconds,
	})
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
func (t *FFmpegTranscoder) TranscodeVideo(ctx context.Context, opts Options, callback ProgressCallback) error {
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
func (t *FFmpegTranscoder) TranscodeAudio(ctx context.Context, opts Options, callback ProgressCallback) error {
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
func (t *FFmpegTranscoder) ExtractSubtitle(ctx context.Context, opts Options) error {
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
func (t *FFmpegTranscoder) ProbeSegmentDurations(ctx context.Context, outputPath string) ([]SegmentInfo, error) {
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
	// (alphabetical sort fails for >999 segments: chunk-1000 comes before chunk-100)
	sort.Slice(segmentFiles, func(i, j int) bool {
		return extractSegmentNumber(segmentFiles[i]) < extractSegmentNumber(segmentFiles[j])
	})

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

// validateCommonOptions validates options common to all transcode types
func (t *FFmpegTranscoder) validateCommonOptions(opts Options) error {
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
func (t *FFmpegTranscoder) validateVideoOptions(opts Options) error {
	if err := t.validateCommonOptions(opts); err != nil {
		return err
	}
	// Dimensions can be 0x0 (keep original) or both must be positive
	if (opts.Width < 0 || opts.Height < 0) || (opts.Width == 0) != (opts.Height == 0) {
		return fmt.Errorf("invalid dimensions: %dx%d (must be both 0 for original or both positive)", opts.Width, opts.Height)
	}
	if opts.CRF <= 0 && opts.VideoBitrate <= 0 {
		return fmt.Errorf("either CRF or video bitrate must be specified")
	}
	return nil
}

// validateAudioOptions validates audio transcoding options
func (t *FFmpegTranscoder) validateAudioOptions(opts Options) error {
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
func (t *FFmpegTranscoder) validateSubtitleOptions(opts Options) error {
	if err := t.validateCommonOptions(opts); err != nil {
		return err
	}
	if opts.SourceStreamIndex < 0 {
		return fmt.Errorf("source stream index required for subtitle track")
	}
	return nil
}

// isHardwareEncoder returns true if the encoder is a hardware-accelerated encoder
func isHardwareEncoder(encoder string) bool {
	return strings.HasSuffix(encoder, "_vaapi") ||
		strings.HasSuffix(encoder, "_qsv") ||
		strings.HasSuffix(encoder, "_nvenc") ||
		strings.HasSuffix(encoder, "_amf") ||
		strings.HasSuffix(encoder, "_videotoolbox")
}

// getEncoderType returns the hardware acceleration type from the encoder name
// Returns: "vaapi", "qsv", "nvenc", "amf", "videotoolbox", or "software"
func getEncoderType(encoder string) string {
	switch {
	case strings.HasSuffix(encoder, "_vaapi"):
		return "vaapi"
	case strings.HasSuffix(encoder, "_qsv"):
		return "qsv"
	case strings.HasSuffix(encoder, "_nvenc"):
		return "nvenc"
	case strings.HasSuffix(encoder, "_amf"):
		return "amf"
	case strings.HasSuffix(encoder, "_videotoolbox"):
		return "videotoolbox"
	default:
		return "software"
	}
}

// getQualityArgs returns the appropriate quality argument for the encoder
// Different encoders use different quality parameters:
// - libx264/libx265: -crf (Constant Rate Factor)
// - h264_vaapi: -qp (Quantization Parameter)
// - h264_qsv: -global_quality (Intel QSV quality)
// - h264_nvenc: -cq (Constant Quality)
func getQualityArgs(encoder string, crf int) []string {
	encoderType := getEncoderType(encoder)

	switch encoderType {
	case "vaapi":
		// VAAPI uses -qp for constant quality mode
		return []string{"-qp", fmt.Sprintf("%d", crf)}
	case "qsv":
		// Intel QSV uses -global_quality with ICQ (Intelligent Constant Quality) mode
		return []string{"-global_quality", fmt.Sprintf("%d", crf)}
	case "nvenc":
		// NVENC uses -cq for constant quality mode
		return []string{"-cq", fmt.Sprintf("%d", crf)}
	default:
		// Software encoders (libx264, libx265) use -crf
		return []string{"-crf", fmt.Sprintf("%d", crf)}
	}
}

// getScaleFilter returns the appropriate scale filter for the given encoder and config
// scaleFilter can be: "auto", "software", "vaapi", "qsv"
// When "auto", the filter is chosen based on the encoder type
func getScaleFilter(encoder, scaleFilter string, width, height int) string {
	// Determine which scale filter to use
	filterType := scaleFilter
	if filterType == "" || filterType == "auto" {
		filterType = getEncoderType(encoder)
	}

	switch filterType {
	case "vaapi":
		// VAAPI scale filter - format option ensures correct pixel format
		return fmt.Sprintf("scale_vaapi=w=%d:h=%d", width, height)
	case "qsv":
		// QSV scale filter
		return fmt.Sprintf("scale_qsv=w=%d:h=%d", width, height)
	default:
		// Software scale filter (default)
		return fmt.Sprintf("scale=%d:%d", width, height)
	}
}

// buildVideoArgs builds FFmpeg arguments for video-only transcoding
func (t *FFmpegTranscoder) buildVideoArgs(opts Options) []string {
	args := []string{}

	// Add custom input arguments before -i (e.g., hardware acceleration)
	if len(opts.FFmpegInputArgs) > 0 {
		args = append(args, opts.FFmpegInputArgs...)
	}

	args = append(args,
		"-i", opts.InputPath,
		"-y",                  // Overwrite output
		"-progress", "pipe:2", // Progress to stderr
	)

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

	isHWEncoder := isHardwareEncoder(videoCodec)

	args = append(args, "-c:v", videoCodec)

	// Add encoder-specific options (profile, level, preset, pix_fmt)
	// These are only reliably supported by software encoders
	if !isHWEncoder {
		args = append(args,
			"-profile:v", "main",
			"-level", "4.0",
			"-pix_fmt", "yuv420p",
		)

		// Preset is only supported by software encoders and NVENC
		preset := opts.Preset
		if preset == "" {
			preset = "medium"
		}
		args = append(args, "-preset", preset)
	} else {
		// For NVENC, preset is supported but uses different values
		encoderType := getEncoderType(videoCodec)
		if encoderType == "nvenc" {
			preset := opts.Preset
			if preset == "" {
				preset = "p4" // NVENC default balanced preset
			}
			args = append(args, "-preset", preset)
		}
		// VAAPI and QSV don't support -preset
	}

	// CRF/quality mode (quality-based) or bitrate mode
	if opts.CRF > 0 {
		// Add encoder-appropriate quality parameter
		args = append(args, getQualityArgs(videoCodec, opts.CRF)...)

		// Add rate control constraints
		args = append(args,
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

	// Scaling (only if dimensions are specified)
	if opts.Width > 0 && opts.Height > 0 {
		scaleFilter := getScaleFilter(videoCodec, opts.ScaleFilter, opts.Width, opts.Height)
		args = append(args, "-vf", scaleFilter)
	}

	// GOP settings
	args = append(args,
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
func (t *FFmpegTranscoder) buildAudioArgs(opts Options) []string {
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
		"-af", "aresample=async=1:first_pts=0",
		"-vn",
		"-sn",
	)

	// Add CMAF segmentation
	return t.addCMAFSegmentingArgs(args, opts)
}

// buildSubtitleArgs builds FFmpeg arguments for subtitle extraction
func (t *FFmpegTranscoder) buildSubtitleArgs(opts Options) []string {
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
func (t *FFmpegTranscoder) addCMAFSegmentingArgs(args []string, opts Options) []string {
	segmentTime := opts.SegmentTime
	if segmentTime == 0 {
		segmentTime = 4
	}

	segmentPattern := opts.SegmentPattern
	if segmentPattern == "" {
		segmentPattern = "chunk-%05d.m4s"
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
func (t *FFmpegTranscoder) executeTranscode(ctx context.Context, args []string, opts Options, callback ProgressCallback) error {
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
func (t *FFmpegTranscoder) verifyOutput(opts Options) error {
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
func (t *FFmpegTranscoder) trackProgress(stderr io.Reader, callback ProgressCallback) {
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

	progress := Progress{}
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

// VerifyOutput is a standalone function to verify transcoded output exists
// This can be used by the server to validate worker-produced output
func VerifyOutput(outputPath string, trackType string) error {
	// Check if this is subtitle extraction
	if strings.HasSuffix(outputPath, ".vtt") || strings.Contains(trackType, "subtitle") {
		checkPath := outputPath
		if !strings.HasSuffix(checkPath, ".vtt") {
			checkPath = checkPath + ".vtt"
		}
		if _, err := os.Stat(checkPath); err != nil {
			return fmt.Errorf("subtitle file not found: %w", err)
		}
		return nil
	}

	// For video/audio, verify CMAF segments
	initSegmentPath := filepath.Join(outputPath, "init.mp4")
	if _, err := os.Stat(initSegmentPath); err != nil {
		return fmt.Errorf("init segment (init.mp4) not found: %w", err)
	}

	// Verify media segments exist
	entries, err := os.ReadDir(outputPath)
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
		return fmt.Errorf("no CMAF media segments found in output directory")
	}

	return nil
}

// extractSegmentNumber extracts the numeric segment number from a filename like "chunk-000.m4s"
// Returns 0 if the number cannot be extracted
func extractSegmentNumber(filename string) int {
	// Remove .m4s extension
	name := strings.TrimSuffix(filename, ".m4s")
	// Find the last dash and extract the number after it
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
