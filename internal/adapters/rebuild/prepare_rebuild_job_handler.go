package rebuild

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/rebuild"
)

// NewPrepareRebuildJobHandler creates a job handler for periodic rebuild export jobs.
// This handler is registered with the job manager to process "prepare_rebuild" job types.
// It exports users and metadata links to rebuild.json for backup/recovery purposes.
func NewPrepareRebuildJobHandler(useCase *rebuild.PrepareUseCase) ports.JobHandler {
	return func(ctx context.Context, job *domain.Job) error {
		return useCase.Execute(ctx)
	}
}
