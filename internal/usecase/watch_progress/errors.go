package watch_progress

import "errors"

var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrInvalidDuration = errors.New("invalid duration")
	ErrInvalidPosition = errors.New("invalid position")
	ErrNotFound        = errors.New("watch progress not found")
)
