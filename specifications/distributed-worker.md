# Distributed Transcoding Worker Specification

## Overview

This specification describes the implementation of a distributed transcoding worker system for vimesrv. The system allows transcoding jobs to be offloaded from the main server to remote worker machines that have access to the media library via shared storage (e.g., NFS).

### Goals

1. Offload CPU/GPU-intensive transcoding from the main server
2. Allow horizontal scaling by adding multiple worker machines
3. Maintain compatibility with existing job queue system
4. Provide visibility into worker status and job progress

### Non-Goals

1. Worker auto-discovery (workers must be configured with server URL)
2. Load balancing between workers (first-come-first-served claiming)
3. GPU-specific optimizations (future enhancement)

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Main Server                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   SQLite    │  │ Job Manager │  │ Worker API  │  │ Worker Registry     │  │
│  │   (jobs)    │◄─┤  (local)    │  │  Handlers   │  │ (in-memory)         │  │
│  └─────────────┘  └─────────────┘  └──────┬──────┘  └─────────────────────┘  │
│                                           │                                   │
└───────────────────────────────────────────┼───────────────────────────────────┘
                                            │ HTTP API (REST)
                    ┌───────────────────────┼───────────────────────┐
                    │                       │                       │
              ┌─────▼─────┐           ┌─────▼─────┐           ┌─────▼─────┐
              │  Worker 1 │           │  Worker 2 │           │  Worker N │
              │           │           │           │           │           │
              └───────────┘           └───────────┘           └───────────┘
                    │                       │                       │
                    └───────────────────────┼───────────────────────┘
                                            │ Shared Storage (NFS)
                                    ┌───────▼───────┐
                                    │ Media Library │
                                    │   /media/     │
                                    └───────────────┘
```

### Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Worker state persistence | In-memory only | Simplicity; workers re-register on server restart |
| Job claiming | Pull-based (workers poll) | Simpler than push; works through NAT/firewalls |
| Progress reporting | REST calls | Workers POST progress; server broadcasts via WebSocket |
| Shared code | `pkg/transcoding` package | Both binaries share FFmpeg logic |
| Job exclusivity | Workers-only when enabled | Server not suited for heavy transcoding |
| Concurrent jobs per worker | Configurable | Utilize multi-core/multi-GPU machines |

---

## Configuration

### Server Configuration

Add to `internal/shared/config/config.go`:

```go
type WorkerConfig struct {
    Enabled                  bool   `mapstructure:"enabled"`
    AuthToken                string `mapstructure:"auth_token"`
    HeartbeatTimeoutSeconds  int    `mapstructure:"heartbeat_timeout_seconds"`
    FallbackToLocal          bool   `mapstructure:"fallback_to_local"`
    FallbackAfterMinutes     int    `mapstructure:"fallback_after_minutes"`
}
```

Add to main `Config` struct:

```go
type Config struct {
    // ... existing fields ...
    Worker WorkerConfig `mapstructure:"worker"`
}
```

Default values in `loader.go`:

```go
v.SetDefault("worker.enabled", false)
v.SetDefault("worker.auth_token", "")
v.SetDefault("worker.heartbeat_timeout_seconds", 60)
v.SetDefault("worker.fallback_to_local", false)
v.SetDefault("worker.fallback_after_minutes", 30)
```

Example `configs/default.yaml`:

```yaml
worker:
  enabled: false                      # Enable worker API and exclusive worker processing
  auth_token: ""                      # Shared secret for worker authentication (required if enabled)
  heartbeat_timeout_seconds: 60       # Worker considered dead if no heartbeat/progress
  fallback_to_local: false            # Process locally when no workers available
  fallback_after_minutes: 30          # Wait time before local fallback (if enabled)
```

### Worker Binary Configuration

New file `configs/worker.yaml`:

```yaml
server:
  url: "http://localhost:8080"        # Main server URL
  auth_token: ""                      # Must match server's worker.auth_token

worker:
  id: ""                              # Auto-generated UUID if empty
  name: "worker-1"                    # Human-readable name for logging
  poll_interval_seconds: 5            # How often to poll for jobs
  heartbeat_interval_seconds: 30      # How often to send heartbeat when idle
  concurrency: 2                      # Max concurrent transcode jobs
  progress_interval_seconds: 5        # How often to report progress

media:
  media_path: "/mnt/nfs/media"        # Path to media library (must match server's media_path)

transcoding:
  ffmpeg_path: "ffmpeg"               # Path to ffmpeg binary
  ffprobe_path: "ffprobe"             # Path to ffprobe binary
  timeout_seconds: 7200               # Max time for a single transcode job

logging:
  level: "info"
  format: "console"                   # "console" or "json"
```

Worker config struct (`internal/worker/config/config.go`):

```go
type Config struct {
    Server      ServerConfig      `mapstructure:"server"`
    Worker      WorkerSettings    `mapstructure:"worker"`
    Media       MediaConfig       `mapstructure:"media"`
    Transcoding TranscodingConfig `mapstructure:"transcoding"`
    Logging     LoggingConfig     `mapstructure:"logging"`
}

