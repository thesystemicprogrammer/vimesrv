# Specification: Smart Playback with Direct Play / Direct Stream

This specification describes the implementation of intelligent playback mode selection, similar to Plex/Jellyfin's approach: attempt direct play of the original file when possible, with automatic fallback to DASH adaptive streaming when needed.

---

## 1. Overview

### 1.1 Current State

The player currently:
- Always uses DASH streaming via dash.js
- Requires completed transcodes before playback is possible
- Has no codec capability detection
- Has no bandwidth detection (relies on dash.js ABR)
- Has minimal error recovery (shows error, stops playback)

**Key files:**
- `web/pwa/src/app/features/player/player.component.ts` - Main player component
- `internal/adapters/http/dash_handler.go` - DASH manifest and segment serving
- `internal/adapters/http/media_handler.go` - Media API endpoints
- `internal/domain/media_file.go` - Media domain model (has codec info, not exposed via API)

### 1.2 Target State

The player will:
- Attempt direct play of original files when conditions allow
- Support direct stream (on-the-fly remux) for MKV/AVI with compatible codecs
- Fall back to DASH when direct play isn't viable
- Monitor playback health and seamlessly switch modes if needed
- Use already-transcoded VTT subtitles during direct play

### 1.3 Playback Modes

| Mode | When Used | Server Action | Client |
|------|-----------|---------------|--------|
| **Direct Play** | MP4 container + browser-supported codec + sufficient bandwidth | Serve original file with Range support | Native `<video src>` |
| **Direct Stream** | MKV/AVI container + browser-supported codec + sufficient bandwidth | On-the-fly remux to fragmented MP4 (no transcode) | Native `<video src>` |
| **DASH** | Codec unsupported OR bandwidth insufficient OR fallback triggered | Serve pre-transcoded segments | dash.js |

---

## 2. Architecture

### 2.1 Decision Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              PLAYBACK DECISION FLOW                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. Fetch Media Metadata (includes source codec info)                       │
│         ↓                                                                   │
│  2. Check Container                                                         │
│      ├── MP4/MOV → candidate for Direct Play                                │
│      └── MKV/AVI → candidate for Direct Stream                              │
│         ↓                                                                   │
│  3. Check Codec Support (MediaCapabilities API)                             │
│      ├── Supported → continue                                               │
│      └── Unsupported → DASH                                                 │
│         ↓                                                                   │
│  4. Check Subtitles                                                         │
│      ├── No image-based subs → continue                                     │
│      └── Has PGS/VOBSUB → DASH                                              │
│         ↓                                                                   │
│  5. Bandwidth Probe                                                         │
│      ├── Sufficient (bitrate × 1.3) → continue                              │
│      └── Insufficient → DASH                                                │
│         ↓                                                                   │
│  6. Start Playback                                                          │
│      ├── MP4 → Direct Play                                                  │
│      └── MKV/AVI → Direct Stream (remux)                                    │
│         ↓                                                                   │
│  7. Monitor Playback Health                                                 │
│      ├── Healthy → continue                                                 │
│      └── Problems (stalls, buffer issues) → Fallback to DASH                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                  SERVER (Go)                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐  │
│  │   Media Handler     │  │   Direct Handler    │  │   Remux Handler     │  │
│  │                     │  │                     │  │                     │  │
│  │ GET /api/v1/media/  │  │ GET /stream/direct/ │  │ GET /stream/remux/  │  │
│  │ :id                 │  │ :id                 │  │ :id                 │  │
│  │                     │  │                     │  │                     │  │
│  │ Returns:            │  │ - Range requests    │  │ - ffmpeg remux      │  │
│  │ - source codec info │  │ - Original MP4 file │  │ - Fragmented MP4    │  │
│  │ - direct play URLs  │  │ - Stream token auth │  │ - Stream token auth │  │
│  └─────────────────────┘  └─────────────────────┘  └─────────────────────┘  │
│                                                                             │
│  ┌─────────────────────┐  ┌─────────────────────┐                           │
│  │   Probe Handler     │  │   DASH Handler      │                           │
│  │                     │  │   (existing)        │                           │
│  │ GET /stream/probe   │  │                     │                           │
│  │                     │  │ Manifest + segments │                           │
│  │ - Bandwidth test    │  │                     │                           │
│  └─────────────────────┘  └─────────────────────┘                           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT (Angular)                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐  │
│  │ PlaybackCapability  │  │   Bandwidth         │  │ PlaybackDecision    │  │
│  │ Service             │  │   Service           │  │ Service             │  │
│  │                     │  │                     │  │                     │  │
│  │ - MediaCapabilities │  │ - Probe endpoint    │  │ - Orchestrates      │  │
│  │ - Codec detection   │  │ - Throughput calc   │  │   capability +      │  │
│  │ - Container check   │  │ - Caching           │  │   bandwidth checks  │  │
│  │ - Subtitle check    │  │                     │  │ - Returns decision  │  │
│  └─────────────────────┘  └─────────────────────┘  └─────────────────────┘  │
│                                                                             │
│  ┌─────────────────────┐  ┌─────────────────────────────────────────────┐   │
│  │ PlaybackMonitor     │  │              Player Component               │   │
│  │ Service             │  │                                             │   │
│  │                     │  │ - Runs decision logic on load               │   │
│  │ - Buffer tracking   │  │ - Initializes direct or DASH player         │   │
│  │ - Stall detection   │  │ - Handles fallback when needed              │   │
│  │ - Fallback trigger  │  │ - Loads VTT subtitles for direct play       │   │
│  └─────────────────────┘  └─────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Server-Side Implementation

