package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SQLiteMetadataCandidateRepository implements MetadataCandidateRepository using SQLite
type SQLiteMetadataCandidateRepository struct {
	db *sql.DB
}

// NewSQLiteMetadataCandidateRepository creates a new SQLite metadata candidate repository
func NewSQLiteMetadataCandidateRepository(db *sql.DB) *SQLiteMetadataCandidateRepository {
	return &SQLiteMetadataCandidateRepository{db: db}
}

// Create inserts a new metadata candidate
func (r *SQLiteMetadataCandidateRepository) Create(ctx context.Context, candidate *domain.MetadataCandidate) error {
	query := `
		INSERT INTO metadata_candidates (
			media_file_id, candidate_type, tmdb_id, title, release_date,
			overview, poster_path, confidence_score, season_number,
			episode_number, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		candidate.MediaFileID,
		candidate.CandidateType,
		candidate.TMDBID,
		candidate.Title,
		candidate.ReleaseDate,
		candidate.Overview,
		candidate.PosterPath,
		candidate.ConfidenceScore,
		candidate.SeasonNumber,
		candidate.EpisodeNumber,
		candidate.Status,
		now,
	)
	if err != nil {
		return fmt.Errorf("insert metadata candidate: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}

	candidate.ID = id
	candidate.CreatedAt = now

	return nil
}

// Get retrieves a candidate by its ID
func (r *SQLiteMetadataCandidateRepository) Get(ctx context.Context, id int64) (*domain.MetadataCandidate, error) {
	query := `
		SELECT id, media_file_id, candidate_type, tmdb_id, title, release_date,
			   overview, poster_path, confidence_score, season_number,
			   episode_number, status, created_at
		FROM metadata_candidates
		WHERE id = ?
	`

	return r.scanCandidate(ctx, query, id)
}

// ListByMediaFileID retrieves all candidates for a given media file
func (r *SQLiteMetadataCandidateRepository) ListByMediaFileID(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error) {
	query := `
		SELECT id, media_file_id, candidate_type, tmdb_id, title, release_date,
			   overview, poster_path, confidence_score, season_number,
			   episode_number, status, created_at
		FROM metadata_candidates
		WHERE media_file_id = ?
		ORDER BY confidence_score DESC
	`

	return r.scanCandidates(ctx, query, mediaFileID)
}

// ListPendingByMediaFileID retrieves pending candidates for a given media file
func (r *SQLiteMetadataCandidateRepository) ListPendingByMediaFileID(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error) {
	query := `
		SELECT id, media_file_id, candidate_type, tmdb_id, title, release_date,
			   overview, poster_path, confidence_score, season_number,
			   episode_number, status, created_at
		FROM metadata_candidates
		WHERE media_file_id = ? AND status = ?
		ORDER BY confidence_score DESC
	`

	return r.scanCandidates(ctx, query, mediaFileID, domain.CandidateStatusPending)
}

// scanCandidate scans a single candidate from a query
func (r *SQLiteMetadataCandidateRepository) scanCandidate(ctx context.Context, query string, args ...interface{}) (*domain.MetadataCandidate, error) {
	candidate := &domain.MetadataCandidate{}

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&candidate.ID,
		&candidate.MediaFileID,
		&candidate.CandidateType,
		&candidate.TMDBID,
		&candidate.Title,
		&candidate.ReleaseDate,
		&candidate.Overview,
		&candidate.PosterPath,
		&candidate.ConfidenceScore,
		&candidate.SeasonNumber,
		&candidate.EpisodeNumber,
		&candidate.Status,
		&candidate.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, shared.ErrNotFound
		}
		return nil, fmt.Errorf("query metadata candidate: %w", err)
	}

	return candidate, nil
}

// scanCandidates scans multiple candidates from a query
func (r *SQLiteMetadataCandidateRepository) scanCandidates(ctx context.Context, query string, args ...interface{}) ([]domain.MetadataCandidate, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query metadata candidates: %w", err)
	}
	defer rows.Close()

	var candidates []domain.MetadataCandidate
	for rows.Next() {
		var c domain.MetadataCandidate
		if err := rows.Scan(
			&c.ID,
			&c.MediaFileID,
			&c.CandidateType,
			&c.TMDBID,
			&c.Title,
			&c.ReleaseDate,
			&c.Overview,
			&c.PosterPath,
			&c.ConfidenceScore,
			&c.SeasonNumber,
			&c.EpisodeNumber,
			&c.Status,
			&c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan metadata candidate: %w", err)
		}
		candidates = append(candidates, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metadata candidates: %w", err)
	}

	return candidates, nil
}

// Update updates an existing candidate
func (r *SQLiteMetadataCandidateRepository) Update(ctx context.Context, candidate *domain.MetadataCandidate) error {
	query := `
		UPDATE metadata_candidates SET
			media_file_id = ?, candidate_type = ?, tmdb_id = ?, title = ?,
			release_date = ?, overview = ?, poster_path = ?, confidence_score = ?,
			season_number = ?, episode_number = ?, status = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		candidate.MediaFileID,
		candidate.CandidateType,
		candidate.TMDBID,
		candidate.Title,
		candidate.ReleaseDate,
		candidate.Overview,
		candidate.PosterPath,
		candidate.ConfidenceScore,
		candidate.SeasonNumber,
		candidate.EpisodeNumber,
		candidate.Status,
		candidate.ID,
	)
	if err != nil {
		return fmt.Errorf("update metadata candidate: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrNotFound
	}

	return nil
}

