# Building VimeSrv

## Prerequisites

### Required

| Dependency | Version | Purpose |
|------------|---------|---------|
| Go | 1.23+ | Backend compilation |
| FFmpeg | 4.4+ | Video transcoding |
| FFprobe | 4.4+ | Media analysis (included with FFmpeg) |

### Optional (for PWA development)

| Dependency | Version | Purpose |
|------------|---------|---------|
| Node.js | 18+ | PWA build tooling |
| npm | 9+ | Package management |

## Build Commands

### Development Build

```bash
make build
```

Builds the binary to `./bin/vimesrv` with debug symbols.

### Production Build

```bash
make build-prod
```

Builds an optimized binary with `-trimpath` and stripped debug symbols.

### Production Build with PWA

```bash
make build-pwa
```

Builds the Angular PWA first, then embeds it into the Go binary. This creates a single self-contained executable.

## Running

### Direct Run

```bash
make run
```

Builds and runs the server in one command.

### Development Mode (Hot Reload)

```bash
make dev
```

Requires [air](https://github.com/cosmtrek/air) for hot reload:

```bash
go install github.com/cosmtrek/air@latest
```

## PWA Development

### Install Dependencies

```bash
make pwa-install
```

### Development Server

```bash
make pwa-dev
```

Starts the Angular dev server at `http://localhost:4200` with hot module replacement.

### Build PWA Only

```bash
make pwa-build
```

Builds the PWA to `web/pwa/dist/vimesrv-client/browser/`.

## Dependency Management

```bash
make deps          # Download dependencies
make deps-tidy     # Tidy go.mod and go.sum
make deps-update   # Update all dependencies
```

## Build Output

| Command | Output Path |
|---------|-------------|
| `make build` | `./bin/vimesrv` |
| `make build-prod` | `./bin/vimesrv` |
| `make build-pwa` | `./bin/vimesrv` (with embedded PWA) |
| `make pwa-build` | `web/pwa/dist/vimesrv-client/browser/` |

## Version Information

The build includes version metadata via ldflags:

- `version`: Semantic version (from Makefile)
- `commit`: Git commit hash
- `date`: Build timestamp

View with:

```bash
./bin/vimesrv --version
```
