package http

import (
	"testing"
)

func TestParseContentPath(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		wantNil         bool
		wantFilePath    string
		wantContentType string
	}{
		{
			name:            "subtitle file",
			path:            "subtitle-0.vtt",
			wantFilePath:    "transcoded/subtitle-0.vtt",
			wantContentType: "text/vtt",
		},
		{
			name:            "subtitle file with higher index",
			path:            "subtitle-5.vtt",
			wantFilePath:    "transcoded/subtitle-5.vtt",
			wantContentType: "text/vtt",
		},
		{
			name:            "audio init segment",
			path:            "audio-0/init.mp4",
			wantFilePath:    "transcoded/audio-0/init.mp4",
			wantContentType: "audio/mp4",
		},
		{
			name:            "audio chunk segment",
			path:            "audio-1/chunk-001.m4s",
			wantFilePath:    "transcoded/audio-1/chunk-001.m4s",
			wantContentType: "audio/iso.segment",
		},
		{
			name:            "video init segment",
			path:            "720p/video/init.mp4",
			wantFilePath:    "transcoded/720p/video/init.mp4",
			wantContentType: "video/mp4",
		},
		{
			name:            "video chunk segment",
			path:            "1080p/video/chunk-005.m4s",
			wantFilePath:    "transcoded/1080p/video/chunk-005.m4s",
			wantContentType: "video/iso.segment",
		},
		{
			name:            "video segment with 480p quality",
			path:            "480p/video/chunk-000.m4s",
			wantFilePath:    "transcoded/480p/video/chunk-000.m4s",
			wantContentType: "video/iso.segment",
		},
		{
			name:    "invalid empty path",
			path:    "",
			wantNil: true,
		},
		{
			name:    "invalid single part without prefix",
			path:    "somefile.mp4",
			wantNil: true,
		},
		{
			name:    "invalid subtitle without .vtt suffix",
			path:    "subtitle-0.srt",
			wantNil: true,
		},
		{
			name:    "invalid two parts without audio prefix",
			path:    "video-0/chunk-001.m4s",
			wantNil: true,
		},
		{
			name:    "invalid three parts without video in middle",
			path:    "720p/audio/chunk-001.m4s",
			wantNil: true,
		},
		{
			name:    "invalid too many parts",
			path:    "720p/video/extra/chunk-001.m4s",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseContentPath(tt.path)

			if tt.wantNil {
				if result != nil {
					t.Errorf("parseContentPath(%q) = %+v, want nil", tt.path, result)
				}
				return
			}

			if result == nil {
				t.Fatalf("parseContentPath(%q) = nil, want non-nil", tt.path)
			}

			if result.FilePath != tt.wantFilePath {
				t.Errorf("parseContentPath(%q).FilePath = %q, want %q", tt.path, result.FilePath, tt.wantFilePath)
			}

			if result.ContentType != tt.wantContentType {
				t.Errorf("parseContentPath(%q).ContentType = %q, want %q", tt.path, result.ContentType, tt.wantContentType)
			}
		})
	}
}
