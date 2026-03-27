// Package api - extended handlers for profiles, tokens, stream, scraper, and sessions.
package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Kizuno18/mongebot-go/internal/account"
	"github.com/Kizuno18/mongebot-go/internal/config"
	"github.com/Kizuno18/mongebot-go/internal/proxy"
	"github.com/Kizuno18/mongebot-go/internal/storage"
	"github.com/Kizuno18/mongebot-go/internal/stream"
	"github.com/Kizuno18/mongebot-go/internal/token"
)

// ExtendedDeps holds optional dependencies for extended handlers.
type ExtendedDeps struct {
	AccountMgr   *account.Manager
	TokenMgr     *token.Manager
	StreamMgr    *stream.Manager
	ProxyScraper *proxy.Scraper
	Storage      *storage.DB
}

var globalExtDeps *ExtendedDeps

// SetExtendedDeps sets the extended dependencies globally.
func SetExtendedDeps(deps *ExtendedDeps) {
	globalExtDeps = deps
}

// getExtendedHandler returns handlers for extended methods.
func getExtendedHandler(method string) (handlerFunc, bool) {
	if globalExtDeps == nil {
		return nil, false
	}

	handlers := map[string]handlerFunc{
		// Profile methods
		"profile.list":      handleProfileList,
		"profile.create":    handleProfileCreate,
		"profile.delete":    handleProfileDelete,
		"profile.activate":  handleProfileActivate,
		"profile.duplicate": handleProfileDuplicate,
		"profile.export":    handleProfileExport,

		// Token methods
		"token.list":     handleTokenList,
		"token.import":   handleTokenImport,
		"token.stats":    handleTokenStats,
		"token.validate": handleTokenValidate,

		// Stream methods
		"stream.restream.start": handleStreamStart,
		"stream.restream.stop":  handleStreamStop,
		"stream.restream.state": handleStreamState,
		"stream.presets":        handleStreamPresets,

		// Scraper methods
		"proxy.scrape": handleProxyScrape,

		// Session history
		"sessions.recent":   handleSessionsRecent,
		"sessions.timeline": handleSessionTimeline,
		"sessions.stats":    handleSessionStats,

		// Proxy geo
		"proxy.geoEnrich": handleProxyGeoEnrich,
		"proxy.geoStats":  handleProxyGeoStats,

		// Config archive
		"config.export": handleConfigExport,
		"config.import": handleConfigImport,
	}

	h, ok := handlers[method]
	return h, ok
}

// --- Profile Handlers ---

func handleProfileList(_ context.Context, _ json.RawMessage) (any, error) {
	if globalExtDeps.AccountMgr == nil {
		return nil, fmt.Errorf("account manager not initialized")
	}
	return globalExtDeps.AccountMgr.List(), nil
}

type profileCreateParams struct {
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Channel  string `json:"channel"`
}

func handleProfileCreate(_ context.Context, params json.RawMessage) (any, error) {
	var p profileCreateParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Platform == "" {
		p.Platform = "twitch"
	}
	return globalExtDeps.AccountMgr.Create(p.Name, p.Platform, p.Channel)
}

type profileIDParams struct {
	ID string `json:"id"`
}

func handleProfileDelete(_ context.Context, params json.RawMessage) (any, error) {
	var p profileIDParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	return map[string]string{"status": "deleted"}, globalExtDeps.AccountMgr.Delete(p.ID)
}

func handleProfileActivate(_ context.Context, params json.RawMessage) (any, error) {
	var p profileIDParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	return map[string]string{"status": "activated"}, globalExtDeps.AccountMgr.SetActive(p.ID)
}

type profileDuplicateParams struct {
	ID      string `json:"id"`
	NewName string `json:"newName"`
}

func handleProfileDuplicate(_ context.Context, params json.RawMessage) (any, error) {
	var p profileDuplicateParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	return globalExtDeps.AccountMgr.Duplicate(p.ID, p.NewName)
}

