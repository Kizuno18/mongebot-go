# MongeBot Go v2.0

A modular, multi-account, multi-platform viewer bot rewritten from Python to Go with a Tauri 2.0 desktop interface.

## Features

- **Multi-Platform**: Twitch (full), Kick (basic) — extensible plugin architecture
- **Multi-Account**: Profile system with per-channel config overrides
- **Desktop UI**: Tauri 2.0 + React + TailwindCSS (~2MB bundle)
- **Headless CLI**: Terminal TUI with ANSI worker bars
- **Real-time Dashboard**: Recharts graphs, worker grid visualization, live metrics
- **Proxy Management**: 4 rotation strategies, concurrent health checker, auto-scraper
- **Token Vault**: AES-256-GCM encrypted storage with PBKDF2 key derivation
- **Stream Monitor**: Auto-detect online/offline, toast notifications
- **FFmpeg Restream**: 5 quality presets (potato → ultra), proxy support
- **Session History**: SQLite-backed metrics persistence
- **Ad Watching**: HLS playlist parsing for stitched ad detection

## Architecture

```
Tauri Desktop App
├── React Frontend (TypeScript)
│   ├── Dashboard (charts, metrics, controls)
│   ├── Profiles (multi-account management)
│   ├── Proxy Manager (import, health check, scrape)
│   ├── Token Vault (encrypted storage)
│   ├── Stream Monitor (status, FFmpeg restream)
│   ├── Session History (past sessions)
│   ├── Log Viewer (real-time, filterable)
│   └── Settings (feature toggles, engine config)
│
└── Go Sidecar Backend
    ├── Platform Layer (Twitch, Kick, YouTube)
    ├── Engine (worker pool, auto-restart, scaling)
    ├── Proxy Manager (pool, rotation, checker, scraper)
    ├── Token Manager (pool, validation, quarantine)
    ├── API Server (WebSocket JSON-RPC 2.0)
    ├── Storage (SQLite, session metrics)
    └── Stream (FFmpeg process management)
```

## Quick Start

### Prerequisites

- Go 1.26+
- Node.js 22+
- Rust (for Tauri)
- FFmpeg (optional, for restreaming)

### Setup

```bash
# Clone
git clone https://github.com/Kizuno18/mongebot-go.git
cd mongebot-go

# Copy sample data files
cp data/proxies.txt.example data/proxies.txt
cp data/tokens.txt.example data/tokens.txt
cp data/user-agents.txt.example data/user-agents.txt

# Build backend
go mod tidy
make build

# Install frontend
cd frontend && npm install
```

### Run Desktop App (Tauri)

```bash
cd frontend
npm run tauri dev
```

### Run Headless (CLI)

```bash
# With TUI
./bin/mongebot-cli --channel streamer_name --workers 50

# As daemon
./bin/mongebot --mode headless --channel streamer_name --workers 50
```

### Run with Docker

```bash
docker compose up -d --build
```

## Configuration

Config is stored in `data/config.json` (auto-created with defaults on first run).

Key settings:
- `engine.maxWorkers`: Max concurrent viewers (default: 50)
- `engine.features.*`: Toggle individual behaviors (ads, chat, pubsub, segments, gql, spade)
- `api.port`: WebSocket API port (default: 9800)

## IPC API (JSON-RPC 2.0)

The backend exposes a WebSocket JSON-RPC API at `ws://127.0.0.1:9800/ws`.

### Methods

| Category | Method | Description |
|----------|--------|-------------|
| Engine | `engine.start` | Start viewer engine |
| | `engine.stop` | Stop all viewers |
| | `engine.status` | Get metrics |
| | `engine.setWorkers` | Dynamic scaling |
| Profile | `profile.list/create/delete/activate/duplicate/export` | CRUD |
| Proxy | `proxy.list/import/check/scrape` | Management |
| Token | `token.list/import/stats/validate` | Management |
| Stream | `stream.restream.start/stop/state` | FFmpeg |
| | `stream.presets` | Quality presets |
| Config | `config.get/set` | Configuration |
| Logs | `logs.history` | Log entries |
| Sessions | `sessions.recent` | History |

### Events (server → client)

- `event.metrics` — Real-time engine metrics (every 5s)
- `event.log` — Log entries
- `event.stream` — Stream online/offline transitions
- `event.error` — Error notifications

## Project Structure

```
mongebot-go/
├── cmd/mongebot/       # Main entry point (sidecar + headless)
├── cmd/cli/            # CLI with TUI dashboard
├── internal/
│   ├── account/        # Multi-account profile management
│   ├── api/            # WebSocket JSON-RPC server
│   ├── config/         # Configuration management
│   ├── engine/         # Worker pool orchestrator
│   ├── logger/         # Structured logging + ring buffer
│   ├── platform/       # Plugin interface + registry
│   │   ├── twitch/     # Full Twitch implementation
│   │   ├── kick/       # Kick.com implementation
│   │   └── youtube/    # YouTube (future)
│   ├── proxy/          # Proxy pool, checker, scraper
│   ├── storage/        # SQLite persistence
│   ├── stream/         # FFmpeg restreaming
│   ├── token/          # Token pool + validation
│   └── vault/          # AES-256-GCM encrypted storage
├── pkg/                # Reusable packages
├── frontend/           # Tauri + React UI
├── docker/             # Dockerfiles
└── data/               # Runtime data (gitignored)
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26 |
| Frontend | React + TypeScript + TailwindCSS |
| Desktop | Tauri 2.0 (Rust) |
| WebSocket | coder/websocket |
| Database | SQLite (modernc.org/sqlite, pure Go) |
| Charts | Recharts |
| Icons | Lucide React |

## License

Private project.
