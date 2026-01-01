package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// AudioStreamRepository implements audio stream persistence operations
type AudioStreamRepository struct {
	db *sql.DB
}

// NewAudioStreamRepository creates a new AudioStreamRepository instance
func NewAudioStreamRepository(db *database.DB) ports.AudioStreamRepository {
	return &AudioStreamRepository{
		db: db.DB,
	}
}

// Create inserts a new audio stream record
func (r *AudioStreamRepository) Create(ctx context.Context, stream *domain.AudioStream) error {
	query := `
		INSERT INTO audio_streams (
			media_id, stream_index, codec, language, channels,
			channel_layout, sample_rate, title, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		stream.MediaID,
		stream.StreamIndex,
		stream.Codec,
		stream.Language,
		stream.Channels,
		stream.ChannelLayout,
		stream.SampleRate,
		stream.Title,
		stream.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert audio stream: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	stream.ID = id
	return nil
}

// GetByMediaID retrieves all audio streams for a media file
func (r *AudioStreamRepository) GetByMediaID(ctx context.Context, mediaID string) ([]*domain.AudioStream, error) {
	query := `
		SELECT 
			id, media_id, stream_index, codec, language, channels,
			channel_layout, sample_rate, title, created_at
		FROM audio_streams
		WHERE media_id = ?
		ORDER BY stream_index ASC
	`

	rows, err := r.db.QueryContext(ctx, query, mediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audio streams: %w", err)
	}
	defer rows.Close()

	var streams []*domain.AudioStream
	for rows.Next() {
		var stream domain.AudioStream
		err := rows.Scan(
			&stream.ID,
			&stream.MediaID,
			&stream.StreamIndex,
			&stream.Codec,
			&stream.Language,
			&stream.Channels,
			&stream.ChannelLayout,
			&stream.SampleRate,
			&stream.Title,
			&stream.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audio stream: %w", err)
		}
		streams = append(streams, &stream)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audio streams: %w", err)
	}

	return streams, nil
}

// DeleteByMediaID deletes all audio streams for a media file
func (r *AudioStreamRepository) DeleteByMediaID(ctx context.Context, mediaID string) error {
	query := `DELETE FROM audio_streams WHERE media_id = ?`

	_, err := r.db.ExecContext(ctx, query, mediaID)
	if err != nil {
		return fmt.Errorf("failed to delete audio streams: %w", err)
	}

	return nil
}
