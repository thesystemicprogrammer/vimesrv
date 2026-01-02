package metadata

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// SkipEnrichmentInput contains the input parameters for skipping enrichment
type SkipEnrichmentInput struct {
	MediaID string
}

// SkipEnrichmentOutput contains the result of skipping enrichment
type SkipEnrichmentOutput struct {
	MediaID string
	Message string
}

// SkipEnrichmentUseCase marks a media file as having skipped enrichment
type SkipEnrichmentUseCase struct {
	mediaRepository             ports.MediaRepository
	metadataCandidateRepository ports.MetadataCandidateRepository
}

// NewSkipEnrichmentUseCase creates a new instance of SkipEnrichmentUseCase
func NewSkipEnrichmentUseCase(
	mediaRepository ports.MediaRepository,
	metadataCandidateRepository ports.MetadataCandidateRepository,
) *SkipEnrichmentUseCase {
	return &SkipEnrichmentUseCase{
		mediaRepository:             mediaRepository,
		metadataCandidateRepository: metadataCandidateRepository,
	}
}

// Execute skips enrichment for a media file
func (uc *SkipEnrichmentUseCase) Execute(ctx context.Context, input SkipEnrichmentInput) (*SkipEnrichmentOutput, error) {
	logger.Info().
		Str("media_id", input.MediaID).
		Msg("Skipping metadata enrichment")

	// Get the media file
	media, err := uc.mediaRepository.Get(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media file: %w", err)
	}
	if media == nil {
		return nil, fmt.Errorf("media file not found: %s", input.MediaID)
	}

	// Reject any existing candidates
	if err := uc.metadataCandidateRepository.RejectAll(ctx, input.MediaID); err != nil {
		logger.Warn().Err(err).Str("media_id", input.MediaID).Msg("Failed to reject candidates")
	}

	// Mark media as skipped
	media.SetEnrichmentSkipped()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	return &SkipEnrichmentOutput{
		MediaID: input.MediaID,
		Message: "Enrichment skipped",
	}, nil
}

// ResetEnrichmentInput contains the input parameters for resetting enrichment
type ResetEnrichmentInput struct {
	MediaID string
}

// ResetEnrichmentOutput contains the result of resetting enrichment
type ResetEnrichmentOutput struct {
	MediaID string
	Message string
}

// ResetEnrichmentUseCase resets enrichment status to allow re-processing
type ResetEnrichmentUseCase struct {
	mediaRepository             ports.MediaRepository
	metadataCandidateRepository ports.MetadataCandidateRepository
}

// NewResetEnrichmentUseCase creates a new instance of ResetEnrichmentUseCase
func NewResetEnrichmentUseCase(
	mediaRepository ports.MediaRepository,
	metadataCandidateRepository ports.MetadataCandidateRepository,
) *ResetEnrichmentUseCase {
	return &ResetEnrichmentUseCase{
		mediaRepository:             mediaRepository,
		metadataCandidateRepository: metadataCandidateRepository,
	}
}

// Execute resets the enrichment status of a media file
func (uc *ResetEnrichmentUseCase) Execute(ctx context.Context, input ResetEnrichmentInput) (*ResetEnrichmentOutput, error) {
	logger.Info().
		Str("media_id", input.MediaID).
		Msg("Resetting metadata enrichment")

	// Get the media file
	media, err := uc.mediaRepository.Get(ctx, input.MediaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media file: %w", err)
	}
	if media == nil {
		return nil, fmt.Errorf("media file not found: %s", input.MediaID)
	}

	// Delete existing candidates
	if err := uc.metadataCandidateRepository.DeleteByMediaFileID(ctx, input.MediaID); err != nil {
		return nil, fmt.Errorf("failed to delete candidates: %w", err)
	}

	// Clear metadata link and reset status
	media.ClearMetadataLink()
	if err := uc.mediaRepository.Update(ctx, media); err != nil {
		return nil, fmt.Errorf("failed to update media file: %w", err)
	}

	return &ResetEnrichmentOutput{
		MediaID: input.MediaID,
		Message: "Enrichment reset to pending",
	}, nil
}
