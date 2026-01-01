package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/media"
)

// DASHHandler handles DASH streaming endpoints
type DASHHandler struct {
	getMediaUC      *media.GetMediaUseCase
	config          *config.Config
	segmentDuration int
}

// NewDASHHandler creates a new DASH handler
func NewDASHHandler(getMediaUC *media.GetMediaUseCase, config *config.Config) *DASHHandler {
	segmentDuration := config.Transcoding.SegmentDuration
	if segmentDuration == 0 {
		segmentDuration = 4 // default
	}
	return &DASHHandler{
		getMediaUC:      getMediaUC,
		config:          config,
		segmentDuration: segmentDuration,
	}
}

// RegisterRoutes registers DASH streaming routes
func (h *DASHHandler) RegisterRoutes(router *gin.Engine) {
	// Manifest endpoint (separate from content)
	router.GET("/stream/dash/:id/manifest.mpd", h.ServeManifest)

	// All content uses a wildcard pattern under /stream/dash/content/
	// Handles:
	//   - /stream/dash/content/{id}/{quality}/video/{segment} (video)
	//   - /stream/dash/content/{id}/audio-{idx}/{segment} (audio)
	//   - /stream/dash/content/{id}/subtitle-{idx}.vtt (subtitle)
	// Use Match to register both GET and HEAD methods
	router.Match([]string{"GET", "HEAD"}, "/stream/dash/content/:id/*path", h.ServeContent)

	logger.Debug().Msg("DASH routes registered")
}

// ServeManifest handles GET /stream/dash/:id/manifest.mpd
func (h *DASHHandler) ServeManifest(c *gin.Context) {
	id := c.Param("id")
	qualityFilter := c.Query("quality") // Optional: filter by quality

	logger.Debug().
		Str("media_id", id).
		Str("quality_filter", qualityFilter).
		Msg("serving DASH manifest")

	// Get media and all related data
	result, err := h.getMediaUC.Execute(c.Request.Context(), media.GetMediaInput{MediaID: id})
	if err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to get media")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"MEDIA_NOT_FOUND",
			"Media not found",
			err.Error(),
		))
		return
	}

	// Filter for completed transcodes
	completedTranscodes := filterCompletedTranscodes(result.Transcodes)
	if len(completedTranscodes) == 0 {
		logger.Warn().Str("media_id", id).Msg("no completed transcode jobs found")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"NO_TRANSCODES",
			"No completed transcode jobs found",
			"",
		))
		return
	}

	// Group transcodes by track type
	grouped := groupTranscodesByTrack(completedTranscodes)

	// Filter by quality if specified
	if qualityFilter != "" {
		grouped = filterGroupedByQuality(grouped, qualityFilter)
	}

	// Generate MPD manifest
	mpd, err := h.generateMPD(c.Request.Context(), id, result.Media, grouped, result.AudioStreams, result.SubtitleStreams)
	if err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to generate MPD")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"MPD_GENERATION_FAILED",
			"Failed to generate DASH manifest",
			err.Error(),
		))
		return
	}

	c.Header("Content-Type", "application/dash+xml")
	c.Header("Cache-Control", "max-age=3600")
	c.String(http.StatusOK, mpd)
}

