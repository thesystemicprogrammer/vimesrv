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

// scanner is an interface that abstracts sql.Row and sql.Rows for row scanning
type scanner interface {
	Scan(dest ...interface{}) error
}

// MediaRepository implements media file persistence operations
type MediaRepository struct {
	db *sql.DB
}

// scanMediaRow scans a row into a MediaFile struct and unmarshals JSON fields
func (r *MediaRepository) scanMediaRow(s scanner) (*domain.MediaFile, error) {
	var media domain.MediaFile
	var audioCodecsJSON, subtitleLanguagesJSON string
	var enrichmentStatus, metadataType, edition sql.NullString
	var movieMetadataID, episodeMetadataID sql.NullInt64

	err := s.Scan(
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
		&enrichmentStatus,
		&metadataType,
		&movieMetadataID,
		&episodeMetadataID,
		&edition,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON array fields
	if err := json.Unmarshal([]byte(audioCodecsJSON), &media.AudioCodecs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal audio codecs: %w", err)
	}

	if err := json.Unmarshal([]byte(subtitleLanguagesJSON), &media.SubtitleLanguages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subtitle languages: %w", err)
	}

	// Handle nullable enrichment fields
	if enrichmentStatus.Valid {
		media.EnrichmentStatus = enrichmentStatus.String
	} else {
		media.EnrichmentStatus = domain.EnrichmentStatusPending
	}

	if metadataType.Valid {
		media.MetadataType = metadataType.String
	} else {
		media.MetadataType = domain.MetadataTypeNone
	}

	if movieMetadataID.Valid {
		media.MovieMetadataID = &movieMetadataID.Int64
	}

	if episodeMetadataID.Valid {
		media.EpisodeMetadataID = &episodeMetadataID.Int64
	}

	if edition.Valid {
		media.Edition = edition.String
	}

	return &media, nil
}

// marshalArrayFields marshals the AudioCodecs and SubtitleLanguages slices to JSON strings
func (r *MediaRepository) marshalArrayFields(media *domain.MediaFile) (audioJSON, subtitleJSON string, err error) {
	audioCodecsBytes, err := json.Marshal(media.AudioCodecs)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal audio codecs: %w", err)
	}

	subtitleLanguagesBytes, err := json.Marshal(media.SubtitleLanguages)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal subtitle languages: %w", err)
	}

	return string(audioCodecsBytes), string(subtitleLanguagesBytes), nil
}

// NewMediaRepository creates a new MediaRepository instance
func NewMediaRepository(db *database.DB) ports.MediaRepository {
	return &MediaRepository{
		db: db.DB,
	}
}

// Create inserts a new media file record
func (r *MediaRepository) Create(ctx context.Context, media *domain.MediaFile) error {
	audioCodecsJSON, subtitleLanguagesJSON, err := r.marshalArrayFields(media)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO media_files (
			id, fingerprint, file_path, original_filename, filename,
			title, duration, file_size, format, video_codec, audio_codecs,
			resolution, width, height, bitrate, audio_tracks, subtitle_tracks,
			subtitle_languages, status, created_at, updated_at, scanned_at,
			enrichment_status, metadata_type, movie_metadata_id, episode_metadata_id, edition
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
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
		audioCodecsJSON,
		media.Resolution,
		media.Width,
		media.Height,
		media.Bitrate,
		media.AudioTracks,
		media.SubtitleTracks,
		subtitleLanguagesJSON,
		media.Status,
		media.CreatedAt,
		media.UpdatedAt,
		media.ScannedAt,
		media.EnrichmentStatus,
		media.MetadataType,
		media.MovieMetadataID,
		media.EpisodeMetadataID,
		media.Edition,
	)

	if err != nil {
		return fmt.Errorf("failed to insert media file: %w", err)
	}

	return nil
}

// Get retrieves a media file by its ID
func (r *MediaRepository) Get(ctx context.Context, id string) (*domain.MediaFile, error) {
	query := `
		SELECT 
			id, fingerprint, file_path, original_filename, filename,
			title, duration, file_size, format, video_codec, audio_codecs,
			resolution, width, height, bitrate, audio_tracks, subtitle_tracks,
			subtitle_languages, status, created_at, updated_at, scanned_at,
			enrichment_status, metadata_type, movie_metadata_id, episode_metadata_id, edition
		FROM media_files
		WHERE id = ?
	`

	media, err := r.scanMediaRow(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("media file not found with id: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query media file: %w", err)
	}

	return media, nil
}

// List retrieves all media files with pagination
func (r *MediaRepository) List(ctx context.Context, page, perPage int) ([]*domain.MediaFile, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM media_files`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count media files: %w", err)
	}

	// Calculate offset
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	// Get paginated media files
	query := `
		SELECT 
			id, fingerprint, file_path, original_filename, filename,
			title, duration, file_size, format, video_codec, audio_codecs,
			resolution, width, height, bitrate, audio_tracks, subtitle_tracks,
			subtitle_languages, status, created_at, updated_at, scanned_at,
			enrichment_status, metadata_type, movie_metadata_id, episode_metadata_id, edition
		FROM media_files
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query media files: %w", err)
	}
	defer rows.Close()

	var mediaFiles []*domain.MediaFile
	for rows.Next() {
		media, err := r.scanMediaRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan media file: %w", err)
		}
		mediaFiles = append(mediaFiles, media)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating media files: %w", err)
	}

	return mediaFiles, total, nil
}

