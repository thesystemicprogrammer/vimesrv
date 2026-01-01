package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// TranscodeRepository implements transcode persistence operations
type TranscodeRepository struct {
	db *sql.DB
}

// NewTranscodeRepository creates a new TranscodeRepository instance
func NewTranscodeRepository(db *database.DB) ports.TranscodeRepository {
	return &TranscodeRepository{
		db: db.DB,
	}
}

// Create inserts a new transcode record
func (r *TranscodeRepository) Create(ctx context.Context, transcode *domain.Transcode) error {
	query := `
		INSERT INTO transcodes (
			id, media_id, quality, track_type, track_index, status,
			output_path, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		transcode.ID,
		transcode.MediaID,
		transcode.Quality,
		string(transcode.TrackType),
		transcode.TrackIndex,
		string(transcode.Status),
		transcode.OutputPath,
		transcode.CreatedAt,
		transcode.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert transcode: %w", err)
	}

	return nil
}

// Get retrieves a transcode by its ID
func (r *TranscodeRepository) Get(ctx context.Context, id string) (*domain.Transcode, error) {
	query := `
		SELECT 
			id, media_id, quality, track_type, track_index, status,
			output_path, created_at, updated_at
		FROM transcodes
		WHERE id = ?
	`

	var transcode domain.Transcode
	var trackType, status string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&transcode.ID,
		&transcode.MediaID,
		&transcode.Quality,
		&trackType,
		&transcode.TrackIndex,
		&status,
		&transcode.OutputPath,
		&transcode.CreatedAt,
		&transcode.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transcode not found with id: %s", id)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query transcode: %w", err)
	}

	transcode.TrackType = domain.TrackType(trackType)
	transcode.Status = domain.TranscodeStatus(status)

	return &transcode, nil
}

// GetByMediaID retrieves all transcodes for a media file
func (r *TranscodeRepository) GetByMediaID(ctx context.Context, mediaID string) ([]*domain.Transcode, error) {
	query := `
		SELECT 
			id, media_id, quality, track_type, track_index, status,
			output_path, created_at, updated_at
		FROM transcodes
		WHERE media_id = ?
		ORDER BY quality, track_type, track_index
	`

	rows, err := r.db.QueryContext(ctx, query, mediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transcodes: %w", err)
	}
	defer rows.Close()

	var transcodes []*domain.Transcode
	for rows.Next() {
		var transcode domain.Transcode
		var trackType, status string

		err := rows.Scan(
			&transcode.ID,
			&transcode.MediaID,
			&transcode.Quality,
			&trackType,
			&transcode.TrackIndex,
			&status,
			&transcode.OutputPath,
			&transcode.CreatedAt,
			&transcode.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transcode: %w", err)
		}

		transcode.TrackType = domain.TrackType(trackType)
		transcode.Status = domain.TranscodeStatus(status)
		transcodes = append(transcodes, &transcode)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transcodes: %w", err)
	}

	return transcodes, nil
}

// UpdateStatus updates the status of a transcode
func (r *TranscodeRepository) UpdateStatus(ctx context.Context, id string, status domain.TranscodeStatus) error {
	query := `
		UPDATE transcodes 
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, string(status), id)
	if err != nil {
		return fmt.Errorf("failed to update transcode status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transcode not found with id: %s", id)
	}

	return nil
}

// UpdateProgress is removed - progress tracking is done in the jobs table via payload

// MarkProcessing marks a transcode as currently processing
func (r *TranscodeRepository) MarkProcessing(ctx context.Context, id string) error {
	query := `
		UPDATE transcodes 
		SET status = 'processing', 
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark transcode as processing: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transcode not found with id: %s", id)
	}

	return nil
}

// MarkCompleted marks a transcode as successfully completed
func (r *TranscodeRepository) MarkCompleted(ctx context.Context, id string, outputPath string) error {
	query := `
		UPDATE transcodes 
		SET status = 'completed',
		    output_path = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, outputPath, id)
	if err != nil {
		return fmt.Errorf("failed to mark transcode as completed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transcode not found with id: %s", id)
	}

	return nil
}

// MarkFailed marks a transcode as failed
func (r *TranscodeRepository) MarkFailed(ctx context.Context, id string) error {
	query := `
		UPDATE transcodes 
		SET status = 'failed',
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark transcode as failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transcode not found with id: %s", id)
	}

	return nil
}

// Delete deletes a transcode
func (r *TranscodeRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM transcodes WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete transcode: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transcode not found with id: %s", id)
	}

	return nil
}

// ListPending retrieves pending transcodes with a limit
func (r *TranscodeRepository) ListPending(ctx context.Context, limit int) ([]*domain.Transcode, error) {
	query := `
		SELECT 
			id, media_id, quality, track_type, track_index, status,
			output_path, created_at, updated_at
		FROM transcodes
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending transcodes: %w", err)
	}
	defer rows.Close()

	var transcodes []*domain.Transcode
	for rows.Next() {
		var transcode domain.Transcode
		var trackType, status string

		err := rows.Scan(
			&transcode.ID,
			&transcode.MediaID,
			&transcode.Quality,
			&trackType,
			&transcode.TrackIndex,
			&status,
			&transcode.OutputPath,
			&transcode.CreatedAt,
			&transcode.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transcode: %w", err)
		}

		transcode.TrackType = domain.TrackType(trackType)
		transcode.Status = domain.TranscodeStatus(status)
		transcodes = append(transcodes, &transcode)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transcodes: %w", err)
	}

	return transcodes, nil
}
