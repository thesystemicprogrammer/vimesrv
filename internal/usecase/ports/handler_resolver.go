package ports

import (
	"context"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

type JobHandler func(ctx context.Context, j *domain.Job) error

type HandlerResolver interface {
	Get(jobType string) (JobHandler, bool)
}
