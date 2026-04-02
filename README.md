# MongeBot Go v2.0

Modular, multi-platform viewer bot rewritten from Python to Go with a Tauri 2.0 desktop interface.

## Features

**Engine**
- Multi-channel simultaneous viewer pools with independent goroutine workers
- Auto-reconnection with exponential backoff and jitter
- 5 behavior profiles (lurker, active, engaged, stealth, rotating)
- Global rate limit detection with coordinated cooldowns
- Formal viewer state machine (idle → connecting → active → reconnecting → stopped/error)
- Graceful shutdown with connection drain and final metrics snapshot

**Platforms**
- **Twitch** (full): GQL API, HLS segments, Spade analytics, PubSub WebSocket, IRC Chat, ad detection/watching, channel points auto-claim, drops tracking
- **Kick** (full): REST API, HLS segment fetching, liveness monitoring
- **YouTube** (stub): Innertube API, live detection, ready for expansion

**Proxy Management**
- 4 rotation strategies: round-robin, random, least-used, fastest
- Concurrent health checker with latency measurement
- IP geolocation enrichment (country tagging, flag mapping)
- Proxy chain support for multi-hop anonymization

**Token Management**
- Pool with rotation, validation, quarantine, and auto-quarantine after 3 errors
- Multi-format import: raw text, JSON array, Netscape cookies, EditThisCookie (browser extension)
- Concurrent batch validation with progress reporting

**Anti-Detection**
- TLS fingerprint rotation across 4 browser profiles (Chrome, Firefox, Safari, Edge)
- Cipher suite shuffling per connection
- Canvas, WebGL, screen resolution, timezone, and language randomization
- Automatic user-agent updater (fetches latest Chrome stable version every 24h)

**Desktop UI** (Tauri 2.0 + React + TypeScript + TailwindCSS v4)
- 10 pages: Dashboard, Profiles, Proxies, Tokens, Stream Monitor, Session History, Scheduler, Logs, Settings, About
- Real-time Recharts graphs (viewers over time, bandwidth)
- Multi-channel cards with independent metrics
- Toast notifications (Discord-style, 6 types)
- Channel search autocomplete with live/offline indicators
- Dark/light theme with 6 accent colors
- Keyboard shortcuts (Ctrl+1-9 navigation, ? help overlay)
- First-run onboarding wizard
- Error boundaries, connection overlay, skeleton loaders

**Scheduler**
- Stream-live trigger: auto-starts when streamer goes online
- Time-based trigger: daily schedules with weekday filtering
- Configurable max session duration
- Webhook notifications (Discord, Telegram, generic HTTP)

**Infrastructure**
- WebSocket JSON-RPC 2.0 API with 51 methods
- Prometheus `/metrics` endpoint with 20+ metrics
- SQLite persistence (sessions, metrics snapshots, profiles)
- Docker multi-stage build (Alpine, CGO_ENABLED=0)
- GitHub Actions CI/CD (test, lint, cross-compile 6 platforms, Tauri bundle)
- Makefile with 25+ targets
- Environment variable overrides (.env support)
- Config version migration system (v0 → v1 → v2)

## Quick Start

### Prerequisites

- Go 1.24+ (1.26 recommended)
- Node.js 22+ (for frontend)
- Rust (for Tauri desktop builds)
- Docker (optional, for containerized deployment)
- FFmpeg (optional, for restreaming)

### Setup

```bash
git clone https://github.com/Kizuno18/mongebot-go.git
cd mongebot-go

# Install backend dependencies
go mod tidy

# Install frontend dependencies
cd frontend && npm install && cd ..

# Copy sample data files
cp data/proxies.txt.example data/proxies.txt
cp data/tokens.txt.example data/tokens.txt
cp data/user-agents.txt.example data/user-agents.txt
```

### Run Desktop App (Tauri)

```bash
cd frontend
npm run tauri dev
```

### Run Headless CLI

```bash
make build
./bin/mongebot --mode headless --channel streamer_name --workers 50
```

### Run CLI with TUI Dashboard

```bash
make build-cli
./bin/mongebot-cli --channel streamer_name --workers 50
```

### Run API Server (Sidecar Mode)

```bash
./bin/mongebot --mode sidecar --port 9800
```

### Run with Docker

```bash
docker compose up -d --build
```

## Configuration

Config is stored in `data/config.json` (auto-created with defaults on first run).

### Environment Variables