// ServeContent handles GET /stream/dash/:id/*path
// Unified handler for all DASH content:
//   - /stream/dash/{id}/{quality}/video/{segment} (video segments)
//   - /stream/dash/{id}/audio-{idx}/{segment} (audio segments)
//   - /stream/dash/{id}/subtitle-{idx}.vtt (subtitle files)
func (h *DASHHandler) ServeContent(c *gin.Context) {
	id := c.Param("id")
	path := c.Param("path") // Wildcard path after /:id/

	// Remove leading slash from wildcard path
	path = strings.TrimPrefix(path, "/")

	logger.Debug().
		Str("media_id", id).
		Str("path", path).
		Str("method", c.Request.Method).
		Msg("serving DASH content")

	// Get media to verify it exists
	_, err := h.getMediaUC.Execute(c.Request.Context(), media.GetMediaInput{MediaID: id})
	if err != nil {
		logger.Error().Err(err).Str("media_id", id).Msg("failed to get media")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"MEDIA_NOT_FOUND",
			"Media not found",
			err.Error(),
		))
		return
	}

	// Parse path to determine content type and build file path
	pathInfo := parseContentPath(path)
	if pathInfo == nil {
		logger.Warn().Str("path", path).Msg("invalid DASH content path")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_PATH",
			"Invalid content path",
			"",
		))
		return
	}

	mediaPath := h.config.Media.MediaPath
	filePath := filepath.Join(mediaPath, id, pathInfo.FilePath)

	// Security: prevent directory traversal
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		logger.Error().Err(err).Str("path", filePath).Msg("failed to resolve absolute path")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_PATH",
			"Invalid content path",
			"",
		))
		return
	}

	absMediaPath, err := filepath.Abs(filepath.Join(mediaPath, id))
	if err != nil || !strings.HasPrefix(absFilePath, absMediaPath) {
		logger.Warn().
			Str("file_path", absFilePath).
			Str("media_path", absMediaPath).
			Msg("potential directory traversal attempt")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_PATH",
			"Invalid content path",
			"",
		))
		return
	}

	// Check if file exists
	if _, err := os.Stat(absFilePath); os.IsNotExist(err) {
		logger.Warn().Str("path", absFilePath).Msg("content file not found")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"CONTENT_NOT_FOUND",
			"Content not found",
			"",
		))
		return
	}

	logger.Debug().Str("media_id", id).Str("path", absFilePath).Msg("serving content file")

	c.Header("Content-Type", pathInfo.ContentType)
	c.Header("Cache-Control", "max-age=31536000") // Content is immutable
	c.File(absFilePath)
}

// generateMPD creates a DASH MPD manifest
func (h *DASHHandler) generateMPD(
	ctx context.Context,
	mediaID string,
	mediaFile *domain.MediaFile,
	grouped *GroupedTranscodes,
	audioStreams []*domain.AudioStream,
	subtitleStreams []*domain.SubtitleStream,
) (string, error) {
	qualities := getQualitiesFromGroup(grouped)
	durationISO := fmt.Sprintf("PT%dS", mediaFile.Duration)

	var sb strings.Builder

	// Write MPD header
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString("\n")
	sb.WriteString(`<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" `)
	sb.WriteString(`profiles="urn:mpeg:dash:profile:isoff-on-demand:2011" `)
	sb.WriteString(`type="static" `)
	sb.WriteString(fmt.Sprintf(`mediaPresentationDuration="%s">`, durationISO))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(`  <Period duration="%s">`, durationISO))
	sb.WriteString("\n")

	// Build adaptation sets
	sb.WriteString(h.buildVideoAdaptationSet(mediaID, qualities, grouped))
	sb.WriteString(h.buildAudioAdaptationSets(mediaID, audioStreams, grouped))
	sb.WriteString(h.buildSubtitleAdaptationSets(mediaID, subtitleStreams, grouped))

	// Close Period and MPD
	sb.WriteString(`  </Period>`)
	sb.WriteString("\n")
	sb.WriteString(`</MPD>`)
	sb.WriteString("\n")

	return sb.String(), nil
}

// buildVideoAdaptationSet builds the video adaptation set XML for the MPD
func (h *DASHHandler) buildVideoAdaptationSet(mediaID string, qualities []string, grouped *GroupedTranscodes) string {
	if len(qualities) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`    <AdaptationSet mimeType="video/mp4" segmentAlignment="true" startWithSAP="1">`)
	sb.WriteString("\n")

	for _, quality := range qualities {
		transcode := grouped.VideoTranscodes[quality]
		if transcode == nil {
			continue
		}

		segmentInfo, err := h.loadSegmentInfo(transcode.OutputPath)
		if err != nil {
			logger.Warn().Err(err).Str("quality", quality).Msg("failed to load segment info")
			continue
		}

		height := extractHeight(quality)
		sb.WriteString(fmt.Sprintf(`      <Representation id="video_%s" bandwidth="%d" width="%d" height="%d" codecs="avc1.4d401f">`,
			quality, 5000000, height*16/9, height))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf(`        <BaseURL>/stream/dash/content/%s/%s/video/</BaseURL>`, mediaID, quality))
		sb.WriteString("\n")
		sb.WriteString(`        <SegmentTemplate timescale="1000" initialization="init.mp4" media="chunk-$Number%03d$.m4s" startNumber="0">`)
		sb.WriteString("\n")
		sb.WriteString(generateSegmentTimeline(segmentInfo.Segments))
		sb.WriteString("        </SegmentTemplate>")
		sb.WriteString("\n")
		sb.WriteString(`      </Representation>`)
		sb.WriteString("\n")
	}

	sb.WriteString(`    </AdaptationSet>`)
	sb.WriteString("\n")

	return sb.String()
}

