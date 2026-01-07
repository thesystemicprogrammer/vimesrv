# Configuration

VimeSrv uses YAML configuration with environment variable overrides. The default configuration is in `configs/default.yaml`.

## Configuration File

Create a custom config file and specify it at startup:

```bash
./vimesrv --config /path/to/config.yaml
```

## Configuration Sections

### Server

```yaml
server:
  host: "127.0.0.1"        # Bind address
  port: 8080               # HTTP port
  shutdown_timeout_seconds: 30  # Graceful shutdown timeout
```

### Authentication

```yaml
auth:
  enabled: true
  username: "admin"
  password_hash: "$2b$10$..."  # bcrypt hash
  jwt_secret: "change-me"      # Override with AUTH_JWT_SECRET env var
  token_expiry_hours: 24       # API token lifetime
  stream_token_mins: 60        # Stream token lifetime
```

Generate a password hash:

```bash
htpasswd -nbB "" "your-password" | cut -d: -f2
```

### Job Processing

```yaml
job:
  worker_count: 2                    # Concurrent job workers
  polling_interval_in_seconds: 2     # Job queue poll interval
  scheduler_batch: 5                 # Jobs to process per tick
  scheduler_interval_in_seconds: 2   # Scheduler tick interval
  max_attempts: 3                    # Retry attempts for failed jobs
  backoff_base_seconds: 2            # Exponential backoff base
  backoff_max_seconds: 300           # Maximum backoff delay
  stuck_job_threshold_minutes: 480   # Mark jobs as stuck after 8 hours
  stuck_job_check_interval_minutes: 5
```

### Media Paths

```yaml
media:
  library_path: "./library"      # Organized media storage
  media_path: "./media"          # Raw media files
  staging_path: "./staging"      # Incoming files for scanning
  trash_path: "/trash"           # Deleted files (relative to library_path)
  transcode_output_pattern: "{media_path}/{media_id}/transcoded"
  ffprobe_timeout_seconds: 30
  transcode_timeout_seconds: 7200  # 2 hours
  supported_formats:
    - ".mp4"
    - ".mkv"
    - ".avi"
    - ".mov"
    - ".webm"
    - ".flv"
    - ".wmv"
    - ".m4v"
  library_scan:
    enabled: true
    cron_spec: "0 * * * * *"     # Every minute (6-field cron)
    priority: 0
```

### Transcoding

```yaml
transcoding:
  segment_duration: 4  # DASH segment length in seconds
  video_encoder: "libx264"  # Video encoder (see below)
  scale_filter: "auto"      # Scale filter mode (see below)
  encoder_preset: "medium"  # Encoder preset (see below)
  ffmpeg_input_args: []     # FFmpeg arguments to add before -i (see below)
  quality_profiles:
    - name: "1080p"
      enabled: false
      resolution: "1920x1080"
      crf: 21                  # Constant Rate Factor (lower = better)
      max_bitrate: "5500k"
      audio_bitrate: "192k"
    - name: "720p"
      enabled: false
      resolution: "1280x720"
      crf: 23
      max_bitrate: "2800k"
      audio_bitrate: "128k"
    - name: "480p"
      enabled: true
      resolution: "854x480"
      crf: 24
      max_bitrate: "1500k"
      audio_bitrate: "128k"
    - name: "360p"
      enabled: true
      resolution: "640x360"
      crf: 25
      max_bitrate: "900k"
      audio_bitrate: "96k"
```

**Video Encoder:**

The `video_encoder` option specifies which encoder to use for transcoding:

| Encoder | Type | Platform | Description |
|---------|------|----------|-------------|
| `libx264` | Software | All | Default CPU encoder, best compatibility |
| `libx265` | Software | All | HEVC encoder, better compression but slower |
| `h264_vaapi` | Hardware | Linux (Intel/AMD) | VAAPI hardware encoder |
| `h264_qsv` | Hardware | Linux/Windows (Intel) | Intel Quick Sync Video |
| `h264_nvenc` | Hardware | Linux/Windows (NVIDIA) | NVIDIA NVENC |
| `h264_amf` | Hardware | Linux/Windows (AMD) | AMD AMF encoder |
| `h264_videotoolbox` | Hardware | macOS | Apple VideoToolbox |

**Scale Filter:**

The `scale_filter` option controls which scaling filter is used for resizing video:

- `auto` (default): Automatically selects based on encoder
  - `h264_vaapi` → `scale_vaapi`
  - `h264_qsv` → `scale_qsv`
  - Others → software `scale`
- `software`: Always use CPU-based scaling
- `vaapi`: Use VAAPI hardware scaling
- `qsv`: Use Intel QSV hardware scaling

**Encoder Preset:**

The `encoder_preset` option controls the speed/quality tradeoff:

- For `libx264`/`libx265`: `ultrafast`, `superfast`, `veryfast`, `faster`, `fast`, `medium`, `slow`, `slower`, `veryslow`
- For `h264_nvenc`: `p1`-`p7` (p1=fastest, p7=slowest)
- For `h264_vaapi`/`h264_qsv`: Not used (ignored)

