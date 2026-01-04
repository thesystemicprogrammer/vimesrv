package linker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// EpisodeLinker handles creating and linking episode metadata from TMDB
// This includes the full hierarchy: series -> season -> episode
type EpisodeLinker struct {
	config                    config.TMDBConfig
	tmdbClient                ports.TMDBClient
	imageDownloader           ports.ImageDownloader
	seriesMetadataRepository  ports.SeriesMetadataRepository
	seasonMetadataRepository  ports.SeasonMetadataRepository
	episodeMetadataRepository ports.EpisodeMetadataRepository
	seriesCreditRepository    ports.SeriesCreditRepository // optional, for fetching credits
	searchRepository          ports.SearchRepository       // optional, for FTS indexing
}

// NewEpisodeLinker creates a new EpisodeLinker instance
func NewEpisodeLinker(
	config config.TMDBConfig,
	tmdbClient ports.TMDBClient,
	imageDownloader ports.ImageDownloader,
	seriesMetadataRepository ports.SeriesMetadataRepository,
	seasonMetadataRepository ports.SeasonMetadataRepository,
	episodeMetadataRepository ports.EpisodeMetadataRepository,
	seriesCreditRepository ports.SeriesCreditRepository,
	searchRepository ports.SearchRepository,
) *EpisodeLinker {
	return &EpisodeLinker{
		config:                    config,
		tmdbClient:                tmdbClient,
		imageDownloader:           imageDownloader,
		seriesMetadataRepository:  seriesMetadataRepository,
		seasonMetadataRepository:  seasonMetadataRepository,
		episodeMetadataRepository: episodeMetadataRepository,
		seriesCreditRepository:    seriesCreditRepository,
		searchRepository:          searchRepository,
	}
}

// EpisodeLinkResult contains the result of an episode linking operation
type EpisodeLinkResult struct {
	SeriesMetadata  *domain.SeriesMetadata
	SeasonMetadata  *domain.SeasonMetadata
	EpisodeMetadata *domain.EpisodeMetadata
	SeriesDetails   *ports.TMDBSeriesDetails // Includes title information for output messages
	SeasonNumber    int
	EpisodeNumber   int
	SeriesCreated   bool // true if the series was newly created (not reused from existing)
}

// Link fetches episode metadata from TMDB and creates/retrieves the local records
// for the full series -> season -> episode hierarchy.
// It also downloads images if configured.
// Returns the episode metadata record (existing or newly created).
func (l *EpisodeLinker) Link(ctx context.Context, seriesTMDBID, seasonNumber, episodeNumber int) (*EpisodeLinkResult, error) {
	logger.Debug().
		Int("series_tmdb_id", seriesTMDBID).
		Int("season", seasonNumber).
		Int("episode", episodeNumber).
		Msg("Linking episode metadata")

	// Get detailed series info from TMDB
	seriesDetails, err := l.tmdbClient.GetSeriesDetails(ctx, seriesTMDBID, l.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get series details: %w", err)
	}

	// Check if series already exists (to track if we created it)
	existingSeries, _ := l.seriesMetadataRepository.GetByTMDBID(ctx, seriesTMDBID)
	seriesCreated := existingSeries == nil

	// Get or create series metadata
	seriesMetadata, err := l.getOrCreateSeriesMetadata(ctx, seriesDetails)
	if err != nil {
		return nil, err
	}

	// Get season details from TMDB
	seasonDetails, err := l.tmdbClient.GetSeasonDetails(ctx, seriesTMDBID, seasonNumber, l.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get season details: %w", err)
	}

	// Get or create season metadata
	seasonMetadata, err := l.getOrCreateSeasonMetadata(ctx, seriesMetadata.ID, seasonDetails)
	if err != nil {
		return nil, err
	}

	// Get or create episode metadata
	episodeMetadata, err := l.getOrCreateEpisodeMetadata(ctx, seasonMetadata.ID, seriesTMDBID, seasonNumber, episodeNumber)
	if err != nil {
		return nil, err
	}

	// Index new series for full-text search
	if seriesCreated {
		l.indexSeriesForSearch(ctx, seriesMetadata.ID, seriesDetails)
	}

	// Download images if configured
	l.downloadImages(ctx, seriesDetails, seasonDetails, seasonNumber, episodeNumber)

	return &EpisodeLinkResult{
		SeriesMetadata:  seriesMetadata,
		SeasonMetadata:  seasonMetadata,
		EpisodeMetadata: episodeMetadata,
		SeriesDetails:   seriesDetails,
		SeasonNumber:    seasonNumber,
		EpisodeNumber:   episodeNumber,
		SeriesCreated:   seriesCreated,
	}, nil
}