### 3.1 Extend Media API Response

**File:** `internal/adapters/http/media_handler.go`

Update `MediaDetailResponse` struct:

```go
type MediaDetailResponse struct {
    // Existing fields
    ID                 string           `json:"id"`
    Title              string           `json:"title"`
    Filename           string           `json:"filename"`
    Duration           int              `json:"duration"`
    Resolution         string           `json:"resolution"`
    Width              int              `json:"width"`
    Height             int              `json:"height"`
    Status             string           `json:"status"`
    DashManifestURL    string           `json:"dash_manifest_url"`
    AudioStreams       []AudioStreamDTO `json:"audio_streams"`
    SubtitleStreams    []SubtitleDTO    `json:"subtitle_streams"`
    AvailableQualities []string         `json:"available_qualities"`

    // NEW: Source file info for playback decision
    Format      string   `json:"format"`        // "mp4", "mkv", "avi", "mov"
    VideoCodec  string   `json:"video_codec"`   // "h264", "hevc", "av1", "vp9"
    AudioCodecs []string `json:"audio_codecs"`  // ["aac", "ac3", "dts", "eac3"]
    Bitrate     int      `json:"bitrate"`       // Total bitrate in bits/sec
    FileSize    int64    `json:"file_size"`     // File size in bytes

    // NEW: Direct play eligibility (computed server-side)
    DirectPlaySupported   bool   `json:"direct_play_supported"`    // true if MP4/MOV
    DirectStreamSupported bool   `json:"direct_stream_supported"`  // true if MKV/AVI (needs remux)
    DirectPlayURL         string `json:"direct_play_url,omitempty"`   // /stream/direct/{id}
    DirectStreamURL       string `json:"direct_stream_url,omitempty"` // /stream/remux/{id}
}
```

### 3.2 Direct Play Handler

**New file:** `internal/adapters/http/direct_handler.go`

- Authenticate via stream token (same as DASH)
- Serve original file using `http.ServeContent` (handles Range requests automatically)
- Set proper headers: `Accept-Ranges: bytes`, correct `Content-Type`
- Security: validate path, prevent directory traversal

### 3.3 Remux Handler (Direct Stream)

**New file:** `internal/adapters/http/remux_handler.go`

- For MKV/AVI files with browser-compatible codecs
- Use ffmpeg to remux on-the-fly: `ffmpeg -i input.mkv -c copy -f mp4 -movflags frag_keyframe+empty_moov -`
- Stream the output directly to the HTTP response
- Note: Range requests are NOT supported for remux streams (seeking limited to forward only)

### 3.4 Bandwidth Probe Handler

**New file:** `internal/adapters/http/probe_handler.go`

- Generate N bytes of random data (max 10MB)
- Set `Cache-Control: no-store`
- Stream token auth

---

## 4. Client-Side Implementation

### 4.1 Update API Types

**File:** `web/pwa/src/app/core/services/api.service.ts`

Update `MediaDetail` interface with new fields and add `probe()` method.

### 4.2 Playback Capability Service

**New file:** `web/pwa/src/app/core/services/playback-capability.service.ts`

- Uses `MediaCapabilities.decodingInfo()` API
- Maps codec names to MediaCapabilities codec strings
- Checks container format (MP4 only for direct play)
- Checks for image-based subtitles (PGS/VOBSUB)

### 4.3 Bandwidth Service

**New file:** `web/pwa/src/app/core/services/bandwidth.service.ts`

- `measure()` - fetches probe endpoint, measures throughput
- `getCached()` - returns recent measurement if available
- `isSufficient(bitrate)` - compares with 1.3x safety margin
- Persists measurements in localStorage

