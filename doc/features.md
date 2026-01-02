# Features

## Library Scanning

VimeSrv automatically scans a staging directory for new video files.

### How It Works

1. **Detection**: Recursively scans `staging_path` for supported video formats
2. **Validation**: Uses FFprobe to validate media and extract metadata
3. **Fingerprinting**: Generates Blake2b hash for deduplication
4. **Organization**: Moves valid files to `library_path`
5. **Database**: Creates media file record with extracted metadata

### Supported Formats

- MP4, MKV, AVI, MOV
- WebM, FLV, WMV, M4V

### Scheduling

Library scans run:
- Once at startup (if enabled)
- Periodically via cron schedule

Configure in `config.yaml`:

```yaml
media:
  library_scan:
    enabled: true
    cron_spec: "0 * * * * *"  # Every minute
```

### Manual Trigger

```bash
curl -X POST http://localhost:8080/api/v1/scanlib \
  -H "Authorization: Bearer <token>"
```

## Video Transcoding

VimeSrv transcodes videos for adaptive streaming via DASH.

### Quality Profiles

Configure multiple quality levels:

| Profile | Resolution | CRF | Max Bitrate |
|---------|------------|-----|-------------|
| 2160p | 3840x2160 | 18 | 16 Mbps |
| 1440p | 2560x1440 | 20 | 9 Mbps |
| 1080p | 1920x1080 | 21 | 5.5 Mbps |
| 720p | 1280x720 | 23 | 2.8 Mbps |
| 480p | 854x480 | 24 | 1.5 Mbps |
| 360p | 640x360 | 25 | 900 Kbps |

Enable/disable profiles in configuration.

### Transcoding Process

1. **Job Creation**: After library scan, transcode jobs are queued
2. **Video Encoding**: H.264 with CRF-based quality
3. **Segmentation**: Split into 4-second DASH segments
4. **Audio Extraction**: Separate audio tracks (preserves all languages)
5. **Subtitle Extraction**: Convert to WebVTT format

### Output Structure

```
media/{media_id}/transcoded/
├── 1080p/
│   └── video/
│       ├── init.mp4
│       ├── chunk-001.m4s
│       └── ...
├── 720p/
│   └── video/
│       └── ...
├── audio-0/
│   ├── init.mp4
│   └── ...
├── audio-1/
│   └── ...
└── subtitle-0.vtt
```

## Metadata Enrichment

Automatic metadata fetching from TMDB (The Movie Database).

### Setup

1. Get an API key from [themoviedb.org](https://www.themoviedb.org/settings/api)
2. Set environment variable: `export TMDB_API_KEY=your-key`
3. Enable in config:

```yaml
tmdb:
  enabled: true
  language: "en-US"
```

### Matching Process

1. **Filename Parsing**: Extract title, year, season/episode from filename
2. **TMDB Search**: Query API with extracted information
3. **Confidence Scoring**: Rank results by match quality
4. **Auto-Linking**: High-confidence matches (>70%) link automatically
5. **Manual Review**: Low-confidence matches require user approval

### Supported Media Types

- **Movies**: Title, year, overview, poster, backdrop
- **TV Series**: Series info, season, episode details
- **Episodes**: Individual episode metadata

### Manual Metadata Management

**Get candidates:**
```bash
curl http://localhost:8080/api/v1/media/{id}/candidates \
  -H "Authorization: Bearer <token>"
```

**Link to candidate:**
```bash
curl -X POST http://localhost:8080/api/v1/media/{id}/link \
  -H "Authorization: Bearer <token>" \
  -d '{"candidate_id": 12345}'
```

**Manual search:**
```bash
curl -X POST http://localhost:8080/api/v1/media/{id}/search \
  -H "Authorization: Bearer <token>" \
  -d '{"query": "The Matrix", "year": 1999}'
```

### Image Caching

Posters and backdrops are downloaded and cached locally:

```yaml
tmdb:
  image_cache_path: "./cache/tmdb"
  download_images: true
  poster_size: "w500"
  backdrop_size: "w1280"
```

## VimeSrv Client (PWA)

Angular-based Progressive Web App for browsing and streaming.

### Features

- **Authentication**: Login with username/password
- **Library Browser**: Grid view with posters
- **Media Details**: Movie/series info, cast, overview
- **Video Player**: Adaptive streaming with quality selection
- **Audio Selection**: Switch between audio tracks
- **Subtitles**: Toggle subtitle tracks
- **Responsive**: Works on desktop and mobile
- **Offline**: PWA caching for offline access

### Access

The PWA is embedded in the server binary and served at the root URL:

```
http://localhost:8080/
```

### Development

Run the PWA dev server separately:

```bash
make pwa-dev
```

Access at `http://localhost:4200` with hot module replacement.

## DASH Streaming

Dynamic Adaptive Streaming over HTTP.

### Manifest

Request the MPD manifest:

```
GET /stream/dash/{id}/manifest.mpd?token=<stream-token>
```

Optional quality filter:

```
GET /stream/dash/{id}/manifest.mpd?quality=720p&token=<stream-token>
```

### Content URLs

Video segments:
```
/stream/dash/content/{id}/1080p/video/chunk-001.m4s?token=<stream-token>
```

Audio segments:
```
/stream/dash/content/{id}/audio-0/chunk-001.m4s?token=<stream-token>
```

Subtitles:
```
/stream/dash/content/{id}/subtitle-0.vtt?token=<stream-token>
```

### Stream Tokens

Stream tokens are short-lived (default 60 minutes) for URL security:

```bash
# Get stream token
curl -X POST http://localhost:8080/api/v1/auth/stream-token \
  -H "Authorization: Bearer <api-token>"

# Use in streaming URLs
curl "http://localhost:8080/stream/dash/{id}/manifest.mpd?token=<stream-token>"
```
