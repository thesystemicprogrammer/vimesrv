package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/library"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
)

func registerJobs(useCases *UseCases, adapters *Adapters) {
	// Scan Library
	scanLibraryJobHandler := library.NewScanLibraryJobHandler(useCases.ScanLibraryUseCase)
	adapters.HandlerRegistry.Register(shared.JobTypeScanLibrary, scanLibraryJobHandler)
}
