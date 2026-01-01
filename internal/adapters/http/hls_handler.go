package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/media"
)

// HLSHandler handles HLS streaming endpoints
type HLSHandler struct {
	getMediaUC      *media.GetMediaUseCase
	config          *config.Config
	segmentDuration int
}

// NewHLSHandler creates a new HLS handler
func NewHLSHandler(getMediaUC *media.GetMediaUseCase, config *config.Config) *HLSHandler {
	segmentDuration := config.Transcoding.SegmentDuration
	if segmentDuration == 0 {
		segmentDuration = 4 // default
	}
	return &HLSHandler{
		getMediaUC:      getMediaUC,
		config:          config,
		segmentDuration: segmentDuration,
	}
}

// RegisterRoutes registers HLS streaming routes
func (h *HLSHandler) RegisterRoutes(router *gin.Engine) {
	// Master playlist (separate route)
	router.GET("/stream/hls/:id/master.m3u8", h.ServeMasterPlaylist)

	// All content uses a wildcard pattern under /stream/hls/content/
	// Handles:
	//   - /stream/hls/content/{id}/{quality}/video/stream.m3u8 (video playlist)
	//   - /stream/hls/content/{id}/{quality}/video/{segment} (video segment)
	//   - /stream/hls/content/{id}/audio-{idx}/stream.m3u8 (audio playlist)
	//   - /stream/hls/content/{id}/audio-{idx}/{segment} (audio segment)
	//   - /stream/hls/content/{id}/subtitle-{idx}/{file}.vtt (subtitle file)
	// Register both GET and HEAD methods for proper HTTP handling
	router.Match([]string{"GET", "HEAD"}, "/stream/hls/content/:id/*path", h.ServeContent)

	logger.Debug().Msg("HLS routes registered")
}

// ServeMasterPlaylist handles GET /stream/hls/:id/master.m3u8
func (h *HLSHandler) ServeMasterPlaylist(c *gin.Context) {
	id := c.Param("id")
	qualityFilter := c.Query("quality") // Optional: filter by quality

	logger.Debug().
		Str("media_id", id).
		Str("quality_filter", qualityFilter).
		Msg("serving HLS master playlist")

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
		if len(grouped.VideoTranscodes) == 0 {
			logger.Warn().
				Str("media_id", id).
				Str("quality", qualityFilter).
				Msg("no completed transcode jobs for requested quality")
			c.JSON(http.StatusNotFound, server.ErrorResponse(
				"QUALITY_NOT_FOUND",
				"Requested quality not available",
				fmt.Sprintf("Quality %s is not available", qualityFilter),
			))
			return
		}
	}

	// Generate master playlist
	playlist := h.generateMasterPlaylist(id, grouped, result.AudioStreams, result.SubtitleStreams)

	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "max-age=3600")
	c.String(http.StatusOK, playlist)
}

