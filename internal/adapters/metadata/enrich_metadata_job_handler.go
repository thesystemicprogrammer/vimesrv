package metadata

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/metadata"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// EnrichMetadataJobPayload represents the payload structure for metadata enrichment jobs
type EnrichMetadataJobPayload struct {
	MediaID  string `json:"media_id"`
	Filename string `json:"filename"`
}

// NewEnrichMetadataJobHandler creates a job handler for metadata enrichment jobs.
// This handler is registered with the job manager to process "enrich_metadata" job types.
func NewEnrichMetadataJobHandler(useCase *metadata.EnrichMediaFileUseCase) ports.JobHandler {
	return func(ctx context.Context, job *domain.Job) error {
		// Parse job payload
		var payload EnrichMetadataJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to parse enrich metadata job payload: %w", err)
		}

		// Validate payload
		if payload.MediaID == "" {
			return fmt.Errorf("media_id is required in job payload")
		}

		// Execute enrich media file use case
		input := metadata.EnrichMediaFileInput{
			MediaID: payload.MediaID,
		}

		_, err := useCase.Execute(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to enrich media file: %w", err)
		}

		return nil
	}
}