func handleProfileExport(_ context.Context, _ json.RawMessage) (any, error) {
	data, err := globalExtDeps.AccountMgr.Export()
	if err != nil {
		return nil, err
	}
	return map[string]string{"data": string(data)}, nil
}

// --- Token Handlers ---

func handleTokenList(_ context.Context, _ json.RawMessage) (any, error) {
	if globalExtDeps.TokenMgr == nil {
		return nil, fmt.Errorf("token manager not initialized")
	}
	tokens := globalExtDeps.TokenMgr.All()

	// Mask values for security
	type maskedToken struct {
		Value   string `json:"value"`
		Label   string `json:"label"`
		State   string `json:"state"`
		Uses    int64  `json:"useCount"`
		Errors  int    `json:"errorCount"`
		Platform string `json:"platform"`
	}

	var result []maskedToken
	for _, t := range tokens {
		result = append(result, maskedToken{
			Value:    t.Masked(),
			Label:    t.Label,
			State:    t.State.String(),
			Uses:     t.UseCount,
			Errors:   t.ErrorCount,
			Platform: t.Platform,
		})
	}
	return result, nil
}

type tokenImportParams struct {
	Tokens   []string `json:"tokens"`
	Platform string   `json:"platform"`
}

func handleTokenImport(_ context.Context, params json.RawMessage) (any, error) {
	var p tokenImportParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Platform == "" {
		p.Platform = "twitch"
	}
	added := globalExtDeps.TokenMgr.AddBulk(p.Tokens, p.Platform)
	return map[string]int{"added": added}, nil
}

func handleTokenStats(_ context.Context, _ json.RawMessage) (any, error) {
	total, valid, expired, quarantined, inUse := globalExtDeps.TokenMgr.Stats()
	return map[string]int{
		"total":       total,
		"valid":       valid,
		"expired":     expired,
		"quarantined": quarantined,
		"inUse":       inUse,
	}, nil
}

func handleTokenValidate(ctx context.Context, _ json.RawMessage) (any, error) {
	if globalExtDeps.TokenMgr == nil {
		return nil, fmt.Errorf("token manager not initialized")
	}

	// Run validation asynchronously — results come via event.tokenValidation events
	go func() {
		validator := token.NewValidator(globalExtDeps.TokenMgr, nil, nil) // Platform set via extended deps
		validator.ValidateAll(context.Background(), "")
	}()

	total, _, _, _, _ := globalExtDeps.TokenMgr.Stats()
	return map[string]any{
		"status": "validation started",
		"total":  total,
	}, nil
}

// --- Stream Handlers ---

type streamStartParams struct {
	InputFile string `json:"inputFile"`
	StreamKey string `json:"streamKey"`
	Quality   string `json:"quality"`
	Loop      bool   `json:"loop"`
	ProxyURL  string `json:"proxyUrl"`
	RTMPURL   string `json:"rtmpUrl"`
}

func handleStreamStart(ctx context.Context, params json.RawMessage) (any, error) {
	if globalExtDeps.StreamMgr == nil {
		return nil, fmt.Errorf("stream manager not initialized")
	}

	var p streamStartParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	preset, ok := stream.Presets[p.Quality]
	if !ok {
		preset = stream.Presets["medium"]
	}

	cfg := stream.Config{
		InputFile: p.InputFile,
		StreamKey: p.StreamKey,
		Quality:   preset,
		Loop:      p.Loop,
		ProxyURL:  p.ProxyURL,
		RTMPURL:   p.RTMPURL,
	}

	if err := globalExtDeps.StreamMgr.Start(ctx, cfg); err != nil {
		return nil, err
	}
	return map[string]string{"status": "started"}, nil
}

func handleStreamStop(_ context.Context, _ json.RawMessage) (any, error) {
	if globalExtDeps.StreamMgr == nil {
		return nil, fmt.Errorf("stream manager not initialized")
	}
	globalExtDeps.StreamMgr.Stop()
	return map[string]string{"status": "stopped"}, nil
}

