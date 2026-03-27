# Changelog

All notable changes to MongeBot Go are documented here.

## [2.0.0] - 2026-03-27

### Added

**Backend (Go 1.26)**
- Complete rewrite from Python to Go
- Plugin-based platform architecture (Twitch, Kick, YouTube)
- Multi-channel engine with independent goroutine worker pools
- Multi-account profile system with per-channel config overrides
- Proxy management: 4 rotation strategies, concurrent health checker, auto-scraper (3 public APIs)
- Token management: pool with rotation, validation, quarantine, auto-quarantine after 3 errors
- Token import: 4 formats (raw text, JSON array, Netscape cookies, EditThisCookie)
- Encrypted vault: AES-256-GCM with PBKDF2 key derivation (600k iterations)
- SQLite persistence: sessions, metrics snapshots, profiles, proxy health log
- FFmpeg restreaming: 5 quality presets (potato to ultra), proxy support, auto-restart
- Scheduler: stream-live auto-detect trigger, time-of-day schedules, weekday filters, max duration
- Event bus: typed publish/subscribe with topic isolation
- Circuit breaker pattern for graceful degradation
- Exponential backoff with jitter for retries
- TLS fingerprint rotation: 4 browser profiles, cipher shuffle, canvas/WebGL randomization
- Typed error system: 7 categories with retry policies
- Proxy geolocation enrichment via ip-api.com
- Config migration system (v0 -> v1 -> v2)
- Config archive: encrypted export/import for portability
- Environment variable overrides with .env file support
- Graceful shutdown with connection drain and final metrics
- WebSocket JSON-RPC 2.0 API: 40 methods
- Structured logging (slog) with ring buffer for UI streaming
- Rate limiter middleware (token bucket per-IP)
- CORS, security headers, request logging middleware
- Twitch: GQL, HLS segments, Spade analytics, PubSub, IRC Chat, ad detection
- Kick: API integration, channel metadata, HLS-based viewer
- YouTube: Innertube API, live detection, live chat stub
- Channel search with GQL autocomplete

**Frontend (React + TypeScript + TailwindCSS)**
- 10 pages: Dashboard, Profiles, Proxies, Tokens, Stream Monitor, Session History, Scheduler, Logs, Settings, About
- Real-time Recharts dashboard (viewers over time, bandwidth)
- Multi-channel cards with independent metrics
- Toast notification system (6 types)
- Channel search with autocomplete and live/offline indicators
- Status bar with live metrics and keyboard hints
- Dark/light theme with 6 accent colors
- Keyboard shortcuts (Ctrl+1-9 navigation, Escape stop)
- Dynamic window title (channel + viewer count)
- First-run onboarding wizard (4 steps)
- Error boundaries with fallback UI
- Connection overlay for disconnection states
- Skeleton loaders for loading states
- Proxy auto-scraper UI with inline results
- Token import with 3 format tabs (Raw/Cookies/JSON)
- FFmpeg quality preset selector (5 presets)
- Schedule rule creator (stream-live + time-based triggers)
- Session history with expandable metrics charts
- Settings: feature toggles, engine config, theme, data export/import

**Infrastructure**
- Tauri 2.0 desktop app with Go sidecar
- Docker multi-stage build (Alpine, CGO_ENABLED=0)
- Docker Compose with persistent volumes
- GitHub Actions CI/CD: test, lint, cross-compile (6 platforms), Tauri bundle (3 OS)
- Makefile with 25+ targets
- golangci-lint configuration (22 linters)
- .gitattributes (LF everywhere) + .editorconfig

## [1.0.0] - Previous Python Version

Original Python implementation (mongebot.py). See `/mongebot/` directory for the original source.