type ServerConfig struct {
    URL       string `mapstructure:"url"`
    AuthToken string `mapstructure:"auth_token"`
}

type WorkerSettings struct {
    ID                       string `mapstructure:"id"`
    Name                     string `mapstructure:"name"`
    PollIntervalSeconds      int    `mapstructure:"poll_interval_seconds"`
    HeartbeatIntervalSeconds int    `mapstructure:"heartbeat_interval_seconds"`
    Concurrency              int    `mapstructure:"concurrency"`
    ProgressIntervalSeconds  int    `mapstructure:"progress_interval_seconds"`
}

type MediaConfig struct {
    MediaPath string `mapstructure:"media_path"`
}

type TranscodingConfig struct {
    FFmpegPath      string `mapstructure:"ffmpeg_path"`
    FFprobePath     string `mapstructure:"ffprobe_path"`
    TimeoutSeconds  int    `mapstructure:"timeout_seconds"`
}

type LoggingConfig struct {
    Level  string `mapstructure:"level"`
    Format string `mapstructure:"format"`
}
```

---

## API Specification

### Authentication

All worker API endpoints require authentication via Bearer token:

```
Authorization: Bearer <auth_token>
```

The token must match `worker.auth_token` in server configuration.

Middleware implementation:

```go
func WorkerAuthMiddleware(cfg *config.WorkerConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        if !cfg.Enabled {
            c.JSON(http.StatusNotFound, gin.H{"error": "Worker API not enabled"})
            c.Abort()
            return
        }

        authHeader := c.GetHeader("Authorization")
        if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid Authorization header"})
            c.Abort()
            return
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")
        if token != cfg.AuthToken {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid auth token"})
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### Endpoints

#### POST /api/v1/worker/register

Register a worker with the server. Called once when worker starts.

**Request:**
```json
{
  "worker_id": "uuid-string",
  "name": "transcoder-1"
}
```

**Response (200 OK):**
```json
{
  "registered": true,
  "message": "Worker registered successfully"
}
```

**Response (401 Unauthorized):**
```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing auth token"
}
```

---

#### POST /api/v1/worker/heartbeat

Send heartbeat to indicate worker is alive. Called periodically when idle (not actively processing jobs). Also returns server status for informational purposes.

**Request:**
```json
{
  "worker_id": "uuid-string",
  "name": "transcoder-1",
  "active_jobs": 1,
  "capacity": 2
}
```

**Response (200 OK):**
```json
{
  "ok": true,
  "server_time": 1704067200,
  "queued_jobs": 5
}
```

---

#### POST /api/v1/worker/jobs/claim

Claim the next available transcode job. Workers poll this endpoint when they have capacity for more jobs.

**Request:**
```json
{
  "worker_id": "uuid-string"
}
```

**Response (200 OK) - Video job available:**
```json
{
  "job": {
    "job_id": 123,
    "transcode_id": "abc-123",
    "track_type": "video",
    "track_index": 0,
    "quality": "720p",
    "input_path": "/media/uuid-456/movie.mkv",
    "output_path": "/media/uuid-456/transcoded/720p/video",
    "media_duration": 7200.5,
    "transcode_options": {
      "width": 1280,
      "height": 720,
      "video_codec": "libx264",
      "crf": 23,
      "max_bitrate": 4000,
      "preset": "medium",
      "segment_time": 4
    }
  }
}
```

**Response (200 OK) - Audio job available:**
```json
{
  "job": {
    "job_id": 124,
    "transcode_id": "abc-124",
    "track_type": "audio",
    "track_index": 1,
    "quality": "",
    "input_path": "/media/uuid-456/movie.mkv",
    "output_path": "/media/uuid-456/transcoded/audio-1",
    "media_duration": 7200.5,
    "transcode_options": {
      "audio_codec": "aac",
      "audio_bitrate": 128,
      "segment_time": 4
    }
  }
}
```

**Response (200 OK) - Subtitle job available:**
```json
{
  "job": {
    "job_id": 125,
    "transcode_id": "abc-125",
    "track_type": "subtitle",
    "track_index": 0,
    "quality": "",
    "input_path": "/media/uuid-456/movie.mkv",
    "output_path": "/media/uuid-456/transcoded/subtitle-0.vtt",
    "media_duration": 7200.5,
    "transcode_options": {}
  }
}
```

**Response (200 OK) - No jobs available:**
```json
{
  "job": null
}
```

---

#### POST /api/v1/worker/jobs/:id/progress

Report progress on an active job. This also serves as an implicit heartbeat - the worker's `LastSeen` timestamp is updated.

**Request:**
```json
{
  "worker_id": "uuid-string",
  "percentage": 45.5,
  "speed": "1.2x",
  "eta_seconds": 3600
}
```

**Response (200 OK):**
```json
{
  "ok": true
}
```

**Response (404 Not Found):**
```json
{
  "error": "JOB_NOT_FOUND",
  "message": "Job not found or not owned by this worker"
}
```

---

#### POST /api/v1/worker/jobs/:id/complete

Mark a job as successfully completed. Server validates output exists before accepting.

**Request:**
```json
{
  "worker_id": "uuid-string",
  "segment_count": 42,
  "output_files": [
    "init.mp4",
    "chunk-000.m4s",
    "chunk-001.m4s"
  ]
}
```

**Response (200 OK):**
```json
{
  "ok": true,
  "message": "Job completed successfully"
}
```

**Response (400 Bad Request) - Validation failed:**
```json
{
  "error": "VALIDATION_FAILED",
  "message": "Output validation failed: init.mp4 not found"
}
```

---

#### POST /api/v1/worker/jobs/:id/fail

Mark a job as failed. Can optionally request retry.

**Request:**
```json
{
  "worker_id": "uuid-string",
  "error": "FFmpeg exited with code 1: No such file or directory",
  "retry": true
}
```

**Response (200 OK):**
```json
{
  "ok": true,
  "message": "Job marked as failed, will retry"
}
```

---

## Domain Types

### Worker State (In-Memory)

```go
// internal/domain/worker.go

// WorkerState represents a registered worker's current state
type WorkerState struct {
    ID           string
    Name         string
    LastSeen     time.Time
    ActiveJobs   int
    Capacity     int
    RegisteredAt time.Time
}

// IsAlive checks if the worker has sent a heartbeat/progress within the timeout
func (w *WorkerState) IsAlive(timeout time.Duration) bool {
    return time.Since(w.LastSeen) < timeout
}
```

### Worker Job (API Transfer Object)

```go
// internal/adapters/http/worker_types.go

// WorkerJob contains all information a worker needs to process a transcode job
type WorkerJob struct {
    JobID            int64                  `json:"job_id"`
    TranscodeID      string                 `json:"transcode_id"`
    TrackType        string                 `json:"track_type"`
    TrackIndex       int                    `json:"track_index"`
    Quality          string                 `json:"quality,omitempty"`
    InputPath        string                 `json:"input_path"`
    OutputPath       string                 `json:"output_path"`
    MediaDuration    float64                `json:"media_duration"`
    TranscodeOptions WorkerTranscodeOptions `json:"transcode_options"`
}

// WorkerTranscodeOptions contains FFmpeg parameters for transcoding
type WorkerTranscodeOptions struct {
    // Video options
    Width      int    `json:"width,omitempty"`
    Height     int    `json:"height,omitempty"`
    VideoCodec string `json:"video_codec,omitempty"`
    CRF        int    `json:"crf,omitempty"`
    MaxBitrate int    `json:"max_bitrate,omitempty"`
    Preset     string `json:"preset,omitempty"`

    // Audio options
    AudioCodec   string `json:"audio_codec,omitempty"`
    AudioBitrate int    `json:"audio_bitrate,omitempty"`

    // Segmentation
    SegmentTime int `json:"segment_time"`
}
```

---

## Worker Registry

The worker registry is an in-memory store that tracks active workers.

```go
// internal/adapters/worker/registry.go

type WorkerRegistry struct {
    mu      sync.RWMutex
    workers map[string]*domain.WorkerState
    timeout time.Duration
}

func NewWorkerRegistry(timeout time.Duration) *WorkerRegistry {
    return &WorkerRegistry{
        workers: make(map[string]*domain.WorkerState),
        timeout: timeout,
    }
}

// Register adds or updates a worker in the registry
func (r *WorkerRegistry) Register(workerID, name string, capacity int) {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    if w, ok := r.workers[workerID]; ok {
        w.Name = name
        w.LastSeen = now
        w.Capacity = capacity
    } else {
        r.workers[workerID] = &domain.WorkerState{
            ID:           workerID,
            Name:         name,
            LastSeen:     now,
            Capacity:     capacity,
            RegisteredAt: now,
        }
    }
}

// Touch updates the LastSeen timestamp for a worker
// Called on heartbeat and progress reports
func (r *WorkerRegistry) Touch(workerID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    if w, ok := r.workers[workerID]; ok {
        w.LastSeen = time.Now()
        return true
    }
    return false
}

// SetActiveJobs updates the active job count for a worker
func (r *WorkerRegistry) SetActiveJobs(workerID string, count int) {
    r.mu.Lock()
    defer r.mu.Unlock()

    if w, ok := r.workers[workerID]; ok {
        w.ActiveJobs = count
    }
}

// HasAliveWorkers returns true if any worker has been seen within the timeout
func (r *WorkerRegistry) HasAliveWorkers() bool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    for _, w := range r.workers {
        if w.IsAlive(r.timeout) {
            return true
        }
    }
    return false
}

// AliveWorkerCount returns the number of alive workers
func (r *WorkerRegistry) AliveWorkerCount() int {
    r.mu.RLock()
    defer r.mu.RUnlock()

    count := 0
    for _, w := range r.workers {
        if w.IsAlive(r.timeout) {
            count++
        }
    }
    return count
}

// GetWorker returns a worker by ID, or nil if not found
func (r *WorkerRegistry) GetWorker(workerID string) *domain.WorkerState {
    r.mu.RLock()
    defer r.mu.RUnlock()

    if w, ok := r.workers[workerID]; ok {
        // Return a copy to avoid race conditions
        copy := *w
        return &copy
    }
    return nil
}

// ListAliveWorkers returns all workers that are currently alive
func (r *WorkerRegistry) ListAliveWorkers() []*domain.WorkerState {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var result []*domain.WorkerState
    for _, w := range r.workers {
        if w.IsAlive(r.timeout) {
            copy := *w
            result = append(result, &copy)
        }
    }
    return result
}

// Cleanup removes workers that haven't been seen for a long time (e.g., 24h)
// Should be called periodically to prevent memory leaks
func (r *WorkerRegistry) Cleanup(maxAge time.Duration) int {
    r.mu.Lock()
    defer r.mu.Unlock()

    removed := 0
    for id, w := range r.workers {
        if time.Since(w.LastSeen) > maxAge {
            delete(r.workers, id)
            removed++
        }
    }
    return removed
}
```

---

## Job Processing Flow

### When Workers Are Enabled

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Job Created                                    │
│                      (transcode_video type)                              │
└─────────────────────────────────────┬───────────────────────────────────┘
                                      │
                                      ▼
                          ┌───────────────────────┐
                          │ worker.enabled = true? │
                          └───────────┬───────────┘
                                      │
                    ┌─────────────────┴─────────────────┐
                    │ NO                                │ YES
                    ▼                                   ▼
          ┌─────────────────┐               ┌─────────────────────┐
          │ Process locally │               │ Job waits in queue  │
          │ (current flow)  │               │ for worker claim    │
          └─────────────────┘               └──────────┬──────────┘
                                                       │
                                                       ▼
                                           ┌───────────────────────┐
                                           │ Worker claims job     │
                                           │ via /jobs/claim       │
                                           └───────────┬───────────┘
                                                       │
                                                       ▼
                                           ┌───────────────────────┐
                                           │ Worker processes job  │
                                           │ (FFmpeg transcoding)  │
                                           └───────────┬───────────┘
                                                       │
                                            ┌──────────┴──────────┐
                                            │                     │
                                            ▼                     ▼
                                   ┌─────────────────┐   ┌─────────────────┐
                                   │ POST /complete  │   │ POST /fail      │
                                   │ (validate output)│   │ (retry or dead) │
                                   └─────────────────┘   └─────────────────┘
```

### Local Fallback (Optional)

When `worker.fallback_to_local: true` and no workers are available:

```go
func (uc *ProcessNextJobUseCase) shouldSkipForWorkers(job *domain.Job) bool {
    // Not a worker-managed job type
    if job.Type != shared.JobTypeTranscodeVideo {
        return false
    }

    // Workers not enabled
    if !uc.config.Worker.Enabled {
        return false
    }

    // Workers are available - let them handle it
    if uc.workerRegistry.HasAliveWorkers() {
        return true
    }

    // No workers available - check fallback settings
    if !uc.config.Worker.FallbackToLocal {
        // No fallback - skip this job, log warning
        logger.Warn().
            Int64("job_id", job.ID).
            Msg("No workers available and fallback disabled, job waiting")
        return true
    }

    // Fallback enabled - check if job has waited long enough
    waitTime := time.Since(job.CreatedAt)
    fallbackDelay := time.Duration(uc.config.Worker.FallbackAfterMinutes) * time.Minute

    if waitTime < fallbackDelay {
        return true // Still waiting for workers
    }

    // Fallback: process locally
    logger.Info().
        Int64("job_id", job.ID).
        Dur("wait_time", waitTime).
        Msg("No workers available, falling back to local processing")
    return false
}
```

### Job Claiming for Workers

When a worker claims a job via `/api/v1/worker/jobs/claim`:

```go
func (uc *ClaimJobForWorkerUseCase) Execute(ctx context.Context, workerID string) (*WorkerJob, error) {
    // 1. Verify worker is registered
    if !uc.workerRegistry.Touch(workerID) {
        return nil, fmt.Errorf("worker not registered: %s", workerID)
    }

    // 2. Claim next transcode job atomically
    // Use a modified query that only selects transcode_video jobs
    job, ok, err := uc.jobRepository.ClaimNextTranscodeJob(ctx, workerID)
    if err != nil {
        return nil, err
    }
    if !ok {
        return nil, nil // No jobs available
    }

    // 3. Parse job payload to get transcode ID
    var payload TranscodeJobPayload
    if err := json.Unmarshal(job.Payload, &payload); err != nil {
        // Release job and return error
        uc.jobRepository.Reschedule(ctx, job.ID, time.Now(), "invalid payload")
        return nil, fmt.Errorf("invalid job payload: %w", err)
    }

    // 4. Fetch transcode record
    transcode, err := uc.transcodeRepo.Get(ctx, payload.TranscodeID)
    if err != nil {
        uc.jobRepository.Reschedule(ctx, job.ID, time.Now(), err.Error())
        return nil, fmt.Errorf("transcode not found: %w", err)
    }

    // 5. Fetch media file
    media, err := uc.mediaRepo.Get(ctx, transcode.MediaID)
    if err != nil {
        uc.jobRepository.Reschedule(ctx, job.ID, time.Now(), err.Error())
        return nil, fmt.Errorf("media not found: %w", err)
    }

    // 6. Build transcode options based on track type and quality
    opts, err := uc.buildTranscodeOptions(transcode, media)
    if err != nil {
        uc.jobRepository.Reschedule(ctx, job.ID, time.Now(), err.Error())
        return nil, err
    }

    // 7. Mark transcode as processing
    if err := uc.transcodeRepo.MarkProcessing(ctx, transcode.ID); err != nil {
        logger.Warn().Err(err).Msg("failed to mark transcode as processing")
    }

    // 8. Notify job started
    uc.jobNotifier.NotifyJobStarted(job)

    // 9. Return complete WorkerJob
    return &WorkerJob{
        JobID:            job.ID,
        TranscodeID:      transcode.ID,
        TrackType:        string(transcode.TrackType),
        TrackIndex:       transcode.TrackIndex,
        Quality:          transcode.Quality,
        InputPath:        media.FilePath,
        OutputPath:       uc.buildOutputPath(media, transcode),
        MediaDuration:    media.Duration,
        TranscodeOptions: opts,
    }, nil
}
```

---

## Output Validation

When a worker reports job completion, the server validates the output exists:

```go
// internal/usecase/worker/complete_job.go

func (uc *CompleteWorkerJobUseCase) validateOutput(
    outputPath string,
    trackType domain.TrackType,
) error {
    switch trackType {
    case domain.TrackTypeVideo, domain.TrackTypeAudio:
        // Check init.mp4 exists
        initPath := filepath.Join(outputPath, "init.mp4")
        if _, err := os.Stat(initPath); os.IsNotExist(err) {
            return fmt.Errorf("init.mp4 not found at %s", initPath)
        }

        // Check at least one segment exists
        pattern := filepath.Join(outputPath, "chunk-*.m4s")
        segments, err := filepath.Glob(pattern)
        if err != nil {
            return fmt.Errorf("failed to glob segments: %w", err)
        }
        if len(segments) == 0 {
            return fmt.Errorf("no segment files found matching %s", pattern)
        }

    case domain.TrackTypeSubtitle:
        // outputPath is the .vtt file itself
        if _, err := os.Stat(outputPath); os.IsNotExist(err) {
            return fmt.Errorf("subtitle file not found at %s", outputPath)
        }
    }

    return nil
}
```

After validation passes:

```go
func (uc *CompleteWorkerJobUseCase) Execute(ctx context.Context, input CompleteJobInput) error {
    // 1. Get job and verify ownership
    job, err := uc.jobRepo.Get(ctx, input.JobID)
    if err != nil {
        return fmt.Errorf("job not found: %w", err)
    }
    if job.WorkerID.String != input.WorkerID {
        return fmt.Errorf("job not owned by worker")
    }

    // 2. Parse payload to get transcode ID
    var payload TranscodeJobPayload
    json.Unmarshal(job.Payload, &payload)

    // 3. Get transcode record
    transcode, _ := uc.transcodeRepo.Get(ctx, payload.TranscodeID)

    // 4. Validate output
    if err := uc.validateOutput(transcode.OutputPath, transcode.TrackType); err != nil {
        return fmt.Errorf("output validation failed: %w", err)
    }

    // 5. For video/audio: probe segment durations and save segments.json
    if transcode.TrackType == domain.TrackTypeVideo || transcode.TrackType == domain.TrackTypeAudio {
        segments, err := uc.transcoder.ProbeSegmentDurations(ctx, transcode.OutputPath)
        if err != nil {
            logger.Warn().Err(err).Msg("failed to probe segment durations")
        } else {
            uc.saveSegmentsJSON(transcode.OutputPath, segments)
        }
    }

    // 6. Mark transcode as completed
    uc.transcodeRepo.MarkCompleted(ctx, transcode.ID)

    // 7. Mark job as succeeded
    uc.jobRepo.MarkSuccess(ctx, job.ID)

    // 8. Notify via WebSocket
    uc.jobNotifier.NotifyJobCompleted(job)

    return nil
}
```

---

## Worker Binary Implementation

### Entry Point

```go
// cmd/worker/main.go

func main() {
    // Parse flags
    configPath := flag.String("config", "", "Path to config file")
    flag.Parse()

    // Load configuration
    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize logger
    logger.Init(cfg.Logging.Level, cfg.Logging.Format)

    // Validate FFmpeg is available
    transcoder := transcoding.NewFFmpegTranscoder(cfg.Transcoding)
    if err := transcoder.IsAvailable(); err != nil {
        log.Fatalf("FFmpeg not available: %v", err)
    }

    // Generate worker ID if not set
    workerID := cfg.Worker.ID
    if workerID == "" {
        workerID = uuid.New().String()
        logger.Info().Str("worker_id", workerID).Msg("Generated worker ID")
    }

    // Create server client
    client := client.NewServerClient(cfg.Server.URL, cfg.Server.AuthToken)

    // Create worker
    worker := worker.New(workerID, cfg, client, transcoder)

    // Setup signal handling
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    // Start worker
    if err := worker.Start(ctx); err != nil {
        log.Fatalf("Worker failed: %v", err)
    }

    logger.Info().Msg("Worker shutdown complete")
}
```

### Worker Core

```go
// internal/worker/worker.go

type Worker struct {
    id          string
    name        string
    config      *config.Config
    client      *client.ServerClient
    transcoder  *transcoding.FFmpegTranscoder

    activeJobs  int32          // atomic counter
    jobs        chan *WorkerJob
    wg          sync.WaitGroup
}

func New(id string, cfg *config.Config, client *client.ServerClient, transcoder *transcoding.FFmpegTranscoder) *Worker {
    return &Worker{
        id:         id,
        name:       cfg.Worker.Name,
        config:     cfg,
        client:     client,
        transcoder: transcoder,
        jobs:       make(chan *WorkerJob, cfg.Worker.Concurrency),
    }
}

func (w *Worker) Start(ctx context.Context) error {
    // Register with server
    if err := w.client.Register(ctx, w.id, w.name); err != nil {
        return fmt.Errorf("failed to register: %w", err)
    }
    logger.Info().Str("worker_id", w.id).Str("name", w.name).Msg("Registered with server")

    // Start worker goroutines
    for i := 0; i < w.config.Worker.Concurrency; i++ {
        w.wg.Add(1)
        go w.processLoop(ctx, i)
    }

    // Start poll loop
    w.wg.Add(1)
    go w.pollLoop(ctx)

    // Start heartbeat loop
    w.wg.Add(1)
    go w.heartbeatLoop(ctx)

    // Wait for shutdown
    <-ctx.Done()
    logger.Info().Msg("Shutdown signal received, waiting for jobs to complete...")

    // Close jobs channel to signal workers to stop
    close(w.jobs)

    // Wait for all goroutines
    w.wg.Wait()

    return nil
}

func (w *Worker) pollLoop(ctx context.Context) {
    defer w.wg.Done()

    ticker := time.NewTicker(time.Duration(w.config.Worker.PollIntervalSeconds) * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Only poll if we have capacity
            active := atomic.LoadInt32(&w.activeJobs)
            if int(active) >= w.config.Worker.Concurrency {
                continue
            }

            job, err := w.client.ClaimJob(ctx, w.id)
            if err != nil {
                logger.Error().Err(err).Msg("Failed to claim job")
                continue
            }
            if job == nil {
                continue // No jobs available
            }

            logger.Info().
                Int64("job_id", job.JobID).
                Str("track_type", job.TrackType).
                Str("quality", job.Quality).
                Msg("Claimed job")

            // Send to worker
            select {
            case w.jobs <- job:
                atomic.AddInt32(&w.activeJobs, 1)
            default:
                // Channel full, shouldn't happen due to capacity check
                logger.Warn().Msg("Job channel full, releasing job")
                w.client.FailJob(ctx, job.JobID, w.id, "worker channel full", true)
            }
        }
    }
}

func (w *Worker) processLoop(ctx context.Context, workerNum int) {
    defer w.wg.Done()

    for job := range w.jobs {
        w.processJob(ctx, job, workerNum)
        atomic.AddInt32(&w.activeJobs, -1)
    }
}

func (w *Worker) processJob(ctx context.Context, job *WorkerJob, workerNum int) {
    logger := logger.With().
        Int64("job_id", job.JobID).
        Str("track_type", job.TrackType).
        Int("worker_num", workerNum).
        Logger()

    logger.Info().Msg("Processing job")

    // Create progress callback
    lastProgress := time.Now()
    progressInterval := time.Duration(w.config.Worker.ProgressIntervalSeconds) * time.Second

    progressCallback := func(p transcoding.Progress) {
        if time.Since(lastProgress) < progressInterval {
            return
        }
        lastProgress = time.Now()

        if err := w.client.ReportProgress(ctx, job.JobID, w.id, p.Percentage, p.Speed); err != nil {
            logger.Warn().Err(err).Msg("Failed to report progress")
        }
    }

    // Execute transcoding based on track type
    var err error
    switch job.TrackType {
    case "video":
        err = w.transcoder.TranscodeVideo(ctx, job.ToTranscodeOptions(), progressCallback)
    case "audio":
        err = w.transcoder.TranscodeAudio(ctx, job.ToTranscodeOptions(), progressCallback)
    case "subtitle":
        err = w.transcoder.ExtractSubtitle(ctx, job.ToTranscodeOptions())
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

    // Report completion
    if err := w.client.CompleteJob(ctx, job.JobID, w.id); err != nil {
        logger.Error().Err(err).Msg("Failed to report job completion")
        return
    }

    logger.Info().Msg("Job completed successfully")
}

func (w *Worker) heartbeatLoop(ctx context.Context) {
    defer w.wg.Done()

    ticker := time.NewTicker(time.Duration(w.config.Worker.HeartbeatIntervalSeconds) * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            active := int(atomic.LoadInt32(&w.activeJobs))
            resp, err := w.client.Heartbeat(ctx, w.id, w.name, active, w.config.Worker.Concurrency)
            if err != nil {
                logger.Warn().Err(err).Msg("Heartbeat failed")
                continue
            }
            logger.Debug().
                Int("queued_jobs", resp.QueuedJobs).
                Msg("Heartbeat sent")
        }
    }
}
```

### Server Client

```go
// internal/worker/client/client.go

type ServerClient struct {
    baseURL    string
    authToken  string
    httpClient *http.Client
}

func NewServerClient(baseURL, authToken string) *ServerClient {
    return &ServerClient{
        baseURL:   strings.TrimSuffix(baseURL, "/"),
        authToken: authToken,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *ServerClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
    var bodyReader io.Reader
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return fmt.Errorf("failed to marshal request: %w", err)
        }
        bodyReader = bytes.NewReader(jsonBody)
    }

    req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Authorization", "Bearer "+c.authToken)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
    }

    if result != nil {
        if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
            return fmt.Errorf("failed to decode response: %w", err)
        }
    }

    return nil
}