All values from `config.json` can be overridden via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MONGEBOT_MODE` | `sidecar` | Run mode: `sidecar` (API) or `headless` (CLI) |
| `MONGEBOT_API_PORT` | `9800` | WebSocket API port |
| `MONGEBOT_API_HOST` | `127.0.0.1` | API bind address (`0.0.0.0` for Docker) |
| `MONGEBOT_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `MONGEBOT_MAX_WORKERS` | `50` | Maximum concurrent viewers |
| `MONGEBOT_CHANNEL` | — | Default channel (headless mode) |
| `MONGEBOT_PLATFORM` | `twitch` | Default platform |
| `MONGEBOT_ENABLE_ADS` | `true` | Enable ad watching |
| `MONGEBOT_ENABLE_CHAT` | `true` | Enable IRC chat |
| `MONGEBOT_ENABLE_PUBSUB` | `true` | Enable PubSub WebSocket |

See `.env.example` for the full list.

## API Reference

The backend exposes a WebSocket JSON-RPC 2.0 API at `ws://HOST:PORT/ws` plus HTTP endpoints.

### HTTP Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check (`{"status":"ok"}`) |
| `GET /metrics` | Prometheus metrics (text format) |

### WebSocket JSON-RPC Methods (51 total)

<details>
<summary>Engine Control (4)</summary>

| Method | Params | Description |
|--------|--------|-------------|
| `engine.start` | `{channel, workers}` | Start viewer engine |
| `engine.stop` | — | Stop all viewers |
| `engine.status` | — | Get metrics snapshot |
| `engine.setWorkers` | `{count}` | Dynamic worker scaling |

</details>

<details>
<summary>Multi-Channel (6)</summary>

| Method | Params | Description |
|--------|--------|-------------|
| `multi.start` | `{channel, workers}` | Start channel engine |
| `multi.stop` | `{channel}` | Stop specific channel |
| `multi.stopAll` | — | Stop all channels |
| `multi.status` | — | All channel statuses |
| `multi.channels` | — | List running channels |
| `multi.workers` | `{channel, count}` | Adjust per-channel workers |

</details>

<details>
<summary>Profiles (6)</summary>

| Method | Params | Description |
|--------|--------|-------------|
| `profile.list` | — | All profiles |
| `profile.create` | `{name, platform, channel}` | Create profile |
| `profile.delete` | `{id}` | Delete profile |
| `profile.activate` | `{id}` | Set active profile |
| `profile.duplicate` | `{id, newName}` | Clone profile |
| `profile.export` | — | Export all profiles |

</details>

<details>
<summary>Proxy (6)</summary>

| Method | Params | Description |
|--------|--------|-------------|
| `proxy.list` | — | Pool stats + proxy list |
| `proxy.import` | `{proxies: []}` | Bulk import |
| `proxy.check` | — | Run health check |
| `proxy.geoEnrich` | — | Fetch country data for all proxies |
| `proxy.geoStats` | — | Country distribution |

</details>

<details>
<summary>Tokens (4)</summary>

| Method | Params | Description |
|--------|--------|-------------|
| `token.list` | — | Masked token list |
| `token.import` | `{tokens: [], platform}` | Bulk import |
| `token.stats` | — | Pool statistics |
| `token.validate` | — | Batch validate all tokens |

</details>

<details>
<summary>Stream / Restream (4)</summary>

| Method | Params | Description |
|--------|--------|-------------|
| `stream.restream.start` | `{inputFile, streamKey, quality}` | Start FFmpeg |
| `stream.restream.stop` | — | Stop FFmpeg |
| `stream.restream.state` | — | Current state |
| `stream.presets` | — | Quality presets |

</details>

<details>
<summary>Scheduler (5)</summary>

| Method | Params | Description |
|--------|--------|-------------|
| `scheduler.list` | — | All rules |
| `scheduler.add` | `{ScheduleRule}` | Add rule |
| `scheduler.remove` | `{id}` | Remove rule |
| `scheduler.start` | — | Start scheduler |
| `scheduler.stop` | — | Stop scheduler |

</details>

<details>
<summary>Sessions (4)</summary>

| Method | Params | Description |
|--------|--------|-------------|
| `sessions.recent` | `{limit}` | Recent sessions |
| `sessions.timeline` | `{sessionId}` | Metrics timeline |
| `sessions.stats` | — | Aggregate stats |
| `sessions.export` | `{format, limit}` | Export CSV/JSON |

</details>

<details>
<summary>Config (4)</summary>

| Method | Params | Description |
|--------|--------|-------------|
| `config.get` | — | Current config |
| `config.set` | `{...overrides}` | Update config |
| `config.export` | — | Plain JSON archive |
| `config.import` | `{data}` | Import archive |

</details>

<details>
<summary>System (5)</summary>

| Method | Params | Description |
|--------|--------|-------------|
| `system.health` | — | Health + uptime |
| `system.healthcheck` | — | Deep check (DB, tokens, memory) |
| `system.version` | — | Version + Go + OS info |
| `system.info` | — | Full system info + memory stats |
| `system.uptime` | — | Process uptime |

