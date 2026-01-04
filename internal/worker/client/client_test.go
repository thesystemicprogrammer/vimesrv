package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServerClient(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		authToken string
		wantURL   string
	}{
		{
			name:      "without trailing slash",
			baseURL:   "http://localhost:8080",
			authToken: "test-token",
			wantURL:   "http://localhost:8080",
		},
		{
			name:      "with trailing slash",
			baseURL:   "http://localhost:8080/",
			authToken: "test-token",
			wantURL:   "http://localhost:8080",
		},
		{
			name:      "with path and trailing slash",
			baseURL:   "http://localhost:8080/api/",
			authToken: "secret",
			wantURL:   "http://localhost:8080/api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewServerClient(tt.baseURL, tt.authToken)
			if client.baseURL != tt.wantURL {
				t.Errorf("baseURL = %q, want %q", client.baseURL, tt.wantURL)
			}
			if client.authToken != tt.authToken {
				t.Errorf("authToken = %q, want %q", client.authToken, tt.authToken)
			}
			if client.httpClient == nil {
				t.Error("httpClient should not be nil")
			}
		})
	}
}

func TestServerClient_Register(t *testing.T) {
	tests := []struct {
		name       string
		workerID   string
		workerName string
		capacity   int
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful registration",
			workerID:   "worker-1",
			workerName: "Test Worker",
			capacity:   4,
			statusCode: http.StatusOK,
			response:   `{"registered": true}`,
			wantErr:    false,
		},
		{
			name:       "server error",
			workerID:   "worker-1",
			workerName: "Test Worker",
			capacity:   4,
			statusCode: http.StatusInternalServerError,
			response:   `{"error": "internal error"}`,
			wantErr:    true,
		},
		{
			name:       "unauthorized",
			workerID:   "worker-1",
			workerName: "Test Worker",
			capacity:   4,
			statusCode: http.StatusUnauthorized,
			response:   `{"error": "invalid token"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/worker/register" {
					t.Errorf("expected /api/v1/worker/register, got %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("expected Bearer test-token, got %s", r.Header.Get("Authorization"))
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Verify body
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("failed to decode body: %v", err)
				}
				if body["worker_id"] != tt.workerID {
					t.Errorf("expected worker_id %s, got %v", tt.workerID, body["worker_id"])
				}
				if body["name"] != tt.workerName {
					t.Errorf("expected name %s, got %v", tt.workerName, body["name"])
				}
				if int(body["capacity"].(float64)) != tt.capacity {
					t.Errorf("expected capacity %d, got %v", tt.capacity, body["capacity"])
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewServerClient(server.URL, "test-token")
			err := client.Register(context.Background(), tt.workerID, tt.workerName, tt.capacity)

			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerClient_Heartbeat(t *testing.T) {
	tests := []struct {
		name       string
		workerID   string
		workerName string
		activeJobs int
		capacity   int
		statusCode int
		response   string
		wantErr    bool
		wantOK     bool
		wantQueued int
	}{
		{
			name:       "successful heartbeat",
			workerID:   "worker-1",
			workerName: "Test Worker",
			activeJobs: 2,
			capacity:   4,
			statusCode: http.StatusOK,
			response:   `{"ok": true, "server_time": 1704326400, "queued_jobs": 5}`,
			wantErr:    false,
			wantOK:     true,
			wantQueued: 5,
		},
		{
			name:       "server error",
			workerID:   "worker-1",
			workerName: "Test Worker",
			activeJobs: 0,
			capacity:   4,
			statusCode: http.StatusInternalServerError,
			response:   `{"error": "internal error"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/worker/heartbeat" {
					t.Errorf("expected /api/v1/worker/heartbeat, got %s", r.URL.Path)
				}

				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("failed to decode body: %v", err)
				}
				if body["worker_id"] != tt.workerID {
					t.Errorf("expected worker_id %s, got %v", tt.workerID, body["worker_id"])
				}
				if int(body["active_jobs"].(float64)) != tt.activeJobs {
					t.Errorf("expected active_jobs %d, got %v", tt.activeJobs, body["active_jobs"])
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewServerClient(server.URL, "test-token")
			resp, err := client.Heartbeat(context.Background(), tt.workerID, tt.workerName, tt.activeJobs, tt.capacity)

			if (err != nil) != tt.wantErr {
				t.Errorf("Heartbeat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp.OK != tt.wantOK {
					t.Errorf("Heartbeat() OK = %v, want %v", resp.OK, tt.wantOK)
				}
				if resp.QueuedJobs != tt.wantQueued {
					t.Errorf("Heartbeat() QueuedJobs = %d, want %d", resp.QueuedJobs, tt.wantQueued)
				}
			}
		})
	}
}

