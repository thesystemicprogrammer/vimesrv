package metadata

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// FetchTranslationsJobPayload represents the payload structure for translation fetch jobs
type FetchTranslationsJobPayload struct {
	Language string `json:"language"`
}

// FetchTranslationsJobHandler handles fetch_translations job execution
type FetchTranslationsJobHandler struct {
	movieMetadataRepo   ports.MovieMetadataRepository
	seriesMetadataRepo  ports.SeriesMetadataRepository
	seasonMetadataRepo  ports.SeasonMetadataRepository
	episodeMetadataRepo ports.EpisodeMetadataRepository
	tmdbClient          ports.TMDBClient
	imageDownloader     ports.ImageDownloader
	downloadImages      bool
}

// NewFetchTranslationsJobHandler creates a new FetchTranslationsJobHandler
func NewFetchTranslationsJobHandler(
	movieMetadataRepo ports.MovieMetadataRepository,
	seriesMetadataRepo ports.SeriesMetadataRepository,
	seasonMetadataRepo ports.SeasonMetadataRepository,
	episodeMetadataRepo ports.EpisodeMetadataRepository,
	tmdbClient ports.TMDBClient,
	imageDownloader ports.ImageDownloader,
	downloadImages bool,
) ports.JobHandler {
	h := &FetchTranslationsJobHandler{
		movieMetadataRepo:   movieMetadataRepo,
		seriesMetadataRepo:  seriesMetadataRepo,
		seasonMetadataRepo:  seasonMetadataRepo,
		episodeMetadataRepo: episodeMetadataRepo,
		tmdbClient:          tmdbClient,
		imageDownloader:     imageDownloader,
		downloadImages:      downloadImages,
	}
	return h.Handle
}

// Handle processes a fetch_translations job
func (h *FetchTranslationsJobHandler) Handle(ctx context.Context, job *domain.Job) error {
	// Parse job payload
	var payload FetchTranslationsJobPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to parse fetch translations job payload: %w", err)
	}

	if payload.Language == "" {
		return fmt.Errorf("language is required in job payload")
	}

	logger.Info().
		Str("language", payload.Language).
		Msg("starting translation fetch job")

	// Fetch movie translations
	if err := h.fetchMovieTranslations(ctx, payload.Language); err != nil {
		logger.Error().Err(err).Msg("failed to fetch movie translations")
		// Continue with other types even if one fails
	}

	// Fetch series translations
	if err := h.fetchSeriesTranslations(ctx, payload.Language); err != nil {
		logger.Error().Err(err).Msg("failed to fetch series translations")
	}

	// Fetch season translations
	if err := h.fetchSeasonTranslations(ctx, payload.Language); err != nil {
		logger.Error().Err(err).Msg("failed to fetch season translations")
	}

	// Fetch episode translations
	if err := h.fetchEpisodeTranslations(ctx, payload.Language); err != nil {
		logger.Error().Err(err).Msg("failed to fetch episode translations")
	}

	logger.Info().
		Str("language", payload.Language).
		Msg("translation fetch job completed")

	return nil
}