// FindByFingerprint retrieves a media file by its fingerprint
func (r *MediaRepository) FindByFingerprint(ctx context.Context, fingerprint string) (*domain.MediaFile, error) {
	query := `
		SELECT 
			id, fingerprint, file_path, original_filename, filename,
			title, duration, file_size, format, video_codec, audio_codecs,
			resolution, width, height, bitrate, audio_tracks, subtitle_tracks,
			subtitle_languages, status, created_at, updated_at, scanned_at,
			enrichment_status, metadata_type, movie_metadata_id, episode_metadata_id, edition
		FROM media_files
		WHERE fingerprint = ?
	`

	media, err := r.scanMediaRow(r.db.QueryRowContext(ctx, query, fingerprint))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query media file: %w", err)
	}

	return media, nil
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
	audioCodecsJSON, subtitleLanguagesJSON, err := r.marshalArrayFields(media)
	if err != nil {
		return err
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
			scanned_at = ?,
			enrichment_status = ?,
			metadata_type = ?,
			movie_metadata_id = ?,
			episode_metadata_id = ?,
			edition = ?
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
		audioCodecsJSON,
		media.Resolution,
		media.Width,
		media.Height,
		media.Bitrate,
		media.AudioTracks,
		media.SubtitleTracks,
		subtitleLanguagesJSON,
		media.Status,
		media.UpdatedAt,
		media.ScannedAt,
		media.EnrichmentStatus,
		media.MetadataType,
		media.MovieMetadataID,
		media.EpisodeMetadataID,
		media.Edition,
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

// Delete removes a media file record by its ID.
// Related records (audio_streams, subtitle_streams, transcodes, metadata_candidates)
// are automatically deleted via CASCADE.
func (r *MediaRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM media_files WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete media file: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("media file not found with id: %s", id)
	}

	return nil
}

// FindByEpisodeMetadataIDs retrieves all media files linked to any of the given episode metadata IDs
func (r *MediaRepository) FindByEpisodeMetadataIDs(ctx context.Context, episodeMetadataIDs []int64) ([]*domain.MediaFile, error) {
	if len(episodeMetadataIDs) == 0 {
		return []*domain.MediaFile{}, nil
	}

	// Build placeholders for IN clause
	placeholders := ""
	args := make([]interface{}, len(episodeMetadataIDs))
	for i, id := range episodeMetadataIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT 
			id, fingerprint, file_path, original_filename, filename,
			title, duration, file_size, format, video_codec, audio_codecs,
			resolution, width, height, bitrate, audio_tracks, subtitle_tracks,
			subtitle_languages, status, created_at, updated_at, scanned_at,
			enrichment_status, metadata_type, movie_metadata_id, episode_metadata_id, edition
		FROM media_files
		WHERE episode_metadata_id IN (%s)
	`, placeholders)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query media files by episode metadata IDs: %w", err)
	}
	defer rows.Close()

	var mediaFiles []*domain.MediaFile
	for rows.Next() {
		media, err := r.scanMediaRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media file: %w", err)
		}
		mediaFiles = append(mediaFiles, media)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating media files: %w", err)
	}

	return mediaFiles, nil
}

// CountBySeasonMetadataID counts media files linked to episodes in the given season
func (r *MediaRepository) CountBySeasonMetadataID(ctx context.Context, seasonMetadataID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM media_files mf
		INNER JOIN episode_metadata em ON mf.episode_metadata_id = em.id
		WHERE em.season_id = ?
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, seasonMetadataID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count media files by season: %w", err)
	}

	return count, nil
}

// CountBySeriesMetadataID counts media files linked to episodes in any season of the given series
func (r *MediaRepository) CountBySeriesMetadataID(ctx context.Context, seriesMetadataID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM media_files mf
		INNER JOIN episode_metadata em ON mf.episode_metadata_id = em.id
		INNER JOIN season_metadata sm ON em.season_id = sm.id
		WHERE sm.series_id = ?
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, seriesMetadataID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count media files by series: %w", err)
	}

	return count, nil
}

// Search searches media files by title or filename
// Returns up to limit results, ordered by relevance (exact matches first, then partial matches)
func (r *MediaRepository) Search(ctx context.Context, searchQuery string, limit int) ([]*domain.MediaFile, error) {
	if limit <= 0 {
		limit = 20
	}

	// Use LIKE for case-insensitive search on title and filename
	// Order by exact match first, then by created_at descending
	query := `
		SELECT 
			id, fingerprint, file_path, original_filename, filename,
			title, duration, file_size, format, video_codec, audio_codecs,
			resolution, width, height, bitrate, audio_tracks, subtitle_tracks,
			subtitle_languages, status, created_at, updated_at, scanned_at,
			enrichment_status, metadata_type, movie_metadata_id, episode_metadata_id, edition
		FROM media_files
		WHERE title LIKE ? OR filename LIKE ?
		ORDER BY 
			CASE 
				WHEN LOWER(title) = LOWER(?) THEN 0
				WHEN LOWER(filename) = LOWER(?) THEN 1
				ELSE 2
			END,
			created_at DESC
		LIMIT ?
	`

	searchPattern := "%" + searchQuery + "%"
	rows, err := r.db.QueryContext(ctx, query, searchPattern, searchPattern, searchQuery, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search media files: %w", err)
	}
	defer rows.Close()

	var mediaFiles []*domain.MediaFile
	for rows.Next() {
		media, err := r.scanMediaRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media file: %w", err)
		}
		mediaFiles = append(mediaFiles, media)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating media files: %w", err)
	}

	return mediaFiles, nil
}
