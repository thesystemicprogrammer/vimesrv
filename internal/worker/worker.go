// Package worker implements the distributed transcoding worker.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/thesystemicprogrammer/vimesrv/internal/worker/client"
	"github.com/thesystemicprogrammer/vimesrv/internal/worker/config"
	"github.com/thesystemicprogrammer/vimesrv/pkg/transcoding"
)

// Worker represents a distributed transcoding worker
type Worker struct {
	id         string
	name       string
	config     *config.Config
	client     *client.ServerClient
	transcoder *transcoding.FFmpegTranscoder
	logger     zerolog.Logger

	activeJobs int32 // atomic counter
	jobs       chan *client.WorkerJob
	wg         sync.WaitGroup
}

// New creates a new Worker
func New(id string, cfg *config.Config, serverClient *client.ServerClient, transcoder *transcoding.FFmpegTranscoder, logger zerolog.Logger) *Worker {
	return &Worker{
		id:         id,
		name:       cfg.Worker.Name,
		config:     cfg,
		client:     serverClient,
		transcoder: transcoder,
		logger:     logger.With().Str("component", "worker").Str("worker_id", id).Logger(),
		jobs:       make(chan *client.WorkerJob, cfg.Worker.Concurrency),
	}
}

// Start starts the worker and blocks until shutdown
func (w *Worker) Start(ctx context.Context) error {
	// Register with server
	if err := w.client.Register(ctx, w.id, w.name, w.config.Worker.Concurrency); err != nil {
		return fmt.Errorf("failed to register with server: %w", err)
	}
	w.logger.Info().Str("name", w.name).Msg("Registered with server")

	// Start worker goroutines for concurrent job processing
	for i := 0; i < w.config.Worker.Concurrency; i++ {
		w.wg.Add(1)
		go w.processLoop(ctx, i)
	}

	// Start poll loop to claim jobs
	w.wg.Add(1)
	go w.pollLoop(ctx)

	// Start heartbeat loop
	w.wg.Add(1)
	go w.heartbeatLoop(ctx)

	// Wait for shutdown signal
	<-ctx.Done()
	w.logger.Info().Msg("Shutdown signal received, waiting for active jobs to complete...")

	// Close jobs channel to signal workers to stop accepting new jobs
	close(w.jobs)

	// Wait for all goroutines to finish
	w.wg.Wait()

	return nil
}

// pollLoop polls the server for new jobs
func (w *Worker) pollLoop(ctx context.Context) {
	defer w.wg.Done()

	pollInterval := time.Duration(w.config.Worker.PollIntervalSeconds) * time.Second
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tryClaimJob(ctx)
		}
	}
}

// tryClaimJob attempts to claim a job if there is capacity
func (w *Worker) tryClaimJob(ctx context.Context) {
	// Check if we have capacity
	active := atomic.LoadInt32(&w.activeJobs)
	if int(active) >= w.config.Worker.Concurrency {
		return
	}

	// Try to claim a job
	job, err := w.client.ClaimJob(ctx, w.id)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to claim job")
		return
	}
	if job == nil {
		// No jobs available
		return
	}

	w.logger.Info().
		Int64("job_id", job.JobID).
		Str("transcode_id", job.TranscodeID).
		Str("track_type", job.TrackType).
		Str("quality", job.Quality).
		Msg("Claimed job")

	// Send to processing channel
	select {
	case w.jobs <- job:
		atomic.AddInt32(&w.activeJobs, 1)
	default:
		// Channel full - shouldn't happen due to capacity check but handle gracefully
		w.logger.Warn().Int64("job_id", job.JobID).Msg("Job channel full, releasing job")
		if err := w.client.FailJob(ctx, job.JobID, w.id, "worker channel full", true); err != nil {
			w.logger.Error().Err(err).Msg("Failed to release job")
		}
	}
}

// processLoop processes jobs from the jobs channel
func (w *Worker) processLoop(ctx context.Context, workerNum int) {
	defer w.wg.Done()

	for job := range w.jobs {
		w.processJob(ctx, job, workerNum)
		atomic.AddInt32(&w.activeJobs, -1)
	}
}

