# VimeSrv

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev/)

Home network video media server with DASH transcoding and TMDB metadata enrichment.

## Features

- **Library Scanning** - Auto-detect and fingerprint video files
- **Transcoding** - FFmpeg-based multi-resolution DASH streaming
- **Metadata Enrichment** - TMDB integration for movies and TV series
- **VimeSrv Client** - Angular PWA for browsing and playback
- **JWT Authentication** - Secure API and stream token management

## Quick Start

```bash
# Build
make build

# Run
./bin/vimesrv

# Build with embedded PWA
make build-pwa
```

Access at `http://localhost:8080`

Default credentials: `admin` / `admin123`

## Documentation

| Guide                                 | Description                              |
| ------------------------------------- | ---------------------------------------- |
| [Building](doc/building.md)           | Prerequisites and build commands         |
| [Configuration](doc/configuration.md) | YAML config and environment variables    |
| [Architecture](doc/architecture.md)   | Clean architecture and project structure |
| [Features](doc/features.md)           | Library scan, transcoding, metadata, PWA |
| [Development](doc/development.md)     | Testing, linting, and contributing       |
| [Deployment](doc/deployment.md)       | Production setup and security            |

## API

See [api/openapi.yaml](api/openapi.yaml) for the complete API specification.

### Key Endpoints

| Endpoint                            | Description                |
| ----------------------------------- | -------------------------- |
| `POST /api/v1/auth/login`           | Authenticate and get token |
| `GET /api/v1/media`                 | List media files           |
| `GET /api/v1/media/:id`             | Get media details          |
| `POST /api/v1/scanlib`              | Trigger library scan       |
| `GET /stream/dash/:id/manifest.mpd` | DASH streaming manifest    |

## Requirements

- Go 1.23+
- FFmpeg 4.4+ (with FFprobe)
- Node.js 18+ (for PWA development)

## License

[MIT](LICENSE)
