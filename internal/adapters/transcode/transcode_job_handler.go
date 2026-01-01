package transcode

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/transcode"
)

// TranscodeJobPayload represents the payload structure for transcode jobs
type TranscodeJobPayload struct {
	TranscodeID string `json:"transcode_id"`
}

// NewTranscodeVideoJobHandler creates a job handler for video transcoding jobs.
// This handler is registered with the job manager to process "transcode_video" job types.
func NewTranscodeVideoJobHandler(useCase *transcode.ProcessTranscodeUseCase) ports.JobHandler {
	return func(ctx context.Context, job *domain.Job) error {
		// Parse job payload
		var payload TranscodeJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to parse transcode job payload: %w", err)
		}

		// Validate payload
		if payload.TranscodeID == "" {
			return fmt.Errorf("transcode_id is required in job payload")
		}

		// Execute transcode use case
		input := transcode.ProcessTranscodeInput{
			TranscodeID: payload.TranscodeID,
		}

		_, err := useCase.Execute(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to process transcode: %w", err)
		}

		return nil
	}
}
