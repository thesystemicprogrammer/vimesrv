package metadata

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// RegexFilenameParser implements ports.FilenameParser using regex patterns
type RegexFilenameParser struct{}

// NewRegexFilenameParser creates a new RegexFilenameParser
func NewRegexFilenameParser() *RegexFilenameParser {
	return &RegexFilenameParser{}
}

// Compile all regex patterns at package level for efficiency
var (
	// Series patterns - order matters, most specific first
	// S01E02, S1E2, s01e02
	seriesPatternSE = regexp.MustCompile(`(?i)[.\s_-]?S(\d{1,2})[\s._-]?E(\d{1,3})`)
	// 1x02, 01x02
	seriesPatternX = regexp.MustCompile(`(?i)[.\s_-](\d{1,2})x(\d{1,3})`)
	// Season 1 Episode 2, Season.1.Episode.2
	seriesPatternFull = regexp.MustCompile(`(?i)Season[\s._-]?(\d{1,2})[\s._-]?Episode[\s._-]?(\d{1,3})`)
	// E01, E1 (episode only, assume season 1)
	seriesPatternEOnly = regexp.MustCompile(`(?i)[.\s_-]E(\d{1,3})(?:[.\s_-]|$)`)

	// Year patterns
	// (2019), [2019]
	yearPatternBracket = regexp.MustCompile(`[\[(](\d{4})[\])]`)
	// .2019., -2019-, _2019_ or year at end of filename
	yearPatternDot = regexp.MustCompile(`[.\s_-](\d{4})(?:[.\s_-]|$)`)

	// Quality patterns - use word boundary or common separators
	qualityPattern = regexp.MustCompile(`(?i)(?:^|[.\s_\-])(2160p|4K|UHD|1080p|720p|480p|576p)(?:$|[.\s_\-])`)

	// Source patterns - use word boundary or common separators
	sourcePattern = regexp.MustCompile(`(?i)(?:^|[.\s_\-])(BluRay|Blu-Ray|BDRip|BRRip|WEB-DL|WEBDL|WEBRip|WEB|HDRip|HDTV|DVDRip|DVD|DVDR|CAM|TS|TC|SCR|R5|PPVRip)(?:$|[.\s_\-])`)

	// Edition patterns - use word boundary or common separators
	editionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:^|[.\s_\-])(Directors?[\s._\-]*Cut)(?:$|[.\s_\-])`),
		regexp.MustCompile(`(?i)(?:^|[.\s_\-])(Extended[\s._\-]*(Edition|Cut)?)(?:$|[.\s_\-])`),
		regexp.MustCompile(`(?i)(?:^|[.\s_\-])(Theatrical[\s._\-]*(Edition|Cut)?)(?:$|[.\s_\-])`),
		regexp.MustCompile(`(?i)(?:^|[.\s_\-])(Unrated[\s._\-]*(Edition|Cut)?)(?:$|[.\s_\-])`),
		regexp.MustCompile(`(?i)(?:^|[.\s_\-])(Uncut)(?:$|[.\s_\-])`),
		regexp.MustCompile(`(?i)(?:^|[.\s_\-])(Remastered)(?:$|[.\s_\-])`),
		regexp.MustCompile(`(?i)(?:^|[.\s_\-])(Special[\s._\-]*Edition)(?:$|[.\s_\-])`),
		regexp.MustCompile(`(?i)(?:^|[.\s_\-])(Final[\s._\-]*Cut)(?:$|[.\s_\-])`),
		regexp.MustCompile(`(?i)(?:^|[.\s_\-])(Ultimate[\s._\-]*(Edition|Cut)?)(?:$|[.\s_\-])`),
		regexp.MustCompile(`(?i)(?:^|[.\s_\-])(Criterion[\s._\-]*(Collection)?)(?:$|[.\s_\-])`),
		regexp.MustCompile(`(?i)(?:^|[.\s_\-])(IMAX([\s._\-]*Edition)?)(?:$|[.\s_\-])`),
	}

	// Cleanup patterns - things to remove from title
	cleanupPatterns = []*regexp.Regexp{
		// Video codecs
		regexp.MustCompile(`(?i)\b(x264|x265|h\.?264|h\.?265|HEVC|AVC|XviD|DivX|VP9|AV1)\b`),
		// Audio codecs
		regexp.MustCompile(`(?i)\b(AAC|AC3|DTS|DTS-HD|TrueHD|Atmos|FLAC|MP3|EAC3|DD5\.?1|DD7\.?1|5\.1|7\.1)\b`),
		// HDR formats
		regexp.MustCompile(`(?i)\b(HDR|HDR10|HDR10\+|Dolby\s*Vision|DV)\b`),
		// Release groups (common patterns)
		regexp.MustCompile(`(?i)-[A-Z0-9]+$`),
		// File size indicators
		regexp.MustCompile(`(?i)\b\d+(\.\d+)?\s*(GB|MB)\b`),
		// Misc tags
		regexp.MustCompile(`(?i)\b(PROPER|REPACK|REAL|INTERNAL|LIMITED|COMPLETE|REMUX)\b`),
		// 3D indicators
		regexp.MustCompile(`(?i)\b(3D|SBS|HSBS|OU|HOU)\b`),
		// Subtitles indicators
		regexp.MustCompile(`(?i)\b(SUBBED|DUBBED|MULTI|MULTi)\b`),
	}

	// Separator normalization
	separatorPattern = regexp.MustCompile(`[._]`)

	// Multiple spaces
	multiSpacePattern = regexp.MustCompile(`\s{2,}`)

	// Brackets and their contents (for cleanup)
	bracketPattern = regexp.MustCompile(`[\[\(][^\]\)]*[\]\)]`)
)

// Parse extracts structured information from a video filename
func (p *RegexFilenameParser) Parse(filename string) *ports.ParsedFilename {
	result := &ports.ParsedFilename{}

	// Remove file extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	original := name

	// Extract series info first (before other processing)
	result.IsSeries, result.SeasonNumber, result.EpisodeNumber = p.extractSeriesInfo(name)

	// Extract year
	result.Year = p.extractYear(name)

	// Extract quality
	result.Quality = p.extractQuality(name)

	// Extract source
	result.Source = p.extractSource(name)

	// Extract edition
	result.Edition = p.extractEdition(name)

	// Extract and clean title
	result.Title = p.extractTitle(original, result)
	result.CleanTitle = p.cleanForSearch(result.Title)

	return result
}

// extractSeriesInfo extracts season and episode numbers
func (p *RegexFilenameParser) extractSeriesInfo(name string) (isSeries bool, season, episode int) {
	// Try S01E02 pattern
	if matches := seriesPatternSE.FindStringSubmatch(name); len(matches) == 3 {
		season, _ = strconv.Atoi(matches[1])
		episode, _ = strconv.Atoi(matches[2])
		return true, season, episode
	}

	// Try 1x02 pattern
	if matches := seriesPatternX.FindStringSubmatch(name); len(matches) == 3 {
		season, _ = strconv.Atoi(matches[1])
		episode, _ = strconv.Atoi(matches[2])
		return true, season, episode
	}

	// Try Season 1 Episode 2 pattern
	if matches := seriesPatternFull.FindStringSubmatch(name); len(matches) == 3 {
		season, _ = strconv.Atoi(matches[1])
		episode, _ = strconv.Atoi(matches[2])
		return true, season, episode
	}

	// Try E01 pattern (assume season 1)
	if matches := seriesPatternEOnly.FindStringSubmatch(name); len(matches) == 2 {
		episode, _ = strconv.Atoi(matches[1])
		return true, 1, episode
	}

	return false, 0, 0
}

// extractYear extracts the release year
func (p *RegexFilenameParser) extractYear(name string) int {
	// Try bracketed year first (more reliable)
	if matches := yearPatternBracket.FindStringSubmatch(name); len(matches) == 2 {
		year, _ := strconv.Atoi(matches[1])
		if year >= 1900 && year <= 2100 {
			return year
		}
	}

	// Try dotted year
	if matches := yearPatternDot.FindStringSubmatch(name); len(matches) == 2 {
		year, _ := strconv.Atoi(matches[1])
		if year >= 1900 && year <= 2100 {
			return year
		}
	}

	return 0
}

// extractQuality extracts video quality
func (p *RegexFilenameParser) extractQuality(name string) string {
	if matches := qualityPattern.FindStringSubmatch(name); len(matches) > 0 {
		quality := strings.ToLower(matches[1])
		// Normalize quality strings
		switch quality {
		case "4k", "uhd":
			return "2160p"
		default:
			return strings.ToLower(matches[1])
		}
	}
	return ""
}

// extractSource extracts video source
func (p *RegexFilenameParser) extractSource(name string) string {
	if matches := sourcePattern.FindStringSubmatch(name); len(matches) > 0 {
		source := strings.ToUpper(matches[1])
		// Normalize source strings
		switch strings.ToUpper(source) {
		case "BLURAY", "BLU-RAY", "BDRIP", "BRRIP":
			return "BluRay"
		case "WEB-DL", "WEBDL":
			return "WEB-DL"
		case "WEBRIP", "WEB":
			return "WEBRip"
		case "DVDRIP", "DVD", "DVDR":
			return "DVD"
		default:
			return source
		}
	}
	return ""
}

// extractEdition extracts special edition info
func (p *RegexFilenameParser) extractEdition(name string) string {
	for _, pattern := range editionPatterns {
		if matches := pattern.FindStringSubmatch(name); len(matches) > 0 {
			edition := strings.TrimSpace(matches[1])
			// Normalize edition strings
			lower := strings.ToLower(edition)
			switch {
			case strings.Contains(lower, "director"):
				return "Director's Cut"
			case strings.Contains(lower, "extended"):
				return "Extended Edition"
			case strings.Contains(lower, "theatrical"):
				return "Theatrical Cut"
			case strings.Contains(lower, "unrated"):
				return "Unrated"
			case strings.Contains(lower, "remaster"):
				return "Remastered"
			case strings.Contains(lower, "special"):
				return "Special Edition"
			case strings.Contains(lower, "final"):
				return "Final Cut"
			case strings.Contains(lower, "ultimate"):
				return "Ultimate Edition"
			case strings.Contains(lower, "criterion"):
				return "Criterion Collection"
			case strings.Contains(lower, "imax"):
				return "IMAX"
			case strings.Contains(lower, "uncut"):
				return "Uncut"
			default:
				return edition
			}
		}
	}
	return ""
}

// extractTitle extracts the clean title from filename
func (p *RegexFilenameParser) extractTitle(name string, parsed *ports.ParsedFilename) string {
	title := name

	// For series, extract title before the series pattern
	if parsed.IsSeries {
		// Find where the series pattern starts and take everything before it
		patterns := []*regexp.Regexp{seriesPatternSE, seriesPatternX, seriesPatternFull, seriesPatternEOnly}
		earliestIdx := len(title)
		for _, pattern := range patterns {
			if loc := pattern.FindStringIndex(title); loc != nil && loc[0] < earliestIdx {
				earliestIdx = loc[0]
			}
		}
		if earliestIdx > 0 && earliestIdx < len(title) {
			title = title[:earliestIdx]
		}
	}

	// For both series and movies, strip the year from the title if present
	if parsed.Year > 0 {
		yearStr := strconv.Itoa(parsed.Year)
		// Find year position (with possible brackets or separators)
		yearPatterns := []string{
			"(" + yearStr + ")",
			"[" + yearStr + "]",
			"." + yearStr + ".",
			" " + yearStr + " ",
			"_" + yearStr + "_",
			"-" + yearStr + "-",
			"." + yearStr,
			" " + yearStr,
			"_" + yearStr,
			"-" + yearStr,
		}
		for _, yp := range yearPatterns {
			if idx := strings.Index(title, yp); idx > 0 {
				title = title[:idx]
				break
			}
		}
	}

	// Remove brackets and their contents
	title = bracketPattern.ReplaceAllString(title, " ")

	// Remove quality, source, edition if still present
	if parsed.Quality != "" {
		title = qualityPattern.ReplaceAllString(title, " ")
	}
	if parsed.Source != "" {
		title = sourcePattern.ReplaceAllString(title, " ")
	}
	for _, pattern := range editionPatterns {
		title = pattern.ReplaceAllString(title, " ")
	}

	// Apply cleanup patterns
	for _, pattern := range cleanupPatterns {
		title = pattern.ReplaceAllString(title, " ")
	}

	// Replace separators with spaces
	title = separatorPattern.ReplaceAllString(title, " ")

	// Clean up multiple spaces
	title = multiSpacePattern.ReplaceAllString(title, " ")

	// Trim
	title = strings.TrimSpace(title)

	return title
}

// cleanForSearch prepares a title for TMDB search
func (p *RegexFilenameParser) cleanForSearch(title string) string {
	// Convert to lowercase
	clean := strings.ToLower(title)

	// Remove special characters except spaces and alphanumerics
	var result strings.Builder
	for _, r := range clean {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			result.WriteRune(r)
		} else if r == '&' {
			result.WriteString(" and ")
		}
	}

	// Clean up multiple spaces
	clean = multiSpacePattern.ReplaceAllString(result.String(), " ")

	return strings.TrimSpace(clean)
}

// Ensure RegexFilenameParser implements FilenameParser
var _ ports.FilenameParser = (*RegexFilenameParser)(nil)