// processJob processes a single job
func (w *Worker) processJob(ctx context.Context, job *client.WorkerJob, workerNum int) {
	logger := w.logger.With().
		Int64("job_id", job.JobID).
		Str("transcode_id", job.TranscodeID).
		Str("track_type", job.TrackType).
		Str("quality", job.Quality).
		Int("worker_num", workerNum).
		Logger()

	logger.Info().Msg("Processing job")
	startTime := time.Now()

	// Create progress callback
	lastProgress := time.Now()
	progressInterval := time.Duration(w.config.Worker.ProgressIntervalSeconds) * time.Second

	progressCallback := func(p transcoding.Progress) {
		if time.Since(lastProgress) < progressInterval {
			return
		}
		lastProgress = time.Now()

		// Calculate percentage based on time if not provided
		percentage := p.Percentage
		if percentage == 0 && job.MediaDuration > 0 && p.Time != "" {
			// Parse time string HH:MM:SS.ms
			currentTime := parseTimeToSeconds(p.Time)
			if currentTime > 0 {
				percentage = (currentTime / job.MediaDuration) * 100
				if percentage > 100 {
					percentage = 100
				}
			}
		}

		// Calculate ETA
		var etaSeconds int
		if percentage > 0 && percentage < 100 {
			elapsed := time.Since(startTime)
			remaining := float64(elapsed) * (100 - percentage) / percentage
			etaSeconds = int(remaining / float64(time.Second))
		}

		if err := w.client.ReportProgress(ctx, job.JobID, w.id, percentage, p.Speed, etaSeconds); err != nil {
			logger.Warn().Err(err).Msg("Failed to report progress")
		}
	}

	// Build transcoding options from job
	opts := w.buildTranscodeOptions(job)

	// Execute transcoding based on track type
	var err error
	switch job.TrackType {
	case "video":
		err = w.transcoder.TranscodeVideo(ctx, opts, progressCallback)
	case "audio":
		err = w.transcoder.TranscodeAudio(ctx, opts, progressCallback)
	case "subtitle":
		err = w.transcoder.ExtractSubtitle(ctx, opts)
	default:
		err = fmt.Errorf("unknown track type: %s", job.TrackType)
	}

	if err != nil {
		logger.Error().Err(err).Msg("Transcoding failed")
		if reportErr := w.client.FailJob(ctx, job.JobID, w.id, err.Error(), true); reportErr != nil {
			logger.Error().Err(reportErr).Msg("Failed to report job failure")
		}
		return
	}

	// Count output files for reporting
	segmentCount, outputFiles := countOutputFiles(job.OutputPath, job.TrackType)

	// For video/audio tracks, probe segment durations and save to segments.json locally
	if job.TrackType == "video" || job.TrackType == "audio" {
		resolvedOutputPath := w.resolvePath(job.OutputPath)
		segments, err := w.transcoder.ProbeSegmentDurations(ctx, resolvedOutputPath)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to probe segment durations")
			if reportErr := w.client.FailJob(ctx, job.JobID, w.id, "segment probing failed: "+err.Error(), true); reportErr != nil {
				logger.Error().Err(reportErr).Msg("Failed to report job failure")
			}
			return
		}
		if err := saveSegmentsJSON(resolvedOutputPath, segments); err != nil {
			logger.Error().Err(err).Msg("Failed to save segments.json")
			if reportErr := w.client.FailJob(ctx, job.JobID, w.id, "failed to save segments.json: "+err.Error(), true); reportErr != nil {
				logger.Error().Err(reportErr).Msg("Failed to report job failure")
			}
			return
		}
		logger.Info().Int("segment_count", len(segments)).Msg("Probed and saved segment durations")
	}

	// Report completion with retry logic
	if err := w.completeJobWithRetry(ctx, job.JobID, segmentCount, outputFiles, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to report job completion after retries")
		return
	}

	elapsed := time.Since(startTime)
	logger.Info().
		Dur("duration", elapsed).
		Int("segments", segmentCount).
		Msg("Job completed successfully")
}

// completeJobWithRetry attempts to report job completion with exponential backoff retry
func (w *Worker) completeJobWithRetry(ctx context.Context, jobID int64, segmentCount int, outputFiles []string, logger zerolog.Logger) error {
	completeFunc := func() error {
		return w.client.CompleteJob(ctx, jobID, w.id, segmentCount, outputFiles)
	}
	return retryWithExponentialBackoff(ctx, completeFunc, 3, 2*time.Second, logger)
}

// retryWithExponentialBackoff executes the given function with exponential backoff retry.
// maxRetries is the maximum number of attempts, baseDelay is the initial delay (doubles each retry).
func retryWithExponentialBackoff(ctx context.Context, fn func() error, maxRetries int, baseDelay time.Duration, logger zerolog.Logger) error {
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := fn()
		if err == nil {
			if attempt > 1 {
				logger.Info().Int("attempt", attempt).Msg("Operation succeeded after retry")
			}
			return nil
		}

		lastErr = err
		if attempt < maxRetries {
			delay := baseDelay * time.Duration(1<<(attempt-1)) // Exponential: 2s, 4s, 8s
			logger.Warn().
				Err(err).
				Int("attempt", attempt).
				Int("max_retries", maxRetries).
				Dur("retry_delay", delay).
				Msg("Operation failed, retrying...")

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled while retrying: %w", ctx.Err())
			case <-time.After(delay):
				// Continue to next retry
			}
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries, lastErr)
}