// getOrCreateSeriesMetadata retrieves existing series metadata or creates new from TMDB details
func (l *EpisodeLinker) getOrCreateSeriesMetadata(ctx context.Context, details *ports.TMDBSeriesDetails) (*domain.SeriesMetadata, error) {
	// Check if we already have this series
	existing, err := l.seriesMetadataRepository.GetByTMDBID(ctx, details.ID)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing series: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// Create new series metadata
	series := domain.NewSeriesMetadata(details.ID, details.OriginalName)
	series.FirstAirDate = details.FirstAirDate
	series.LastAirDate = details.LastAirDate
	series.Status = details.Status
	series.PosterPath = details.PosterPath
	series.BackdropPath = details.BackdropPath
	series.VoteAverage = details.VoteAverage
	series.VoteCount = details.VoteCount
	series.Popularity = details.Popularity
	series.NumberOfSeasons = details.NumberOfSeasons
	series.NumberOfEpisodes = details.NumberOfEpisodes
	series.OriginalLang = details.OriginalLang

	genres := make([]string, len(details.Genres))
	for i, g := range details.Genres {
		genres[i] = g.Name
	}
	series.Genres = genres

	networks := make([]string, len(details.Networks))
	for i, n := range details.Networks {
		networks[i] = n.Name
	}
	series.Networks = networks

	if err := l.seriesMetadataRepository.Create(ctx, series); err != nil {
		// Handle race condition: if another worker created the series concurrently,
		// fetch and return the existing record instead of failing
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			existing, fetchErr := l.seriesMetadataRepository.GetByTMDBID(ctx, details.ID)
			if fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch series after concurrent create: %w", fetchErr)
			}
			return existing, nil
		}
		return nil, fmt.Errorf("failed to create series metadata: %w", err)
	}

	// Create translation
	translation := domain.NewSeriesMetadataTranslation(series.ID, l.config.Language, details.Name)
	translation.Overview = details.Overview
	if err := l.seriesMetadataRepository.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create series translation: %w", err)
	}

	return series, nil
}

// getOrCreateSeasonMetadata retrieves existing season metadata or creates new from TMDB details
func (l *EpisodeLinker) getOrCreateSeasonMetadata(ctx context.Context, seriesID int64, details *ports.TMDBSeasonDetails) (*domain.SeasonMetadata, error) {
	// Check if we already have this season
	existing, err := l.seasonMetadataRepository.GetBySeriesAndNumber(ctx, seriesID, details.SeasonNumber)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing season: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// Create new season metadata
	season := domain.NewSeasonMetadata(seriesID, details.ID, details.SeasonNumber)
	season.AirDate = details.AirDate
	season.PosterPath = details.PosterPath
	season.EpisodeCount = len(details.Episodes)

	if err := l.seasonMetadataRepository.Create(ctx, season); err != nil {
		// Handle race condition: if another worker created the season concurrently,
		// fetch and return the existing record instead of failing
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			existing, fetchErr := l.seasonMetadataRepository.GetBySeriesAndNumber(ctx, seriesID, details.SeasonNumber)
			if fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch season after concurrent create: %w", fetchErr)
			}
			return existing, nil
		}
		return nil, fmt.Errorf("failed to create season metadata: %w", err)
	}

	// Create translation
	translation := domain.NewSeasonMetadataTranslation(season.ID, l.config.Language, details.Name)
	translation.Overview = details.Overview
	if err := l.seasonMetadataRepository.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create season translation: %w", err)
	}

	return season, nil
}

