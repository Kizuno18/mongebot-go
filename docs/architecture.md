# Architecture — MongeBot Go v2.0

## Overview

MongeBot is a modular, multi-platform viewer bot built as a Go backend with a Tauri 2.0 desktop frontend. The architecture follows a plugin-based design where each streaming platform is a self-contained provider implementing a common interface.

## System Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                     TAURI DESKTOP APP                               │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                 React Frontend (TypeScript)                    │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        │  │
│  │  │Dashboard │ │ Profiles │ │ Proxies  │ │ Tokens   │  ...    │  │
│  │  │(Charts)  │ │ (CRUD)   │ │(Pool)    │ │(Import)  │ 10 pgs │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘        │  │
│  │  ┌──────────────────────────────────────────────────┐        │  │
│  │  │ IPC Service (WebSocket JSON-RPC 2.0 Client)      │        │  │
│  │  └──────────────────────┬───────────────────────────┘        │  │
│  └─────────────────────────┼─────────────────────────────────────┘  │
│                            │ ws://127.0.0.1:9800/ws                 │
│  ┌─────────────────────────▼─────────────────────────────────────┐  │
│  │                 Go Sidecar (Backend Engine)                    │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                             │
             ┌───────────────┼───────────────┐
             ▼               ▼               ▼
      ┌────────────┐  ┌────────────┐  ┌────────────┐
      │  Twitch    │  │   Kick     │  │  YouTube   │
      │  Provider  │  │  Provider  │  │  Provider  │
      │ (12 files) │  │ (2 files)  │  │ (1 file)   │
      └────────────┘  └────────────┘  └────────────┘
             │               │               │
             ▼               ▼               ▼
      ┌──────────────────────────────────────────┐
      │        Streaming Platforms (Internet)      │
      └──────────────────────────────────────────┘
```

## Backend Modules

```
internal/
├── config/       Config, env overrides, migration, archive
├── platform/     Plugin interface + registry + 3 providers
│   ├── twitch/   GQL, HLS, Spade, PubSub, Chat, Ads, Points, Drops
│   ├── kick/     REST API, HLS segments, liveness
│   └── youtube/  Innertube API, live detection
├── engine/       Worker pool, multi-channel, scheduler, FSM, reconnect,
│                 ratelimit, behavior profiles, webhook, eventbus, persistence
├── proxy/        Pool (4 strategies), checker, geo, chains
├── token/        Pool, validator, multi-format importer
├── account/      Multi-account profiles (CRUD, clone, export)
├── storage/      SQLite (WAL, migrations, repository, export)
├── stream/       FFmpeg restreaming (5 quality presets)
├── api/          WebSocket JSON-RPC (50 methods, middleware, Prometheus)
└── logger/       slog structured + ring buffer with pub/sub

pkg/
├── fingerprint/  Device ID, TLS rotation (4 browser profiles)
├── netutil/      Retry, circuit breaker, typed errors, IP utils
└── useragent/    UA pool, auto-updater (Chrome stable API)
```

## Viewer Lifecycle

```
Engine.Start(channel, N)
  │
  ├─ for each worker:
  │   ├─ Acquire proxy (rotation strategy)
  │   ├─ Acquire token (round-robin)
  │   ├─ Generate fingerprint (TLS profile, device ID, UA)
  │   └─ Wrap in ReconnectingViewer
  │
  ├─ ReconnectingViewer.Start(ctx)
  │   ├─ FSM: idle → connecting
  │   ├─ Platform.Connect(config) → Viewer
  │   │
  │   ├─ Viewer.Start(ctx)              [Twitch]
  │   │   ├─ GQL: metadata + auth + stream token
  │   │   ├─ HTTP: M3U8 master playlist
  │   │   ├─ Spade: video-play event
  │   │   │
  │   │   └─ Concurrent goroutines:
  │   │       ├─ heartbeatLoop (Spade minute-watched, 60s)
  │   │       ├─ segmentFetcherLoop (HLS .ts chunks, 4-8s)
  │   │       ├─ gqlPulseLoop (WatchTrackQuery, 3-7min)
  │   │       ├─ livenessLoop (HEAD check, 40s)
  │   │       ├─ PubSub WebSocket (ad events)
  │   │       ├─ IRC Chat (channel presence)
  │   │       └─ PointsClaimer (bonus check, 5min)
  │   │
  │   └─ On disconnect:
  │       ├─ FSM: active → reconnecting
  │       ├─ Exponential backoff with jitter
  │       └─ Retry (up to maxAttempts)
  │
  └─ MonitorLoop (check dead workers, restart)
```

## IPC Protocol

WebSocket JSON-RPC 2.0 at `ws://HOST:PORT/ws`.

50 methods in 12 groups: engine (4), multi (6), profile (6), proxy (5), token (4), stream (4), scheduler (5), config (4), sessions (4), system (5), search/behavior/drops (3+4), webhook (4), logs (1).

Server pushes events: `event.metrics` (5s), `event.log`, `event.stream`, `event.error`.

## Database Schema

4 tables in SQLite (WAL mode, pure Go driver):

- `profiles` — multi-account configs (FK target for sessions)
- `sessions` — bot session records with final metrics
- `metrics_snapshots` — point-in-time metrics per session
- `proxy_health_log` — proxy check results over time

## Security

- API: localhost-only, rate limiter, CORS whitelist, security headers
- Config archive: export/import for portability
- Proxy list: masked in API responses (counts only, no IPs)
- TLS: randomized cipher suites per connection

## Key Design Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| Go sidecar vs embedded | Go sidecar | Reuse Go expertise, Tauri sidecar support |
| SQLite vs Postgres | SQLite pure Go | No CGO, portable single file, WAL concurrency |
| JSON-RPC vs REST | JSON-RPC over WS | Bidirectional, real-time events, single connection |
| Per-viewer goroutine | Yes | Go's goroutine model handles thousands cheaply |
| Plugin interface | `platform.Platform` | Add Kick/YouTube without touching engine code |
| Prometheus text format | Native /metrics | No deps, industry standard, Grafana compatible |