**FFmpeg Input Arguments:**

The `ffmpeg_input_args` option allows you to specify arbitrary FFmpeg arguments that are inserted before the `-i` input flag. This is required for hardware-accelerated decoding and encoding.

**Hardware Acceleration Examples:**

For hardware encoding to work correctly, you typically need both:
1. `ffmpeg_input_args` for hardware-accelerated decoding
2. `video_encoder` set to a hardware encoder

```yaml
# Full VAAPI pipeline (Intel/AMD on Linux)
transcoding:
  video_encoder: "h264_vaapi"
  scale_filter: "auto"  # Will use scale_vaapi
  ffmpeg_input_args:
    - "-hwaccel"
    - "vaapi"
    - "-hwaccel_device"
    - "/dev/dri/renderD128"
    - "-hwaccel_output_format"
    - "vaapi"

# Full Intel QSV pipeline
transcoding:
  video_encoder: "h264_qsv"
  scale_filter: "auto"  # Will use scale_qsv
  ffmpeg_input_args:
    - "-hwaccel"
    - "qsv"
    - "-qsv_device"
    - "/dev/dri/renderD128"

# NVIDIA NVENC
transcoding:
  video_encoder: "h264_nvenc"
  encoder_preset: "p4"  # Balanced preset
  ffmpeg_input_args:
    - "-hwaccel"
    - "cuda"
    - "-hwaccel_output_format"
    - "cuda"

# macOS VideoToolbox
transcoding:
  video_encoder: "h264_videotoolbox"
  ffmpeg_input_args:
    - "-hwaccel"
    - "videotoolbox"

# Software decoding with hardware encoding (fallback if HW decode fails)
transcoding:
  video_encoder: "h264_vaapi"
  scale_filter: "software"  # Force software scaling
  ffmpeg_input_args:
    - "-vaapi_device"
    - "/dev/dri/renderD128"
```

Leave `ffmpeg_input_args` empty (`[]`) and `video_encoder` as `libx264` for pure software encoding (default).

**Note:** Each machine (server or worker) can have its own hardware encoding configuration, allowing different hardware setups per machine in a distributed transcoding environment. If transcoding fails with hardware encoding, the error will be visible in the job logs.

**CRF Guidelines:**

The CRF (Constant Rate Factor) quality setting is automatically mapped to the appropriate parameter for each encoder:
- `libx264`/`libx265`: Uses `-crf`
- `h264_vaapi`: Uses `-qp`
- `h264_qsv`: Uses `-global_quality`
- `h264_nvenc`: Uses `-cq`

Recommended CRF values:
- 18-20: Near-lossless quality
- 21-23: Excellent quality
- 24-26: Good quality (smaller files)

### Database

```yaml
database:
  path: "./data/vimesrv.db"  # SQLite database file
```

### Logging

```yaml
logging:
  level: "info"       # debug, info, warn, error
  format: "console"   # console or json
  file: ""            # Optional log file path
```

### TMDB Integration

```yaml
tmdb:
  enabled: false                  # Enable metadata enrichment
  api_key: ""                     # Set via TMDB_API_KEY env var
  language: "en-US"
  auto_search: true               # Auto-search on new media
  auto_link_threshold: 70         # Confidence % for auto-linking
  max_candidates: 5
  image_cache_path: "./cache/tmdb"
  download_images: true
  poster_size: "w500"
  backdrop_size: "w1280"
  requests_per_10s: 35            # Rate limiting
```

### Database Rebuild

```yaml
rebuild:
  allow_rebuild: false          # Safety flag - must be true to enable --rebuild-from-dump
  tmdb_requests_per_10s: 15     # Conservative rate limit during rebuild
```

See [Database Rebuild](rebuild.md) for usage guide.

## Environment Variables

Override config values with environment variables:

| Variable | Config Path | Description |
|----------|-------------|-------------|
| `AUTH_JWT_SECRET` | `auth.jwt_secret` | JWT signing secret |
| `TMDB_API_KEY` | `tmdb.api_key` | TMDB API key |

## Example Production Config

```yaml
server:
  host: "0.0.0.0"
  port: 8080

auth:
  enabled: true
  username: "admin"
  password_hash: "$2b$10$your-secure-hash"
  jwt_secret: ""  # Use AUTH_JWT_SECRET env var

media:
  library_path: "/data/library"
  media_path: "/data/media"
  staging_path: "/data/staging"

transcoding:
  quality_profiles:
    - name: "1080p"
      enabled: true
      resolution: "1920x1080"
      crf: 21
      max_bitrate: "5500k"
      audio_bitrate: "192k"
    - name: "720p"
      enabled: true
      resolution: "1280x720"
      crf: 23
      max_bitrate: "2800k"
      audio_bitrate: "128k"

database:
  path: "/data/vimesrv.db"

logging:
  level: "info"
  format: "json"
  file: "/var/log/vimesrv.log"

tmdb:
  enabled: true
  language: "en-US"
```
