package shared

import "testing"

func TestLanguageCodeToName(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"eng", "English"},
		{"fra", "French"},
		{"deu", "German"},
		{"spa", "Spanish"},
		{"ita", "Italian"},
		{"jpn", "Japanese"},
		{"zho", "Chinese"},
		{"kor", "Korean"},
		{"rus", "Russian"},
		{"por", "Portuguese"},
		{"nld", "Dutch"},
		{"swe", "Swedish"},
		{"nor", "Norwegian Bokmål"},
		{"dan", "Danish"},
		{"fin", "Finnish"},
		// ISO 639-1 codes should also work
		{"en", "English"},
		{"fr", "French"},
		{"de", "German"},
		// Empty string
		{"", ""},
		// Unknown code returns original
		{"xyz", "xyz"},
		// Undefined
		{"und", "Unknown language"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := LanguageCodeToName(tt.code)
			if result != tt.expected {
				t.Errorf("LanguageCodeToName(%q) = %q, want %q", tt.code, result, tt.expected)
			}
		})
	}
}
