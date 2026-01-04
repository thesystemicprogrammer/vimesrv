package domain

import (
	"testing"
	"time"
)

func TestNewTranscode(t *testing.T) {
	id := "test-123"
	mediaID := "media-456"
	quality := "720p"
	trackType := TrackTypeVideo
	trackIndex := 0

	transcode := NewTranscode(id, mediaID, quality, trackType, trackIndex)

	if transcode.ID != id {
		t.Errorf("expected ID %s, got %s", id, transcode.ID)
	}
	if transcode.MediaID != mediaID {
		t.Errorf("expected MediaID %s, got %s", mediaID, transcode.MediaID)
	}
	if transcode.Quality != quality {
		t.Errorf("expected Quality %s, got %s", quality, transcode.Quality)
	}
	if transcode.TrackType != trackType {
		t.Errorf("expected TrackType %s, got %s", trackType, transcode.TrackType)
	}
	if transcode.TrackIndex != trackIndex {
		t.Errorf("expected TrackIndex %d, got %d", trackIndex, transcode.TrackIndex)
	}
	if transcode.Status != TranscodePending {
		t.Errorf("expected Status %s, got %s", TranscodePending, transcode.Status)
	}
	if transcode.OutputPath != "" {
		t.Errorf("expected empty OutputPath, got %s", transcode.OutputPath)
	}
}

func TestTranscode_MarkProcessing(t *testing.T) {
	transcode := NewTranscode("test-1", "media-1", "360p", TrackTypeVideo, 0)
	before := time.Now()
	outputPath := "/path/to/output"

	transcode.MarkProcessing(outputPath)

	if transcode.Status != TranscodeProcessing {
		t.Errorf("expected Status %s, got %s", TranscodeProcessing, transcode.Status)
	}
	if transcode.OutputPath != outputPath {
		t.Errorf("expected OutputPath %s, got %s", outputPath, transcode.OutputPath)
	}
	if transcode.UpdatedAt.Before(before) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestTranscode_MarkCompleted(t *testing.T) {
	transcode := NewTranscode("test-1", "media-1", "360p", TrackTypeVideo, 0)
	outputPath := "/path/to/output"
	before := time.Now()

	transcode.MarkCompleted(outputPath)

	if transcode.Status != TranscodeCompleted {
		t.Errorf("expected Status %s, got %s", TranscodeCompleted, transcode.Status)
	}
	if transcode.OutputPath != outputPath {
		t.Errorf("expected OutputPath %s, got %s", outputPath, transcode.OutputPath)
	}
	if transcode.UpdatedAt.Before(before) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestTranscode_MarkFailed(t *testing.T) {
	transcode := NewTranscode("test-1", "media-1", "360p", TrackTypeVideo, 0)
	before := time.Now()

	transcode.MarkFailed()

	if transcode.Status != TranscodeFailed {
		t.Errorf("expected Status %s, got %s", TranscodeFailed, transcode.Status)
	}
	if transcode.UpdatedAt.Before(before) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestTranscode_MarkCancelled(t *testing.T) {
	transcode := NewTranscode("test-1", "media-1", "360p", TrackTypeVideo, 0)
	before := time.Now()

	transcode.MarkCancelled()

	if transcode.Status != TranscodeCancelled {
		t.Errorf("expected Status %s, got %s", TranscodeCancelled, transcode.Status)
	}
	if transcode.UpdatedAt.Before(before) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestTranscode_StatusCheckers(t *testing.T) {
	tests := []struct {
		name         string
		status       TranscodeStatus
		isPending    bool
		isProcessing bool
		isCompleted  bool
		isFailed     bool
		isCancelled  bool
	}{
		{
			name:         "pending",
			status:       TranscodePending,
			isPending:    true,
			isProcessing: false,
			isCompleted:  false,
			isFailed:     false,
			isCancelled:  false,
		},
		{
			name:         "processing",
			status:       TranscodeProcessing,
			isPending:    false,
			isProcessing: true,
			isCompleted:  false,
			isFailed:     false,
			isCancelled:  false,
		},
		{
			name:         "completed",
			status:       TranscodeCompleted,
			isPending:    false,
			isProcessing: false,
			isCompleted:  true,
			isFailed:     false,
			isCancelled:  false,
		},
		{
			name:         "failed",
			status:       TranscodeFailed,
			isPending:    false,
			isProcessing: false,
			isCompleted:  false,
			isFailed:     true,
			isCancelled:  false,
		},
		{
			name:         "cancelled",
			status:       TranscodeCancelled,
			isPending:    false,
			isProcessing: false,
			isCompleted:  false,
			isFailed:     false,
			isCancelled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transcode := NewTranscode("test-1", "media-1", "360p", TrackTypeVideo, 0)
			transcode.Status = tt.status

			if got := transcode.IsPending(); got != tt.isPending {
				t.Errorf("IsPending() = %v, want %v", got, tt.isPending)
			}
			if got := transcode.IsProcessing(); got != tt.isProcessing {
				t.Errorf("IsProcessing() = %v, want %v", got, tt.isProcessing)
			}
			if got := transcode.IsCompleted(); got != tt.isCompleted {
				t.Errorf("IsCompleted() = %v, want %v", got, tt.isCompleted)
			}
			if got := transcode.IsFailed(); got != tt.isFailed {
				t.Errorf("IsFailed() = %v, want %v", got, tt.isFailed)
			}
			if got := transcode.IsCancelled(); got != tt.isCancelled {
				t.Errorf("IsCancelled() = %v, want %v", got, tt.isCancelled)
			}
		})
	}
}
