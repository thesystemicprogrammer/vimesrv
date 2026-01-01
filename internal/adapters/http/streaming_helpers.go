package http

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
)

// GroupedTranscodes holds transcodes organized by track type and quality
type GroupedTranscodes struct {
	VideoTranscodes    map[string]*domain.Transcode         // quality -> transcode
	AudioTranscodes    map[int]map[string]*domain.Transcode // trackIndex -> quality -> transcode
	SubtitleTranscodes map[int]*domain.Transcode            // trackIndex -> transcode
}

// filterCompletedTranscodes returns only completed transcode jobs
func filterCompletedTranscodes(transcodes []*domain.Transcode) []*domain.Transcode {
	var completed []*domain.Transcode
	for _, t := range transcodes {
		if t.IsCompleted() {
			completed = append(completed, t)
		}
	}
	return completed
}

// groupTranscodesByTrack groups transcode jobs by track type
func groupTranscodesByTrack(transcodes []*domain.Transcode) *GroupedTranscodes {
	grouped := &GroupedTranscodes{
		VideoTranscodes:    make(map[string]*domain.Transcode),
		AudioTranscodes:    make(map[int]map[string]*domain.Transcode),
		SubtitleTranscodes: make(map[int]*domain.Transcode),
	}

	for _, t := range transcodes {
		switch t.TrackType {
		case domain.TrackTypeVideo:
			grouped.VideoTranscodes[t.Quality] = t

		case domain.TrackTypeAudio:
			if grouped.AudioTranscodes[t.TrackIndex] == nil {
				grouped.AudioTranscodes[t.TrackIndex] = make(map[string]*domain.Transcode)
			}
			grouped.AudioTranscodes[t.TrackIndex][t.Quality] = t

		case domain.TrackTypeSubtitle:
			grouped.SubtitleTranscodes[t.TrackIndex] = t
		}
	}

	return grouped
}

// filterGroupedByQuality filters grouped transcodes by quality
// Returns only transcodes for the specified quality
func filterGroupedByQuality(grouped *GroupedTranscodes, quality string) *GroupedTranscodes {
	filtered := &GroupedTranscodes{
		VideoTranscodes:    make(map[string]*domain.Transcode),
		AudioTranscodes:    make(map[int]map[string]*domain.Transcode),
		SubtitleTranscodes: grouped.SubtitleTranscodes, // Subtitles don't have quality variations
	}

	// Filter video for this quality
	if transcode, exists := grouped.VideoTranscodes[quality]; exists {
		filtered.VideoTranscodes[quality] = transcode
	}

	// Filter audio for this quality
	for audioIdx, qualityMap := range grouped.AudioTranscodes {
		if transcode, exists := qualityMap[quality]; exists {
			filtered.AudioTranscodes[audioIdx] = make(map[string]*domain.Transcode)
			filtered.AudioTranscodes[audioIdx][quality] = transcode
		}
	}

	return filtered
}

// extractHeight extracts height from quality string (e.g., "720p" -> 720)
func extractHeight(quality string) int {
	// Remove 'p' suffix if present
	quality = strings.TrimSuffix(quality, "p")

	var height int
	_, err := fmt.Sscanf(quality, "%d", &height)
	if err != nil {
		return 0
	}
	return height
}

// getQualitiesFromGroup returns a sorted list of available qualities from grouped transcodes
func getQualitiesFromGroup(grouped *GroupedTranscodes) []string {
	// Use a map to deduplicate and collect qualities
	qualityMap := make(map[string]bool)
	for quality := range grouped.VideoTranscodes {
		qualityMap[quality] = true
	}

	// Convert to slice and sort by height (descending)
	qualities := make([]string, 0, len(qualityMap))
	for quality := range qualityMap {
		qualities = append(qualities, quality)
	}

	// Sort by height descending (1080p, 720p, 480p, 360p)
	sort.Slice(qualities, func(i, j int) bool {
		return extractHeight(qualities[i]) > extractHeight(qualities[j])
	})

	return qualities
}
