package metadata

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// GetCandidatesInput contains the input parameters for getting candidates
type GetCandidatesInput struct {
	MediaID     string
	PendingOnly bool // If true, only return pending candidates
}

// CandidateDTO represents a candidate in the API response
type CandidateDTO struct {
	ID              int64  `json:"id"`
	TMDBID          int    `json:"tmdb_id"`
	CandidateType   string `json:"candidate_type"` // "movie" or "series"
	Title           string `json:"title"`
	ReleaseDate     string `json:"release_date,omitempty"`
	Overview        string `json:"overview,omitempty"`
	PosterPath      string `json:"poster_path,omitempty"`
	PosterURL       string `json:"poster_url,omitempty"` // Full URL for display
	ConfidenceScore int    `json:"confidence_score"`
	SeasonNumber    *int   `json:"season_number,omitempty"`
	EpisodeNumber   *int   `json:"episode_number,omitempty"`
	Status          string `json:"status"`
}

// GetCandidatesOutput contains the candidates for a media file
type GetCandidatesOutput struct {
	MediaID          string         `json:"media_id"`
	EnrichmentStatus string         `json:"enrichment_status"`
	Candidates       []CandidateDTO `json:"candidates"`
	Count            int            `json:"count"`
}

// GetCandidatesUseCase retrieves metadata candidates for a media file
type GetCandidatesUseCase struct {
	config                      config.TMDBConfig
	tmdbClient                  ports.TMDBClient
	mediaRepository             ports.MediaRepository
	metadataCandidateRepository ports.MetadataCandidateRepository
}

// NewGetCandidatesUseCase creates a new instance of GetCandidatesUseCase
func NewGetCandidatesUseCase(
	config config.TMDBConfig,
	tmdbClient ports.TMDBClient,
	mediaRepository ports.MediaRepository,
	metadataCandidateRepository ports.MetadataCandidateRepository,
) *GetCandidatesUseCase {
	return &GetCandidatesUseCase{
		config:                      config,
		tmdbClient:                  tmdbClient,
		mediaRepository:             mediaRepository,
		metadataCandidateRepository: metadataCandidateRepository,
	}
}

// Execute retrieves candidates for a media file
func (uc *GetCandidatesUseCase) Execute(ctx context.Context, input GetCandidatesInput) (*GetCandidatesOutput, error) {
	logger.Debug().
		Str("media_id", input.MediaID).
		Bool("pending_only", input.PendingOnly).
		Msg("Getting metadata candidates")

	// Get the media file to verify it exists and get its status
	media, err := uc.mediaRepository.Get(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media file: %w", err)
	}
	if media == nil {
		return nil, fmt.Errorf("media file not found: %s", input.MediaID)
	}

	// Get candidates
	var candidates []domain.MetadataCandidate
	if input.PendingOnly {
		candidates, err = uc.metadataCandidateRepository.ListPendingByMediaFileID(ctx, input.MediaID)
	} else {
		candidates, err = uc.metadataCandidateRepository.ListByMediaFileID(ctx, input.MediaID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get candidates: %w", err)
	}

	// Convert to DTOs
	dtos := make([]CandidateDTO, len(candidates))
	for i, c := range candidates {
		dtos[i] = CandidateDTO{
			ID:              c.ID,
			TMDBID:          c.TMDBID,
			CandidateType:   c.CandidateType,
			Title:           c.Title,
			ReleaseDate:     c.ReleaseDate,
			Overview:        c.Overview,
			PosterPath:      c.PosterPath,
			PosterURL:       uc.tmdbClient.GetImageURL(c.PosterPath, uc.config.PosterSize),
			ConfidenceScore: c.ConfidenceScore,
			SeasonNumber:    c.SeasonNumber,
			EpisodeNumber:   c.EpisodeNumber,
			Status:          c.Status,
		}
	}

	return &GetCandidatesOutput{
		MediaID:          input.MediaID,
		EnrichmentStatus: media.EnrichmentStatus,
		Candidates:       dtos,
		Count:            len(dtos),
	}, nil
}
