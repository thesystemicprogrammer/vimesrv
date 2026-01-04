package transcode

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// MockMediaRepository for testing
type MockMediaRepository struct {
	GetFn func(ctx context.Context, id string) (*domain.MediaFile, error)
}

func (m *MockMediaRepository) Create(ctx context.Context, media *domain.MediaFile) error {
	return nil
}

func (m *MockMediaRepository) FindByFingerprint(ctx context.Context, fingerprint string) (*domain.MediaFile, error) {
	return nil, nil
}

func (m *MockMediaRepository) ExistsByFingerprint(ctx context.Context, fingerprint string) (bool, error) {
	return false, nil
}

func (m *MockMediaRepository) Update(ctx context.Context, media *domain.MediaFile) error {
	return nil
}

func (m *MockMediaRepository) Get(ctx context.Context, id string) (*domain.MediaFile, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, id)
	}
	return &domain.MediaFile{
		ID:       id,
		FilePath: "/media/test.mp4",
	}, nil
}

func (m *MockMediaRepository) List(ctx context.Context, page, perPage int) ([]*domain.MediaFile, int, error) {
	return nil, 0, nil
}

// MockTranscodeRepository for testing
type MockTranscodeRepository struct {
	CreatedTranscodes []*domain.Transcode
	CreateFn          func(ctx context.Context, transcode *domain.Transcode) error
}

func (m *MockTranscodeRepository) Create(ctx context.Context, transcode *domain.Transcode) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, transcode)
	}
	m.CreatedTranscodes = append(m.CreatedTranscodes, transcode)
	return nil
}

func (m *MockTranscodeRepository) Get(ctx context.Context, id string) (*domain.Transcode, error) {
	return nil, nil
}

func (m *MockTranscodeRepository) GetByMediaID(ctx context.Context, mediaID string) ([]*domain.Transcode, error) {
	return nil, nil
}

func (m *MockTranscodeRepository) UpdateStatus(ctx context.Context, id string, status domain.TranscodeStatus) error {
	return nil
}

func (m *MockTranscodeRepository) MarkProcessing(ctx context.Context, id string) error {
	return nil
}

func (m *MockTranscodeRepository) MarkCompleted(ctx context.Context, id string, outputPath string) error {
	return nil
}

func (m *MockTranscodeRepository) MarkFailed(ctx context.Context, id string) error {
	return nil
}

func (m *MockTranscodeRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *MockTranscodeRepository) ListPending(ctx context.Context, limit int) ([]*domain.Transcode, error) {
	return nil, nil
}

// MockJobRepository for testing
type MockJobRepository struct {
	EnqueuedJobs []*domain.Job
	EnqueueFn    func(ctx context.Context, job *domain.Job) (int64, error)
}

func (m *MockJobRepository) Enqueue(ctx context.Context, job *domain.Job) (int64, error) {
	if m.EnqueueFn != nil {
		return m.EnqueueFn(ctx, job)
	}
	m.EnqueuedJobs = append(m.EnqueuedJobs, job)
	return int64(len(m.EnqueuedJobs)), nil
}

func (m *MockJobRepository) ClaimNextJobDue(ctx context.Context, workerID string) (*domain.Job, bool, error) {
	return nil, false, nil
}

func (m *MockJobRepository) ClaimNextJobDueExcludingTypes(ctx context.Context, workerID string, excludeTypes []string) (*domain.Job, bool, error) {
	return nil, false, nil
}

func (m *MockJobRepository) MarkSuccess(ctx context.Context, jobID int64) error {
	return nil
}

func (m *MockJobRepository) Reschedule(ctx context.Context, jobID int64, runAt time.Time, lastErr string) error {
	return nil
}

func (m *MockJobRepository) MarkDead(ctx context.Context, jobID int64, lastErr string) error {
	return nil
}

func (m *MockJobRepository) FindStuckJobs(ctx context.Context, threshold time.Duration) ([]*domain.Job, error) {
	return nil, nil
}

func (m *MockJobRepository) ResetStuckJob(ctx context.Context, jobID int64) error {
	return nil
}

func (m *MockJobRepository) ExistsPendingJobByType(ctx context.Context, jobType string, payload string) (bool, error) {
	return false, nil
}

func (m *MockJobRepository) ListJobs(ctx context.Context, filter ports.JobListFilter) (*ports.JobListResult, error) {
	return &ports.JobListResult{Jobs: nil, Total: 0}, nil
}

