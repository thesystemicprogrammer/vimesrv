package library

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

// DeleteSeasonInput contains the input parameters for deleting a season
type DeleteSeasonInput struct {
	SeasonID int64
}

// DeleteSeasonResult contains the result of the delete operation
type DeleteSeasonResult struct {
	DeletedMediaCount int
}

// DeleteSeasonUseCase handles the deletion of all media files for a season
type DeleteSeasonUseCase struct {
	seasonRepo    ports.SeasonMetadataRepository
	episodeRepo   ports.EpisodeMetadataRepository
	mediaRepo     ports.MediaRepository
	transcodeRepo ports.TranscodeRepository
	fsService     ports.FileSystemService
	config        *config.Config
}

// NewDeleteSeasonUseCase creates a new DeleteSeasonUseCase instance
func NewDeleteSeasonUseCase(
	seasonRepo ports.SeasonMetadataRepository,
	episodeRepo ports.EpisodeMetadataRepository,
	mediaRepo ports.MediaRepository,
	transcodeRepo ports.TranscodeRepository,
	fsService ports.FileSystemService,
	cfg *config.Config,
) *DeleteSeasonUseCase {
	return &DeleteSeasonUseCase{
		seasonRepo:    seasonRepo,
		episodeRepo:   episodeRepo,
		mediaRepo:     mediaRepo,
		transcodeRepo: transcodeRepo,
		fsService:     fsService,
		config:        cfg,
	}
}

// Execute deletes all media files for a season.
// It performs the following steps:
// 1. Verify season exists and get all episodes
// 2. Get all media files linked to episodes
// 3. Check ALL media files for running transcode jobs (block if any)
// 4. Delete each media file (move source to trash, delete transcodes)
// 5. Return count of deleted media files
func (uc *DeleteSeasonUseCase) Execute(ctx context.Context, input DeleteSeasonInput) (*DeleteSeasonResult, error) {
	if input.SeasonID == 0 {
		return nil, fmt.Errorf("season ID is required")
	}

	// 1. Get season to verify it exists
	season, err := uc.seasonRepo.Get(ctx, input.SeasonID)
	if err != nil {
		return nil, fmt.Errorf("season not found: %w", err)
	}

	logger.Info().
		Int64("season_id", input.SeasonID).
		Int("season_number", season.SeasonNumber).
		Msg("Starting season media deletion")

	// 2. Get all episodes for this season
	episodes, err := uc.episodeRepo.ListBySeasonID(ctx, input.SeasonID)
	if err != nil {
		return nil, fmt.Errorf("failed to get episodes for season: %w", err)
	}

	if len(episodes) == 0 {
		logger.Info().Int64("season_id", input.SeasonID).Msg("No episodes found for season")
		return &DeleteSeasonResult{DeletedMediaCount: 0}, nil
	}

	// Collect episode metadata IDs
	episodeIDs := make([]int64, len(episodes))
	for i, ep := range episodes {
		episodeIDs[i] = ep.ID
	}

	// 3. Get all media files linked to these episodes
	mediaFiles, err := uc.mediaRepo.FindByEpisodeMetadataIDs(ctx, episodeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get media files for episodes: %w", err)
	}

	if len(mediaFiles) == 0 {
		logger.Info().Int64("season_id", input.SeasonID).Msg("No media files found for season")
		return &DeleteSeasonResult{DeletedMediaCount: 0}, nil
	}

	logger.Info().
		Int64("season_id", input.SeasonID).
		Int("media_count", len(mediaFiles)).
		Msg("Found media files for season")

	// 4. Pre-check: Check ALL media files for running transcode jobs before deleting ANY
	for _, media := range mediaFiles {
		processing, err := uc.transcodeRepo.GetProcessingByMediaID(ctx, media.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check processing transcodes for media %s: %w", media.ID, err)
		}
		if len(processing) > 0 {
			logger.Warn().
				Str("media_id", media.ID).
				Int64("season_id", input.SeasonID).
				Int("processing_count", len(processing)).
				Msg("Cannot delete season: media has running transcode jobs")
			return nil, shared.ErrMediaHasRunningJobs
		}
	}

	// 5. Delete each media file
	deletedCount := 0
	trashDir := filepath.Join(uc.config.Media.LibraryPath, "trash")
	if err := uc.fsService.CreateDir(trashDir); err != nil {
		logger.Warn().Err(err).Str("path", trashDir).Msg("Failed to create trash directory")
	}

	for _, media := range mediaFiles {
		// Move source file to trash
		if uc.fsService.FileExists(media.FilePath) {
			sourceFilename := filepath.Base(media.FilePath)
			timestamp := time.Now().Format("20060102_150405")
			trashFilePath := filepath.Join(trashDir, fmt.Sprintf("%s_%s", timestamp, sourceFilename))

			if err := uc.fsService.Rename(media.FilePath, trashFilePath); err != nil {
				logger.Warn().
					Err(err).
					Str("source", media.FilePath).
					Str("trash", trashFilePath).
					Msg("Failed to move source file to trash")
			} else {
				logger.Info().
					Str("source", media.FilePath).
					Str("trash", trashFilePath).
					Msg("Source file moved to trash")
			}
		}

		// Delete from database
		if err := uc.mediaRepo.Delete(ctx, media.ID); err != nil {
			logger.Error().Err(err).Str("media_id", media.ID).Msg("Failed to delete media record")
			continue
		}

		// Delete media directory with transcoded files
		mediaDir := filepath.Join(uc.config.Media.MediaPath, media.ID)
		if uc.fsService.FileExists(mediaDir) {
			if err := uc.fsService.RemoveDir(mediaDir); err != nil {
				logger.Warn().Err(err).Str("path", mediaDir).Msg("Failed to delete media directory")
			} else {
				logger.Info().Str("path", mediaDir).Msg("Deleted media directory")
			}
		}

		deletedCount++
		logger.Info().
			Str("media_id", media.ID).
			Str("filename", media.Filename).
			Msg("Media file deleted")
	}

	logger.Info().
		Int64("season_id", input.SeasonID).
		Int("deleted_count", deletedCount).
		Msg("Season media deletion completed")

	return &DeleteSeasonResult{DeletedMediaCount: deletedCount}, nil
}
