package shared

import "errors"

var (
	// ErrNotFound is returned when a resource is not found
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists is returned when a resource already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")

	// ErrInvalidState is returned when an operation cannot be performed in the current state
	ErrInvalidState = errors.New("invalid state")

	// ErrMediaNotFound is returned when a media file is not found
	ErrMediaNotFound = errors.New("media not found")

	// ErrTranscodeJobNotFound is returned when a transcode job is not found
	ErrTranscodeJobNotFound = errors.New("transcode job not found")

	// ErrInvalidMediaPath is returned when a media path is invalid
	ErrInvalidMediaPath = errors.New("invalid media path")

	// ErrUnsupportedFormat is returned when a media format is not supported
	ErrUnsupportedFormat = errors.New("unsupported media format")

	// ErrTranscodingFailed is returned when transcoding fails
	ErrTranscodingFailed = errors.New("transcoding failed")

	// ErrTranscodingInProgress is returned when attempting an operation on a job that is still processing
	ErrTranscodingInProgress = errors.New("transcoding in progress")

	// ErrMediaHasRunningJobs is returned when attempting to delete media that has running transcode jobs
	ErrMediaHasRunningJobs = errors.New("media has running transcode jobs")
)