// getOrCreateEpisodeMetadata retrieves existing episode metadata or creates new from TMDB
func (l *EpisodeLinker) getOrCreateEpisodeMetadata(ctx context.Context, seasonID int64, seriesID int, seasonNumber int, episodeNumber int) (*domain.EpisodeMetadata, error) {
	// Check if we already have this episode
	existing, err := l.episodeMetadataRepository.GetBySeasonAndNumber(ctx, seasonID, episodeNumber)
	if err != nil && !errors.Is(err, shared.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing episode: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// Get episode details from TMDB
	details, err := l.tmdbClient.GetEpisodeDetails(ctx, seriesID, seasonNumber, episodeNumber, l.config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get episode details: %w", err)
	}

	// Create new episode metadata
	episode := domain.NewEpisodeMetadata(seasonID, details.ID, episodeNumber)
	episode.AirDate = details.AirDate
	episode.StillPath = details.StillPath
	episode.Runtime = details.Runtime
	episode.VoteAverage = details.VoteAverage
	episode.VoteCount = details.VoteCount

	if err := l.episodeMetadataRepository.Create(ctx, episode); err != nil {
		// Handle race condition: if another worker created the episode concurrently,
		// fetch and return the existing record instead of failing
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			existing, fetchErr := l.episodeMetadataRepository.GetBySeasonAndNumber(ctx, seasonID, episodeNumber)
			if fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch episode after concurrent create: %w", fetchErr)
			}
			return existing, nil
		}
		return nil, fmt.Errorf("failed to create episode metadata: %w", err)
	}

	// Create translation
	translation := domain.NewEpisodeMetadataTranslation(episode.ID, l.config.Language, details.Name)
	translation.Overview = details.Overview
	if err := l.episodeMetadataRepository.CreateTranslation(ctx, translation); err != nil {
		return nil, fmt.Errorf("failed to create episode translation: %w", err)
	}

	return episode, nil
}

// downloadImages downloads series, season, and episode images
func (l *EpisodeLinker) downloadImages(ctx context.Context, seriesDetails *ports.TMDBSeriesDetails, seasonDetails *ports.TMDBSeasonDetails, seasonNumber, episodeNumber int) {
	if !l.config.DownloadImages || l.imageDownloader == nil {
		return
	}

	// Series images
	if seriesDetails.PosterPath != "" {
		if _, err := l.imageDownloader.DownloadImage(ctx, seriesDetails.PosterPath, ports.ImageTypeSeriesPoster, seriesDetails.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Msg("Failed to download series poster")
		}
	}
	if seriesDetails.BackdropPath != "" {
		if _, err := l.imageDownloader.DownloadImage(ctx, seriesDetails.BackdropPath, ports.ImageTypeSeriesBackdrop, seriesDetails.ID); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Msg("Failed to download series backdrop")
		}
	}

	// Season poster
	if seasonDetails.PosterPath != "" {
		if _, err := l.imageDownloader.DownloadSeasonImage(ctx, seasonDetails.PosterPath, seriesDetails.ID, seasonNumber); err != nil {
			logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Int("season", seasonNumber).Msg("Failed to download season poster")
		}
	}

	// Episode still
	for _, ep := range seasonDetails.Episodes {
		if ep.EpisodeNumber == episodeNumber && ep.StillPath != "" {
			if _, err := l.imageDownloader.DownloadEpisodeImage(ctx, ep.StillPath, seriesDetails.ID, seasonNumber, episodeNumber); err != nil {
				logger.Warn().Err(err).Int("tmdb_id", seriesDetails.ID).Int("season", seasonNumber).Int("episode", episodeNumber).Msg("Failed to download episode still")
			}
			break
		}
	}
}

// indexSeriesForSearch adds the series to the FTS search index
func (l *EpisodeLinker) indexSeriesForSearch(ctx context.Context, seriesMetadataID int64, details *ports.TMDBSeriesDetails) {
	if l.searchRepository == nil {
		return
	}

	// Get credits from the database to build searchable cast/crew strings
	// Credits may not exist yet for new series (they're fetched on-demand when viewing series details)
	var castNames, crewNames []string
	if l.seriesCreditRepository != nil {
		credits, err := l.seriesCreditRepository.GetBySeriesMetadataID(ctx, seriesMetadataID)
		if err != nil {
			logger.Debug().Err(err).Int64("series_id", seriesMetadataID).Msg("No credits available for search indexing yet")
		} else {
			for _, credit := range credits {
				if credit.CreditType == domain.CreditTypeCast {
					castNames = append(castNames, credit.Name)
				} else {
					crewNames = append(crewNames, credit.Name)
				}
			}
		}
	}

	if err := l.searchRepository.IndexSeries(
		ctx,
		seriesMetadataID,
		details.Name,
		details.OriginalName,
		strings.Join(castNames, " "),
		strings.Join(crewNames, " "),
	); err != nil {
		logger.Warn().Err(err).Int64("series_id", seriesMetadataID).Msg("Failed to index series for search")
	}
}
