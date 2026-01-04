package media

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// DeleteMediaInput contains the input parameters for deleting media
type DeleteMediaInput struct {
	MediaID string
}

// DeleteMediaUseCase handles the deletion of a media file and all associated data
type DeleteMediaUseCase struct {
	mediaRepo     ports.MediaRepository
	transcodeRepo ports.TranscodeRepository
	episodeRepo   ports.EpisodeMetadataRepository
	seasonRepo    ports.SeasonMetadataRepository
	seriesRepo    ports.SeriesMetadataRepository
	fsService     ports.FileSystemService
	config        *config.Config
}

// NewDeleteMediaUseCase creates a new DeleteMediaUseCase instance
func NewDeleteMediaUseCase(
	mediaRepo ports.MediaRepository,
	transcodeRepo ports.TranscodeRepository,
	episodeRepo ports.EpisodeMetadataRepository,
	seasonRepo ports.SeasonMetadataRepository,
	seriesRepo ports.SeriesMetadataRepository,
	fsService ports.FileSystemService,
	cfg *config.Config,
) *DeleteMediaUseCase {
	return &DeleteMediaUseCase{
		mediaRepo:     mediaRepo,
		transcodeRepo: transcodeRepo,
		episodeRepo:   episodeRepo,
		seasonRepo:    seasonRepo,
		seriesRepo:    seriesRepo,
		fsService:     fsService,
		config:        cfg,
	}
}

// Execute deletes a media file and all associated data.
// It performs the following steps:
// 1. Verify media exists
// 2. Check for running transcode jobs (block if any)
// 3. Move source file to trash with timestamp
// 4. Delete database records (CASCADE handles related tables)
// 5. Delete media directory with transcoded files
// 6. Cascade cleanup: delete empty season/series metadata
func (uc *DeleteMediaUseCase) Execute(ctx context.Context, input DeleteMediaInput) error {
	if input.MediaID == "" {
		return fmt.Errorf("media ID is required")
	}

	// 1. Get media file to verify it exists and get file path
	media, err := uc.mediaRepo.Get(ctx, input.MediaID)
	if err != nil {
		return fmt.Errorf("%w: %s", shared.ErrMediaNotFound, input.MediaID)
	}

	// Store episode metadata ID before deletion for cascade cleanup
	episodeMetadataID := media.EpisodeMetadataID

	logger.Info().
		Str("media_id", input.MediaID).
		Str("file_path", media.FilePath).
		Msg("Starting media deletion")

	// 2. Check for processing transcodes - block deletion if any
	processing, err := uc.transcodeRepo.GetProcessingByMediaID(ctx, input.MediaID)
	if err != nil {
		return fmt.Errorf("failed to check processing transcodes: %w", err)
	}
	if len(processing) > 0 {
		logger.Warn().
			Str("media_id", input.MediaID).
			Int("processing_count", len(processing)).
			Msg("Cannot delete media with running transcode jobs")
		return shared.ErrMediaHasRunningJobs
	}

	// 3. Move source file to trash with timestamp
	// Trash path: {library_path}/trash/{timestamp}_{original_filename}
	trashDir := filepath.Join(uc.config.Media.LibraryPath, "trash")
	if err := uc.fsService.CreateDir(trashDir); err != nil {
		logger.Warn().Err(err).Str("path", trashDir).Msg("Failed to create trash directory")
		// Continue anyway - we'll try to delete the file directly
	}

	sourceFilename := filepath.Base(media.FilePath)
	timestamp := time.Now().Format("20060102_150405")
	trashFilePath := filepath.Join(trashDir, fmt.Sprintf("%s_%s", timestamp, sourceFilename))

	// Try to move source file to trash
	if uc.fsService.FileExists(media.FilePath) {
		if err := uc.fsService.Rename(media.FilePath, trashFilePath); err != nil {
			logger.Warn().
				Err(err).
				Str("source", media.FilePath).
				Str("trash", trashFilePath).
				Msg("Failed to move source file to trash, will delete directly")
			// If move fails, we'll still proceed with deletion
			// The file might be on a different filesystem
		} else {
			logger.Info().
				Str("source", media.FilePath).
				Str("trash", trashFilePath).
				Msg("Source file moved to trash")
		}
	} else {
		logger.Warn().Str("path", media.FilePath).Msg("Source file not found, skipping trash")
	}

	// 4. Delete from database - CASCADE handles:
	// - audio_streams
	// - subtitle_streams
	// - transcodes
	// - metadata_candidates
	if err := uc.mediaRepo.Delete(ctx, input.MediaID); err != nil {
		return fmt.Errorf("failed to delete media record: %w", err)
	}

	logger.Info().Str("media_id", input.MediaID).Msg("Deleted media record from database")

	// 5. Delete media directory with transcoded files
	// Path: {media_path}/{media_id}/
	mediaDir := filepath.Join(uc.config.Media.MediaPath, input.MediaID)
	if uc.fsService.FileExists(mediaDir) {
		if err := uc.fsService.RemoveDir(mediaDir); err != nil {
			// Log warning but don't fail - DB record is already deleted
			logger.Warn().
				Err(err).
				Str("path", mediaDir).
				Msg("Failed to delete media directory")
		} else {
			logger.Info().Str("path", mediaDir).Msg("Deleted media directory")
		}
	}

	// 6. Cascade cleanup: delete empty season/series metadata
	if episodeMetadataID != nil {
		uc.cascadeCleanupMetadata(ctx, *episodeMetadataID)
	}

	logger.Info().
		Str("media_id", input.MediaID).
		Str("filename", sourceFilename).
		Msg("Media deletion completed successfully")

	return nil
}

