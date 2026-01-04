package metadata

import (
	"context"
	"fmt"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/metadata/linker"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// LinkMetadataInput contains the input parameters for linking a media file to metadata
type LinkMetadataInput struct {
	MediaID     string
	CandidateID int64 // If > 0, link using an existing candidate
}

// LinkMetadataOutput contains the result of the link operation
type LinkMetadataOutput struct {
	MediaID      string
	MetadataType string
	Title        string
	Message      string
}

// LinkMetadataUseCase handles linking a media file to metadata (user selection from candidates)
type LinkMetadataUseCase struct {
	movieLinker                 *linker.MovieLinker
	episodeLinker               *linker.EpisodeLinker
	mediaRepository             ports.MediaRepository
	metadataCandidateRepository ports.MetadataCandidateRepository
	searchRepository            ports.SearchRepository
	movieCreditRepository       ports.MovieCreditRepository
	seriesCreditRepository      ports.SeriesCreditRepository
}

// NewLinkMetadataUseCase creates a new instance of LinkMetadataUseCase
func NewLinkMetadataUseCase(
	movieLinker *linker.MovieLinker,
	episodeLinker *linker.EpisodeLinker,
	mediaRepository ports.MediaRepository,
	metadataCandidateRepository ports.MetadataCandidateRepository,
	searchRepository ports.SearchRepository,
	movieCreditRepository ports.MovieCreditRepository,
	seriesCreditRepository ports.SeriesCreditRepository,
) *LinkMetadataUseCase {
	return &LinkMetadataUseCase{
		movieLinker:                 movieLinker,
		episodeLinker:               episodeLinker,
		mediaRepository:             mediaRepository,
		metadataCandidateRepository: metadataCandidateRepository,
		searchRepository:            searchRepository,
		movieCreditRepository:       movieCreditRepository,
		seriesCreditRepository:      seriesCreditRepository,
	}
}

// Execute links a media file to metadata using a candidate
func (uc *LinkMetadataUseCase) Execute(ctx context.Context, input LinkMetadataInput) (*LinkMetadataOutput, error) {
	logger.Info().
		Str("media_id", input.MediaID).
		Int64("candidate_id", input.CandidateID).
		Msg("Linking metadata to media file")

	// Get the media file
	media, err := uc.mediaRepository.Get(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media file: %w", err)
	}
	if media == nil {
		return nil, fmt.Errorf("media file not found: %s", input.MediaID)
	}

	// Get the candidate
	candidate, err := uc.metadataCandidateRepository.Get(ctx, input.CandidateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate: %w", err)
	}
	if candidate == nil {
		return nil, fmt.Errorf("candidate not found: %d", input.CandidateID)
	}

	// Verify candidate belongs to this media file
	if candidate.MediaFileID != input.MediaID {
		return nil, fmt.Errorf("candidate %d does not belong to media file %s", input.CandidateID, input.MediaID)
	}

	// Mark candidate as selected and reject others
	if err := uc.metadataCandidateRepository.MarkSelected(ctx, input.CandidateID); err != nil {
		return nil, fmt.Errorf("failed to mark candidate as selected: %w", err)
	}

	// Link based on candidate type
	var output *LinkMetadataOutput
	if candidate.IsMovie() {
		output, err = uc.linkToMovie(ctx, media, candidate)
	} else {
		output, err = uc.linkToSeries(ctx, media, candidate)
	}

	if err != nil {
		return nil, err
	}

	return output, nil
}

// linkToMovie uses MovieLinker to create/fetch movie metadata and link the media file
func (uc *LinkMetadataUseCase) linkToMovie(ctx context.Context, media *domain.MediaFile, candidate *domain.MetadataCandidate) (*LinkMetadataOutput, error) {
	logger.Debug().
		Str("media_id", media.ID).
		Int("tmdb_id", candidate.TMDBID).
		Msg("Linking media to movie")

	// Use MovieLinker to handle all movie metadata creation (including credits, certifications, collection)
	result, err := uc.movieLinker.Link(ctx, candidate.TMDBID)
	if err != nil {
		return nil, fmt.Errorf("failed to link movie: %w", err)
	}

	// Link media file to movie
	media.LinkToMovie(result.MovieMetadata.ID)
	media.SetEnrichmentLinked()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	// Index movie for full-text search
	uc.indexMovieForSearch(ctx, media.ID, result.MovieMetadata.ID, result.Details.Title, result.Details.OriginalTitle)

	return &LinkMetadataOutput{
		MediaID:      media.ID,
		MetadataType: domain.MetadataTypeMovie,
		Title:        result.Details.Title,
		Message:      fmt.Sprintf("Linked to movie: %s", result.Details.Title),
	}, nil
}

// linkToSeries uses EpisodeLinker to create/fetch series metadata and link the media file to an episode
func (uc *LinkMetadataUseCase) linkToSeries(ctx context.Context, media *domain.MediaFile, candidate *domain.MetadataCandidate) (*LinkMetadataOutput, error) {
	seasonNumber := 1
	episodeNumber := 1
	if candidate.SeasonNumber != nil {
		seasonNumber = *candidate.SeasonNumber
	}
	if candidate.EpisodeNumber != nil {
		episodeNumber = *candidate.EpisodeNumber
	}

	logger.Debug().
		Str("media_id", media.ID).
		Int("tmdb_id", candidate.TMDBID).
		Int("season", seasonNumber).
		Int("episode", episodeNumber).
		Msg("Linking media to series episode")

	// Use EpisodeLinker to handle all episode metadata creation
	result, err := uc.episodeLinker.Link(ctx, candidate.TMDBID, seasonNumber, episodeNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to link episode: %w", err)
	}

	// Link media file to episode
	media.LinkToEpisode(result.EpisodeMetadata.ID)
	media.SetEnrichmentLinked()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	// Index series for full-text search (if it was newly created)
	if result.SeriesCreated {
		uc.indexSeriesForSearch(ctx, result.SeriesMetadata.ID, result.SeriesDetails.Name, result.SeriesDetails.OriginalName)
	}

	seriesName := result.SeriesDetails.Name
	return &LinkMetadataOutput{
		MediaID:      media.ID,
		MetadataType: domain.MetadataTypeEpisode,
		Title:        fmt.Sprintf("%s S%02dE%02d", seriesName, seasonNumber, episodeNumber),
		Message:      fmt.Sprintf("Linked to %s S%02dE%02d", seriesName, seasonNumber, episodeNumber),
	}, nil
}

// indexMovieForSearch adds the movie to the FTS search index
func (uc *LinkMetadataUseCase) indexMovieForSearch(ctx context.Context, mediaID string, movieMetadataID int64, title, originalTitle string) {
	if uc.searchRepository == nil {
		return
	}

	// Get credits from the database to build searchable cast/crew strings
	var castNames, crewNames []string
	if uc.movieCreditRepository != nil {
		credits, err := uc.movieCreditRepository.GetByMovieMetadataID(ctx, movieMetadataID)
		if err != nil {
			logger.Debug().Err(err).Int64("movie_id", movieMetadataID).Msg("No credits available for search indexing")
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

	if err := uc.searchRepository.IndexMovie(
		ctx,
		mediaID,
		movieMetadataID,
		title,
		originalTitle,
		strings.Join(castNames, " "),
		strings.Join(crewNames, " "),
	); err != nil {
		logger.Warn().Err(err).Int64("movie_id", movieMetadataID).Msg("Failed to index movie for search")
	}
}

// indexSeriesForSearch adds the series to the FTS search index
func (uc *LinkMetadataUseCase) indexSeriesForSearch(ctx context.Context, seriesMetadataID int64, name, originalName string) {
	if uc.searchRepository == nil {
		return
	}

	// Get credits from the database to build searchable cast/crew strings
	var castNames, crewNames []string
	if uc.seriesCreditRepository != nil {
		credits, err := uc.seriesCreditRepository.GetBySeriesMetadataID(ctx, seriesMetadataID)
		if err != nil {
			logger.Debug().Err(err).Int64("series_id", seriesMetadataID).Msg("No credits available for search indexing")
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

	if err := uc.searchRepository.IndexSeries(
		ctx,
		seriesMetadataID,
		name,
		originalName,
		strings.Join(castNames, " "),
		strings.Join(crewNames, " "),
	); err != nil {
		logger.Warn().Err(err).Int64("series_id", seriesMetadataID).Msg("Failed to index series for search")
	}
}
