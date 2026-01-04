// Package client provides an HTTP client for communicating with the main vimesrv server.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ServerClient handles HTTP communication with the main vimesrv server
type ServerClient struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewServerClient creates a new ServerClient
func NewServerClient(baseURL, authToken string) *ServerClient {
	return &ServerClient{
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WorkerJob contains all information needed to process a transcode job
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

// HeartbeatResponse is the response from the heartbeat endpoint
type HeartbeatResponse struct {
	OK         bool  `json:"ok"`
	ServerTime int64 `json:"server_time"`
	QueuedJobs int   `json:"queued_jobs"`
}

// doRequest performs an HTTP request with JSON body and parses the response
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

	// Read body for error messages
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Register registers the worker with the server
func (c *ServerClient) Register(ctx context.Context, workerID, name string, capacity int) error {
	req := map[string]interface{}{
		"worker_id": workerID,
		"name":      name,
		"capacity":  capacity,
	}
	return c.doRequest(ctx, "POST", "/api/v1/worker/register", req, nil)
}

// Heartbeat sends a heartbeat to the server
func (c *ServerClient) Heartbeat(ctx context.Context, workerID, name string, activeJobs, capacity int) (*HeartbeatResponse, error) {
	req := map[string]interface{}{
		"worker_id":   workerID,
		"name":        name,
		"active_jobs": activeJobs,
		"capacity":    capacity,
	}
	var resp HeartbeatResponse
	err := c.doRequest(ctx, "POST", "/api/v1/worker/heartbeat", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ClaimJob attempts to claim the next available job
// Returns nil if no jobs are available
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

// ReportProgress reports job progress to the server
func (c *ServerClient) ReportProgress(ctx context.Context, jobID int64, workerID string, percentage float64, speed string, etaSeconds int) error {
	req := map[string]interface{}{
		"worker_id":   workerID,
		"percentage":  percentage,
		"speed":       speed,
		"eta_seconds": etaSeconds,
	}
	path := fmt.Sprintf("/api/v1/worker/jobs/%d/progress", jobID)
	return c.doRequest(ctx, "POST", path, req, nil)
}

// CompleteJob marks a job as successfully completed
func (c *ServerClient) CompleteJob(ctx context.Context, jobID int64, workerID string, segmentCount int, outputFiles []string) error {
	req := map[string]interface{}{
		"worker_id":     workerID,
		"segment_count": segmentCount,
		"output_files":  outputFiles,
	}
	path := fmt.Sprintf("/api/v1/worker/jobs/%d/complete", jobID)
	return c.doRequest(ctx, "POST", path, req, nil)
}

// FailJob marks a job as failed
func (c *ServerClient) FailJob(ctx context.Context, jobID int64, workerID, errMsg string, retry bool) error {
	req := map[string]interface{}{
		"worker_id": workerID,
		"error":     errMsg,
		"retry":     retry,
	}
	path := fmt.Sprintf("/api/v1/worker/jobs/%d/fail", jobID)
	return c.doRequest(ctx, "POST", path, req, nil)
}