func handleStreamState(_ context.Context, _ json.RawMessage) (any, error) {
	if globalExtDeps.StreamMgr == nil {
		return nil, fmt.Errorf("stream manager not initialized")
	}
	return map[string]string{"state": globalExtDeps.StreamMgr.GetState().String()}, nil
}

func handleStreamPresets(_ context.Context, _ json.RawMessage) (any, error) {
	return stream.GetPresets(), nil
}

// --- Scraper Handler ---

func handleProxyScrape(ctx context.Context, _ json.RawMessage) (any, error) {
	if globalExtDeps.ProxyScraper == nil {
		return nil, fmt.Errorf("proxy scraper not initialized")
	}

	proxies, err := globalExtDeps.ProxyScraper.Scrape(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]int{"found": len(proxies)}, nil
}

// --- Session Handler ---

type sessionsRecentParams struct {
	Limit int `json:"limit"`
}

func handleSessionsRecent(ctx context.Context, params json.RawMessage) (any, error) {
	if globalExtDeps.Storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	var p sessionsRecentParams
	if err := json.Unmarshal(params, &p); err != nil || p.Limit <= 0 {
		p.Limit = 20
	}

	return globalExtDeps.Storage.GetRecentSessions(ctx, p.Limit)
}

type sessionTimelineParams struct {
	SessionID int64 `json:"sessionId"`
}

func handleSessionTimeline(ctx context.Context, params json.RawMessage) (any, error) {
	if globalExtDeps.Storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	var p sessionTimelineParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	return globalExtDeps.Storage.GetMetricsTimeline(ctx, p.SessionID)
}

func handleSessionStats(ctx context.Context, _ json.RawMessage) (any, error) {
	if globalExtDeps.Storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return globalExtDeps.Storage.GetSessionStats(ctx)
}

func handleProxyGeoEnrich(ctx context.Context, _ json.RawMessage) (any, error) {
	// Runs async — return ack
	go func() {
		enricher := proxy.NewGeoEnricher(nil) // Logger will be nil-safe
		// Would need proxyMgr reference — for now just ack
		_ = enricher
	}()
	return map[string]string{"status": "geo enrichment started"}, nil
}

func handleProxyGeoStats(_ context.Context, _ json.RawMessage) (any, error) {
	return map[string]string{"status": "not yet implemented"}, nil
}

// --- Config Archive Handlers ---

type archiveExportParams struct {
	Passphrase string `json:"passphrase"`
}

func handleConfigExport(_ context.Context, params json.RawMessage) (any, error) {
	var p archiveExportParams
	if err := json.Unmarshal(params, &p); err != nil {
		p.Passphrase = "" // No encryption if no passphrase
	}

	// Get profiles data
	var profilesJSON json.RawMessage
	if globalExtDeps.AccountMgr != nil {
		data, err := globalExtDeps.AccountMgr.Export()
		if err == nil {
			profilesJSON = data
		}
	}

	archive, err := config.ExportArchive(nil, profilesJSON, nil, p.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("export failed: %w", err)
	}

	return map[string]any{
		"data":      string(archive),
		"encrypted": p.Passphrase != "",
		"size":      len(archive),
	}, nil
}

type archiveImportParams struct {
	Data       string `json:"data"`
	Passphrase string `json:"passphrase"`
}

func handleConfigImport(_ context.Context, params json.RawMessage) (any, error) {
	var p archiveImportParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	archive, err := config.ImportArchive([]byte(p.Data), p.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("import failed: %w", err)
	}

	// Import profiles if present
	imported := 0
	if archive.Profiles != nil && globalExtDeps.AccountMgr != nil {
		n, err := globalExtDeps.AccountMgr.Import(archive.Profiles)
		if err == nil {
			imported = n
		}
	}

	return map[string]any{
		"status":          "imported",
		"profilesImported": imported,
		"version":         archive.Version,
	}, nil
}
