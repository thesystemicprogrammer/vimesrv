# Architecture

VimeSrv follows a clean/hexagonal architecture pattern, separating business logic from external dependencies.

## Directory Structure

```
vimesrv/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── configs/
│   └── default.yaml          # Default configuration
├── internal/
│   ├── app/                  # Application bootstrap
│   │   ├── app.go            # Main application struct
│   │   ├── init_*.go         # Initialization modules
│   │   └── register_*.go     # Route/job registration
│   ├── domain/               # Domain entities
│   │   ├── media_file.go
│   │   ├── job.go
│   │   ├── transcode.go
│   │   └── *_metadata.go
│   ├── usecase/              # Business logic
│   │   ├── ports/            # Interface definitions
│   │   ├── job/              # Job management
│   │   ├── library/          # Library scanning
│   │   ├── media/            # Media queries
│   │   ├── metadata/         # TMDB enrichment
│   │   └── transcode/        # Transcoding logic
│   ├── adapters/             # External implementations
│   │   ├── http/             # HTTP handlers
│   │   ├── job/              # Job manager & scheduling
│   │   ├── library/          # Library scan handler
│   │   ├── media/            # FFmpeg/FFprobe adapters
│   │   ├── metadata/         # TMDB client
│   │   ├── repository/       # SQLite repositories
│   │   └── transcode/        # Transcode job handler
│   ├── infrastructure/
│   │   ├── database/         # SQLite setup & migrations
│   │   └── server/           # HTTP server & middleware
│   └── shared/
│       ├── config/           # Configuration loading
│       ├── logger/           # Structured logging
│       ├── constants.go
│       └── errors.go
├── web/
│   ├── pwa/                  # Angular PWA client
│   └── embed_pwa.go          # Go embed directive
├── api/
│   └── openapi.yaml          # API specification
└── doc/                      # Documentation
```

## Layers

### Domain Layer (`internal/domain/`)

Pure Go structs representing business entities. No external dependencies.

- `MediaFile`: Video file with metadata
- `Job`: Background task with status tracking
- `Schedule`: Cron-based job scheduling
- `Transcode`: Transcoding job parameters and output
- `*Metadata`: Movie/series/episode metadata from TMDB

### Use Case Layer (`internal/usecase/`)

Business logic orchestration. Depends only on domain and port interfaces.

**Ports** (`usecase/ports/`): Interface definitions for external dependencies:
- `MediaRepository`: Media file persistence
- `JobRepository`: Job queue persistence
- `FileSystemService`: File operations
- `FFProbe`: Media analysis
- `Transcoder`: Video transcoding
- `MetadataProvider`: TMDB integration

**Use Cases**:
- `library/`: Scan staging directory, fingerprint files, detect duplicates
- `job/`: Enqueue jobs, process queue, recover stuck jobs
- `media/`: List and get media files
- `metadata/`: Search, match, and link TMDB metadata
- `transcode/`: Create and process transcode jobs

### Adapter Layer (`internal/adapters/`)

Implementations of port interfaces:

- `http/`: Gin HTTP handlers
- `repository/`: SQLite implementations using modernc.org/sqlite
- `media/`: FFmpeg transcoder, FFprobe analyzer, Blake2b hasher
- `metadata/`: TMDB REST client, image downloader
- `job/`: Job manager, cron parser, exponential backoff

### Infrastructure Layer (`internal/infrastructure/`)

Framework and driver code:

- `database/`: SQLite connection, migrations
- `server/`: Gin router, middleware (auth, CORS, logging)

## Data Flow

```
HTTP Request
    │
    ▼
┌─────────────────┐
│  HTTP Handler   │  (adapters/http)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    Use Case     │  (usecase/)
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌───────┐ ┌───────┐
│ Repo  │ │Adapter│  (adapters/repository, adapters/media)
└───┬───┘ └───┬───┘
    │         │
    ▼         ▼
┌───────┐ ┌───────┐
│SQLite │ │FFmpeg │  (external)
└───────┘ └───────┘
```

## Job System

Background processing uses a polling-based job queue:

1. **Scheduler**: Cron-based triggers create jobs
2. **Enqueue**: Jobs inserted into SQLite queue
3. **Workers**: Poll for pending jobs, process with handlers
4. **Handlers**: Type-specific processors (scan, transcode, metadata)
5. **Recovery**: Stuck job detection and retry with exponential backoff

```
┌──────────┐     ┌─────────┐     ┌─────────┐
│Scheduler │────▶│ Enqueue │────▶│  Queue  │
└──────────┘     └─────────┘     └────┬────┘
                                      │
                                      ▼
┌─────────┐     ┌─────────┐     ┌─────────┐
│ Handler │◀────│ Worker  │◀────│  Poll   │
└─────────┘     └─────────┘     └─────────┘
```

## Authentication

Two-tier JWT authentication:

1. **API Token**: Long-lived (24h default) for API requests
2. **Stream Token**: Short-lived (60min default) for streaming URLs

```
Login ──▶ API Token ──▶ Protected Endpoints
                   │
                   ▼
            Stream Token ──▶ /stream/* Endpoints
```

## Streaming Architecture

DASH streaming with adaptive bitrate:

1. Media file transcoded to multiple quality levels
2. Each quality has separate video segments
3. Audio tracks shared across qualities
4. Subtitles extracted to WebVTT
5. MPD manifest generated dynamically

```
Source Video
    │
    ├──▶ 1080p/video/ (segments)
    ├──▶ 720p/video/  (segments)
    ├──▶ 480p/video/  (segments)
    ├──▶ audio-0/     (segments)
    ├──▶ audio-1/     (segments)
    └──▶ subtitle-0.vtt
```