func (m *MockJobRepository) Get(ctx context.Context, jobID int64) (*domain.Job, error) {
	return nil, nil
}

func (m *MockJobRepository) ClaimNextTranscodeJob(ctx context.Context, workerID string) (*domain.Job, error) {
	return nil, nil
}

func (m *MockJobRepository) CountQueuedTranscodeJobs(ctx context.Context) (int, error) {
	return 0, nil
}

// MockClock for testing
type MockClock struct{}

func (m *MockClock) Now() time.Time {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
}

// createEnqueueJobUseCase creates an EnqueueJobUseCase with the given mock job repository
func createEnqueueJobUseCase(mockJobRepo *MockJobRepository) *job.EnqueueJobUseCase {
	cfg := config.JobConfig{MaxAttempts: 3}
	return job.NewEnqueueJobUseCase(cfg, mockJobRepo, &MockClock{}, &ports.NoOpJobNotifier{})
}

// MockFFProbeService for testing
type MockFFProbeService struct {
	GetAudioStreamsFn    func(filePath string) ([]*ports.AudioStreamInfo, error)
	GetSubtitleStreamsFn func(filePath string) ([]*ports.SubtitleStreamInfo, error)
}

func (m *MockFFProbeService) IsAvailable() error {
	return nil
}

func (m *MockFFProbeService) ValidateVideo(filePath string) (bool, error) {
	return true, nil
}

func (m *MockFFProbeService) ExtractMetadata(filePath string) (*ports.VideoMetadata, error) {
	return nil, nil
}

func (m *MockFFProbeService) GetAudioStreams(filePath string) ([]*ports.AudioStreamInfo, error) {
	if m.GetAudioStreamsFn != nil {
		return m.GetAudioStreamsFn(filePath)
	}
	return []*ports.AudioStreamInfo{}, nil
}

func (m *MockFFProbeService) GetSubtitleStreams(filePath string) ([]*ports.SubtitleStreamInfo, error) {
	if m.GetSubtitleStreamsFn != nil {
		return m.GetSubtitleStreamsFn(filePath)
	}
	return []*ports.SubtitleStreamInfo{}, nil
}

// MockAudioStreamRepository for testing
type MockAudioStreamRepository struct {
	CreatedStreams []*domain.AudioStream
	CreateFn       func(ctx context.Context, stream *domain.AudioStream) error
}

func (m *MockAudioStreamRepository) Create(ctx context.Context, stream *domain.AudioStream) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, stream)
	}
	m.CreatedStreams = append(m.CreatedStreams, stream)
	return nil
}

func (m *MockAudioStreamRepository) GetByMediaID(ctx context.Context, mediaID string) ([]*domain.AudioStream, error) {
	return nil, nil
}

func (m *MockAudioStreamRepository) DeleteByMediaID(ctx context.Context, mediaID string) error {
	return nil
}

// MockSubtitleStreamRepository for testing
type MockSubtitleStreamRepository struct {
	CreatedStreams []*domain.SubtitleStream
	CreateFn       func(ctx context.Context, stream *domain.SubtitleStream) error
}

func (m *MockSubtitleStreamRepository) Create(ctx context.Context, stream *domain.SubtitleStream) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, stream)
	}
	m.CreatedStreams = append(m.CreatedStreams, stream)
	return nil
}

func (m *MockSubtitleStreamRepository) GetByMediaID(ctx context.Context, mediaID string) ([]*domain.SubtitleStream, error) {
	return nil, nil
}

func (m *MockSubtitleStreamRepository) DeleteByMediaID(ctx context.Context, mediaID string) error {
	return nil
}

