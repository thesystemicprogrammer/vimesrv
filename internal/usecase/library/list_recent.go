package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ListRecentInput contains the input parameters for listing recently added media
type ListRecentInput struct {
	Language string
}

// ListRecentOutput contains the recently added media list
type ListRecentOutput struct {
	Items []ports.RecentlyAddedItem `json:"items"`
}

// ListRecentUseCase retrieves recently added media
type ListRecentUseCase struct {
	libraryRepository ports.LibraryRepository
	libraryConfig     config.LibraryConfig
}

// NewListRecentUseCase creates a new ListRecentUseCase instance
func NewListRecentUseCase(libraryRepository ports.LibraryRepository, libraryConfig config.LibraryConfig) *ListRecentUseCase {
	return &ListRecentUseCase{
		libraryRepository: libraryRepository,
		libraryConfig:     libraryConfig,
	}
}

// Execute retrieves recently added media
func (uc *ListRecentUseCase) Execute(ctx context.Context, input ListRecentInput) (*ListRecentOutput, error) {
	language := input.Language
	if language == "" {
		language = "en"
	}

	limit := uc.libraryConfig.RecentlyAddedCount
	if limit <= 0 {
		limit = 20 // fallback default
	}

	items, err := uc.libraryRepository.ListRecentlyAdded(ctx, language, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list recently added: %w", err)
	}

	return &ListRecentOutput{
		Items: items,
	}, nil
}