// cascadeCleanupMetadata deletes empty season and series metadata after media deletion.
// This is called when deleting media linked to an episode.
// If the season has no more media, delete the season metadata.
// If the series has no more media, delete the series metadata.
func (uc *DeleteMediaUseCase) cascadeCleanupMetadata(ctx context.Context, episodeMetadataID int64) {
	// Get the episode to find the season ID
	episode, err := uc.episodeRepo.Get(ctx, episodeMetadataID)
	if err != nil {
		// Episode might already be deleted or not exist - that's OK
		logger.Debug().
			Int64("episode_metadata_id", episodeMetadataID).
			Err(err).
			Msg("Could not find episode metadata for cascade cleanup")
		return
	}

	seasonID := episode.SeasonID

	// Get the season to find the series ID
	season, err := uc.seasonRepo.Get(ctx, seasonID)
	if err != nil {
		logger.Debug().
			Int64("season_id", seasonID).
			Err(err).
			Msg("Could not find season metadata for cascade cleanup")
		return
	}

	seriesID := season.SeriesID

	// Check if the season has any remaining media
	seasonMediaCount, err := uc.mediaRepo.CountBySeasonMetadataID(ctx, seasonID)
	if err != nil {
		logger.Warn().
			Int64("season_id", seasonID).
			Err(err).
			Msg("Failed to count remaining media for season")
		return
	}

	if seasonMediaCount > 0 {
		// Season still has media, no cleanup needed
		logger.Debug().
			Int64("season_id", seasonID).
			Int("remaining_media", seasonMediaCount).
			Msg("Season still has media, skipping cleanup")
		return
	}

	// Season is empty - delete it (CASCADE will delete episodes)
	logger.Info().
		Int64("season_id", seasonID).
		Int("season_number", season.SeasonNumber).
		Msg("Season has no more media, deleting season metadata")

	if err := uc.seasonRepo.Delete(ctx, seasonID); err != nil {
		logger.Warn().
			Int64("season_id", seasonID).
			Err(err).
			Msg("Failed to delete empty season metadata")
		return
	}

	// Check if the series has any remaining media
	seriesMediaCount, err := uc.mediaRepo.CountBySeriesMetadataID(ctx, seriesID)
	if err != nil {
		logger.Warn().
			Int64("series_id", seriesID).
			Err(err).
			Msg("Failed to count remaining media for series")
		return
	}

	if seriesMediaCount > 0 {
		// Series still has media, no cleanup needed
		logger.Debug().
			Int64("series_id", seriesID).
			Int("remaining_media", seriesMediaCount).
			Msg("Series still has media, skipping cleanup")
		return
	}

	// Series is empty - delete it (CASCADE will delete seasons and episodes)
	logger.Info().
		Int64("series_id", seriesID).
		Msg("Series has no more media, deleting series metadata")

	if err := uc.seriesRepo.Delete(ctx, seriesID); err != nil {
		logger.Warn().
			Int64("series_id", seriesID).
			Err(err).
			Msg("Failed to delete empty series metadata")
	}
}
