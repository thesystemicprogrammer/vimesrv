package worker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thesystemicprogrammer/vimesrv/internal/worker/client"
	"github.com/thesystemicprogrammer/vimesrv/internal/worker/config"
	"github.com/thesystemicprogrammer/vimesrv/pkg/transcoding"
)

func TestParseTimeToSeconds(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "zero time",
			input:    "00:00:00.00",
			expected: 0,
		},
		{
			name:     "one second",
			input:    "00:00:01.00",
			expected: 1.0,
		},
		{
			name:     "one minute",
			input:    "00:01:00.00",
			expected: 60.0,
		},
		{
			name:     "one hour",
			input:    "01:00:00.00",
			expected: 3600.0,
		},
		{
			name:     "complex time",
			input:    "01:30:45.50",
			expected: 5445.50,
		},
		{
			name:     "two hours",
			input:    "02:00:00.00",
			expected: 7200.0,
		},
		{
			name:     "with milliseconds",
			input:    "00:05:30.25",
			expected: 330.25,
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: 0,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "partial format",
			input:    "00:30",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTimeToSeconds(tt.input)
			if result != tt.expected {
				t.Errorf("parseTimeToSeconds(%q) = %f, want %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCountOutputFiles(t *testing.T) {
	t.Run("subtitle returns single file", func(t *testing.T) {
		outputPath := "/transcodes/abc123/subtitle-0.vtt"
		count, files := countOutputFiles(outputPath, "subtitle")

		if count != 1 {
			t.Errorf("expected count 1, got %d", count)
		}
		if len(files) != 1 || files[0] != outputPath {
			t.Errorf("expected files [%s], got %v", outputPath, files)
		}
	})

	t.Run("video with segments", func(t *testing.T) {
		// Create temp directory with segment files
		tempDir := t.TempDir()

		// Create init.mp4 and segment files
		testFiles := []string{"init.mp4", "chunk-001.m4s", "chunk-002.m4s", "chunk-003.m4s"}
		for _, f := range testFiles {
			if err := os.WriteFile(filepath.Join(tempDir, f), []byte("test"), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}
		}

		count, files := countOutputFiles(tempDir, "video")

		if count != 3 { // Only .m4s files
			t.Errorf("expected segment count 3, got %d", count)
		}
		if len(files) != 4 { // All files including init.mp4
			t.Errorf("expected 4 files, got %d", len(files))
		}
	})

	t.Run("audio with segments", func(t *testing.T) {
		tempDir := t.TempDir()

		testFiles := []string{"init.mp4", "chunk-001.m4s", "chunk-002.m4s"}
		for _, f := range testFiles {
			if err := os.WriteFile(filepath.Join(tempDir, f), []byte("test"), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}
		}

		count, files := countOutputFiles(tempDir, "audio")

		if count != 2 {
			t.Errorf("expected segment count 2, got %d", count)
		}
		if len(files) != 3 {
			t.Errorf("expected 3 files, got %d", len(files))
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		count, files := countOutputFiles("/nonexistent/path", "video")

		if count != 0 {
			t.Errorf("expected count 0, got %d", count)
		}
		if files != nil {
			t.Errorf("expected nil files, got %v", files)
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		tempDir := t.TempDir()

		count, files := countOutputFiles(tempDir, "video")

		if count != 0 {
			t.Errorf("expected count 0, got %d", count)
		}
		if len(files) != 0 {
			t.Errorf("expected empty files, got %v", files)
		}
	})
}

func TestBuildTranscodeOptions(t *testing.T) {
	// Create a minimal worker for testing with empty config (no path translation)
	w := &Worker{
		config: &config.Config{
			Media: config.MediaConfig{
				MediaPath:       "/local/media",
				ServerMediaPath: "", // No translation
			},
		},
	}

	t.Run("video job", func(t *testing.T) {
		job := &client.WorkerJob{
			JobID:       123,
			TranscodeID: "abc123",
			TrackType:   "video",
			TrackIndex:  0,
			Quality:     "1080p",
			InputPath:   "/media/movie.mkv",
			OutputPath:  "/transcodes/abc123/1080p/video",
			TranscodeOptions: client.WorkerTranscodeOptions{
				Width:       1920,
				Height:      1080,
				VideoCodec:  "libx264",
				CRF:         23,
				MaxBitrate:  8000000,
				Preset:      "medium",
				SegmentTime: 4,
			},
		}

		opts := w.buildTranscodeOptions(job)

		if opts.InputPath != job.InputPath {
			t.Errorf("InputPath = %s, want %s", opts.InputPath, job.InputPath)
		}
		if opts.OutputPath != job.OutputPath {
			t.Errorf("OutputPath = %s, want %s", opts.OutputPath, job.OutputPath)
		}
		if opts.SourceStreamIndex != job.TrackIndex {
			t.Errorf("SourceStreamIndex = %d, want %d", opts.SourceStreamIndex, job.TrackIndex)
		}
		if opts.TrackType != "video" {
			t.Errorf("TrackType = %s, want video", opts.TrackType)
		}
		if opts.Width != 1920 {
			t.Errorf("Width = %d, want 1920", opts.Width)
		}
		if opts.Height != 1080 {
			t.Errorf("Height = %d, want 1080", opts.Height)
		}
		if opts.VideoCodec != "libx264" {
			t.Errorf("VideoCodec = %s, want libx264", opts.VideoCodec)
		}
		if opts.CRF != 23 {
			t.Errorf("CRF = %d, want 23", opts.CRF)
		}
		if opts.MaxBitrate != 8000000 {
			t.Errorf("MaxBitrate = %d, want 8000000", opts.MaxBitrate)
		}
		if opts.Preset != "medium" {
			t.Errorf("Preset = %s, want medium", opts.Preset)
		}
		if opts.SegmentTime != 4 {
			t.Errorf("SegmentTime = %d, want 4", opts.SegmentTime)
		}
		// Audio fields should not be set for video
		if opts.AudioCodec != "" {
			t.Errorf("AudioCodec should be empty for video, got %s", opts.AudioCodec)
		}
		if opts.AudioBitrate != 0 {
			t.Errorf("AudioBitrate should be 0 for video, got %d", opts.AudioBitrate)
		}
	})

	t.Run("audio job", func(t *testing.T) {
		job := &client.WorkerJob{
			JobID:       124,
			TranscodeID: "abc123",
			TrackType:   "audio",
			TrackIndex:  1,
			InputPath:   "/media/movie.mkv",
			OutputPath:  "/transcodes/abc123/audio-0",
			TranscodeOptions: client.WorkerTranscodeOptions{
				AudioCodec:   "aac",
				AudioBitrate: 128000,
				SegmentTime:  4,
			},
		}

		opts := w.buildTranscodeOptions(job)

		if opts.TrackType != "audio" {
			t.Errorf("TrackType = %s, want audio", opts.TrackType)
		}
		if opts.SourceStreamIndex != 1 {
			t.Errorf("SourceStreamIndex = %d, want 1", opts.SourceStreamIndex)
		}
		if opts.AudioCodec != "aac" {
			t.Errorf("AudioCodec = %s, want aac", opts.AudioCodec)
		}
		if opts.AudioBitrate != 128000 {
			t.Errorf("AudioBitrate = %d, want 128000", opts.AudioBitrate)
		}
		// Video fields should not be set for audio
		if opts.Width != 0 {
			t.Errorf("Width should be 0 for audio, got %d", opts.Width)
		}
		if opts.VideoCodec != "" {
			t.Errorf("VideoCodec should be empty for audio, got %s", opts.VideoCodec)
		}
	})

	t.Run("subtitle job", func(t *testing.T) {
		job := &client.WorkerJob{
			JobID:       125,
			TranscodeID: "abc123",
			TrackType:   "subtitle",
			TrackIndex:  2,
			InputPath:   "/media/movie.mkv",
			OutputPath:  "/transcodes/abc123/subtitle-0.vtt",
			TranscodeOptions: client.WorkerTranscodeOptions{
				SegmentTime: 4,
			},
		}

		opts := w.buildTranscodeOptions(job)

		if opts.TrackType != "subtitle" {
			t.Errorf("TrackType = %s, want subtitle", opts.TrackType)
		}
		if opts.SourceStreamIndex != 2 {
			t.Errorf("SourceStreamIndex = %d, want 2", opts.SourceStreamIndex)
		}
		// No video or audio fields for subtitle
		if opts.Width != 0 || opts.VideoCodec != "" || opts.AudioCodec != "" {
			t.Error("Video/audio fields should not be set for subtitle")
		}
	})
}

func TestWorkerTranscodeOptionsMapping(t *testing.T) {
	// Verify that WorkerTranscodeOptions maps correctly to transcoding.Options
	// This is a type-level test to ensure the structs are compatible

	workerOpts := client.WorkerTranscodeOptions{
		Width:        1920,
		Height:       1080,
		VideoCodec:   "libx264",
		CRF:          23,
		MaxBitrate:   8000000,
		Preset:       "medium",
		AudioCodec:   "aac",
		AudioBitrate: 128000,
		SegmentTime:  4,
	}

	// Create transcoding options with the same fields
	transcodeOpts := transcoding.Options{
		Width:        workerOpts.Width,
		Height:       workerOpts.Height,
		VideoCodec:   workerOpts.VideoCodec,
		CRF:          workerOpts.CRF,
		MaxBitrate:   workerOpts.MaxBitrate,
		Preset:       workerOpts.Preset,
		AudioCodec:   workerOpts.AudioCodec,
		AudioBitrate: workerOpts.AudioBitrate,
		SegmentTime:  workerOpts.SegmentTime,
	}

	// Verify the mapping
	if transcodeOpts.Width != 1920 {
		t.Errorf("Width mapping failed: got %d", transcodeOpts.Width)
	}
	if transcodeOpts.Height != 1080 {
		t.Errorf("Height mapping failed: got %d", transcodeOpts.Height)
	}
	if transcodeOpts.VideoCodec != "libx264" {
		t.Errorf("VideoCodec mapping failed: got %s", transcodeOpts.VideoCodec)
	}
	if transcodeOpts.CRF != 23 {
		t.Errorf("CRF mapping failed: got %d", transcodeOpts.CRF)
	}
	if transcodeOpts.MaxBitrate != 8000000 {
		t.Errorf("MaxBitrate mapping failed: got %d", transcodeOpts.MaxBitrate)
	}
	if transcodeOpts.Preset != "medium" {
		t.Errorf("Preset mapping failed: got %s", transcodeOpts.Preset)
	}
	if transcodeOpts.AudioCodec != "aac" {
		t.Errorf("AudioCodec mapping failed: got %s", transcodeOpts.AudioCodec)
	}
	if transcodeOpts.AudioBitrate != 128000 {
		t.Errorf("AudioBitrate mapping failed: got %d", transcodeOpts.AudioBitrate)
	}
	if transcodeOpts.SegmentTime != 4 {
		t.Errorf("SegmentTime mapping failed: got %d", transcodeOpts.SegmentTime)
	}
}

func TestTranslatePath(t *testing.T) {
	tests := []struct {
		name            string
		serverMediaPath string
		localMediaPath  string
		inputPath       string
		expectedPath    string
	}{
		{
			name:            "no translation when server_media_path is empty",
			serverMediaPath: "",
			localMediaPath:  "/local/media",
			inputPath:       "/srv/media/abc123/movie.mkv",
			expectedPath:    "/srv/media/abc123/movie.mkv",
		},
		{
			name:            "translates matching prefix",
			serverMediaPath: "/srv/media",
			localMediaPath:  "/mnt/nfs/media",
			inputPath:       "/srv/media/abc123/movie.mkv",
			expectedPath:    "/mnt/nfs/media/abc123/movie.mkv",
		},
		{
			name:            "translates exact match",
			serverMediaPath: "/srv/media",
			localMediaPath:  "/mnt/nfs/media",
			inputPath:       "/srv/media",
			expectedPath:    "/mnt/nfs/media",
		},
		{
			name:            "does not translate non-matching path",
			serverMediaPath: "/srv/media",
			localMediaPath:  "/mnt/nfs/media",
			inputPath:       "/other/path/movie.mkv",
			expectedPath:    "/other/path/movie.mkv",
		},
		{
			name:            "handles nested paths correctly",
			serverMediaPath: "/mnt/video/vimesrv/library/media",
			localMediaPath:  "/nfs/vimesrv/media",
			inputPath:       "/mnt/video/vimesrv/library/media/884cc0da/IT_Welcome.mkv",
			expectedPath:    "/nfs/vimesrv/media/884cc0da/IT_Welcome.mkv",
		},
		{
			name:            "does not match partial directory names",
			serverMediaPath: "/srv/media",
			localMediaPath:  "/mnt/nfs/media",
			inputPath:       "/srv/media2/movie.mkv",
			expectedPath:    "/srv/media2/movie.mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Worker{
				config: &config.Config{
					Media: config.MediaConfig{
						MediaPath:       tt.localMediaPath,
						ServerMediaPath: tt.serverMediaPath,
					},
				},
			}

			result := w.translatePath(tt.inputPath)
			if result != tt.expectedPath {
				t.Errorf("translatePath(%q) = %q, want %q", tt.inputPath, result, tt.expectedPath)
			}
		})
	}
}

func TestBuildTranscodeOptionsWithPathTranslation(t *testing.T) {
	w := &Worker{
		config: &config.Config{
			Media: config.MediaConfig{
				MediaPath:       "/nfs/media",
				ServerMediaPath: "/srv/media",
			},
		},
	}

	job := &client.WorkerJob{
		JobID:       123,
		TranscodeID: "abc123",
		TrackType:   "video",
		TrackIndex:  0,
		Quality:     "1080p",
		InputPath:   "/srv/media/abc123/movie.mkv",
		OutputPath:  "/srv/media/abc123/transcoded/1080p/video",
		TranscodeOptions: client.WorkerTranscodeOptions{
			Width:       1920,
			Height:      1080,
			VideoCodec:  "libx264",
			SegmentTime: 4,
		},
	}

	opts := w.buildTranscodeOptions(job)

	expectedInputPath := "/nfs/media/abc123/movie.mkv"
	expectedOutputPath := "/nfs/media/abc123/transcoded/1080p/video"

	if opts.InputPath != expectedInputPath {
		t.Errorf("InputPath = %q, want %q", opts.InputPath, expectedInputPath)
	}
	if opts.OutputPath != expectedOutputPath {
		t.Errorf("OutputPath = %q, want %q", opts.OutputPath, expectedOutputPath)
	}
}
