package library

import "github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"

type ScanLibraryRepository struct{}

func NewScanLibraryScanRepository() *ScanLibraryRepository {
	return &ScanLibraryRepository{}
}

func (repo *ScanLibraryRepository) Scan(libraryPath string) error {
	logger.Info().Str("path", libraryPath).Msg("YES SCANNING WAS CALLED")
	return nil
}