// TestCreateTranscodeJobsUseCase_Execute_Success tests successful job creation with multiple tracks
func TestCreateTranscodeJobsUseCase_Execute_Success(t *testing.T) {
	cfg := &config.Config{
		Media: config.MediaConfig{
			MediaPath:              "/media",
			TranscodeOutputPattern: "{media_path}/{media_id}/transcoded",
		},
		Transcoding: config.TranscodingConfig{
			SegmentDuration: 4,
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k", AudioBitrate: "96k"},
				{Name: "480p", Enabled: true, Resolution: "854x480", CRF: 24, MaxBitrate: "1500k", AudioBitrate: "128k"},
			},
		},
	}

	mockMediaRepo := &MockMediaRepository{
		GetFn: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return &domain.MediaFile{
				ID:       id,
				FilePath: "/media/test.mp4",
			}, nil
		},
	}

	mockTranscodeRepo := &MockTranscodeRepository{
		CreatedTranscodes: make([]*domain.Transcode, 0),
	}

	mockJobRepo := &MockJobRepository{
		EnqueuedJobs: make([]*domain.Job, 0),
	}

	mockAudioStreamRepo := &MockAudioStreamRepository{
		CreatedStreams: make([]*domain.AudioStream, 0),
	}

	mockSubtitleStreamRepo := &MockSubtitleStreamRepository{
		CreatedStreams: make([]*domain.SubtitleStream, 0),
	}

	mockFFProbe := &MockFFProbeService{
		GetAudioStreamsFn: func(filePath string) ([]*ports.AudioStreamInfo, error) {
			return []*ports.AudioStreamInfo{
				{StreamIndex: 0, Codec: "aac", Language: "eng", Channels: 2, ChannelLayout: "stereo"},
				{StreamIndex: 1, Codec: "aac", Language: "spa", Channels: 6, ChannelLayout: "5.1(side)"},
			}, nil
		},
		GetSubtitleStreamsFn: func(filePath string) ([]*ports.SubtitleStreamInfo, error) {
			return []*ports.SubtitleStreamInfo{
				{StreamIndex: 0, Codec: "subrip", Language: "eng"},
			}, nil
		},
	}

	useCase := NewCreateTranscodeJobsUseCase(
		mockMediaRepo,
		mockTranscodeRepo,
		createEnqueueJobUseCase(mockJobRepo),
		mockAudioStreamRepo,
		mockSubtitleStreamRepo,
		mockFFProbe,
		cfg,
	)

	output, err := useCase.Execute(context.Background(), CreateTranscodeJobsInput{
		MediaID: "media123",
	})

	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.Equal(t, "media123", output.MediaID)
	assert.Equal(t, 5, output.TotalJobs, "Should create 2 video + 2 audio + 1 subtitle = 5 jobs")
	assert.Equal(t, 2, output.VideoJobs, "Should create 2 video jobs (360p, 480p)")
	assert.Equal(t, 2, output.AudioJobs, "Should create 2 audio jobs")
	assert.Equal(t, 1, output.SubtitleJobs, "Should create 1 subtitle job")

	// Verify transcode records created
	assert.Equal(t, 5, len(mockTranscodeRepo.CreatedTranscodes))

	// Verify video transcodes
	videoCount := 0
	for _, tc := range mockTranscodeRepo.CreatedTranscodes {
		if tc.TrackType == domain.TrackTypeVideo {
			videoCount++
			assert.Contains(t, []string{"360p", "480p"}, tc.Quality)
			assert.Equal(t, 0, tc.TrackIndex)
		}
	}
	assert.Equal(t, 2, videoCount)

	// Verify audio transcodes
	audioCount := 0
	for _, tc := range mockTranscodeRepo.CreatedTranscodes {
		if tc.TrackType == domain.TrackTypeAudio {
			audioCount++
			assert.Equal(t, "", tc.Quality, "Audio tracks should have empty quality")
			assert.Contains(t, []int{0, 1}, tc.TrackIndex)
		}
	}
	assert.Equal(t, 2, audioCount)

	// Verify subtitle transcodes
	subtitleCount := 0
	for _, tc := range mockTranscodeRepo.CreatedTranscodes {
		if tc.TrackType == domain.TrackTypeSubtitle {
			subtitleCount++
			assert.Equal(t, "", tc.Quality, "Subtitle tracks should have empty quality")
			assert.Equal(t, 0, tc.TrackIndex)
		}
	}
	assert.Equal(t, 1, subtitleCount)

	// Verify jobs created
	assert.Equal(t, 5, len(mockJobRepo.EnqueuedJobs))
	for _, job := range mockJobRepo.EnqueuedJobs {
		assert.Equal(t, "transcode_video", job.Type)
		assert.NotNil(t, job.Payload)

		// Verify payload contains transcode_id
		var payload struct {
			TranscodeID   string `json:"transcode_id"`
			Language      string `json:"language,omitempty"`
			ChannelLayout string `json:"channel_layout,omitempty"`
		}
		err := json.Unmarshal(job.Payload, &payload)
		require.NoError(t, err)
		assert.NotEmpty(t, payload.TranscodeID)

		// Verify audio jobs have language and channel_layout
		if strings.Contains(payload.TranscodeID, "-audio-") {
			assert.NotEmpty(t, payload.Language, "Audio job should have language")
			assert.NotEmpty(t, payload.ChannelLayout, "Audio job should have channel_layout")
		}
	}
}