func TestServerClient_ClaimJob(t *testing.T) {
	tests := []struct {
		name       string
		workerID   string
		statusCode int
		response   string
		wantErr    bool
		wantNil    bool
		wantJobID  int64
	}{
		{
			name:       "job available",
			workerID:   "worker-1",
			statusCode: http.StatusOK,
			response: `{
				"job": {
					"job_id": 123,
					"transcode_id": "abc123",
					"track_type": "video",
					"track_index": 0,
					"quality": "1080p",
					"input_path": "/media/movie.mkv",
					"output_path": "/transcodes/abc123/1080p/video",
					"media_duration": 7200.5,
					"transcode_options": {
						"width": 1920,
						"height": 1080,
						"video_codec": "libx264",
						"crf": 23,
						"preset": "medium",
						"segment_time": 4
					}
				}
			}`,
			wantErr:   false,
			wantNil:   false,
			wantJobID: 123,
		},
		{
			name:       "no jobs available",
			workerID:   "worker-1",
			statusCode: http.StatusOK,
			response:   `{"job": null}`,
			wantErr:    false,
			wantNil:    true,
		},
		{
			name:       "server error",
			workerID:   "worker-1",
			statusCode: http.StatusInternalServerError,
			response:   `{"error": "internal error"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/worker/jobs/claim" {
					t.Errorf("expected /api/v1/worker/jobs/claim, got %s", r.URL.Path)
				}

				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("failed to decode body: %v", err)
				}
				if body["worker_id"] != tt.workerID {
					t.Errorf("expected worker_id %s, got %v", tt.workerID, body["worker_id"])
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewServerClient(server.URL, "test-token")
			job, err := client.ClaimJob(context.Background(), tt.workerID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ClaimJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.wantNil && job != nil {
					t.Errorf("ClaimJob() job should be nil")
				}
				if !tt.wantNil {
					if job == nil {
						t.Errorf("ClaimJob() job should not be nil")
						return
					}
					if job.JobID != tt.wantJobID {
						t.Errorf("ClaimJob() JobID = %d, want %d", job.JobID, tt.wantJobID)
					}
					if job.TrackType != "video" {
						t.Errorf("ClaimJob() TrackType = %s, want video", job.TrackType)
					}
					if job.TranscodeOptions.Width != 1920 {
						t.Errorf("ClaimJob() Width = %d, want 1920", job.TranscodeOptions.Width)
					}
				}
			}
		})
	}
}

func TestServerClient_ReportProgress(t *testing.T) {
	tests := []struct {
		name       string
		jobID      int64
		workerID   string
		percentage float64
		speed      string
		etaSeconds int
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "successful progress report",
			jobID:      123,
			workerID:   "worker-1",
			percentage: 45.5,
			speed:      "2.5x",
			etaSeconds: 300,
			statusCode: http.StatusOK,
			response:   `{"ok": true}`,
			wantErr:    false,
		},
		{
			name:       "job not found",
			jobID:      999,
			workerID:   "worker-1",
			percentage: 50.0,
			speed:      "1.0x",
			etaSeconds: 100,
			statusCode: http.StatusNotFound,
			response:   `{"error": "job not found"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				expectedPath := "/api/v1/worker/jobs/123/progress"
				if tt.jobID == 999 {
					expectedPath = "/api/v1/worker/jobs/999/progress"
				}
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s, got %s", expectedPath, r.URL.Path)
				}

				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("failed to decode body: %v", err)
				}
				if body["worker_id"] != tt.workerID {
					t.Errorf("expected worker_id %s, got %v", tt.workerID, body["worker_id"])
				}
				if body["percentage"].(float64) != tt.percentage {
					t.Errorf("expected percentage %f, got %v", tt.percentage, body["percentage"])
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewServerClient(server.URL, "test-token")
			err := client.ReportProgress(context.Background(), tt.jobID, tt.workerID, tt.percentage, tt.speed, tt.etaSeconds)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportProgress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerClient_CompleteJob(t *testing.T) {
	tests := []struct {
		name         string
		jobID        int64
		workerID     string
		segmentCount int
		outputFiles  []string
		statusCode   int
		response     string
		wantErr      bool
	}{
		{
			name:         "successful completion",
			jobID:        123,
			workerID:     "worker-1",
			segmentCount: 50,
			outputFiles:  []string{"init.mp4", "chunk-001.m4s", "chunk-002.m4s"},
			statusCode:   http.StatusOK,
			response:     `{"ok": true}`,
			wantErr:      false,
		},
		{
			name:         "validation failed",
			jobID:        123,
			workerID:     "worker-1",
			segmentCount: 0,
			outputFiles:  []string{},
			statusCode:   http.StatusBadRequest,
			response:     `{"error": "validation failed"}`,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/worker/jobs/123/complete" {
					t.Errorf("expected /api/v1/worker/jobs/123/complete, got %s", r.URL.Path)
				}

				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("failed to decode body: %v", err)
				}
				if body["worker_id"] != tt.workerID {
					t.Errorf("expected worker_id %s, got %v", tt.workerID, body["worker_id"])
				}
				if int(body["segment_count"].(float64)) != tt.segmentCount {
					t.Errorf("expected segment_count %d, got %v", tt.segmentCount, body["segment_count"])
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewServerClient(server.URL, "test-token")
			err := client.CompleteJob(context.Background(), tt.jobID, tt.workerID, tt.segmentCount, tt.outputFiles)

			if (err != nil) != tt.wantErr {
				t.Errorf("CompleteJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerClient_FailJob(t *testing.T) {
	tests := []struct {
		name       string
		jobID      int64
		workerID   string
		errMsg     string
		retry      bool
		statusCode int
		response   string
		wantErr    bool
	}{
		{
			name:       "fail with retry",
			jobID:      123,
			workerID:   "worker-1",
			errMsg:     "transcoding failed: codec error",
			retry:      true,
			statusCode: http.StatusOK,
			response:   `{"ok": true}`,
			wantErr:    false,
		},
		{
			name:       "fail without retry",
			jobID:      123,
			workerID:   "worker-1",
			errMsg:     "permanent failure",
			retry:      false,
			statusCode: http.StatusOK,
			response:   `{"ok": true}`,
			wantErr:    false,
		},
		{
			name:       "server error",
			jobID:      123,
			workerID:   "worker-1",
			errMsg:     "some error",
			retry:      true,
			statusCode: http.StatusInternalServerError,
			response:   `{"error": "internal error"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/worker/jobs/123/fail" {
					t.Errorf("expected /api/v1/worker/jobs/123/fail, got %s", r.URL.Path)
				}

				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("failed to decode body: %v", err)
				}
				if body["worker_id"] != tt.workerID {
					t.Errorf("expected worker_id %s, got %v", tt.workerID, body["worker_id"])
				}
				if body["error"] != tt.errMsg {
					t.Errorf("expected error %s, got %v", tt.errMsg, body["error"])
				}
				if body["retry"].(bool) != tt.retry {
					t.Errorf("expected retry %v, got %v", tt.retry, body["retry"])
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewServerClient(server.URL, "test-token")
			err := client.FailJob(context.Background(), tt.jobID, tt.workerID, tt.errMsg, tt.retry)

			if (err != nil) != tt.wantErr {
				t.Errorf("FailJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response - won't actually wait due to context cancellation
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewServerClient(server.URL, "test-token")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.Register(ctx, "worker-1", "test", 4)
	if err == nil {
		t.Error("expected error due to cancelled context")
	}
}
