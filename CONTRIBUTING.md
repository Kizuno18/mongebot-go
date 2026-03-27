# Contributing to MongeBot Go

## Development Setup

### Prerequisites

- Go 1.26+
- Node.js 22+
- Rust (for Tauri builds)
- Docker (optional)
- FFmpeg (optional, for restreaming)

### Quick Start

```bash
# Clone the repo
git clone https://github.com/Kizuno18/mongebot-go.git
cd mongebot-go

# Install all dependencies
make install

# Copy sample data files
cp data/proxies.txt.example data/proxies.txt
cp data/tokens.txt.example data/tokens.txt

# Run backend in development mode
make dev

# In another terminal, run frontend
make frontend-dev

# Or run both together
make dev-all

# Run with Tauri desktop
make frontend-tauri-dev
```

### Running Tests

```bash
# All tests
make test

# With coverage report
make test-cover

# Short tests only
make test-short

# Benchmarks
make bench

# Linting
make lint
```

### Building

```bash
# Local build
make build-all

# Cross-compile for all platforms
make release-all

# Specific platform
make release-linux
make release-darwin
make release-windows

# Docker
make docker-up
```

## Project Structure

```
cmd/           Entry points (mongebot, cli)
internal/      Private application code
  account/     Multi-account profile management
  api/         WebSocket JSON-RPC API server
  config/      Configuration, env, migration, archive
  engine/      Worker pool, scheduler, event bus, shutdown
  logger/      Structured logging with ring buffer
  platform/    Plugin interface + providers (twitch, kick, youtube)
  proxy/       Proxy pool, checker, scraper, geolocation
  storage/     SQLite persistence
  stream/      FFmpeg restreaming
  token/       Token pool, validator, importer
  vault/       Encrypted storage
pkg/           Public reusable packages
  fingerprint/ Device ID, TLS fingerprint rotation
  netutil/     Retry, circuit breaker, errors, IP utils
  useragent/   User-agent pool
frontend/      React + Tauri UI
```

## Code Standards

- Go code follows standard `gofmt` formatting
- All code and comments in English
- camelCase for variables/functions, PascalCase for types
- kebab-case for file names
- Tests in `*_test.go` files alongside source
- No hardcoded secrets — use config, env vars, or vault

## Adding a New Platform

1. Create `internal/platform/<name>/provider.go`
2. Implement the `platform.Platform` interface
3. Register in `cmd/mongebot/main.go`: `registry.Register(yourpkg.NewProvider(logger))`
4. Add tests

## Adding a New IPC Method

1. Choose the right handler file in `internal/api/`
2. Add handler function matching `handlerFunc` signature
3. Register in the appropriate `get*Handler()` map
4. Add TypeScript types in `frontend/src/types/`
5. Call from frontend via `ipc.call("method.name", params)`