// TestCreateTranscodeJobsUseCase_Execute_NoAudioOrSubtitles tests with video only
func TestCreateTranscodeJobsUseCase_Execute_NoAudioOrSubtitles(t *testing.T) {
	cfg := &config.Config{
		Media: config.MediaConfig{
			MediaPath:              "/media",
			TranscodeOutputPattern: "{media_path}/{media_id}/transcoded",
		},
		Transcoding: config.TranscodingConfig{
			SegmentDuration: 4,
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k", AudioBitrate: "96k"},
			},
		},
	}

	mockMediaRepo := &MockMediaRepository{}
	mockTranscodeRepo := &MockTranscodeRepository{CreatedTranscodes: make([]*domain.Transcode, 0)}
	mockJobRepo := &MockJobRepository{EnqueuedJobs: make([]*domain.Job, 0)}
	mockFFProbe := &MockFFProbeService{
		GetAudioStreamsFn: func(filePath string) ([]*ports.AudioStreamInfo, error) {
			return []*ports.AudioStreamInfo{}, nil
		},
		GetSubtitleStreamsFn: func(filePath string) ([]*ports.SubtitleStreamInfo, error) {
			return []*ports.SubtitleStreamInfo{}, nil
		},
	}

	useCase := NewCreateTranscodeJobsUseCase(
		mockMediaRepo,
		mockTranscodeRepo,
		createEnqueueJobUseCase(mockJobRepo),
		&MockAudioStreamRepository{},
		&MockSubtitleStreamRepository{},
		mockFFProbe,
		cfg,
	)

	output, err := useCase.Execute(context.Background(), CreateTranscodeJobsInput{
		MediaID: "media123",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, output.TotalJobs, "Should create only 1 video job")
	assert.Equal(t, 1, output.VideoJobs)
	assert.Equal(t, 0, output.AudioJobs)
	assert.Equal(t, 0, output.SubtitleJobs)
}

// TestCreateTranscodeJobsUseCase_Execute_MediaNotFound tests error when media doesn't exist
func TestCreateTranscodeJobsUseCase_Execute_MediaNotFound(t *testing.T) {
	cfg := &config.Config{
		Transcoding: config.TranscodingConfig{
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true},
			},
		},
	}

	mockMediaRepo := &MockMediaRepository{
		GetFn: func(ctx context.Context, id string) (*domain.MediaFile, error) {
			return nil, errors.New("media not found")
		},
	}

	useCase := NewCreateTranscodeJobsUseCase(
		mockMediaRepo,
		&MockTranscodeRepository{},
		createEnqueueJobUseCase(&MockJobRepository{}),
		&MockAudioStreamRepository{},
		&MockSubtitleStreamRepository{},
		&MockFFProbeService{},
		cfg,
	)

	output, err := useCase.Execute(context.Background(), CreateTranscodeJobsInput{
		MediaID: "nonexistent",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to get media")
}

// TestCreateTranscodeJobsUseCase_Execute_NoEnabledProfiles tests error when no quality profiles enabled
func TestCreateTranscodeJobsUseCase_Execute_NoEnabledProfiles(t *testing.T) {
	cfg := &config.Config{
		Transcoding: config.TranscodingConfig{
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: false},
				{Name: "480p", Enabled: false},
			},
		},
	}

	mockMediaRepo := &MockMediaRepository{}

	useCase := NewCreateTranscodeJobsUseCase(
		mockMediaRepo,
		&MockTranscodeRepository{},
		createEnqueueJobUseCase(&MockJobRepository{}),
		&MockAudioStreamRepository{},
		&MockSubtitleStreamRepository{},
		&MockFFProbeService{},
		cfg,
	)

	output, err := useCase.Execute(context.Background(), CreateTranscodeJobsInput{
		MediaID: "media123",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "no enabled quality profiles found")
}