// ServeContent handles GET /stream/hls/:id/*path
// Unified handler for all HLS content:
//   - /stream/hls/{id}/{quality}/video/stream.m3u8 (video playlist)
//   - /stream/hls/{id}/{quality}/video/{segment} (video segment)
//   - /stream/hls/{id}/audio-{idx}/stream.m3u8 (audio playlist)
//   - /stream/hls/{id}/audio-{idx}/{segment} (audio segment)
//   - /stream/hls/{id}/subtitle-{idx}/{file}.vtt (subtitle file)
func (h *HLSHandler) ServeContent(c *gin.Context) {
	id := c.Param("id")
	path := c.Param("path")

	// Remove leading slash from wildcard path
	path = strings.TrimPrefix(path, "/")

	logger.Debug().
		Str("media_id", id).
		Str("path", path).
		Msg("serving HLS content")

	// Get media to verify it exists
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

	parts := strings.Split(path, "/")

	// Route based on path pattern
	if len(parts) == 2 && strings.HasPrefix(parts[0], "audio-") {
		// Audio: audio-{idx}/stream.m3u8 or audio-{idx}/{segment}
		if parts[1] == "stream.m3u8" {
			h.serveAudioPlaylist(c, id, parts[0], result)
		} else {
			h.serveAudioSegment(c, id, parts[0], parts[1], result)
		}
	} else if len(parts) == 2 && strings.HasPrefix(parts[0], "subtitle-") {
		// Subtitle: subtitle-{idx}/stream.m3u8 (playlist) or subtitle-{idx}/{file}.vtt (file)
		if parts[1] == "stream.m3u8" {
			h.serveSubtitlePlaylist(c, id, parts[0], result)
		} else {
			h.serveSubtitleFile(c, id, parts[1])
		}
	} else if len(parts) == 3 && parts[1] == "video" {
		// Video: {quality}/video/stream.m3u8 or {quality}/video/{segment}
		quality := parts[0]
		if parts[2] == "stream.m3u8" {
			h.serveVideoPlaylist(c, id, quality, result)
		} else {
			h.serveVideoSegment(c, id, quality, parts[2], result)
		}
	} else {
		logger.Warn().Str("path", path).Msg("invalid HLS content path")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_PATH",
			"Invalid content path",
			"",
		))
	}
}

// serveVideoPlaylist serves video media playlist
func (h *HLSHandler) serveVideoPlaylist(c *gin.Context, mediaID, quality string, result *media.GetMediaOutput) {
	// Find video transcode for this quality
	var transcode *domain.Transcode
	for _, t := range result.Transcodes {
		if t.IsCompleted() && t.TrackType == domain.TrackTypeVideo && t.Quality == quality {
			transcode = t
			break
		}
	}

	if transcode == nil {
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"TRANSCODE_NOT_FOUND",
			"Video transcode not found",
			fmt.Sprintf("Quality %s is not available", quality),
		))
		return
	}

	h.serveMediaPlaylistForTranscode(c, transcode)
}

// serveAudioPlaylist serves audio media playlist
func (h *HLSHandler) serveAudioPlaylist(c *gin.Context, mediaID, audioTrack string, result *media.GetMediaOutput) {
	// Parse track index from "audio-{idx}"
	var trackIdx int
	if _, err := fmt.Sscanf(audioTrack, "audio-%d", &trackIdx); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_TRACK",
			"Invalid audio track format",
			"",
		))
		return
	}

	// Find audio transcode for this track (audio has empty quality string)
	var transcode *domain.Transcode
	for _, t := range result.Transcodes {
		if t.IsCompleted() && t.TrackType == domain.TrackTypeAudio && t.TrackIndex == trackIdx {
			transcode = t
			break
		}
	}

	if transcode == nil {
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"TRANSCODE_NOT_FOUND",
			"Audio transcode not found",
			fmt.Sprintf("Audio track %d is not available", trackIdx),
		))
		return
	}

	h.serveMediaPlaylistForTranscode(c, transcode)
}

// serveMediaPlaylistForTranscode generates and serves the media playlist for a transcode
func (h *HLSHandler) serveMediaPlaylistForTranscode(c *gin.Context, transcode *domain.Transcode) {
	// Get list of segments
	entries, err := os.ReadDir(transcode.OutputPath)
	if err != nil {
		logger.Error().
			Err(err).
			Str("output_path", transcode.OutputPath).
			Msg("failed to read output directory")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"INTERNAL_ERROR",
			"Failed to read transcode output",
			"",
		))
		return
	}

	// Collect segment files
	var segments []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".m4s") {
			segments = append(segments, entry.Name())
		}
	}

	// Sort segments by name
	sort.Strings(segments)

	if len(segments) == 0 {
		logger.Error().
			Str("output_path", transcode.OutputPath).
			Msg("no segments found")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"NO_SEGMENTS",
			"No segments found",
			"",
		))
		return
	}

	// Check for init segment
	initSegmentPath := filepath.Join(transcode.OutputPath, "init.mp4")
	hasInitSegment := false
	if _, err := os.Stat(initSegmentPath); err == nil {
		hasInitSegment = true
	}

	// Load segment timing info
	segmentInfo, err := h.loadSegmentInfo(transcode.OutputPath)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("output_path", transcode.OutputPath).
			Msg("failed to load segment timing info, using default duration")
		segmentInfo = nil
	}

	// Generate media playlist with actual timing data
	playlist := h.generateMediaPlaylist(segments, hasInitSegment, segmentInfo)

	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "max-age=3600")
	c.String(http.StatusOK, playlist)
}

