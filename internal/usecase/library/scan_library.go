package library

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type ScanLibraryUseCase struct {
	config                config.MediaConfig
	scanLibraryRepository ports.ScanLibraryRepository
}

func NewScanLibraryUseCase(config config.MediaConfig, scanLibraryRepository ports.ScanLibraryRepository) *ScanLibraryUseCase {
	return &ScanLibraryUseCase{
		config:                config,
		scanLibraryRepository: scanLibraryRepository,
	}
}

func (uc *ScanLibraryUseCase) Execute() error {
	return uc.scanLibraryRepository.Scan(uc.config.LibraryPath)
}
