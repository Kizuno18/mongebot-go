# MongeBot Go — Architecture Plan

## Overview

MongeBot is a **modular, multi-account, multi-platform viewer bot** rewritten from Python to Go with a Tauri 2.0 desktop interface. Designed for extensibility, performance, and user-friendliness.

## Tech Stack

| Layer | Technology | Version | Why |
|-------|-----------|---------|-----|
| **Backend** | Go | 1.26 | Green Tea GC, goroutines, Swiss Tables, native concurrency |
| **Frontend** | Tauri 2.0 + React + TypeScript | 2.10.x | ~2MB bundle, native webview, cross-platform |
| **Styling** | TailwindCSS 4 | latest | CSS-first config, utility classes |
| **WebSocket** | coder/websocket | latest | Context-aware, concurrent-safe, actively maintained |
| **HTTP** | net/http (stdlib) | 1.26 | Native proxy support, mature, zero dependencies |
| **Database** | SQLite (modernc.org/sqlite) | latest | Pure Go, no CGO, local persistence |
| **Encryption** | crypto/aes + crypto/rand | stdlib | Token vault encryption at rest |
| **IPC** | WebSocket + JSON-RPC | custom | Real-time bidirectional Tauri <-> Go sidecar |
| **Build** | Docker + docker-compose | latest | Containerized dev/prod environments |

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                    TAURI DESKTOP APP                          │
│  ┌────────────────────────────────────────────────────────┐  │
│  │              React Frontend (TypeScript)                │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │  │
│  │  │Dashboard │ │ Accounts │ │ Settings │ │  Logs    │ │  │
│  │  │  Panel   │ │  Manager │ │  Panel   │ │  Viewer  │ │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │  │
│  │  │ Proxy    │ │  Token   │ │ Platform │ │ Stream   │ │  │
│  │  │ Manager  │ │  Vault   │ │ Selector │ │ Monitor  │ │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ │  │
│  └────────────────────┬───────────────────────────────────┘  │
│                       │ WebSocket IPC (JSON-RPC)             │
│  ┌────────────────────▼───────────────────────────────────┐  │
│  │              Go Sidecar (Backend Engine)                │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│   Platform   │ │   Platform   │ │   Platform   │
│   Twitch     │ │    Kick      │ │   YouTube    │
│  (Provider)  │ │  (Provider)  │ │  (Provider)  │
└──────────────┘ └──────────────┘ └──────────────┘
```

## Go Backend — Module Architecture

```
mongebot-go/
├── cmd/
│   ├── mongebot/          # Main entry point (CLI + sidecar mode)
│   │   └── main.go
│   └── cli/               # Headless CLI mode (no UI)
│       └── main.go
├── internal/
│   ├── config/            # Configuration management
│   │   ├── config.go      # Config struct, loading, validation
│   │   ├── defaults.go    # Default values
│   │   └── migrate.go     # Config version migration
│   ├── vault/             # Encrypted token storage
│   │   ├── vault.go       # AES-256-GCM encryption/decryption
│   │   └── keyring.go     # OS keyring integration
│   ├── platform/          # Platform abstraction layer
│   │   ├── platform.go    # Interface definitions
│   │   ├── registry.go    # Platform registry (plugin system)
│   │   ├── twitch/        # Twitch implementation
│   │   │   ├── provider.go    # Platform interface impl
│   │   │   ├── gql.go         # GraphQL API client
│   │   │   ├── hls.go         # HLS stream handler
│   │   │   ├── spade.go       # Spade analytics
│   │   │   ├── pubsub.go      # PubSub WebSocket
│   │   │   ├── chat.go        # IRC Chat WebSocket
│   │   │   ├── ads.go         # Ad detection & watching
│   │   │   └── constants.go   # Twitch-specific constants
│   │   ├── kick/          # Kick implementation (future)
│   │   │   └── provider.go
│   │   └── youtube/       # YouTube implementation (future)
│   │       └── provider.go
│   ├── engine/            # Viewer engine core
│   │   ├── engine.go      # Main engine orchestrator
│   │   ├── worker.go      # Viewer worker (goroutine lifecycle)
│   │   ├── pool.go        # Worker pool management
│   │   ├── scheduler.go   # Smart thread scheduling
│   │   └── metrics.go     # Real-time metrics collector
│   ├── proxy/             # Proxy management
│   │   ├── manager.go     # Proxy pool, rotation, health checks
│   │   ├── checker.go     # Proxy validation & speed test
│   │   ├── parser.go      # Multi-format proxy parser
│   │   └── types.go       # Proxy types (HTTP, SOCKS4, SOCKS5)
│   ├── token/             # Token management
│   │   ├── manager.go     # Token pool, rotation, validation
│   │   ├── validator.go   # Token health checking
│   │   └── importer.go    # Multi-format token import
│   ├── account/           # Multi-account management
│   │   ├── manager.go     # Account CRUD, switching
│   │   ├── profile.go     # Account profile (per-streamer configs)
│   │   └── store.go       # SQLite persistence
│   ├── stream/            # Stream restreaming (FFmpeg)
│   │   ├── ffmpeg.go      # FFmpeg process management
│   │   └── config.go      # Stream quality presets
│   ├── api/               # IPC API server
│   │   ├── server.go      # WebSocket server
│   │   ├── handler.go     # JSON-RPC handlers
│   │   ├── events.go      # Real-time event emitter
│   │   └── middleware.go   # Auth, logging, rate limiting
│   ├── storage/           # Database layer
│   │   ├── sqlite.go      # SQLite connection & migrations
│   │   ├── migrations/    # SQL migration files
│   │   └── repository.go  # Data access patterns
│   └── logger/            # Structured logging
│       ├── logger.go      # Slog-based logger
│       └── ring.go        # Ring buffer for UI log viewer
├── pkg/                   # Public packages (reusable)
│   ├── useragent/         # User-agent generator & rotator
│   │   └── useragent.go
│   ├── fingerprint/       # Browser fingerprint generation
│   │   └── fingerprint.go
│   └── netutil/           # Network utilities
│       └── netutil.go
├── frontend/              # Tauri + React frontend
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── components/    # Reusable UI components
│   │   ├── pages/         # Page-level components
│   │   ├── hooks/         # Custom React hooks
│   │   ├── stores/        # State management
│   │   ├── services/      # IPC service layer
│   │   └── types/         # TypeScript types
│   ├── src-tauri/         # Tauri Rust backend
│   │   ├── src/
│   │   │   └── main.rs    # Sidecar launcher
│   │   └── tauri.conf.json
│   ├── package.json
│   ├── tsconfig.json
│   └── tailwind.config.ts
├── data/                  # Runtime data (gitignored)
│   ├── proxies.txt
│   ├── tokens.txt
│   └── user-agents.txt
├── docker/
│   ├── Dockerfile.backend
│   └── Dockerfile.dev
├── docs/
│   ├── architecture.md    # This file
│   └── api.md             # IPC API documentation
├── .gitattributes
├── .editorconfig
├── .gitignore
├── docker-compose.yml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Platform Interface (Plugin System)

