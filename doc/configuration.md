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

**CRF Guidelines:**
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
