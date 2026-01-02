package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// MovieMetadataForTranslation contains info needed to fetch translations
type MovieMetadataForTranslation struct {
	ID            int64
	TMDBID        int
	OriginalTitle string
}

// MovieMetadataRepository defines the interface for movie metadata persistence
type MovieMetadataRepository interface {
	// Create inserts a new movie metadata record
	Create(ctx context.Context, metadata *domain.MovieMetadata) error

	// Get retrieves a movie metadata by its internal ID
	Get(ctx context.Context, id int64) (*domain.MovieMetadata, error)

	// GetByTMDBID retrieves a movie metadata by its TMDB ID
	GetByTMDBID(ctx context.Context, tmdbID int) (*domain.MovieMetadata, error)

	// Update updates an existing movie metadata record
	Update(ctx context.Context, metadata *domain.MovieMetadata) error

	// Delete removes a movie metadata record
	Delete(ctx context.Context, id int64) error

	// ExistsByTMDBID checks if a movie metadata with the given TMDB ID exists
	ExistsByTMDBID(ctx context.Context, tmdbID int) (bool, error)

	// CreateTranslation creates a new translation for a movie
	CreateTranslation(ctx context.Context, translation *domain.MovieMetadataTranslation) error

	// GetTranslation retrieves a specific translation for a movie
	GetTranslation(ctx context.Context, movieMetadataID int64, language string) (*domain.MovieMetadataTranslation, error)

	// GetTranslations retrieves all translations for a movie
	GetTranslations(ctx context.Context, movieMetadataID int64) ([]domain.MovieMetadataTranslation, error)

	// UpsertTranslation creates or updates a translation
	UpsertTranslation(ctx context.Context, translation *domain.MovieMetadataTranslation) error

	// ListIDsWithoutTranslation returns movie metadata IDs that don't have a translation for the given language
	ListIDsWithoutTranslation(ctx context.Context, language string) ([]MovieMetadataForTranslation, error)

	// SetFullCreditsFetched marks that full credits have been fetched from TMDB for this movie
	SetFullCreditsFetched(ctx context.Context, movieMetadataID int64) error

	// HasFullCreditsFetched checks if full credits have been fetched from TMDB for this movie
	HasFullCreditsFetched(ctx context.Context, movieMetadataID int64) (bool, error)

	// GetTMDBIDByID retrieves the TMDB ID for a movie metadata record
	GetTMDBIDByID(ctx context.Context, movieMetadataID int64) (int, error)
}