```go
// internal/platform/platform.go

type Platform interface {
    Name() string
    Connect(ctx context.Context, cfg *ViewerConfig) (Viewer, error)
    ValidateToken(ctx context.Context, token string) (bool, error)
    GetStreamStatus(ctx context.Context, channel string) (StreamStatus, error)
    GetStreamMetadata(ctx context.Context, channel string) (*StreamMetadata, error)
}

type Viewer interface {
    Start(ctx context.Context) error
    Stop() error
    Status() ViewerStatus
    Metrics() ViewerMetrics
}

type ViewerConfig struct {
    Channel     string
    Token       string
    Proxy       *Proxy
    UserAgent   string
    DeviceID    string
    Options     map[string]any // Platform-specific options
}

type StreamStatus int
const (
    StreamOffline StreamStatus = iota
    StreamOnline
    StreamUnknown
)

type StreamMetadata struct {
    BroadcastID string
    ChannelID   string
    ViewerCount int
    Title       string
    Game        string
    StartedAt   time.Time
}

type ViewerMetrics struct {
    Connected       bool
    Uptime          time.Duration
    SegmentsFetched int64
    BytesReceived   int64
    HeartbeatsSent  int64
    AdsWatched      int64
    LastError       error
    LastActivity    time.Time
}

type ViewerStatus int
const (
    ViewerIdle ViewerStatus = iota
    ViewerConnecting
    ViewerActive
    ViewerReconnecting
    ViewerStopped
    ViewerError
)
```

## Engine Architecture

```go
// internal/engine/engine.go

type Engine struct {
    platform    platform.Platform
    proxyMgr    *proxy.Manager
    tokenMgr    *token.Manager
    pool        *WorkerPool
    metrics     *MetricsCollector
    eventBus    *events.Bus
    config      *config.EngineConfig
}

type EngineConfig struct {
    MaxWorkers          int           // Max concurrent viewers
    RestartInterval     time.Duration // Dead worker restart check
    HeartbeatInterval   time.Duration // Viewer heartbeat frequency
    SegmentFetchDelay   MinMax        // Random delay range for segments
    GQLPulseInterval    MinMax        // Random delay range for GQL pulses
    ProxyTimeout        time.Duration // Max proxy connection timeout
    MaxRetries          int           // Max retries per viewer
    EnableAds           bool          // Watch ads or skip
    EnableChat          bool          // Join IRC chat
    EnablePubSub        bool          // Connect to PubSub
    EnableSegments      bool          // Fetch HLS segments
    EnableGQLPulse      bool          // Send GQL heartbeats
    EnableSpade         bool          // Send Spade analytics
}

type MinMax struct {
    Min time.Duration
    Max time.Duration
}
```

