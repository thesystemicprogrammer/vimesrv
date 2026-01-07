package media

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"github.com/thesystemicprogrammer/vimesrv/pkg/transcoding"
)

// ffmpegTranscoderAdapter wraps the shared FFmpegTranscoder to implement ports.Transcoder
type ffmpegTranscoderAdapter struct {
	transcoder *transcoding.FFmpegTranscoder
}

// NewFFmpegTranscoder creates a new FFmpegTranscoder with the specified timeout in seconds
// If timeoutSeconds is 0, defaults to 2 hours
func NewFFmpegTranscoder(timeoutSeconds int) ports.Transcoder {
	return &ffmpegTranscoderAdapter{
		transcoder: transcoding.NewFFmpegTranscoderWithTimeout(timeoutSeconds),
	}
}

// IsAvailable checks if FFmpeg is available and executable
func (a *ffmpegTranscoderAdapter) IsAvailable() error {
	return a.transcoder.IsAvailable()
}

// TranscodeVideo transcodes a video stream to the specified quality with CMAF segmentation
func (a *ffmpegTranscoderAdapter) TranscodeVideo(ctx context.Context, opts ports.TranscodeOptions, callback ports.ProgressCallback) error {
	// Convert ports.TranscodeOptions to transcoding.Options
	tOpts := toTranscodingOptions(opts)

	// Convert callback
	var tCallback transcoding.ProgressCallback
	if callback != nil {
		tCallback = func(p transcoding.Progress) {
			callback(toPortsProgress(p))
		}
	}

	return a.transcoder.TranscodeVideo(ctx, tOpts, tCallback)
}

// TranscodeAudio transcodes an audio stream to AAC with CMAF segmentation
func (a *ffmpegTranscoderAdapter) TranscodeAudio(ctx context.Context, opts ports.TranscodeOptions, callback ports.ProgressCallback) error {
	// Convert ports.TranscodeOptions to transcoding.Options
	tOpts := toTranscodingOptions(opts)

	// Convert callback
	var tCallback transcoding.ProgressCallback
	if callback != nil {
		tCallback = func(p transcoding.Progress) {
			callback(toPortsProgress(p))
		}
	}

	return a.transcoder.TranscodeAudio(ctx, tOpts, tCallback)
}

// ExtractSubtitle extracts a subtitle stream and converts it to WebVTT format
func (a *ffmpegTranscoderAdapter) ExtractSubtitle(ctx context.Context, opts ports.TranscodeOptions) error {
	tOpts := toTranscodingOptions(opts)
	return a.transcoder.ExtractSubtitle(ctx, tOpts)
}

// ProbeSegmentDurations probes all segment files and returns their exact durations
func (a *ffmpegTranscoderAdapter) ProbeSegmentDurations(ctx context.Context, outputPath string) ([]ports.SegmentInfo, error) {
	segments, err := a.transcoder.ProbeSegmentDurations(ctx, outputPath)
	if err != nil {
		return nil, err
	}

	// Convert to ports.SegmentInfo
	result := make([]ports.SegmentInfo, len(segments))
	for i, s := range segments {
		result[i] = ports.SegmentInfo{
			Number:   s.Number,
			Duration: s.Duration,
		}
	}
	return result, nil
}

// toTranscodingOptions converts ports.TranscodeOptions to transcoding.Options
func toTranscodingOptions(opts ports.TranscodeOptions) transcoding.Options {
	return transcoding.Options{
		InputPath:         opts.InputPath,
		SourceStreamIndex: opts.SourceStreamIndex,
		OutputPath:        opts.OutputPath,
		Width:             opts.Width,
		Height:            opts.Height,
		VideoCodec:        opts.VideoCodec,
		CRF:               opts.CRF,
		MaxBitrate:        opts.MaxBitrate,
		VideoBitrate:      opts.VideoBitrate,
		Preset:            opts.Preset,
		FFmpegInputArgs:   opts.FFmpegInputArgs,
		AudioCodec:        opts.AudioCodec,
		AudioBitrate:      opts.AudioBitrate,
		AudioChannels:     opts.AudioChannels,
		SegmentTime:       opts.SegmentTime,
		SegmentPattern:    opts.SegmentPattern,
		TrackType:         opts.TrackType,
	}
}

// toPortsProgress converts transcoding.Progress to ports.TranscodeProgress
func toPortsProgress(p transcoding.Progress) ports.TranscodeProgress {
	return ports.TranscodeProgress{
		Frame:      p.Frame,
		FPS:        p.FPS,
		Bitrate:    p.Bitrate,
		Time:       p.Time,
		Speed:      p.Speed,
		Percentage: p.Percentage,
	}
}