func (c *ServerClient) Register(ctx context.Context, workerID, name string) error {
    req := map[string]string{
        "worker_id": workerID,
        "name":      name,
    }
    return c.doRequest(ctx, "POST", "/api/v1/worker/register", req, nil)
}

func (c *ServerClient) Heartbeat(ctx context.Context, workerID, name string, activeJobs, capacity int) (*HeartbeatResponse, error) {
    req := map[string]interface{}{
        "worker_id":   workerID,
        "name":        name,
        "active_jobs": activeJobs,
        "capacity":    capacity,
    }
    var resp HeartbeatResponse
    err := c.doRequest(ctx, "POST", "/api/v1/worker/heartbeat", req, &resp)
    return &resp, err
}

func (c *ServerClient) ClaimJob(ctx context.Context, workerID string) (*WorkerJob, error) {
    req := map[string]string{
        "worker_id": workerID,
    }
    var resp struct {
        Job *WorkerJob `json:"job"`
    }
    if err := c.doRequest(ctx, "POST", "/api/v1/worker/jobs/claim", req, &resp); err != nil {
        return nil, err
    }
    return resp.Job, nil
}

func (c *ServerClient) ReportProgress(ctx context.Context, jobID int64, workerID string, percentage float64, speed string) error {
    req := map[string]interface{}{
        "worker_id":  workerID,
        "percentage": percentage,
        "speed":      speed,
    }
    path := fmt.Sprintf("/api/v1/worker/jobs/%d/progress", jobID)
    return c.doRequest(ctx, "POST", path, req, nil)
}

