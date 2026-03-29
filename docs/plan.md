# Implementation Plan — MongeBot Go

## Current State (v2.0.0)

- 154 files, 20k+ LOC, 97 tests, 51 IPC methods
- Go backend compiles and runs
- Frontend builds and renders
- Deployed to eu-central-1.kizubot.com:9800
- Prometheus monitoring active
- 841 proxies auto-scraped
- Several modules written but not wired into main flow (see todo.md section "Critical")

---

## Phase 1: Wire Everything (v2.0.1) — Estimated: 1 session

**Goal:** Make all existing modules actually functional end-to-end.

### Tasks

1. **Wire vault into main.go**
   - Replace `loadTokensFromFile("data/tokens.txt")` with `vault.Open()` + `vault.GetValidTokenValues()`
   - Add `--vault-passphrase` flag or prompt at startup
   - Fall back to plain text file if vault doesn't exist

2. **Wire config migration**
   - In `config.Load()`, call `migrator.MigrateIfNeeded(data)` before `json.Unmarshal`
   - Save migrated config back to disk if changed

3. **Wire behavior profiles into viewer**
   - In `engine.spawnWorker`, call `behavior.RandomProfile()` or use `cfg.BehaviorProfile`
   - Pass profile timings into ViewerConfig.Options
   - Viewer reads Options for heartbeat/segment/gql intervals

4. **Wire circuit breaker into viewers**
   - Create one CircuitBreaker per proxy in ProxyManager
   - Wrap GQL and HLS requests in `cb.Execute()`
   - On circuit open, mark proxy as HealthSlow/HealthDead

5. **Wire scheduler auto-start**
   - In main.go after creating scheduler, call `scheduler.Start(ctx)`
   - Load rules from config or SQLite on boot

6. **Wire metrics persister**
   - On `engine.Start()`, call `persister.StartSession()`
   - On `engine.Stop()`, call `persister.EndSession()`
   - Pass sessionID to the snapshot loop

7. **Wire account manager to SQLite**
   - Replace `account.NewManager(filePath)` with `storage.NewProfileRepo(db)`
   - Migrate existing JSON profiles on first run

8. **Wire onboarding wizard IPC**
   - In OnboardingWizard.tsx, call `proxy.import`, `token.import`, `profile.create` on finish
   - Show success/error for each step

### Verification
- [ ] Bot can start with vault-encrypted tokens
- [ ] Config auto-migrates from v1 to v2
- [ ] Different viewers use different behavior profiles
- [ ] Circuit breaker trips after 5 proxy failures
- [ ] Scheduler auto-starts when channel goes live
- [ ] Session appears in SessionHistory after engine.stop
- [ ] Onboarding wizard imports real data

---

## Phase 2: Full Platform Coverage (v2.1.0) — Estimated: 2 sessions

**Goal:** YouTube full viewer + Kick chat + production hardening.

### Tasks

1. **YouTube full viewer**
   - Implement DASH/HLS URL extraction with signature handling
   - Innertube player request for stream manifest
   - Segment fetching loop similar to Twitch
   - Live chat monitoring via Innertube getLiveChat

2. **Kick Pusher chat**
   - Connect to Pusher WebSocket (`wss://ws-us2.pusher.com/app/...`)
   - Subscribe to chatroom channel (`chatrooms.{id}`)
   - Handle presence events

3. **Token auto-refresh**
   - Background loop checking token validity every 30 min
   - Move expired tokens to quarantine
   - Notify frontend via event.tokenExpired
   - Webhook notification on token pool depletion

4. **Proxy auto-maintenance**
   - Schedule: scrape every 6 hours, check every 2 hours
   - Remove dead proxies after 3 consecutive failures
   - Auto-scrape when available pool drops below threshold

5. **Production error handling**
   - Use `netutil.CategorizedError` consistently across all modules
   - Add structured error context (proxy, token, channel) to all errors
   - Rate limit tracker integrated into viewer error handling

### Verification
- [ ] YouTube viewer fetches segments from a live stream
- [ ] Kick viewer joins chat and receives messages
- [ ] Expired tokens auto-quarantine without manual intervention
- [ ] Proxy pool self-maintains above configured minimum
- [ ] All errors logged with category and context

---

## Phase 3: Dashboard & Monitoring (v2.2.0) — Estimated: 1 session

**Goal:** Complete the dashboard integration and Grafana setup.

### Tasks

1. **Merge api-server PR and dash PR**
   - Test api-server-kizubot#1 endpoints
   - Test dash-kizubot#20 page rendering
   - Deploy both

2. **Grafana dashboard template**
   - Create JSON template with panels:
     - Active viewers over time (graph)
     - Engine state (stat)
     - Proxy pool (gauge)
     - Token pool (gauge)
     - Segments/s (rate graph)
     - Bytes/s (rate graph)
     - Goroutines + memory (system panel)
   - Export as `docs/grafana-dashboard.json`

3. **Alertmanager rules**
   - Alert: viewer count drops to 0 when engine is "running"
   - Alert: proxy pool < 10 available
   - Alert: all tokens quarantined
   - Alert: memory > 500MB
   - Webhook to Discord on alert fire

4. **Light theme completion**
   - Audit all 10 pages for hardcoded `gray-900`, `gray-800` etc.
   - Use CSS variables from theme.ts
   - Test toggle between dark/light

5. **Session detail expansion**
   - Click session card to expand with SessionChart
   - Show metrics timeline graph inline
   - Add CSV export per individual session

### Verification
- [ ] Grafana dashboard shows live MongeBot data
- [ ] Alerts fire on proxy depletion (test by removing all proxies)
- [ ] Light theme is fully functional
- [ ] Session timeline chart renders in SessionHistory

---

## Phase 4: Scale & Resilience (v3.0.0) — Estimated: 3+ sessions

**Goal:** Multi-machine deployment, horizontal scaling, advanced features.

### Tasks

1. **Multi-machine coordination**
   - Central coordinator (API server) distributes channels across bot machines
   - Each machine runs independent MongeBot instance
   - Coordinator aggregates metrics from all instances
   - Dashboard shows per-machine and global views

2. **Viewer count targeting**
   - Set target viewer count for a channel
   - Engine auto-scales workers up/down to maintain target
   - Account for natural viewer drop rate
   - Gradual ramp-up to avoid detection

3. **Proxy intelligence**
   - Score proxies by success rate, latency, and uptime
   - Automatic proxy rotation based on score
   - Geographic targeting (prefer proxies from specific regions)
   - Blacklist detection (detect when proxy IP is banned)

4. **Token farming integration**
   - Automated token acquisition pipeline
   - Token validity monitoring dashboard
   - Auto-rotate tokens approaching expiry
   - Token usage analytics (which tokens get rate-limited most)

5. **API versioning**
   - JSON-RPC method versioning (e.g., `engine.v2.start`)
   - Backward compatibility layer
   - OpenAPI/Swagger documentation generation

### Verification
- [ ] 3 machines running coordinated viewers
- [ ] Engine maintains target viewer count within 10%
- [ ] Proxy scoring improves success rate over time
- [ ] Token pool self-maintains without manual intervention

---

## Timeline Summary

| Phase | Version | Focus | Size |
|-------|---------|-------|------|
| 1 | v2.0.1 | Wire existing code | Small (1 session) |
| 2 | v2.1.0 | Full platforms + hardening | Medium (2 sessions) |
| 3 | v2.2.0 | Dashboard + monitoring | Small (1 session) |
| 4 | v3.0.0 | Scale + multi-machine | Large (3+ sessions) |
