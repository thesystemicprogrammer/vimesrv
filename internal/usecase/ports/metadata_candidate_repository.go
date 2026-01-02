package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// MetadataCandidateRepository defines the interface for metadata candidate persistence
type MetadataCandidateRepository interface {
	// Create inserts a new metadata candidate
	Create(ctx context.Context, candidate *domain.MetadataCandidate) error

	// Get retrieves a candidate by its ID
	Get(ctx context.Context, id int64) (*domain.MetadataCandidate, error)

	// ListByMediaFileID retrieves all candidates for a given media file
	ListByMediaFileID(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error)

	// ListPendingByMediaFileID retrieves pending candidates for a given media file
	ListPendingByMediaFileID(ctx context.Context, mediaFileID string) ([]domain.MetadataCandidate, error)

	// Update updates an existing candidate
	Update(ctx context.Context, candidate *domain.MetadataCandidate) error

	// Delete removes a candidate
	Delete(ctx context.Context, id int64) error

	// DeleteByMediaFileID removes all candidates for a given media file
	DeleteByMediaFileID(ctx context.Context, mediaFileID string) error

	// MarkSelected marks a candidate as selected and rejects all others for the same media file
	MarkSelected(ctx context.Context, candidateID int64) error

	// RejectAll marks all candidates for a media file as rejected
	RejectAll(ctx context.Context, mediaFileID string) error

	// CreateBatch creates multiple candidates in a single transaction
	CreateBatch(ctx context.Context, candidates []domain.MetadataCandidate) error
}
