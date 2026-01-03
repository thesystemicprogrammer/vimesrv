package media

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// MockMediaRepository for testing
type mockMediaRepository struct {
	GetFn func(ctx context.Context, id string) (*domain.MediaFile, error)
}

func (m *mockMediaRepository) Create(ctx context.Context, media *domain.MediaFile) error {
	return nil
}

func (m *mockMediaRepository) Get(ctx context.Context, id string) (*domain.MediaFile, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, id)
	}
	return nil, nil
}

func (m *mockMediaRepository) FindByFingerprint(ctx context.Context, fingerprint string) (*domain.MediaFile, error) {
	return nil, nil
}

func (m *mockMediaRepository) ExistsByFingerprint(ctx context.Context, fingerprint string) (bool, error) {
	return false, nil
}

func (m *mockMediaRepository) Update(ctx context.Context, media *domain.MediaFile) error {
	return nil
}

func (m *mockMediaRepository) List(ctx context.Context, page, perPage int) ([]*domain.MediaFile, int, error) {
	return nil, 0, nil
}

// MockTranscodeRepository for testing
type mockTranscodeRepository struct {
	GetByMediaIDFn func(ctx context.Context, mediaID string) ([]*domain.Transcode, error)
}

func (m *mockTranscodeRepository) Create(ctx context.Context, transcode *domain.Transcode) error {
	return nil
}

func (m *mockTranscodeRepository) Get(ctx context.Context, id string) (*domain.Transcode, error) {
	return nil, nil
}

func (m *mockTranscodeRepository) GetByMediaID(ctx context.Context, mediaID string) ([]*domain.Transcode, error) {
	if m.GetByMediaIDFn != nil {
		return m.GetByMediaIDFn(ctx, mediaID)
	}
	return nil, nil
}

func (m *mockTranscodeRepository) UpdateStatus(ctx context.Context, id string, status domain.TranscodeStatus) error {
	return nil
}

func (m *mockTranscodeRepository) MarkProcessing(ctx context.Context, id string) error {
	return nil
}

func (m *mockTranscodeRepository) MarkCompleted(ctx context.Context, id string, outputPath string) error {
	return nil
}

func (m *mockTranscodeRepository) MarkFailed(ctx context.Context, id string) error {
	return nil
}

func (m *mockTranscodeRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockTranscodeRepository) ListPending(ctx context.Context, limit int) ([]*domain.Transcode, error) {
	return nil, nil
}

// MockAudioStreamRepository for testing
type mockAudioStreamRepository struct {
	GetByMediaIDFn func(ctx context.Context, mediaID string) ([]*domain.AudioStream, error)
}

func (m *mockAudioStreamRepository) Create(ctx context.Context, stream *domain.AudioStream) error {
	return nil
}

func (m *mockAudioStreamRepository) GetByMediaID(ctx context.Context, mediaID string) ([]*domain.AudioStream, error) {
	if m.GetByMediaIDFn != nil {
		return m.GetByMediaIDFn(ctx, mediaID)
	}
	return nil, nil
}

func (m *mockAudioStreamRepository) DeleteByMediaID(ctx context.Context, mediaID string) error {
	return nil
}

// MockSubtitleStreamRepository for testing
type mockSubtitleStreamRepository struct {
	GetByMediaIDFn func(ctx context.Context, mediaID string) ([]*domain.SubtitleStream, error)
}

func (m *mockSubtitleStreamRepository) Create(ctx context.Context, stream *domain.SubtitleStream) error {
	return nil
}

func (m *mockSubtitleStreamRepository) GetByMediaID(ctx context.Context, mediaID string) ([]*domain.SubtitleStream, error) {
	if m.GetByMediaIDFn != nil {
		return m.GetByMediaIDFn(ctx, mediaID)
	}
	return nil, nil
}

func (m *mockSubtitleStreamRepository) DeleteByMediaID(ctx context.Context, mediaID string) error {
	return nil
}

func TestGetMediaUseCase_Execute_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mediaID := "test-media-id"

	expectedMedia := &domain.MediaFile{
		ID:       mediaID,
		FilePath: "/media/test.mp4",
		Title:    "Test Video",
	}

	expectedTranscodes := []*domain.Transcode{
		{
			ID:         "transcode-1",
			MediaID:    mediaID,
			Quality:    "720p",
			TrackType:  domain.TrackTypeVideo,
			TrackIndex: 0,
			Status:     domain.TranscodeCompleted,
		},
		{
			ID:         "transcode-2",
			MediaID:    mediaID,
			Quality:    "480p",
			TrackType:  domain.TrackTypeVideo,
			TrackIndex: 0,
			Status:     domain.TranscodeCompleted,
		},
	}

	expectedAudioStreams := []*domain.AudioStream{
		{
			ID:          1,
			MediaID:     mediaID,
			StreamIndex: 0,
			Codec:       "aac",
			Language:    "eng",
		},
	}

	expectedSubtitleStreams := []*domain.SubtitleStream{
		{
			ID:          1,
			MediaID:     mediaID,
			StreamIndex: 0,
			Codec:       "subrip",
			Language:    "eng",
		},
	}

	mediaRepo := &mockMediaRepository{
		GetFn: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return expectedMedia, nil
		},
	}

	transcodeRepo := &mockTranscodeRepository{
		GetByMediaIDFn: func(ctx context.Context, mediaID string) ([]*domain.Transcode, error) {
			return expectedTranscodes, nil
		},
	}

	audioStreamRepo := &mockAudioStreamRepository{
		GetByMediaIDFn: func(ctx context.Context, mediaID string) ([]*domain.AudioStream, error) {
			return expectedAudioStreams, nil
		},
	}

	subtitleStreamRepo := &mockSubtitleStreamRepository{
		GetByMediaIDFn: func(ctx context.Context, mediaID string) ([]*domain.SubtitleStream, error) {
			return expectedSubtitleStreams, nil
		},
	}

	useCase := NewGetMediaUseCase(mediaRepo, transcodeRepo, audioStreamRepo, subtitleStreamRepo)

	// Act
	result, err := useCase.Execute(ctx, GetMediaInput{MediaID: mediaID})

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expectedMedia, result.Media)
	assert.Equal(t, expectedTranscodes, result.Transcodes)
	assert.Equal(t, expectedAudioStreams, result.AudioStreams)
	assert.Equal(t, expectedSubtitleStreams, result.SubtitleStreams)
}

