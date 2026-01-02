package library

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// ListUnmatchedInput contains the input parameters for listing unmatched media
type ListUnmatchedInput struct {
	Page    int
	PerPage int
}

// ListUnmatchedOutput contains the paginated unmatched media list
type ListUnmatchedOutput struct {
	Items   []ports.UnmatchedMediaSummary `json:"items"`
	Total   int                           `json:"total"`
	Page    int                           `json:"page"`
	PerPage int                           `json:"per_page"`
}

// ListUnmatchedUseCase retrieves media files without metadata
type ListUnmatchedUseCase struct {
	libraryRepository ports.LibraryRepository
}

// NewListUnmatchedUseCase creates a new ListUnmatchedUseCase instance
func NewListUnmatchedUseCase(libraryRepository ports.LibraryRepository) *ListUnmatchedUseCase {
	return &ListUnmatchedUseCase{
		libraryRepository: libraryRepository,
	}
}

// Execute retrieves media files without metadata
func (uc *ListUnmatchedUseCase) Execute(ctx context.Context, input ListUnmatchedInput) (*ListUnmatchedOutput, error) {
	// Set defaults
	page := input.Page
	perPage := input.PerPage

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage

	items, total, err := uc.libraryRepository.ListUnmatched(ctx, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list unmatched media: %w", err)
	}

	return &ListUnmatchedOutput{
		Items:   items,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}