func (c *ServerClient) CompleteJob(ctx context.Context, jobID int64, workerID string) error {
    req := map[string]string{
        "worker_id": workerID,
    }
    path := fmt.Sprintf("/api/v1/worker/jobs/%d/complete", jobID)
    return c.doRequest(ctx, "POST", path, req, nil)
}

func (c *ServerClient) FailJob(ctx context.Context, jobID int64, workerID, errMsg string, retry bool) error {
    req := map[string]interface{}{
        "worker_id": workerID,
        "error":     errMsg,
        "retry":     retry,
    }
    path := fmt.Sprintf("/api/v1/worker/jobs/%d/fail", jobID)
    return c.doRequest(ctx, "POST", path, req, nil)
}

type HeartbeatResponse struct {
    OK         bool  `json:"ok"`
    ServerTime int64 `json:"server_time"`
    QueuedJobs int   `json:"queued_jobs"`
}
```

---

## Shared Package: pkg/transcoding

Extract FFmpeg logic from `internal/adapters/media/ffmpeg_transcoder.go` to a shared package.

```
pkg/
└── transcoding/
    ├── transcoder.go      # FFmpegTranscoder implementation
    ├── options.go         # TranscodeOptions struct
    ├── progress.go        # Progress struct and parsing
    └── probe.go           # FFprobe functionality