// buildTranscodeOptions converts a WorkerJob to transcoding.Options
func (w *Worker) buildTranscodeOptions(job *client.WorkerJob) transcoding.Options {
	opts := transcoding.Options{
		InputPath:         w.resolvePath(job.InputPath),
		OutputPath:        w.resolvePath(job.OutputPath),
		SourceStreamIndex: job.TrackIndex,
		SegmentTime:       job.TranscodeOptions.SegmentTime,
		TrackType:         job.TrackType,
		FFmpegInputArgs:   w.config.Transcoding.FFmpegInputArgs,
		ScaleFilter:       w.config.Transcoding.ScaleFilter,
	}

	// Video options
	if job.TrackType == "video" {
		opts.Width = job.TranscodeOptions.Width
		opts.Height = job.TranscodeOptions.Height

		// Use worker's configured encoder, falling back to job's codec, then libx264
		opts.VideoCodec = w.config.Transcoding.VideoEncoder
		if opts.VideoCodec == "" {
			opts.VideoCodec = job.TranscodeOptions.VideoCodec
		}
		if opts.VideoCodec == "" {
			opts.VideoCodec = "libx264"
		}

		opts.CRF = job.TranscodeOptions.CRF
		opts.MaxBitrate = job.TranscodeOptions.MaxBitrate

		// Use worker's configured preset, falling back to job's preset, then medium
		opts.Preset = w.config.Transcoding.EncoderPreset
		if opts.Preset == "" {
			opts.Preset = job.TranscodeOptions.Preset
		}
		if opts.Preset == "" {
			opts.Preset = "medium"
		}
	}

	// Audio options
	if job.TrackType == "audio" {
		opts.AudioCodec = job.TranscodeOptions.AudioCodec
		opts.AudioBitrate = job.TranscodeOptions.AudioBitrate
	}

	return opts
}

// resolvePath converts a relative path from the server to an absolute local path
// by prepending the worker's configured media_path.
func (w *Worker) resolvePath(relativePath string) string {
	return filepath.Join(w.config.Media.MediaPath, relativePath)
}

// heartbeatLoop sends periodic heartbeats when idle
func (w *Worker) heartbeatLoop(ctx context.Context) {
	defer w.wg.Done()

	heartbeatInterval := time.Duration(w.config.Worker.HeartbeatIntervalSeconds) * time.Second
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			active := int(atomic.LoadInt32(&w.activeJobs))
			resp, err := w.client.Heartbeat(ctx, w.id, w.name, active, w.config.Worker.Concurrency)
			if err != nil {
				w.logger.Warn().Err(err).Msg("Heartbeat failed")
				continue
			}
			w.logger.Debug().
				Int("active_jobs", active).
				Int("queued_jobs", resp.QueuedJobs).
				Msg("Heartbeat sent")
		}
	}
}

// parseTimeToSeconds parses FFmpeg time format (HH:MM:SS.ms) to seconds
func parseTimeToSeconds(timeStr string) float64 {
	var hours, minutes int
	var seconds float64

	_, err := fmt.Sscanf(timeStr, "%d:%d:%f", &hours, &minutes, &seconds)
	if err != nil {
		return 0
	}

	return float64(hours)*3600 + float64(minutes)*60 + seconds
}

// countOutputFiles counts segments and lists output files
func countOutputFiles(outputPath, trackType string) (int, []string) {
	if trackType == "subtitle" {
		// For subtitles, just return the single file
		return 1, []string{outputPath}
	}

	// For video/audio, count .m4s segments
	entries, err := os.ReadDir(outputPath)
	if err != nil {
		return 0, nil
	}

	var files []string
	segmentCount := 0
	for _, entry := range entries {
		name := entry.Name()
		files = append(files, name)
		if strings.HasSuffix(name, ".m4s") {
			segmentCount++
		}
	}

	return segmentCount, files
}

// saveSegmentsJSON saves segment timing data to segments.json in the output directory
func saveSegmentsJSON(outputPath string, segments []transcoding.SegmentInfo) error {
	data := struct {
		Segments []transcoding.SegmentInfo `json:"segments"`
	}{
		Segments: segments,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal segments: %w", err)
	}

	return os.WriteFile(filepath.Join(outputPath, "segments.json"), jsonData, 0644)
}
