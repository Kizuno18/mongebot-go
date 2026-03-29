# TODO — MongeBot Go v2.0

## Priority: Critical (Wire Existing Code)

These modules exist but are not connected to the main flow.

- [ ] Wire vault into main.go (replace token file loading with encrypted vault)
- [ ] Wire config migration into config.Load() (call MigrateIfNeeded before parse)
- [ ] Wire behavior profiles into viewer (use profile timings instead of hardcoded)
- [ ] Wire circuit breaker into viewer HTTP calls (wrap GQL/HLS requests)
- [ ] Wire proxy chains into engine (allow chain selection per profile)
- [ ] Wire account manager to SQLite repository (replace JSON file persistence)
- [ ] Wire scheduler auto-start on main.go boot (start monitoring enabled rules)
- [ ] Wire metrics persister StartSession/EndSession in engine start/stop

## Priority: High (Functional Gaps)

- [ ] Token validate handler: pass real platform + proxy reference (currently nil)
- [ ] Proxy geo enrichment handler: implement actual async call to GeoEnricher
- [ ] Onboarding wizard: wire IPC calls for proxy.import, token.import, profile.create
- [ ] Webhook config persistence: save/load webhooks to JSON or SQLite
- [ ] ChannelSearch: pass a valid token for GQL search to work
- [ ] Session export download: test in Tauri webview (blob URL compatibility)
- [ ] MultiChannelCards: add UI to start multiple channels from dashboard

## Priority: Medium (Improvements)

- [ ] Add unit tests for all 51 API handlers with mock dependencies
- [ ] Add E2E tests with Playwright for critical frontend flows
- [ ] Implement YouTube full viewer (DASH/HLS with signed URLs)
- [ ] Implement Kick Pusher WebSocket chat client
- [ ] Light theme: audit all components for hardcoded dark colors
- [ ] Add Grafana dashboard JSON template for MongeBot metrics
- [ ] Add Alertmanager rules (viewer drop, proxy pool depleted, token exhaustion)
- [ ] Token auto-refresh: detect expiring tokens and re-validate periodically
- [ ] Add proxy import from URL (paste URL, fetch list, auto-import)
- [ ] Add session detail expand in SessionHistory with MetricsChart

## Priority: Low (Nice to Have)

- [ ] Add Telegram bot command interface (/start, /stop, /status)
- [ ] Add Discord bot command interface
- [ ] Add browser extension for one-click token extraction
- [ ] Add proxy pool auto-maintain (scrape + check on schedule)
- [ ] Add viewer count target mode (scale workers to maintain target count)
- [ ] Add bandwidth limiting per viewer
- [ ] Add custom GQL operation hash updater (detect when Twitch changes hashes)
- [ ] Add multi-language support (i18n) to frontend
- [ ] Add audit log page (who did what, when)
- [ ] Add profile templates (quick-start configs for common use cases)

## Priority: Technical Debt

- [ ] Remove `_ = monitor` in main.go (actually use it or remove)
- [ ] Consolidate token loading (file vs vault vs manager — pick one path)
- [ ] Standardize error handling (use pkg/netutil/errors everywhere)
- [ ] Add godoc comments to all exported functions
- [ ] Add OpenAPI/Swagger spec for the JSON-RPC API
- [ ] Add `.env.test` for test environment isolation
- [ ] Audit all goroutine leaks (ensure all have ctx cancellation)
- [ ] Add pprof endpoint for profiling in development mode

## Done (Completed in This Session)

- [x] Go rewrite from Python (20k+ LOC)
- [x] Tauri 2.0 desktop app with 10 pages
- [x] 51 WebSocket JSON-RPC methods
- [x] 3 platform providers (Twitch full, Kick full, YouTube stub)
- [x] 97 tests + 15 benchmarks passing
- [x] Docker build + deploy to eu-central-1
- [x] Prometheus monitoring integration
- [x] 841 proxies auto-scraped
- [x] GitHub release v2.0.0 with 6 cross-compiled binaries
- [x] CI/CD pipeline (GitHub Actions)
- [x] Integration PRs for api-server and dashboard