// buildAudioAdaptationSets builds the audio adaptation sets XML for the MPD
func (h *DASHHandler) buildAudioAdaptationSets(mediaID string, audioStreams []*domain.AudioStream, grouped *GroupedTranscodes) string {
	var sb strings.Builder

	for _, audioStream := range audioStreams {
		streamIdx := audioStream.StreamIndex
		if _, exists := grouped.AudioTranscodes[streamIdx]; !exists {
			continue
		}

		lang := audioStream.Language
		if lang == "" {
			lang = "und"
		}

		// Build a display label for the audio track
		label := audioStream.Title
		if label == "" {
			if audioStream.Language != "" {
				label = shared.LanguageCodeToName(audioStream.Language)
			} else {
				label = fmt.Sprintf("Audio %d", streamIdx)
			}
		}

		sb.WriteString(fmt.Sprintf(`    <AdaptationSet mimeType="audio/mp4" segmentAlignment="true" lang="%s" label="%s">`, lang, label))
		sb.WriteString("\n")

		qualityMap := grouped.AudioTranscodes[streamIdx]
		transcode := qualityMap[""] // Audio uses empty quality string

		if transcode == nil {
			logger.Warn().Int("stream_idx", streamIdx).Msg("audio transcode not found, skipping audio track")
			sb.WriteString(`    </AdaptationSet>`)
			sb.WriteString("\n")
			continue
		}

		sb.WriteString(fmt.Sprintf(`      <Representation id="audio_%d" bandwidth="128000" codecs="mp4a.40.2" audioSamplingRate="48000">`,
			streamIdx))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf(`        <BaseURL>/stream/dash/content/%s/audio-%d/</BaseURL>`, mediaID, streamIdx))
		sb.WriteString("\n")

		segmentInfo, err := h.loadSegmentInfo(transcode.OutputPath)
		if err == nil && len(segmentInfo.Segments) > 0 {
			sb.WriteString(`        <SegmentTemplate timescale="1000" initialization="init.mp4" media="chunk-$Number%03d$.m4s" startNumber="0">`)
			sb.WriteString("\n")
			sb.WriteString(generateSegmentTimeline(segmentInfo.Segments))
			sb.WriteString("        </SegmentTemplate>")
		} else {
			logger.Warn().Err(err).Int("stream_idx", streamIdx).Msg("audio segment timing data not available, using fixed duration")
			sb.WriteString(`        <SegmentTemplate timescale="1000" duration="4000" initialization="init.mp4" media="chunk-$Number%03d$.m4s" startNumber="0"/>`)
		}

		sb.WriteString("\n")
		sb.WriteString(`      </Representation>`)
		sb.WriteString("\n")
		sb.WriteString(`    </AdaptationSet>`)
		sb.WriteString("\n")
	}

	return sb.String()
}

// buildSubtitleAdaptationSets builds the subtitle adaptation sets XML for the MPD
func (h *DASHHandler) buildSubtitleAdaptationSets(mediaID string, subtitleStreams []*domain.SubtitleStream, grouped *GroupedTranscodes) string {
	var sb strings.Builder

	for _, subtitleStream := range subtitleStreams {
		streamIdx := subtitleStream.StreamIndex
		if _, exists := grouped.SubtitleTranscodes[streamIdx]; !exists {
			continue
		}

		lang := subtitleStream.Language
		if lang == "" {
			lang = "und"
		}

		// Build a display label for the subtitle track
		label := subtitleStream.Title
		if label == "" {
			if subtitleStream.Language != "" {
				label = shared.LanguageCodeToName(subtitleStream.Language)
			} else {
				label = fmt.Sprintf("Subtitle %d", streamIdx)
			}
		}

		transcode := grouped.SubtitleTranscodes[streamIdx]
		subtitleFilename := filepath.Base(transcode.OutputPath)

		sb.WriteString(fmt.Sprintf(`    <AdaptationSet mimeType="text/vtt" lang="%s" label="%s">`, lang, label))
		sb.WriteString("\n")
		sb.WriteString(`      <Role schemeIdUri="urn:mpeg:dash:role:2011" value="subtitle"/>`)
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf(`      <Representation id="subtitle_%d">`, streamIdx))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf(`        <BaseURL>/stream/dash/content/%s/%s</BaseURL>`, mediaID, subtitleFilename))
		sb.WriteString("\n")
		sb.WriteString(`      </Representation>`)
		sb.WriteString("\n")
		sb.WriteString(`    </AdaptationSet>`)
		sb.WriteString("\n")
	}

	return sb.String()
}