## IPC Protocol (JSON-RPC over WebSocket)

```json
// Request
{
    "jsonrpc": "2.0",
    "method": "engine.start",
    "params": {
        "profileId": "abc123",
        "channel": "streamer_name",
        "workers": 50
    },
    "id": 1
}

// Response
{
    "jsonrpc": "2.0",
    "result": { "status": "started", "workers": 50 },
    "id": 1
}

// Event (server -> client push)
{
    "jsonrpc": "2.0",
    "method": "event.metrics",
    "params": {
        "activeViewers": 47,
        "totalSegments": 12450,
        "totalBytes": "1.2GB",
        "adsWatched": 23,
        "uptime": "2h34m"
    }
}
```

### IPC Methods

| Category | Method | Description |
|----------|--------|-------------|
| **Engine** | `engine.start` | Start viewer engine for a channel |
| | `engine.stop` | Stop all viewers |
| | `engine.pause` | Pause all viewers (keep connections) |
| | `engine.resume` | Resume paused viewers |
| | `engine.status` | Get engine status & metrics |
| | `engine.setWorkers` | Dynamic worker count adjustment |
| **Profile** | `profile.list` | List all saved profiles |
| | `profile.create` | Create new profile |
| | `profile.update` | Update profile settings |
| | `profile.delete` | Delete a profile |
| | `profile.activate` | Switch active profile |
| **Proxy** | `proxy.list` | List all proxies with health status |
| | `proxy.import` | Import proxies (txt, csv, url) |
| | `proxy.remove` | Remove proxies |
| | `proxy.check` | Run proxy health check |
| | `proxy.setRotation` | Set rotation strategy |
| **Token** | `token.list` | List tokens (masked) |
| | `token.import` | Import tokens |
| | `token.validate` | Validate all tokens |
| | `token.remove` | Remove tokens |
| **Stream** | `stream.status` | Check if target is live |
| | `stream.metadata` | Get stream metadata |
| | `stream.restream.start` | Start FFmpeg restreaming |
| | `stream.restream.stop` | Stop restreaming |
| **Config** | `config.get` | Get current config |
| | `config.set` | Update config values |
| | `config.export` | Export config (encrypted) |
| | `config.import` | Import config |
| **Logs** | `logs.subscribe` | Subscribe to log stream |
| | `logs.history` | Get log history (ring buffer) |
| | `logs.setLevel` | Change log level |
| **Platform** | `platform.list` | List available platforms |
| | `platform.select` | Select active platform |

## Frontend Pages

### 1. Dashboard (Home)
- Real-time viewer count gauge
- Active workers / total workers
- Bytes transferred, segments fetched
- Ads watched counter
- Stream status indicator (live/offline)
- Quick start/stop controls
- Mini charts (viewers over time, bandwidth)

### 2. Profiles (Multi-Account)
- Profile cards with per-channel configs
- Quick switch between profiles
- Duplicate/import/export profiles
- Per-profile settings override

### 3. Proxy Manager
- Proxy list with health indicators (green/yellow/red)
- Bulk import (paste, file, URL)
- One-click health check all
- Rotation strategy selector (round-robin, random, least-used, fastest)
- Proxy type filter (HTTP, SOCKS4, SOCKS5)
- Country/region grouping

### 4. Token Vault
- Encrypted token storage
- Token list with validity status
- Bulk import
- Auto-validation on import
- Invalid token quarantine

### 5. Settings
- Engine settings (workers, intervals, timeouts)
- Feature toggles (ads, chat, pubsub, segments, gql, spade)
- Platform selection
- UI preferences (theme, language)
- Data management (export/import/reset)

### 6. Logs
- Real-time log viewer with color-coded levels
- Filter by level, worker, component
- Search through logs
- Export logs

### 7. Stream Monitor
- Target stream info (title, game, viewers)
- Restream controls (FFmpeg)
- Quality presets

## Security Considerations

1. **Token Encryption**: AES-256-GCM at rest, key derived from OS keyring
2. **No Hardcoded Secrets**: All credentials in encrypted vault
3. **Proxy Auth**: Supports authenticated proxies (user:pass)
4. **IPC Security**: Localhost-only WebSocket, random port, auth token
5. **Config Encryption**: Optional encrypted config export/import

## Decisions Made

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Multi-platform | Extensible, Twitch first | Plugin architecture allows future Kick/YouTube |
| Multi-account | Profiles + multi-channel | Each profile has independent config |
| Persistence | SQLite (pure Go) | No CGO, portable, single file |
| IPC | WebSocket JSON-RPC | Bidirectional, real-time events |
| Go sidecar vs Rust | Go sidecar | Reuse Go expertise, Tauri sidecar support |
| Frontend framework | React + TypeScript | Ecosystem, Tauri compatibility |
| Proxy types | HTTP + SOCKS4 + SOCKS5 | Maximum flexibility |
| Log system | slog (stdlib) + ring buffer | Structured logging, UI-friendly |