// serveVideoSegment serves a video segment file
func (h *HLSHandler) serveVideoSegment(c *gin.Context, mediaID, quality, segment string, result *media.GetMediaOutput) {
	// Find video transcode for this quality
	var transcode *domain.Transcode
	for _, t := range result.Transcodes {
		if t.IsCompleted() && t.TrackType == domain.TrackTypeVideo && t.Quality == quality {
			transcode = t
			break
		}
	}

	if transcode == nil {
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"TRANSCODE_NOT_FOUND",
			"Video transcode not found",
			"",
		))
		return
	}

	h.serveSegmentFile(c, transcode.OutputPath, segment, "video")
}

// serveAudioSegment serves an audio segment file
func (h *HLSHandler) serveAudioSegment(c *gin.Context, mediaID, audioTrack, segment string, result *media.GetMediaOutput) {
	// Parse track index from "audio-{idx}"
	var trackIdx int
	if _, err := fmt.Sscanf(audioTrack, "audio-%d", &trackIdx); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_TRACK",
			"Invalid audio track format",
			"",
		))
		return
	}

	// Find audio transcode for this track
	var transcode *domain.Transcode
	for _, t := range result.Transcodes {
		if t.IsCompleted() && t.TrackType == domain.TrackTypeAudio && t.TrackIndex == trackIdx {
			transcode = t
			break
		}
	}

	if transcode == nil {
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"TRANSCODE_NOT_FOUND",
			"Audio transcode not found",
			"",
		))
		return
	}

	h.serveSegmentFile(c, transcode.OutputPath, segment, "audio")
}

// serveSegmentFile serves a segment file with proper security checks
func (h *HLSHandler) serveSegmentFile(c *gin.Context, outputPath, segment, trackType string) {
	// Validate segment filename
	if strings.Contains(segment, "..") || strings.Contains(segment, "/") || strings.Contains(segment, "\\") {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_SEGMENT",
			"Invalid segment filename",
			"",
		))
		return
	}

	validExt := strings.HasSuffix(segment, ".m4s") ||
		strings.HasSuffix(segment, ".mp4") ||
		segment == "init.mp4"

	if !validExt {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_SEGMENT",
			"Invalid segment extension",
			"",
		))
		return
	}

	// Build path to segment file
	segmentPath := filepath.Join(outputPath, segment)

	// Security: verify path is within output directory
	absSegmentPath, err := filepath.Abs(segmentPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"INTERNAL_ERROR",
			"Failed to resolve segment path",
			"",
		))
		return
	}

	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"INTERNAL_ERROR",
			"Failed to resolve output path",
			"",
		))
		return
	}

	if !strings.HasPrefix(absSegmentPath, absOutputPath) {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_SEGMENT",
			"Invalid segment path",
			"",
		))
		return
	}

	// Verify file exists
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"FILE_NOT_FOUND",
			"Segment file not found",
			"",
		))
		return
	}

	// Set proper headers
	if segment == "init.mp4" || strings.HasSuffix(segment, ".mp4") {
		if trackType == "audio" {
			c.Header("Content-Type", "audio/mp4")
		} else {
			c.Header("Content-Type", "video/mp4")
		}
	} else {
		if trackType == "audio" {
			c.Header("Content-Type", "audio/iso.segment")
		} else {
			c.Header("Content-Type", "video/iso.segment")
		}
	}
	c.Header("Cache-Control", "max-age=31536000")
	c.Header("Accept-Ranges", "bytes")

	c.File(segmentPath)
}