// SegmentTiming represents the timing information for a media segment
type SegmentTiming struct {
	Number   int `json:"Number"`
	Duration int `json:"Duration"` // milliseconds
}

// SegmentInfo contains metadata about transcoded segments
type SegmentInfo struct {
	Segments []SegmentTiming `json:"segments"`
}

// loadSegmentInfo loads segment metadata from segments.json
func (h *DASHHandler) loadSegmentInfo(outputPath string) (*SegmentInfo, error) {
	segmentFile := filepath.Join(outputPath, "segments.json")
	data, err := os.ReadFile(segmentFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read segments.json: %w", err)
	}

	var info SegmentInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse segments.json: %w", err)
	}

	return &info, nil
}

// contentPathInfo contains the parsed information from a DASH content path
type contentPathInfo struct {
	FilePath    string // Relative path to the content file
	ContentType string // MIME type for the content
}

// parseContentPath parses a DASH content path and returns the file path and content type.
// Returns nil if the path is invalid.
// Supported patterns:
//   - "{quality}/video/{segment}" -> video segment
//   - "audio-{idx}/{segment}" -> audio segment
//   - "subtitle-{idx}.vtt" -> subtitle file
func parseContentPath(path string) *contentPathInfo {
	parts := strings.Split(path, "/")

	if len(parts) == 1 && strings.HasPrefix(parts[0], "subtitle-") && strings.HasSuffix(parts[0], ".vtt") {
		// Subtitle file: subtitle-{idx}.vtt
		return &contentPathInfo{
			FilePath:    filepath.Join("transcoded", parts[0]),
			ContentType: "text/vtt",
		}
	}

	if len(parts) == 2 && strings.HasPrefix(parts[0], "audio-") {
		// Audio segment: audio-{idx}/{segment}
		contentType := "audio/mp4"
		if strings.HasSuffix(parts[1], ".m4s") {
			contentType = "audio/iso.segment"
		}
		return &contentPathInfo{
			FilePath:    filepath.Join("transcoded", parts[0], parts[1]),
			ContentType: contentType,
		}
	}

	if len(parts) == 3 && parts[1] == "video" {
		// Video segment: {quality}/video/{segment}
		contentType := "video/mp4"
		if strings.HasSuffix(parts[2], ".m4s") {
			contentType = "video/iso.segment"
		}
		return &contentPathInfo{
			FilePath:    filepath.Join("transcoded", parts[0], parts[1], parts[2]),
			ContentType: contentType,
		}
	}

	return nil
}

// generateSegmentTimeline generates the SegmentTimeline XML from timing data
func generateSegmentTimeline(timings []SegmentTiming) string {
	if len(timings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("          <SegmentTimeline>\n")

	// Group consecutive segments with same duration using 'r' attribute
	i := 0
	for i < len(timings) {
		currentDuration := timings[i].Duration
		repeatCount := 0

		// Count how many consecutive segments have the same duration
		j := i + 1
		for j < len(timings) && timings[j].Duration == currentDuration {
			repeatCount++
			j++
		}

		// Generate <S> element
		if i == 0 {
			// First segment includes 't' attribute (start time = 0)
			if repeatCount > 0 {
				sb.WriteString(fmt.Sprintf(`            <S t="0" d="%d" r="%d"/>`, currentDuration, repeatCount))
			} else {
				sb.WriteString(fmt.Sprintf(`            <S t="0" d="%d"/>`, currentDuration))
			}
		} else {
			// Subsequent segments (time is implicit)
			if repeatCount > 0 {
				sb.WriteString(fmt.Sprintf(`            <S d="%d" r="%d"/>`, currentDuration, repeatCount))
			} else {
				sb.WriteString(fmt.Sprintf(`            <S d="%d"/>`, currentDuration))
			}
		}
		sb.WriteString("\n")

		i = j
	}

	sb.WriteString("          </SegmentTimeline>\n")
	return sb.String()
}