```

### Key interfaces

```go
// pkg/transcoding/transcoder.go

type Transcoder interface {
    IsAvailable() error
    TranscodeVideo(ctx context.Context, opts Options, callback ProgressCallback) error
    TranscodeAudio(ctx context.Context, opts Options, callback ProgressCallback) error
    ExtractSubtitle(ctx context.Context, opts Options) error
    ProbeSegmentDurations(ctx context.Context, outputPath string) ([]SegmentInfo, error)
}

type Options struct {
    InputPath         string
    OutputPath        string
    SourceStreamIndex int

    // Video
    Width      int
    Height     int
    VideoCodec string
    CRF        int
    MaxBitrate int
    Preset     string

    // Audio
    AudioCodec   string
    AudioBitrate int

    // Segmentation
    SegmentTime int
}

type Progress struct {
    Percentage float64
    Speed      string
    ETA        time.Duration
}

type ProgressCallback func(Progress)

type SegmentInfo struct {
    Number   int
    Duration int64 // milliseconds
}
```

---

## Project Structure Changes

```
cmd/
├── server/
│   └── main.go              # Existing server entry point
└── worker/
    └── main.go              # NEW: Worker entry point

internal/
├── worker/                  # NEW: Worker-specific code (for worker binary)
│   ├── config/
│   │   └── config.go
│   ├── client/
│   │   └── client.go
│   └── worker.go
├── adapters/
│   ├── worker/              # NEW: Server-side worker components
│   │   └── registry.go
│   └── http/
│       ├── worker_handler.go    # NEW: Worker API handlers
│       └── worker_types.go      # NEW: Request/response types
├── usecase/
│   └── worker/              # NEW: Worker API use cases
│       ├── claim_job.go
│       ├── complete_job.go
│       ├── fail_job.go
│       └── report_progress.go
└── domain/
    └── worker.go            # NEW: WorkerState struct

