package ports

type ScanLibraryRepository interface {
	Scan(libraryPath string) error
}
