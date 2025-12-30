package library

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/library"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// NewScanLibraryJobHandler creates a job handler for library scanning jobs.
// This handler is registered with the job manager to process "scan_library" job types.
func NewScanLibraryJobHandler(useCase *library.ScanLibraryUseCase) ports.JobHandler {
	return func(ctx context.Context, job *domain.Job) error {
		// No payload needed - the use case uses config.Media.LibraryPath
		return useCase.Execute()
	}
}