// fetchMovieTranslations fetches translations for all movies missing the given language
func (h *FetchTranslationsJobHandler) fetchMovieTranslations(ctx context.Context, language string) error {
	movies, err := h.movieMetadataRepo.ListIDsWithoutTranslation(ctx, language)
	if err != nil {
		return fmt.Errorf("list movies without translation: %w", err)
	}

	logger.Debug().
		Int("count", len(movies)).
		Str("language", language).
		Msg("fetching movie translations")

	for _, m := range movies {
		details, err := h.tmdbClient.GetMovieDetails(ctx, m.TMDBID, language)
		if err != nil {
			logger.Warn().
				Err(err).
				Int64("movie_id", m.ID).
				Int("tmdb_id", m.TMDBID).
				Msg("failed to fetch movie details from TMDB")
			continue
		}

		// Store translation if we have any translated content
		// (title may be the same but overview/tagline might be translated)
		if details.Title != "" {
			translation := domain.NewMovieMetadataTranslation(m.ID, language, details.Title)
			translation.Tagline = details.Tagline
			translation.Overview = details.Overview
			translation.PosterPath = details.PosterPath
			translation.BackdropPath = details.BackdropPath

			// Download images if enabled
			if h.downloadImages && h.imageDownloader != nil {
				if details.PosterPath != "" {
					if _, err := h.imageDownloader.DownloadImageWithLanguage(
						ctx, details.PosterPath, ports.ImageTypeMoviePoster, m.TMDBID, language,
					); err != nil {
						logger.Warn().
							Err(err).
							Int64("movie_id", m.ID).
							Msg("failed to download movie poster")
					}
				}

				if details.BackdropPath != "" {
					if _, err := h.imageDownloader.DownloadImageWithLanguage(
						ctx, details.BackdropPath, ports.ImageTypeMovieBackdrop, m.TMDBID, language,
					); err != nil {
						logger.Warn().
							Err(err).
							Int64("movie_id", m.ID).
							Msg("failed to download movie backdrop")
					}
				}
			}

			if err := h.movieMetadataRepo.UpsertTranslation(ctx, translation); err != nil {
				logger.Warn().
					Err(err).
					Int64("movie_id", m.ID).
					Msg("failed to upsert movie translation")
			} else {
				logger.Debug().
					Int64("movie_id", m.ID).
					Str("title", details.Title).
					Msg("saved movie translation")
			}
		}
	}

	return nil
}

// fetchSeriesTranslations fetches translations for all series missing the given language
func (h *FetchTranslationsJobHandler) fetchSeriesTranslations(ctx context.Context, language string) error {
	seriesList, err := h.seriesMetadataRepo.ListIDsWithoutTranslation(ctx, language)
	if err != nil {
		return fmt.Errorf("list series without translation: %w", err)
	}

	logger.Debug().
		Int("count", len(seriesList)).
		Str("language", language).
		Msg("fetching series translations")

	for _, s := range seriesList {
		details, err := h.tmdbClient.GetSeriesDetails(ctx, s.TMDBID, language)
		if err != nil {
			logger.Warn().
				Err(err).
				Int64("series_id", s.ID).
				Int("tmdb_id", s.TMDBID).
				Msg("failed to fetch series details from TMDB")
			continue
		}

		// Store translation if we have any translated content
		// (name may be the same but overview might be translated)
		if details.Name != "" {
			translation := domain.NewSeriesMetadataTranslation(s.ID, language, details.Name)
			translation.Overview = details.Overview
			translation.PosterPath = details.PosterPath
			translation.BackdropPath = details.BackdropPath

			// Download images if enabled
			if h.downloadImages && h.imageDownloader != nil {
				if details.PosterPath != "" {
					if _, err := h.imageDownloader.DownloadImageWithLanguage(
						ctx, details.PosterPath, ports.ImageTypeSeriesPoster, s.TMDBID, language,
					); err != nil {
						logger.Warn().
							Err(err).
							Int64("series_id", s.ID).
							Msg("failed to download series poster")
					}
				}

				if details.BackdropPath != "" {
					if _, err := h.imageDownloader.DownloadImageWithLanguage(
						ctx, details.BackdropPath, ports.ImageTypeSeriesBackdrop, s.TMDBID, language,
					); err != nil {
						logger.Warn().
							Err(err).
							Int64("series_id", s.ID).
							Msg("failed to download series backdrop")
					}
				}
			}

			if err := h.seriesMetadataRepo.UpsertTranslation(ctx, translation); err != nil {
				logger.Warn().
					Err(err).
					Int64("series_id", s.ID).
					Msg("failed to upsert series translation")
			} else {
				logger.Debug().
					Int64("series_id", s.ID).
					Str("name", details.Name).
					Msg("saved series translation")
			}
		}
	}

	return nil
}