// Delete removes a candidate
func (r *SQLiteMetadataCandidateRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM metadata_candidates WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete metadata candidate: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return shared.ErrNotFound
	}

	return nil
}

// DeleteByMediaFileID removes all candidates for a given media file
func (r *SQLiteMetadataCandidateRepository) DeleteByMediaFileID(ctx context.Context, mediaFileID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM metadata_candidates WHERE media_file_id = ?", mediaFileID)
	if err != nil {
		return fmt.Errorf("delete metadata candidates by media file: %w", err)
	}
	return nil
}

// MarkSelected marks a candidate as selected and rejects all others for the same media file
func (r *SQLiteMetadataCandidateRepository) MarkSelected(ctx context.Context, candidateID int64) error {
	// First get the candidate to find the media file ID
	candidate, err := r.Get(ctx, candidateID)
	if err != nil {
		return err
	}

	// Reject all other candidates for this media file
	_, err = r.db.ExecContext(ctx,
		"UPDATE metadata_candidates SET status = ? WHERE media_file_id = ? AND id != ?",
		domain.CandidateStatusRejected,
		candidate.MediaFileID,
		candidateID,
	)
	if err != nil {
		return fmt.Errorf("reject other candidates: %w", err)
	}

	// Mark the selected candidate
	_, err = r.db.ExecContext(ctx,
		"UPDATE metadata_candidates SET status = ? WHERE id = ?",
		domain.CandidateStatusSelected,
		candidateID,
	)
	if err != nil {
		return fmt.Errorf("mark candidate as selected: %w", err)
	}

	return nil
}

// RejectAll marks all candidates for a media file as rejected
func (r *SQLiteMetadataCandidateRepository) RejectAll(ctx context.Context, mediaFileID string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE metadata_candidates SET status = ? WHERE media_file_id = ?",
		domain.CandidateStatusRejected,
		mediaFileID,
	)
	if err != nil {
		return fmt.Errorf("reject all candidates: %w", err)
	}
	return nil
}

// CreateBatch creates multiple candidates in a single transaction
func (r *SQLiteMetadataCandidateRepository) CreateBatch(ctx context.Context, candidates []domain.MetadataCandidate) error {
	if len(candidates) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metadata_candidates (
			media_file_id, candidate_type, tmdb_id, title, release_date,
			overview, poster_path, confidence_score, season_number,
			episode_number, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for i := range candidates {
		result, err := stmt.ExecContext(ctx,
			candidates[i].MediaFileID,
			candidates[i].CandidateType,
			candidates[i].TMDBID,
			candidates[i].Title,
			candidates[i].ReleaseDate,
			candidates[i].Overview,
			candidates[i].PosterPath,
			candidates[i].ConfidenceScore,
			candidates[i].SeasonNumber,
			candidates[i].EpisodeNumber,
			candidates[i].Status,
			now,
		)
		if err != nil {
			return fmt.Errorf("insert candidate %d: %w", i, err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("get last insert id for candidate %d: %w", i, err)
		}

		candidates[i].ID = id
		candidates[i].CreatedAt = now
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Ensure SQLiteMetadataCandidateRepository implements MetadataCandidateRepository
var _ ports.MetadataCandidateRepository = (*SQLiteMetadataCandidateRepository)(nil)
