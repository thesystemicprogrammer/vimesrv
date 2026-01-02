# Development

## Setup

### Clone and Install

```bash
git clone https://github.com/thesystemicprogrammer/vimesrv.git
cd vimesrv
make deps
```

### Install Development Tools

```bash
# Hot reload
go install github.com/cosmtrek/air@latest

# Linting
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Test formatting
go install gotest.tools/gotestsum@latest
```

## Running

### Development Mode

```bash
make dev
```

Uses [air](https://github.com/cosmtrek/air) for hot reload on Go file changes.

### Standard Run

```bash
make run
```

## Testing

### Run All Tests

```bash
make test
```

Runs all tests with race detection and generates `coverage.out`.

### Unit Tests Only

```bash
make test-unit
```

Skips integration tests (faster).

### Integration Tests Only

```bash
make test-integration
```

Runs tests tagged with `Integration` in the name.

### Coverage Report

```bash
make test-coverage
```

Generates `coverage.html` for browser viewing.

### Formatted Output

```bash
make test-form
```

Uses gotestsum for cleaner test output.

## Code Quality

### Format Code

```bash
make fmt
```

Runs `gofmt` on all Go files.

### Static Analysis

```bash
make vet
```

Runs `go vet` for common issues.

### Linting

```bash
make lint
```

Runs golangci-lint with project configuration.

### All Checks

```bash
make check
```

Runs fmt and vet together.

## PWA Development

### Install Dependencies

```bash
make pwa-install
```

### Development Server

```bash
make pwa-dev
```

Starts Angular dev server at `http://localhost:4200`.

Configure proxy to backend in `web/pwa/proxy.conf.json` if needed.

### Build

```bash
make pwa-build
```

## Project Structure

### Adding a New Use Case

1. Define port interfaces in `internal/usecase/ports/`
2. Create use case in `internal/usecase/{domain}/`
3. Implement adapters in `internal/adapters/`
4. Wire up in `internal/app/init_usecases.go`

### Adding an HTTP Endpoint

1. Create handler in `internal/adapters/http/`
2. Register routes in `internal/app/register_http_routes.go`
3. Document in `api/openapi.yaml`

### Adding a Job Type

1. Define job type constant in `internal/shared/constants.go`
2. Create handler in `internal/adapters/{domain}/`
3. Register in `internal/app/register_jobs.go`

## Database

### Migrations

Migrations are embedded and run automatically at startup.

Location: `internal/infrastructure/database/`

### SQLite

Uses pure-Go SQLite driver (modernc.org/sqlite) - no CGO required.

## Test Fixtures

Test video files are in `test/fixtures/`:

```
test/fixtures/
└── sample_video.mp4
```

## REST Client

HTTP request files for manual testing:

```
test/rest/
├── 01-library.http
└── http-client.env.json
```

Use with VS Code REST Client or IntelliJ HTTP Client.

## Common Tasks

### Reset Database

```bash
rm ./data/vimesrv.db
make run
```

### Clear Transcodes

```bash
rm -rf ./media/*/transcoded/
```

### Debug Logging

Set in config:

```yaml
logging:
  level: "debug"
  format: "console"
```

Or temporarily:

```bash
LOG_LEVEL=debug ./bin/vimesrv
```