</details>

<details>
<summary>Other (7)</summary>

| Method | Params | Description |
|--------|--------|-------------|
| `channel.search` | `{query, limit}` | Twitch channel search |
| `behavior.list` | — | Viewer behavior profiles |
| `drops.progress` | — | Twitch drops campaigns |
| `drops.points` | `{channelId}` | Channel points balance |
| `webhook.list` | — | Configured webhooks |
| `webhook.add` | `{WebhookConfig}` | Add webhook |
| `webhook.remove` | `{id}` | Remove webhook |
| `webhook.test` | `{id}` | Send test notification |
| `logs.history` | — | Log ring buffer |

</details>

### Server-Pushed Events

| Event | Description |
|-------|-------------|
| `event.metrics` | Engine metrics (every 5s) |
| `event.log` | Real-time log entries |
| `event.stream` | Stream online/offline transitions |
| `event.error` | Error notifications |

## Deployment

### Docker (Production)

```bash
docker compose up -d --build
# API available at http://localhost:9800
# Health: curl http://localhost:9800/health
# Metrics: curl http://localhost:9800/metrics
```

### Cross-Compilation

```bash
make release-all   # Linux, macOS, Windows (amd64 + arm64)
ls -lh dist/
```

### Prometheus Monitoring

Add to your `prometheus.yml`:

```yaml
- job_name: mongebot
  scrape_interval: 15s
  static_configs:
    - targets: [your-server:9800]
```

Available metrics: `mongebot_active_viewers`, `mongebot_total_workers`, `mongebot_segments_fetched_total`, `mongebot_bytes_received_total`, `mongebot_heartbeats_sent_total`, `mongebot_ads_watched_total`, `mongebot_proxies_total`, `mongebot_tokens_valid`, `mongebot_go_goroutines`, `mongebot_build_info`, and more.

## Project Structure

```
mongebot-go/
├── cmd/
│   ├── mongebot/              # Main entry (sidecar + headless modes)
│   └── cli/                   # Interactive CLI with TUI dashboard
├── internal/
│   ├── account/               # Multi-account profiles (CRUD, clone, export)
│   ├── api/                   # WebSocket JSON-RPC server (51 methods)
│   ├── config/                # Config, env overrides, migration, archive
│   ├── engine/                # Worker pool, multi-engine, scheduler, FSM,
│   │                            reconnect, ratelimit, behavior, webhook,
│   │                            eventbus, persistence, shutdown
│   ├── logger/                # slog structured logging + ring buffer
│   ├── platform/
│   │   ├── twitch/            # Full: GQL, HLS, Spade, PubSub, Chat, Ads, Points, Drops
│   │   ├── kick/              # Full: API, HLS segments, liveness
│   │   └── youtube/           # Stub: Innertube API, live detection
│   ├── proxy/                 # Pool, checker, geolocation
│   ├── storage/               # SQLite persistence, repository, export
│   ├── stream/                # FFmpeg restreaming (5 presets)
│   ├── token/                 # Pool, validator, multi-format importer
│   
├── pkg/
│   ├── fingerprint/           # Device ID, TLS rotation, browser profiles
│   ├── netutil/               # Retry, circuit breaker, typed errors, IP utils
│   └── useragent/             # UA pool, auto-updater
├── frontend/                  # Tauri 2.0 + React + TypeScript
│   ├── src/pages/             # 10 pages
│   ├── src/components/        # 11 reusable components
│   ├── src/hooks/             # 3 custom hooks
│   └── src-tauri/             # Rust sidecar launcher
├── docker/                    # Dockerfiles
├── .github/workflows/         # CI/CD pipeline
└── docs/                      # Architecture, plan, todo
```

## Tech Stack

| Layer | Technology | Why |
|-------|-----------|-----|
| Backend | Go 1.26 | Green Tea GC, goroutines, native concurrency |
| Frontend | React + TypeScript + TailwindCSS v4 | Component ecosystem, type safety |
| Desktop | Tauri 2.0 (Rust) | ~2MB bundle, native webview |
| WebSocket | coder/websocket | Context-aware, concurrent-safe |
| Database | SQLite (modernc.org/sqlite) | Pure Go, no CGO, portable |
| Encryption | None | Simplified architecture |
| Charts | Recharts | React-native charting |
| Icons | Lucide React | Consistent icon set |
| CI/CD | GitHub Actions | Test, lint, build, bundle |
| Monitoring | Prometheus + Grafana | Industry standard |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and code standards.

## License

MIT — see [LICENSE](LICENSE).