func TestGetMediaUseCase_Execute_EmptyMediaID(t *testing.T) {
	// Arrange
	ctx := context.Background()
	useCase := NewGetMediaUseCase(
		&mockMediaRepository{},
		&mockTranscodeRepository{},
		&mockAudioStreamRepository{},
		&mockSubtitleStreamRepository{},
	)

	// Act
	result, err := useCase.Execute(ctx, GetMediaInput{MediaID: ""})

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "media ID is required")
}

func TestGetMediaUseCase_Execute_MediaNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mediaID := "non-existent-id"

	mediaRepo := &mockMediaRepository{
		GetFn: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return nil, nil
		},
	}

	useCase := NewGetMediaUseCase(
		mediaRepo,
		&mockTranscodeRepository{},
		&mockAudioStreamRepository{},
		&mockSubtitleStreamRepository{},
	)

	// Act
	result, err := useCase.Execute(ctx, GetMediaInput{MediaID: mediaID})

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "media not found")
}

func TestGetMediaUseCase_Execute_MediaRepositoryError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mediaID := "test-media-id"
	expectedError := errors.New("database connection failed")

	mediaRepo := &mockMediaRepository{
		GetFn: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return nil, expectedError
		},
	}

	useCase := NewGetMediaUseCase(
		mediaRepo,
		&mockTranscodeRepository{},
		&mockAudioStreamRepository{},
		&mockSubtitleStreamRepository{},
	)

	// Act
	result, err := useCase.Execute(ctx, GetMediaInput{MediaID: mediaID})

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get media")
}

func TestGetMediaUseCase_Execute_TranscodeRepositoryError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mediaID := "test-media-id"
	expectedError := errors.New("transcode query failed")

	mediaRepo := &mockMediaRepository{
		GetFn: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return &domain.MediaFile{ID: mediaID}, nil
		},
	}

	transcodeRepo := &mockTranscodeRepository{
		GetByMediaIDFn: func(ctx context.Context, mediaID string) ([]*domain.Transcode, error) {
			return nil, expectedError
		},
	}

	useCase := NewGetMediaUseCase(
		mediaRepo,
		transcodeRepo,
		&mockAudioStreamRepository{},
		&mockSubtitleStreamRepository{},
	)

	// Act
	result, err := useCase.Execute(ctx, GetMediaInput{MediaID: mediaID})

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get transcodes")
}

func TestGetMediaUseCase_Execute_AudioStreamRepositoryError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mediaID := "test-media-id"
	expectedError := errors.New("audio stream query failed")

	mediaRepo := &mockMediaRepository{
		GetFn: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return &domain.MediaFile{ID: mediaID}, nil
		},
	}

	transcodeRepo := &mockTranscodeRepository{
		GetByMediaIDFn: func(ctx context.Context, mediaID string) ([]*domain.Transcode, error) {
			return []*domain.Transcode{}, nil
		},
	}

	audioStreamRepo := &mockAudioStreamRepository{
		GetByMediaIDFn: func(ctx context.Context, mediaID string) ([]*domain.AudioStream, error) {
			return nil, expectedError
		},
	}

	useCase := NewGetMediaUseCase(
		mediaRepo,
		transcodeRepo,
		audioStreamRepo,
		&mockSubtitleStreamRepository{},
	)

	// Act
	result, err := useCase.Execute(ctx, GetMediaInput{MediaID: mediaID})

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get audio streams")
}

func TestGetMediaUseCase_Execute_SubtitleStreamRepositoryError(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mediaID := "test-media-id"
	expectedError := errors.New("subtitle stream query failed")

	mediaRepo := &mockMediaRepository{
		GetFn: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return &domain.MediaFile{ID: mediaID}, nil
		},
	}

	transcodeRepo := &mockTranscodeRepository{
		GetByMediaIDFn: func(ctx context.Context, mediaID string) ([]*domain.Transcode, error) {
			return []*domain.Transcode{}, nil
		},
	}

	audioStreamRepo := &mockAudioStreamRepository{
		GetByMediaIDFn: func(ctx context.Context, mediaID string) ([]*domain.AudioStream, error) {
			return []*domain.AudioStream{}, nil
		},
	}

	subtitleStreamRepo := &mockSubtitleStreamRepository{
		GetByMediaIDFn: func(ctx context.Context, mediaID string) ([]*domain.SubtitleStream, error) {
			return nil, expectedError
		},
	}

	useCase := NewGetMediaUseCase(
		mediaRepo,
		transcodeRepo,
		audioStreamRepo,
		subtitleStreamRepo,
	)

	// Act
	result, err := useCase.Execute(ctx, GetMediaInput{MediaID: mediaID})

	// Assert
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get subtitle streams")
}