// fetchSeasonTranslations fetches translations for all seasons missing the given language
func (h *FetchTranslationsJobHandler) fetchSeasonTranslations(ctx context.Context, language string) error {
	seasons, err := h.seasonMetadataRepo.ListIDsWithoutTranslation(ctx, language)
	if err != nil {
		return fmt.Errorf("list seasons without translation: %w", err)
	}

	logger.Debug().
		Int("count", len(seasons)).
		Str("language", language).
		Msg("fetching season translations")

	for _, s := range seasons {
		details, err := h.tmdbClient.GetSeasonDetails(ctx, s.SeriesTMDBID, s.SeasonNumber, language)
		if err != nil {
			logger.Warn().
				Err(err).
				Int64("season_id", s.ID).
				Int("series_tmdb_id", s.SeriesTMDBID).
				Int("season_number", s.SeasonNumber).
				Msg("failed to fetch season details from TMDB")
			continue
		}

		// Store season translation if name is not empty
		// Note: Season names are often just "Season X" so we store regardless
		if details.Name != "" {
			translation := domain.NewSeasonMetadataTranslation(s.ID, language, details.Name)
			translation.Overview = details.Overview
			translation.PosterPath = details.PosterPath

			// Download images if enabled
			if h.downloadImages && h.imageDownloader != nil && details.PosterPath != "" {
				if _, err := h.imageDownloader.DownloadSeasonImageWithLanguage(
					ctx, details.PosterPath, s.SeriesTMDBID, s.SeasonNumber, language,
				); err != nil {
					logger.Warn().
						Err(err).
						Int64("season_id", s.ID).
						Msg("failed to download season poster")
				}
			}

			if err := h.seasonMetadataRepo.UpsertTranslation(ctx, translation); err != nil {
				logger.Warn().
					Err(err).
					Int64("season_id", s.ID).
					Msg("failed to upsert season translation")
			} else {
				logger.Debug().
					Int64("season_id", s.ID).
					Str("name", details.Name).
					Msg("saved season translation")
			}
		}
	}

	return nil
}

// fetchEpisodeTranslations fetches translations for all episodes missing the given language
func (h *FetchTranslationsJobHandler) fetchEpisodeTranslations(ctx context.Context, language string) error {
	episodes, err := h.episodeMetadataRepo.ListIDsWithoutTranslation(ctx, language)
	if err != nil {
		return fmt.Errorf("list episodes without translation: %w", err)
	}

	logger.Debug().
		Int("count", len(episodes)).
		Str("language", language).
		Msg("fetching episode translations")

	for _, e := range episodes {
		details, err := h.tmdbClient.GetEpisodeDetails(ctx, e.SeriesTMDBID, e.SeasonNumber, e.EpisodeNumber, language)
		if err != nil {
			logger.Warn().
				Err(err).
				Int64("episode_id", e.ID).
				Int("series_tmdb_id", e.SeriesTMDBID).
				Int("season_number", e.SeasonNumber).
				Int("episode_number", e.EpisodeNumber).
				Msg("failed to fetch episode details from TMDB")
			continue
		}

		// Store episode translation if name is not empty
		if details.Name != "" {
			translation := domain.NewEpisodeMetadataTranslation(e.ID, language, details.Name)
			translation.Overview = details.Overview
			translation.StillPath = details.StillPath

			// Download images if enabled
			if h.downloadImages && h.imageDownloader != nil && details.StillPath != "" {
				if _, err := h.imageDownloader.DownloadEpisodeImageWithLanguage(
					ctx, details.StillPath, e.SeriesTMDBID, e.SeasonNumber, e.EpisodeNumber, language,
				); err != nil {
					logger.Warn().
						Err(err).
						Int64("episode_id", e.ID).
						Msg("failed to download episode still")
				}
			}

			if err := h.episodeMetadataRepo.UpsertTranslation(ctx, translation); err != nil {
				logger.Warn().
					Err(err).
					Int64("episode_id", e.ID).
					Msg("failed to upsert episode translation")
			} else {
				logger.Debug().
					Int64("episode_id", e.ID).
					Str("name", details.Name).
					Msg("saved episode translation")
			}
		}
	}

	return nil
}
