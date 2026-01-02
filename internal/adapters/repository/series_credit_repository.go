package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SQLiteSeriesCreditRepository implements SeriesCreditRepository using SQLite
type SQLiteSeriesCreditRepository struct {
	db *sql.DB
}

// NewSQLiteSeriesCreditRepository creates a new SQLite series credit repository
func NewSQLiteSeriesCreditRepository(db *sql.DB) *SQLiteSeriesCreditRepository {
	return &SQLiteSeriesCreditRepository{db: db}
}

// CreateBatch inserts multiple series credits in a single transaction
func (r *SQLiteSeriesCreditRepository) CreateBatch(ctx context.Context, credits []*domain.SeriesCredit) error {
	if len(credits) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO series_credits (
			series_metadata_id, credit_type, tmdb_person_id, name, roles,
			jobs, department, profile_path, total_episode_count, display_order, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, credit := range credits {
		result, err := stmt.ExecContext(ctx,
			credit.SeriesMetadataID,
			string(credit.CreditType),
			credit.TMDBPersonID,
			credit.Name,
			credit.Roles,
			credit.Jobs,
			credit.Department,
			credit.ProfilePath,
			credit.TotalEpisodeCount,
			credit.DisplayOrder,
			now,
		)
		if err != nil {
			return fmt.Errorf("insert series credit: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("get last insert id: %w", err)
		}

		credit.ID = id
		credit.CreatedAt = now
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetBySeriesMetadataID retrieves all credits for a series
func (r *SQLiteSeriesCreditRepository) GetBySeriesMetadataID(ctx context.Context, seriesMetadataID int64) ([]domain.SeriesCredit, error) {
	query := `
		SELECT id, series_metadata_id, credit_type, tmdb_person_id, name, roles,
			   jobs, department, profile_path, total_episode_count, display_order, created_at
		FROM series_credits
		WHERE series_metadata_id = ?
		ORDER BY credit_type, display_order
	`

	return r.queryCredits(ctx, query, seriesMetadataID)
}

// GetCastBySeriesMetadataID retrieves cast credits for a series, ordered by display_order
func (r *SQLiteSeriesCreditRepository) GetCastBySeriesMetadataID(ctx context.Context, seriesMetadataID int64, limit int) ([]domain.SeriesCredit, error) {
	query := `
		SELECT id, series_metadata_id, credit_type, tmdb_person_id, name, roles,
			   jobs, department, profile_path, total_episode_count, display_order, created_at
		FROM series_credits
		WHERE series_metadata_id = ? AND credit_type = 'cast'
		ORDER BY display_order
		LIMIT ?
	`

	return r.queryCredits(ctx, query, seriesMetadataID, limit)
}

// GetCrewBySeriesMetadataID retrieves crew credits for a series, ordered by display_order
func (r *SQLiteSeriesCreditRepository) GetCrewBySeriesMetadataID(ctx context.Context, seriesMetadataID int64) ([]domain.SeriesCredit, error) {
	query := `
		SELECT id, series_metadata_id, credit_type, tmdb_person_id, name, roles,
			   jobs, department, profile_path, total_episode_count, display_order, created_at
		FROM series_credits
		WHERE series_metadata_id = ? AND credit_type = 'crew'
		ORDER BY display_order
	`

	return r.queryCredits(ctx, query, seriesMetadataID)
}

// DeleteBySeriesMetadataID removes all credits for a series
func (r *SQLiteSeriesCreditRepository) DeleteBySeriesMetadataID(ctx context.Context, seriesMetadataID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM series_credits WHERE series_metadata_id = ?", seriesMetadataID)
	if err != nil {
		return fmt.Errorf("delete series credits: %w", err)
	}

	return nil
}

// HasCredits checks if credits exist for a series
func (r *SQLiteSeriesCreditRepository) HasCredits(ctx context.Context, seriesMetadataID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM series_credits WHERE series_metadata_id = ?",
		seriesMetadataID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("count series credits: %w", err)
	}

	return count > 0, nil
}

// queryCredits is a helper to execute credit queries and scan results
func (r *SQLiteSeriesCreditRepository) queryCredits(ctx context.Context, query string, args ...interface{}) ([]domain.SeriesCredit, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []domain.SeriesCredit{}, nil
		}
		return nil, fmt.Errorf("query series credits: %w", err)
	}
	defer rows.Close()

	var credits []domain.SeriesCredit
	for rows.Next() {
		var c domain.SeriesCredit
		var creditType string
		if err := rows.Scan(
			&c.ID,
			&c.SeriesMetadataID,
			&creditType,
			&c.TMDBPersonID,
			&c.Name,
			&c.Roles,
			&c.Jobs,
			&c.Department,
			&c.ProfilePath,
			&c.TotalEpisodeCount,
			&c.DisplayOrder,
			&c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan series credit: %w", err)
		}
		c.CreditType = domain.CreditType(creditType)
		credits = append(credits, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate series credits: %w", err)
	}

	if credits == nil {
		credits = []domain.SeriesCredit{}
	}

	return credits, nil
}

// Ensure SQLiteSeriesCreditRepository implements SeriesCreditRepository
var _ ports.SeriesCreditRepository = (*SQLiteSeriesCreditRepository)(nil)