// serveSubtitlePlaylist serves a WebVTT subtitle playlist for HLS
// HLS requires subtitles to be served as m3u8 playlists, not directly as VTT files
func (h *HLSHandler) serveSubtitlePlaylist(c *gin.Context, mediaID, subtitleTrack string, result *media.GetMediaOutput) {
	// Parse track index from "subtitle-{idx}"
	var trackIdx int
	if _, err := fmt.Sscanf(subtitleTrack, "subtitle-%d", &trackIdx); err != nil {
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_TRACK",
			"Invalid subtitle track format",
			"",
		))
		return
	}

	// Verify subtitle transcode exists
	var found bool
	for _, t := range result.Transcodes {
		if t.IsCompleted() && t.TrackType == domain.TrackTypeSubtitle && t.TrackIndex == trackIdx {
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"TRANSCODE_NOT_FOUND",
			"Subtitle transcode not found",
			fmt.Sprintf("Subtitle track %d is not available", trackIdx),
		))
		return
	}

	// Get media duration for the subtitle playlist
	duration := result.Media.Duration
	if duration == 0 {
		duration = 3600 // Fallback to 1 hour if unknown
	}

	// For unsegmented WebVTT, reference the whole VTT file as a single segment
	// The VTT file contains timestamped cues so the player will display correct subtitles
	var sb strings.Builder
	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:7\n")
	sb.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", duration+1))
	sb.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	sb.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")
	sb.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n", float64(duration)))
	sb.WriteString(fmt.Sprintf("subtitle-%d.vtt\n", trackIdx))
	sb.WriteString("#EXT-X-ENDLIST\n")

	c.Header("Content-Type", "application/vnd.apple.mpegurl")
	c.Header("Cache-Control", "max-age=3600")
	c.String(http.StatusOK, sb.String())
}

// serveSubtitleFile handles serving subtitle .vtt files
func (h *HLSHandler) serveSubtitleFile(c *gin.Context, mediaID, subtitleFile string) {
	// Build subtitle path: {media_path}/{media_id}/transcoded/{subtitleFile}
	mediaPath := h.config.Media.MediaPath
	subtitlePath := filepath.Join(mediaPath, mediaID, "transcoded", subtitleFile)

	logger.Debug().
		Str("media_id", mediaID).
		Str("subtitle_file", subtitleFile).
		Str("path", subtitlePath).
		Msg("serving subtitle file")

	// Validate path security
	absSubtitlePath, err := filepath.Abs(subtitlePath)
	if err != nil {
		logger.Error().Err(err).Msg("failed to resolve subtitle path")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"INTERNAL_ERROR",
			"Failed to resolve subtitle path",
			"",
		))
		return
	}

	absMediaPath, err := filepath.Abs(filepath.Join(mediaPath, mediaID))
	if err != nil {
		logger.Error().Err(err).Msg("failed to resolve media path")
		c.JSON(http.StatusInternalServerError, server.ErrorResponse(
			"INTERNAL_ERROR",
			"Failed to resolve media path",
			"",
		))
		return
	}

	if !strings.HasPrefix(absSubtitlePath, absMediaPath) {
		logger.Warn().
			Str("subtitle_path", absSubtitlePath).
			Str("media_path", absMediaPath).
			Msg("subtitle path outside media directory")
		c.JSON(http.StatusBadRequest, server.ErrorResponse(
			"INVALID_PATH",
			"Invalid subtitle path",
			"",
		))
		return
	}

	// Verify file exists
	if _, err := os.Stat(subtitlePath); os.IsNotExist(err) {
		logger.Warn().
			Str("subtitle_file", subtitleFile).
			Str("path", subtitlePath).
			Msg("subtitle file not found")
		c.JSON(http.StatusNotFound, server.ErrorResponse(
			"FILE_NOT_FOUND",
			"Subtitle file not found",
			"",
		))
		return
	}

	// Set proper headers for WebVTT
	c.Header("Content-Type", "text/vtt")
	c.Header("Cache-Control", "max-age=31536000")
	c.Header("Accept-Ranges", "bytes")

	c.File(subtitlePath)
}

