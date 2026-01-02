package domain

import "time"

// MovieCertification represents a content rating for a movie in a specific country
type MovieCertification struct {
	ID              int64     `json:"id"`
	MovieMetadataID int64     `json:"movie_metadata_id"`
	Country         string    `json:"country"` // ISO 3166-1 alpha-2 country code (e.g., "US", "DE", "GB")
	Certification   string    `json:"certification"`
	CreatedAt       time.Time `json:"created_at"`
}

// LanguageToCountry maps language codes to their primary country for certification lookup
var LanguageToCountry = map[string]string{
	"en": "US",
	"de": "DE",
	"fr": "FR",
	"es": "ES",
	"it": "IT",
	"pt": "BR",
	"ja": "JP",
	"ko": "KR",
	"zh": "CN",
	"ru": "RU",
	"nl": "NL",
	"pl": "PL",
	"sv": "SE",
	"da": "DK",
	"fi": "FI",
	"no": "NO",
	"cs": "CZ",
	"hu": "HU",
	"tr": "TR",
	"el": "GR",
	"he": "IL",
	"ar": "SA",
	"th": "TH",
	"vi": "VN",
	"id": "ID",
	"ms": "MY",
}

// FallbackCountries defines the priority order for fallback certifications
var FallbackCountries = []string{"US", "GB"}

// NewMovieCertification creates a new MovieCertification with the current timestamp
func NewMovieCertification(movieMetadataID int64, country, certification string) *MovieCertification {
	return &MovieCertification{
		MovieMetadataID: movieMetadataID,
		Country:         country,
		Certification:   certification,
		CreatedAt:       time.Now(),
	}
}

// GetCountryForLanguage returns the country code for a given language code,
// or an empty string if no mapping exists
func GetCountryForLanguage(language string) string {
	if country, ok := LanguageToCountry[language]; ok {
		return country
	}
	return ""
}

// GetCertificationWithFallback returns the certification for the user's language,
// falling back to US/GB if not found. Returns empty string if no certification available.
func GetCertificationWithFallback(certifications []MovieCertification, language string) string {
	// Build a map for quick lookup
	certMap := make(map[string]string)
	for _, cert := range certifications {
		certMap[cert.Country] = cert.Certification
	}

	// Try the user's language country first
	if country := GetCountryForLanguage(language); country != "" {
		if cert, ok := certMap[country]; ok {
			return cert
		}
	}

	// Try fallback countries
	for _, country := range FallbackCountries {
		if cert, ok := certMap[country]; ok {
			return cert
		}
	}

	return ""
}
