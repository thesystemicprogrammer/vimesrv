package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// MediaRepository implements media file persistence operations
type MediaRepository struct {
	db *sql.DB
}

// NewMediaRepository creates a new MediaRepository instance
func NewMediaRepository(db *database.DB) ports.MediaRepository {
	return &MediaRepository{
		db: db.DB,
	}
}

// Create inserts a new media file record
func (r *MediaRepository) Create(ctx context.Context, media *domain.MediaFile) error {
	// Marshal array fields to JSON
	audioCodecsJSON, err := json.Marshal(media.AudioCodecs)
	if err != nil {
		return fmt.Errorf("failed to marshal audio codecs: %w", err)
	}

	subtitleLanguagesJSON, err := json.Marshal(media.SubtitleLanguages)
	if err != nil {
		return fmt.Errorf("failed to marshal subtitle languages: %w", err)
	}

	query := `
		INSERT INTO media_files (
			id, fingerprint, file_path, original_filename, filename,
			title, duration, file_size, format, video_codec, audio_codecs,
			resolution, width, height, bitrate, audio_tracks, subtitle_tracks,
			subtitle_languages, status, created_at, updated_at, scanned_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`

	_, err = r.db.ExecContext(ctx, query,
		media.ID,
		media.Fingerprint,
		media.FilePath,
		media.OriginalFilename,
		media.Filename,
		media.Title,
		media.Duration,
		media.FileSize,
		media.Format,
		media.VideoCodec,
		string(audioCodecsJSON),
		media.Resolution,
		media.Width,
		media.Height,
		media.Bitrate,
		media.AudioTracks,
		media.SubtitleTracks,
		string(subtitleLanguagesJSON),
		media.Status,
		media.CreatedAt,
		media.UpdatedAt,
		media.ScannedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert media file: %w", err)
	}

	return nil
}

// FindByFingerprint retrieves a media file by its fingerprint
func (r *MediaRepository) FindByFingerprint(ctx context.Context, fingerprint string) (*domain.MediaFile, error) {
	query := `
		SELECT 
			id, fingerprint, file_path, original_filename, filename,
			title, duration, file_size, format, video_codec, audio_codecs,
			resolution, width, height, bitrate, audio_tracks, subtitle_tracks,
			subtitle_languages, status, created_at, updated_at, scanned_at
		FROM media_files
		WHERE fingerprint = ?
	`

	var media domain.MediaFile
	var audioCodecsJSON, subtitleLanguagesJSON string

	err := r.db.QueryRowContext(ctx, query, fingerprint).Scan(
		&media.ID,
		&media.Fingerprint,
		&media.FilePath,
		&media.OriginalFilename,
		&media.Filename,
		&media.Title,
		&media.Duration,
		&media.FileSize,
		&media.Format,
		&media.VideoCodec,
		&audioCodecsJSON,
		&media.Resolution,
		&media.Width,
		&media.Height,
		&media.Bitrate,
		&media.AudioTracks,
		&media.SubtitleTracks,
		&subtitleLanguagesJSON,
		&media.Status,
		&media.CreatedAt,
		&media.UpdatedAt,
		&media.ScannedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query media file: %w", err)
	}

	// Unmarshal JSON array fields
	if err := json.Unmarshal([]byte(audioCodecsJSON), &media.AudioCodecs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal audio codecs: %w", err)
	}

	if err := json.Unmarshal([]byte(subtitleLanguagesJSON), &media.SubtitleLanguages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subtitle languages: %w", err)
	}

	return &media, nil
}

// ExistsByFingerprint checks if a media file with the given fingerprint exists
func (r *MediaRepository) ExistsByFingerprint(ctx context.Context, fingerprint string) (bool, error) {
	query := `SELECT COUNT(*) FROM media_files WHERE fingerprint = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, fingerprint).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check fingerprint existence: %w", err)
	}

	return count > 0, nil
}

// Update updates an existing media file record
func (r *MediaRepository) Update(ctx context.Context, media *domain.MediaFile) error {
	// Marshal array fields to JSON
	audioCodecsJSON, err := json.Marshal(media.AudioCodecs)
	if err != nil {
		return fmt.Errorf("failed to marshal audio codecs: %w", err)
	}

	subtitleLanguagesJSON, err := json.Marshal(media.SubtitleLanguages)
	if err != nil {
		return fmt.Errorf("failed to marshal subtitle languages: %w", err)
	}

	query := `
		UPDATE media_files SET
			fingerprint = ?,
			file_path = ?,
			original_filename = ?,
			filename = ?,
			title = ?,
			duration = ?,
			file_size = ?,
			format = ?,
			video_codec = ?,
			audio_codecs = ?,
			resolution = ?,
			width = ?,
			height = ?,
			bitrate = ?,
			audio_tracks = ?,
			subtitle_tracks = ?,
			subtitle_languages = ?,
			status = ?,
			updated_at = ?,
			scanned_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		media.Fingerprint,
		media.FilePath,
		media.OriginalFilename,
		media.Filename,
		media.Title,
		media.Duration,
		media.FileSize,
		media.Format,
		media.VideoCodec,
		string(audioCodecsJSON),
		media.Resolution,
		media.Width,
		media.Height,
		media.Bitrate,
		media.AudioTracks,
		media.SubtitleTracks,
		string(subtitleLanguagesJSON),
		media.Status,
		media.UpdatedAt,
		media.ScannedAt,
		media.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update media file: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("media file not found with id: %s", media.ID)
	}

	return nil
}