// TestCreateTranscodeJobsUseCase_Execute_TranscodeCreateError tests error handling when transcode creation fails
func TestCreateTranscodeJobsUseCase_Execute_TranscodeCreateError(t *testing.T) {
	cfg := &config.Config{
		Transcoding: config.TranscodingConfig{
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k"},
			},
		},
	}

	mockMediaRepo := &MockMediaRepository{}
	mockTranscodeRepo := &MockTranscodeRepository{
		CreateFn: func(ctx context.Context, transcode *domain.Transcode) error {
			return errors.New("database error")
		},
	}
	mockJobRepo := &MockJobRepository{}
	mockAudioStreamRepo := &MockAudioStreamRepository{}
	mockSubtitleStreamRepo := &MockSubtitleStreamRepository{}
	mockFFProbe := &MockFFProbeService{}

	useCase := NewCreateTranscodeJobsUseCase(
		mockMediaRepo,
		mockTranscodeRepo,
		createEnqueueJobUseCase(mockJobRepo),
		mockAudioStreamRepo,
		mockSubtitleStreamRepo,
		mockFFProbe,
		cfg,
	)

	output, err := useCase.Execute(context.Background(), CreateTranscodeJobsInput{
		MediaID: "media123",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to create video transcode record")
}

// TestCreateTranscodeJobsUseCase_Execute_JobEnqueueError tests error when job enqueue fails
func TestCreateTranscodeJobsUseCase_Execute_JobEnqueueError(t *testing.T) {
	cfg := &config.Config{
		Transcoding: config.TranscodingConfig{
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k"},
			},
		},
	}

	mockMediaRepo := &MockMediaRepository{}
	mockTranscodeRepo := &MockTranscodeRepository{CreatedTranscodes: make([]*domain.Transcode, 0)}
	mockJobRepo := &MockJobRepository{
		EnqueueFn: func(ctx context.Context, job *domain.Job) (int64, error) {
			return 0, errors.New("job queue full")
		},
	}
	mockFFProbe := &MockFFProbeService{}

	useCase := NewCreateTranscodeJobsUseCase(
		mockMediaRepo,
		mockTranscodeRepo,
		createEnqueueJobUseCase(mockJobRepo),
		&MockAudioStreamRepository{},
		&MockSubtitleStreamRepository{},
		mockFFProbe,
		cfg,
	)

	output, err := useCase.Execute(context.Background(), CreateTranscodeJobsInput{
		MediaID: "media123",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to create video transcode job")
}

// TestCreateTranscodeJobsUseCase_Execute_FFProbeError tests error when FFProbe fails
func TestCreateTranscodeJobsUseCase_Execute_FFProbeError(t *testing.T) {
	cfg := &config.Config{
		Transcoding: config.TranscodingConfig{
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k"},
			},
		},
	}

	mockMediaRepo := &MockMediaRepository{}
	mockTranscodeRepo := &MockTranscodeRepository{CreatedTranscodes: make([]*domain.Transcode, 0)}
	mockJobRepo := &MockJobRepository{EnqueuedJobs: make([]*domain.Job, 0)}
	mockFFProbe := &MockFFProbeService{
		GetAudioStreamsFn: func(filePath string) ([]*ports.AudioStreamInfo, error) {
			return nil, errors.New("ffprobe failed")
		},
	}

	useCase := NewCreateTranscodeJobsUseCase(
		mockMediaRepo,
		mockTranscodeRepo,
		createEnqueueJobUseCase(mockJobRepo),
		&MockAudioStreamRepository{},
		&MockSubtitleStreamRepository{},
		mockFFProbe,
		cfg,
	)

	output, err := useCase.Execute(context.Background(), CreateTranscodeJobsInput{
		MediaID: "media123",
	})

	require.Error(t, err)
	assert.Nil(t, output)
	assert.Contains(t, err.Error(), "failed to detect audio streams")
}

// TestCreateTranscodeJobsUseCase_Execute_SingleQuality tests with single quality profile
func TestCreateTranscodeJobsUseCase_Execute_SingleQuality(t *testing.T) {
	cfg := &config.Config{
		Transcoding: config.TranscodingConfig{
			QualityProfiles: []config.QualityProfile{
				{Name: "720p", Enabled: true, Resolution: "1280x720", CRF: 23, MaxBitrate: "2500k"},
			},
		},
	}

	mockMediaRepo := &MockMediaRepository{}
	mockTranscodeRepo := &MockTranscodeRepository{CreatedTranscodes: make([]*domain.Transcode, 0)}
	mockJobRepo := &MockJobRepository{EnqueuedJobs: make([]*domain.Job, 0)}
	mockFFProbe := &MockFFProbeService{
		GetAudioStreamsFn: func(filePath string) ([]*ports.AudioStreamInfo, error) {
			return []*ports.AudioStreamInfo{
				{StreamIndex: 0, Codec: "aac"},
			}, nil
		},
		GetSubtitleStreamsFn: func(filePath string) ([]*ports.SubtitleStreamInfo, error) {
			return []*ports.SubtitleStreamInfo{}, nil
		},
	}

	useCase := NewCreateTranscodeJobsUseCase(
		mockMediaRepo,
		mockTranscodeRepo,
		createEnqueueJobUseCase(mockJobRepo),
		&MockAudioStreamRepository{},
		&MockSubtitleStreamRepository{},
		mockFFProbe,
		cfg,
	)

	output, err := useCase.Execute(context.Background(), CreateTranscodeJobsInput{
		MediaID: "media123",
	})

	require.NoError(t, err)
	assert.Equal(t, 2, output.TotalJobs, "Should create 1 video + 1 audio = 2 jobs")
	assert.Equal(t, 1, output.VideoJobs)
	assert.Equal(t, 1, output.AudioJobs)
	assert.Equal(t, 0, output.SubtitleJobs)

	// Verify video transcode uses correct quality
	found720p := false
	for _, tc := range mockTranscodeRepo.CreatedTranscodes {
		if tc.TrackType == domain.TrackTypeVideo && tc.Quality == "720p" {
			found720p = true
		}
	}
	assert.True(t, found720p, "Should create 720p video transcode")
}

// TestCreateTranscodeJobsUseCase_Execute_MultipleSubtitles tests with multiple subtitle tracks
func TestCreateTranscodeJobsUseCase_Execute_MultipleSubtitles(t *testing.T) {
	cfg := &config.Config{
		Transcoding: config.TranscodingConfig{
			QualityProfiles: []config.QualityProfile{
				{Name: "360p", Enabled: true, Resolution: "640x360", CRF: 25, MaxBitrate: "900k"},
			},
		},
	}

	mockMediaRepo := &MockMediaRepository{}
	mockTranscodeRepo := &MockTranscodeRepository{CreatedTranscodes: make([]*domain.Transcode, 0)}
	mockJobRepo := &MockJobRepository{EnqueuedJobs: make([]*domain.Job, 0)}
	mockFFProbe := &MockFFProbeService{
		GetAudioStreamsFn: func(filePath string) ([]*ports.AudioStreamInfo, error) {
			return []*ports.AudioStreamInfo{}, nil
		},
		GetSubtitleStreamsFn: func(filePath string) ([]*ports.SubtitleStreamInfo, error) {
			return []*ports.SubtitleStreamInfo{
				{StreamIndex: 0, Codec: "subrip", Language: "eng"},
				{StreamIndex: 1, Codec: "subrip", Language: "spa"},
				{StreamIndex: 2, Codec: "subrip", Language: "fra"},
			}, nil
		},
	}

	useCase := NewCreateTranscodeJobsUseCase(
		mockMediaRepo,
		mockTranscodeRepo,
		createEnqueueJobUseCase(mockJobRepo),
		&MockAudioStreamRepository{},
		&MockSubtitleStreamRepository{},
		mockFFProbe,
		cfg,
	)

	output, err := useCase.Execute(context.Background(), CreateTranscodeJobsInput{
		MediaID: "media123",
	})

	require.NoError(t, err)
	assert.Equal(t, 4, output.TotalJobs, "Should create 1 video + 3 subtitles = 4 jobs")
	assert.Equal(t, 1, output.VideoJobs)
	assert.Equal(t, 0, output.AudioJobs)
	assert.Equal(t, 3, output.SubtitleJobs)

	// Verify subtitle track indices
	subtitleIndices := make(map[int]bool)
	for _, tc := range mockTranscodeRepo.CreatedTranscodes {
		if tc.TrackType == domain.TrackTypeSubtitle {
			subtitleIndices[tc.TrackIndex] = true
		}
	}
	assert.True(t, subtitleIndices[0])
	assert.True(t, subtitleIndices[1])
	assert.True(t, subtitleIndices[2])
}