### 4.4 Playback Decision Service

**New file:** `web/pwa/src/app/core/services/playback-decision.service.ts`

- `decide(media)` - orchestrates capability + bandwidth checks
- Returns `{ mode, url, reason, fallbackAvailable }`

### 4.5 Playback Monitor Service

**New file:** `web/pwa/src/app/core/services/playback-monitor.service.ts`

- Tracks buffer ahead, stall count, last stall time
- Emits `fallbackNeeded$` when thresholds exceeded
- Thresholds: 2 stalls OR buffer < 3s for 5+ seconds

### 4.6 Update Player Component

**File:** `web/pwa/src/app/features/player/player.component.ts`

- Run decision logic on load
- Initialize direct player using Blob URL approach (fetch with auth header)
- Subscribe to monitor for fallback events
- Seamless switch to DASH when fallback triggered
- Load VTT subtitles for direct play mode

---

## 5. Implementation Phases

### Phase 1: Foundation (Direct Play for MP4)

**Goal:** Direct play works for MP4 files with H.264/AAC

**Server tasks:**
1. Extend `MediaDetailResponse` with source info fields
2. Update `buildMediaDetailResponse()` to populate new fields
3. Create `DirectHandler` with Range request support
4. Create `ProbeHandler` for bandwidth measurement
5. Register new routes

**Client tasks:**
1. Update `MediaDetail` interface with new fields
2. Add `probe()` method to `ApiService`
3. Create `PlaybackCapabilityService`
4. Create `BandwidthService`
5. Create `PlaybackDecisionService`
6. Update `PlayerComponent` with decision logic and direct player

### Phase 2: Monitoring & Fallback

**Goal:** Automatic fallback from direct play to DASH when issues detected

**Client tasks:**
1. Create `PlaybackMonitorService`
2. Implement fallback logic in `PlayerComponent`
3. Add notification UI
4. Handle subtitle loading for direct play

### Phase 3: Direct Stream (Remux)

**Goal:** MKV/AVI files with compatible codecs play without transcoding

**Server tasks:**
1. Create `RemuxHandler` with ffmpeg streaming
2. Handle client disconnect cleanup

**Client tasks:**
1. Update `PlaybackDecisionService` for direct-stream mode
2. Update `PlayerComponent` to handle remux streams
3. Add seek limitation note for remux streams

### Phase 4: Polish & Optimization

**Goal:** Improved user experience and performance

**Tasks:**
1. LAN detection (prefer direct play on local network)
2. Pre-warm DASH manifest while trying direct play
3. Playback mode indicator in UI
4. Debug/stats panel (buffer health, bandwidth, etc.)

---

## 6. File Summary

### New Go Files

| File | Purpose |
|------|---------|
| `internal/adapters/http/direct_handler.go` | Serve original files with Range support |
| `internal/adapters/http/remux_handler.go` | On-the-fly MKV/AVI remux |
| `internal/adapters/http/probe_handler.go` | Bandwidth measurement endpoint |

### Modified Go Files

| File | Changes |
|------|---------|
| `internal/adapters/http/media_handler.go` | Add source info + direct play URLs |
| `internal/app/register_http_routes.go` | Register new endpoints |

### New TypeScript Files

| File | Purpose |
|------|---------|
| `playback-capability.service.ts` | Browser codec detection |
| `bandwidth.service.ts` | Network throughput measurement |
| `playback-decision.service.ts` | Choose playback mode |
| `playback-monitor.service.ts` | Watch playback health |

### Modified TypeScript Files

| File | Changes |
|------|---------|
| `api.service.ts` | New types and probe method |
| `player.component.ts` | Smart playback logic |

---

## 7. Technical Notes

### 7.1 Auth for Direct Play

Uses Blob URL approach:
1. Fetch media with Authorization header
2. Create blob URL from response
3. Set as video src

This is more secure than query param tokens (tokens not visible in logs/network).

### 7.2 Remux Streaming (Option B)

On-the-fly remuxing with no caching:
- Zero additional disk space
- Instant start (no waiting for full remux)
- Seeking is limited (forward only, or restart from beginning)
- CPU usage on every playback

### 7.3 Subtitle Handling

During direct play, use already-transcoded VTT files via `<track>` elements:
```html
<track kind="subtitles" src="/stream/dash/content/{id}/subtitle-{idx}.vtt" />
```

### 7.4 Fallback UX

When switching from direct → DASH mid-playback:
1. Save current position
2. Show brief notification ("Switching to adaptive streaming...")
3. Stop current playback, revoke blob URL
4. Initialize DASH player at saved position
5. Auto-dismiss notification after 3 seconds