// generateMasterPlaylist generates an HLS master playlist
func (h *HLSHandler) generateMasterPlaylist(
	mediaID string,
	grouped *GroupedTranscodes,
	audioStreams []*domain.AudioStream,
	subtitleStreams []*domain.SubtitleStream,
) string {
	var sb strings.Builder

	// HLS master playlist header
	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:7\n")
	sb.WriteString("#EXT-X-INDEPENDENT-SEGMENTS\n")
	sb.WriteString("\n")

	// Audio renditions (EXT-X-MEDIA tags)
	// Use StreamIndex (actual DB index) not slice index
	isFirstAudio := true
	for _, audioStream := range audioStreams {
		streamIdx := audioStream.StreamIndex
		if _, exists := grouped.AudioTranscodes[streamIdx]; !exists {
			continue
		}

		// Get language and name
		lang := audioStream.Language
		if lang == "" {
			lang = "und"
		}

		name := audioStream.Title
		if name == "" {
			if lang != "" && lang != "und" {
				name = lang
			} else {
				name = fmt.Sprintf("Audio %d", streamIdx)
			}
		}

		// Audio is always transcoded to stereo (2 channels)
		// regardless of source channel count (e.g., 5.1 surround -> stereo)
		channels := "2"

		defaultAttr := ""
		if isFirstAudio {
			defaultAttr = ",DEFAULT=YES"
			isFirstAudio = false
		}

		// Audio is transcoded once (no quality variants), use empty string quality
		// Use absolute path since content is served from /stream/hls/content/
		sb.WriteString(fmt.Sprintf("#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID=\"audio\",NAME=\"%s\",LANGUAGE=\"%s\"%s,AUTOSELECT=YES,CHANNELS=\"%s\",URI=\"/stream/hls/content/%s/audio-%d/stream.m3u8\"\n",
			name, lang, defaultAttr, channels, mediaID, streamIdx))
	}

	// Subtitle renditions (EXT-X-MEDIA tags)
	// Use StreamIndex (actual DB index) not slice index
	isFirstSubtitle := true
	for _, subtitleStream := range subtitleStreams {
		streamIdx := subtitleStream.StreamIndex
		if _, exists := grouped.SubtitleTranscodes[streamIdx]; !exists {
			continue
		}

		lang := subtitleStream.Language
		if lang == "" {
			lang = "und"
		}

		name := subtitleStream.Title
		if name == "" {
			if lang != "" && lang != "und" {
				name = lang
			} else {
				name = fmt.Sprintf("Subtitle %d", streamIdx)
			}
		}

		defaultAttr := ""
		if isFirstSubtitle {
			defaultAttr = ",DEFAULT=YES"
			isFirstSubtitle = false
		}

		// Subtitles must be served as m3u8 playlists in HLS, not directly as VTT files
		// FORCED=NO indicates these are regular subtitles, not forced narrative subtitles
		sb.WriteString(fmt.Sprintf("#EXT-X-MEDIA:TYPE=SUBTITLES,GROUP-ID=\"subs\",NAME=\"%s\",LANGUAGE=\"%s\"%s,AUTOSELECT=YES,FORCED=NO,URI=\"/stream/hls/content/%s/subtitle-%d/stream.m3u8\"\n",
			name, lang, defaultAttr, mediaID, streamIdx))
	}

	sb.WriteString("\n")

	// Sort video qualities
	qualities := getQualitiesFromGroup(grouped)

	// Video variants (EXT-X-STREAM-INF tags)
	for _, quality := range qualities {
		height := extractHeight(quality)
		width := height * 16 / 9 // Assume 16:9 aspect ratio
		bandwidth := h.estimateBandwidth(quality)

		audioAttr := ""
		if len(grouped.AudioTranscodes) > 0 {
			audioAttr = ",AUDIO=\"audio\""
		}

		subsAttr := ""
		if len(grouped.SubtitleTranscodes) > 0 {
			subsAttr = ",SUBTITLES=\"subs\""
		}

		sb.WriteString(fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,CODECS=\"avc1.4d401f,mp4a.40.2\"%s%s\n",
			bandwidth, width, height, audioAttr, subsAttr))
		sb.WriteString(fmt.Sprintf("/stream/hls/content/%s/%s/video/stream.m3u8\n", mediaID, quality))
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateMediaPlaylist generates an HLS media playlist
func (h *HLSHandler) generateMediaPlaylist(segments []string, hasInitSegment bool, segmentInfo *HLSSegmentInfo) string {
	var sb strings.Builder

	// Calculate max duration for TARGETDURATION
	maxDuration := h.segmentDuration
	if segmentInfo != nil && len(segmentInfo.Segments) > 0 {
		for _, seg := range segmentInfo.Segments {
			durationSec := (seg.Duration + 999) / 1000 // Round up to seconds
			if durationSec > maxDuration {
				maxDuration = durationSec
			}
		}
	}

	sb.WriteString("#EXTM3U\n")
	sb.WriteString("#EXT-X-VERSION:7\n")
	sb.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", maxDuration+1)) // Add 1 second buffer
	sb.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	sb.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")

	// Add init segment if present
	if hasInitSegment {
		sb.WriteString("#EXT-X-MAP:URI=\"init.mp4\"\n")
	}

	// Add each segment with actual duration from segments.json
	for i, seg := range segments {
		var duration float64
		if segmentInfo != nil && i < len(segmentInfo.Segments) {
			// Use actual duration in seconds (milliseconds / 1000)
			duration = float64(segmentInfo.Segments[i].Duration) / 1000.0
		} else {
			// Fallback to default duration
			duration = float64(h.segmentDuration)
		}
		sb.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n", duration))
		sb.WriteString(fmt.Sprintf("%s\n", seg))
	}

	sb.WriteString("#EXT-X-ENDLIST\n")

	return sb.String()
}

// estimateBandwidth estimates bandwidth for a quality level
func (h *HLSHandler) estimateBandwidth(quality string) int {
	// Find matching quality profile
	for _, profile := range h.config.Transcoding.QualityProfiles {
		if profile.Name == quality {
			// Parse max bitrate (e.g., "1500k" -> 1500000)
			bitrate := profile.MaxBitrate
			bitrate = strings.TrimSuffix(bitrate, "k")
			bitrate = strings.TrimSuffix(bitrate, "K")

			val := 0
			fmt.Sscanf(bitrate, "%d", &val)
			return val * 1000 // Convert to bps
		}
	}

	// Default fallback
	return 2000000
}

// HLSSegmentTiming represents the timing information for a media segment
type HLSSegmentTiming struct {
	Number   int `json:"Number"`
	Duration int `json:"Duration"` // milliseconds
}

// HLSSegmentInfo contains metadata about transcoded segments
type HLSSegmentInfo struct {
	Segments []HLSSegmentTiming `json:"segments"`
}

// loadHLSSegmentInfo loads segment metadata from segments.json
func (h *HLSHandler) loadSegmentInfo(outputPath string) (*HLSSegmentInfo, error) {
	segmentFile := filepath.Join(outputPath, "segments.json")
	data, err := os.ReadFile(segmentFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read segments.json: %w", err)
	}

	var info HLSSegmentInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse segments.json: %w", err)
	}

	return &info, nil
}
