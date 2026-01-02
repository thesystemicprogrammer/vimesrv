package repository

import (
	"testing"
)

func TestLanguageParams(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantExact string
		wantBase  string
	}{
		{
			name:      "full locale with hyphen",
			input:     "en-US",
			wantExact: "en-US",
			wantBase:  "en",
		},
		{
			name:      "full locale with underscore",
			input:     "en_US",
			wantExact: "en_US",
			wantBase:  "en",
		},
		{
			name:      "base language only",
			input:     "en",
			wantExact: "en",
			wantBase:  "en",
		},
		{
			name:      "german locale",
			input:     "de-DE",
			wantExact: "de-DE",
			wantBase:  "de",
		},
		{
			name:      "french locale",
			input:     "fr-CA",
			wantExact: "fr-CA",
			wantBase:  "fr",
		},
		{
			name:      "empty string defaults to en",
			input:     "",
			wantExact: "en",
			wantBase:  "en",
		},
		{
			name:      "chinese simplified",
			input:     "zh-CN",
			wantExact: "zh-CN",
			wantBase:  "zh",
		},
		{
			name:      "portuguese brazilian",
			input:     "pt-BR",
			wantExact: "pt-BR",
			wantBase:  "pt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotExact, gotBase := languageParams(tt.input)

			if gotExact != tt.wantExact {
				t.Errorf("languageParams(%q) exact = %q, want %q", tt.input, gotExact, tt.wantExact)
			}

			if gotBase != tt.wantBase {
				t.Errorf("languageParams(%q) base = %q, want %q", tt.input, gotBase, tt.wantBase)
			}
		})
	}
}

func TestParseGenresJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "multiple genres",
			input: `["Action","Science Fiction","Adventure"]`,
			want:  "Action,Science Fiction,Adventure",
		},
		{
			name:  "single genre",
			input: `["Drama"]`,
			want:  "Drama",
		},
		{
			name:  "empty array",
			input: `[]`,
			want:  "",
		},
		{
			name:  "genres with special characters",
			input: `["Sci-Fi \u0026 Fantasy","Action \u0026 Adventure"]`,
			want:  `Sci-Fi \u0026 Fantasy,Action \u0026 Adventure`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGenresJSON(tt.input)
			if got != tt.want {
				t.Errorf("parseGenresJSON(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
