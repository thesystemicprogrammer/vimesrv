package shared

import (
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// LanguageCodeToName converts an ISO 639 language code (e.g., "eng", "fra", "deu")
// to a human-readable language name (e.g., "English", "French", "German").
// If the code cannot be parsed, it returns the original code as-is.
func LanguageCodeToName(code string) string {
	if code == "" {
		return ""
	}

	tag, err := language.Parse(code)
	if err != nil {
		return code
	}

	// Get the language name in English
	name := display.English.Tags().Name(tag)
	if name == "" {
		return code
	}

	return name
}
