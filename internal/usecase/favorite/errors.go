package favorite

import "errors"

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrInvalidMediaType  = errors.New("invalid media type, must be 'movie' or 'series'")
	ErrInvalidMetadataID = errors.New("invalid metadata ID")
)
