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

// DeleteSeriesInput contains the input parameters for deleting a series
type DeleteSeriesInput struct {
	SeriesID int64
}

// DeleteSeriesResult contains the result of the delete operation
type DeleteSeriesResult struct {
	DeletedMediaCount  int
	DeletedSeasonCount int
}

// DeleteSeriesUseCase handles the deletion of all media files for a series
type DeleteSeriesUseCase struct {
	seriesRepo    ports.SeriesMetadataRepository
	seasonRepo    ports.SeasonMetadataRepository
	episodeRepo   ports.EpisodeMetadataRepository
	mediaRepo     ports.MediaRepository
	transcodeRepo ports.TranscodeRepository
	fsService     ports.FileSystemService
	config        *config.Config
}

// NewDeleteSeriesUseCase creates a new DeleteSeriesUseCase instance
func NewDeleteSeriesUseCase(
	seriesRepo ports.SeriesMetadataRepository,
	seasonRepo ports.SeasonMetadataRepository,
	episodeRepo ports.EpisodeMetadataRepository,
	mediaRepo ports.MediaRepository,
	transcodeRepo ports.TranscodeRepository,
	fsService ports.FileSystemService,
	cfg *config.Config,
) *DeleteSeriesUseCase {
	return &DeleteSeriesUseCase{
		seriesRepo:    seriesRepo,
		seasonRepo:    seasonRepo,
		episodeRepo:   episodeRepo,
		mediaRepo:     mediaRepo,
		transcodeRepo: transcodeRepo,
		fsService:     fsService,
		config:        cfg,
	}
}

// Execute deletes all media files for a series.
// It performs the following steps:
// 1. Verify series exists and get all seasons
// 2. Get all episodes for all seasons
// 3. Get all media files linked to episodes
// 4. Check ALL media files for running transcode jobs (block if any)
// 5. Delete each media file (move source to trash, delete transcodes)
// 6. Return count of deleted media files and seasons
func (uc *DeleteSeriesUseCase) Execute(ctx context.Context, input DeleteSeriesInput) (*DeleteSeriesResult, error) {
	if input.SeriesID == 0 {
		return nil, fmt.Errorf("series ID is required")
	}

	// 1. Get series to verify it exists
	series, err := uc.seriesRepo.Get(ctx, input.SeriesID)
	if err != nil {
		return nil, fmt.Errorf("series not found: %w", err)
	}

	logger.Info().
		Int64("series_id", input.SeriesID).
		Str("series_name", series.OriginalName).
		Msg("Starting series media deletion")

	// 2. Get all seasons for this series
	seasons, err := uc.seasonRepo.ListBySeriesID(ctx, input.SeriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to get seasons for series: %w", err)
	}

	if len(seasons) == 0 {
		logger.Info().Int64("series_id", input.SeriesID).Msg("No seasons found for series")
		return &DeleteSeriesResult{DeletedMediaCount: 0, DeletedSeasonCount: 0}, nil
	}

	// 3. Get all episodes for all seasons
	var allEpisodeIDs []int64
	for _, season := range seasons {
		episodes, err := uc.episodeRepo.ListBySeasonID(ctx, season.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get episodes for season %d: %w", season.ID, err)
		}
		for _, ep := range episodes {
			allEpisodeIDs = append(allEpisodeIDs, ep.ID)
		}
	}

	if len(allEpisodeIDs) == 0 {
		logger.Info().Int64("series_id", input.SeriesID).Msg("No episodes found for series")
		return &DeleteSeriesResult{DeletedMediaCount: 0, DeletedSeasonCount: len(seasons)}, nil
	}

	// 4. Get all media files linked to these episodes
	mediaFiles, err := uc.mediaRepo.FindByEpisodeMetadataIDs(ctx, allEpisodeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get media files for episodes: %w", err)
	}

	if len(mediaFiles) == 0 {
		logger.Info().Int64("series_id", input.SeriesID).Msg("No media files found for series")
		return &DeleteSeriesResult{DeletedMediaCount: 0, DeletedSeasonCount: len(seasons)}, nil
	}

	logger.Info().
		Int64("series_id", input.SeriesID).
		Int("season_count", len(seasons)).
		Int("media_count", len(mediaFiles)).
		Msg("Found media files for series")

	// 5. Pre-check: Check ALL media files for running transcode jobs before deleting ANY
	for _, media := range mediaFiles {
		processing, err := uc.transcodeRepo.GetProcessingByMediaID(ctx, media.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check processing transcodes for media %s: %w", media.ID, err)
		}
		if len(processing) > 0 {
			logger.Warn().
				Str("media_id", media.ID).
				Int64("series_id", input.SeriesID).
				Int("processing_count", len(processing)).
				Msg("Cannot delete series: media has running transcode jobs")
			return nil, shared.ErrMediaHasRunningJobs
		}
	}

	// 6. Delete each media file
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
		Int64("series_id", input.SeriesID).
		Int("deleted_media_count", deletedCount).
		Int("season_count", len(seasons)).
		Msg("Series media deletion completed")

	return &DeleteSeriesResult{
		DeletedMediaCount:  deletedCount,
		DeletedSeasonCount: len(seasons),
	}, nil
}
