package watch_progress

import (
	"context"
	"database/sql"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// RecordWatchProgressUseCase handles recording user watch progress
type RecordWatchProgressUseCase struct {
	repo ports.WatchProgressRepository
}

func NewRecordWatchProgressUseCase(repo ports.WatchProgressRepository) *RecordWatchProgressUseCase {
	return &RecordWatchProgressUseCase{repo: repo}
}

type RecordProgressInput struct {
	UserID          string
	MediaID         *string // For movies
	EpisodeID       *int64  // For episodes
	PositionSeconds int
	DurationSeconds int
}

func (uc *RecordWatchProgressUseCase) Execute(ctx context.Context, input RecordProgressInput) error {
	// Validate input
	if input.UserID == "" {
		return ErrInvalidInput
	}
	if input.MediaID == nil && input.EpisodeID == nil {
		return ErrInvalidInput
	}
	if input.DurationSeconds <= 0 {
		return ErrInvalidDuration
	}
	if input.PositionSeconds < 0 {
		return ErrInvalidPosition
	}

	// Calculate progress percentage
	progressPercent := (float64(input.PositionSeconds) / float64(input.DurationSeconds)) * 100
	if progressPercent > 100 {
		progressPercent = 100
	}

	// Determine completion status (95% threshold)
	completed := progressPercent >= 95.0

	progress := &domain.WatchProgress{
		UserID:          input.UserID,
		PositionSeconds: input.PositionSeconds,
		DurationSeconds: input.DurationSeconds,
		ProgressPercent: progressPercent,
		Completed:       completed,
		LastWatchedAt:   time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if input.MediaID != nil {
		progress.MediaID = sql.NullString{String: *input.MediaID, Valid: true}
	}
	if input.EpisodeID != nil {
		progress.EpisodeMetadataID = sql.NullInt64{Int64: *input.EpisodeID, Valid: true}
	}

	return uc.repo.SaveProgress(ctx, progress)
}