pkg/                         # NEW: Shared packages
└── transcoding/
    ├── transcoder.go
    ├── options.go
    ├── progress.go
    └── probe.go

configs/
├── default.yaml             # Server config (add worker section)
└── worker.yaml              # NEW: Worker config template
```

---

## Build System

Update `Makefile`:

```makefile
.PHONY: build build-server build-worker clean

# Build both binaries
build: build-server build-worker

# Build server binary
build-server:
	go build -o bin/vimesrv ./cmd/server

# Build worker binary
build-worker:
	go build -o bin/vimesrv-worker ./cmd/worker

# Clean build artifacts
clean:
	rm -rf bin/

# Run server
run-server: build-server
	./bin/vimesrv

# Run worker
run-worker: build-worker
	./bin/vimesrv-worker -config configs/worker.yaml
```

---

## Implementation Order

| Phase | Task | Effort | Dependencies |
|-------|------|--------|--------------|
| 1.1 | Add `WorkerConfig` to server configuration | Small | None |
| 1.2 | Create `pkg/transcoding` package (extract FFmpeg code) | Medium | None |
| 1.3 | Add `WorkerState` domain type | Small | None |
| 1.4 | Implement `WorkerRegistry` | Small | 1.3 |
| 1.5 | Implement worker API use cases (claim, complete, fail, progress) | Medium | 1.1, 1.4 |
| 1.6 | Implement worker HTTP handlers with auth middleware | Medium | 1.5 |
| 1.7 | Register worker routes in `register_http_routes.go` | Small | 1.6 |
| 1.8 | Modify `ProcessNextJobUseCase` to skip worker-exclusive jobs | Small | 1.4 |
| 2.1 | Create worker config package | Small | None |
| 2.2 | Implement worker server client | Medium | 1.6 |
| 2.3 | Implement worker core (poll, process, heartbeat loops) | Medium | 1.2, 2.2 |
| 2.4 | Create worker binary entry point | Small | 2.1, 2.3 |
| 3.1 | Add output validation on job completion | Small | 1.5 |
| 3.2 | Update Makefile for dual binary build | Small | 2.4 |
| 3.3 | Create worker.yaml config template | Small | 2.1 |
| 3.4 | Update OpenAPI spec (optional) | Small | 1.6 |
| 3.5 | Add tests | Medium | All |

**Estimated Total:** 2-3 days of focused work

---

## Testing Strategy

### Unit Tests

1. `WorkerRegistry` - test touch, cleanup, alive detection
2. `ClaimJobForWorkerUseCase` - test job claiming, payload building
3. `CompleteWorkerJobUseCase` - test validation, segment probing
4. Worker client - mock HTTP responses

### Integration Tests

1. Full job flow: create job → worker claims → worker completes → job marked succeeded
2. Worker failure: worker claims → worker fails → job retried
3. Worker timeout: worker claims → worker disappears → job reclaimed
4. Output validation: test missing files are detected

### Manual Testing

1. Start server with `worker.enabled: true`
2. Start worker with matching config
3. Import a media file
4. Verify worker claims and processes transcode jobs
5. Verify progress appears in UI
6. Verify playback works after completion

---

## Security Considerations

1. **Auth token:** Must be kept secret; consider environment variable injection
2. **Network:** Worker API should ideally be on internal network only
3. **File paths:** Workers receive file paths from server; ensure paths stay within media_path
4. **Input validation:** Validate all worker requests to prevent injection

---

## Future Enhancements

1. **Worker capabilities:** Allow workers to declare GPU support, specific codecs
2. **Job affinity:** Route specific jobs to specific workers
3. **Priority workers:** Some workers handle high-priority jobs only
4. **Metrics/monitoring:** Prometheus metrics for job processing times, queue depth
5. **Worker dashboard:** UI page showing worker status, active jobs
6. **mTLS authentication:** Replace token auth with mutual TLS for production
