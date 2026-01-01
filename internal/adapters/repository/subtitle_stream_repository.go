package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SubtitleStreamRepository implements subtitle stream persistence operations
type SubtitleStreamRepository struct {
	db *sql.DB
}

// NewSubtitleStreamRepository creates a new SubtitleStreamRepository instance
func NewSubtitleStreamRepository(db *database.DB) ports.SubtitleStreamRepository {
	return &SubtitleStreamRepository{
		db: db.DB,
	}
}

// Create inserts a new subtitle stream record
func (r *SubtitleStreamRepository) Create(ctx context.Context, stream *domain.SubtitleStream) error {
	query := `
		INSERT INTO subtitle_streams (
			media_id, stream_index, codec, language, title, forced, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	forced := 0
	if stream.Forced {
		forced = 1
	}

	result, err := r.db.ExecContext(ctx, query,
		stream.MediaID,
		stream.StreamIndex,
		stream.Codec,
		stream.Language,
		stream.Title,
		forced,
		stream.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert subtitle stream: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	stream.ID = id
	return nil
}

// GetByMediaID retrieves all subtitle streams for a media file
func (r *SubtitleStreamRepository) GetByMediaID(ctx context.Context, mediaID string) ([]*domain.SubtitleStream, error) {
	query := `
		SELECT 
			id, media_id, stream_index, codec, language, title, forced, created_at
		FROM subtitle_streams
		WHERE media_id = ?
		ORDER BY stream_index ASC
	`

	rows, err := r.db.QueryContext(ctx, query, mediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query subtitle streams: %w", err)
	}
	defer rows.Close()

	var streams []*domain.SubtitleStream
	for rows.Next() {
		var stream domain.SubtitleStream
		var forced int
		var language, title sql.NullString
		err := rows.Scan(
			&stream.ID,
			&stream.MediaID,
			&stream.StreamIndex,
			&stream.Codec,
			&language,
			&title,
			&forced,
			&stream.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subtitle stream: %w", err)
		}
		stream.Language = language.String
		stream.Title = title.String
		stream.Forced = forced == 1
		streams = append(streams, &stream)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating subtitle streams: %w", err)
	}

	return streams, nil
}

// DeleteByMediaID deletes all subtitle streams for a media file
func (r *SubtitleStreamRepository) DeleteByMediaID(ctx context.Context, mediaID string) error {
	query := `DELETE FROM subtitle_streams WHERE media_id = ?`

	_, err := r.db.ExecContext(ctx, query, mediaID)
	if err != nil {
		return fmt.Errorf("failed to delete subtitle streams: %w", err)
	}

	return nil
}
