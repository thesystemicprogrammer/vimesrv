package job

import (
	"context"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ListJobsInput defines the input for listing jobs
type ListJobsInput struct {
	Statuses            []string // Filter by status (queued, running, succeeded, dead)
	Types               []string // Filter by job type
	IncludeOlderThan24h bool     // If false, only include jobs from last 24h (for completed/failed)
	Limit               int      // Maximum number of results (default 100)
	Offset              int      // Offset for pagination
}

// ListJobsOutput defines the output for listing jobs
type ListJobsOutput struct {
	Jobs  []*domain.Job
	Total int
}

// ListJobsUseCase handles listing jobs with filtering
type ListJobsUseCase struct {
	jobRepo ports.JobRepository
}

// NewListJobsUseCase creates a new ListJobsUseCase
func NewListJobsUseCase(jobRepo ports.JobRepository) *ListJobsUseCase {
	return &ListJobsUseCase{
		jobRepo: jobRepo,
	}
}

// Execute lists jobs based on the input criteria
func (uc *ListJobsUseCase) Execute(ctx context.Context, input ListJobsInput) (*ListJobsOutput, error) {
	// Set defaults
	limit := input.Limit
	if limit <= 0 {
		limit = 100
	}

	// Build filter
	filter := ports.JobListFilter{
		Statuses: input.Statuses,
		Types:    input.Types,
		Limit:    limit,
		Offset:   input.Offset,
	}

	// For non-active jobs, limit to last 24 hours unless explicitly requested
	if !input.IncludeOlderThan24h {
		filter.Since = time.Now().Add(-24 * time.Hour)
	}

	result, err := uc.jobRepo.ListJobs(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &ListJobsOutput{
		Jobs:  result.Jobs,
		Total: result.Total,
	}, nil
}
