// Package main is the entry point for MongeBot.
// Supports two modes:
//   - sidecar: runs as Tauri sidecar with API server (default)
//   - headless: runs without UI, controlled via API or CLI flags
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/account"
	"github.com/Kizuno18/mongebot-go/internal/api"
	"github.com/Kizuno18/mongebot-go/internal/config"
	"github.com/Kizuno18/mongebot-go/internal/engine"
	logpkg "github.com/Kizuno18/mongebot-go/internal/logger"
	"github.com/Kizuno18/mongebot-go/internal/platform"
	"github.com/Kizuno18/mongebot-go/internal/platform/kick"
	"github.com/Kizuno18/mongebot-go/internal/platform/twitch"
	"github.com/Kizuno18/mongebot-go/internal/platform/youtube"
	"github.com/Kizuno18/mongebot-go/internal/proxy"
	"github.com/Kizuno18/mongebot-go/internal/storage"
	"github.com/Kizuno18/mongebot-go/internal/stream"
	"github.com/Kizuno18/mongebot-go/internal/token"
	"github.com/Kizuno18/mongebot-go/pkg/useragent"
)

const version = "2.0.0"

func main() {
	// Parse flags
	mode := flag.String("mode", "sidecar", "Run mode: sidecar (with API) or headless")
	port := flag.Int("port", 0, "API server port (overrides config)")
	configPath := flag.String("config", "data/config.json", "Path to config file")
	channel := flag.String("channel", "", "Target channel (headless mode)")
	workers := flag.Int("workers", 0, "Number of workers (headless mode)")
	dbPath := flag.String("db", "data/mongebot.db", "SQLite database path")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("MongeBot v%s (Go 1.26)\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	if *port > 0 {
		cfg.API.Port = *port
	}

	// Setup logging
	logger, err := logpkg.Setup(cfg.Logging.Level, cfg.Logging.File)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}
	logRing := logpkg.NewRingBuffer(cfg.Logging.RingBuffer)

	logger.Info("MongeBot starting", "mode", *mode, "version", version)

	// Setup context with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// --- Initialize all modules ---

	// SQLite storage
	db, err := storage.Open(*dbPath, logger)
	if err != nil {
		logger.Warn("SQLite initialization failed (continuing without persistence)", "error", err)
	} else {
		defer db.Close()
		logger.Info("SQLite database initialized", "path", *dbPath)
	}

	// Platform registry
	registry := platform.NewRegistry()
	registry.Register(twitch.NewProvider(logger))
	registry.Register(kick.NewProvider(logger))
	registry.Register(youtube.NewProvider(logger))
	logger.Info("platforms registered", "platforms", registry.List())

	// Proxy manager
	proxyMgr := proxy.NewManager(proxy.RotationRandom)
	if err := proxyMgr.LoadFromFile("data/proxies.txt"); err != nil {
		logger.Warn("failed to load proxies", "error", err)
	}
	total, _, _ := proxyMgr.Count()
	logger.Info("proxies loaded", "count", total)



	// User-agent pool with auto-updater
	uaPool := useragent.NewPool()
	uaPool.LoadFromFile("data/user-agents.txt")
	logger.Info("user agents loaded", "count", uaPool.Count())

	// Start UA auto-updater in background (refreshes every 24h)
	uaUpdater := useragent.NewUpdater(uaPool, logger)
	go uaUpdater.AutoUpdate(ctx, 24*time.Hour)

	// Token manager — load from plain text
	tokenMgr := token.NewManager(logger)
	rawTokens := loadTokensFromFile("data/tokens.txt")
	if len(rawTokens) > 0 {
		tokenMgr.AddBulk(rawTokens, "twitch")
	}
	tTotal, tValid, _, _, _ := tokenMgr.Stats()
	logger.Info("tokens loaded", "total", tTotal, "valid", tValid)

	// Account manager - try SQLite first, fallback to JSON
	var accountMgr *account.Manager
	if db != nil {
		repo := account.NewSQLiteRepo(db)
		accountMgr, err = account.NewManagerWithRepo(repo, logger)
		if err != nil {
			logger.Warn("account manager with SQLite failed, trying JSON", "error", err)
		}
	}
	if accountMgr == nil {
		accountMgr, err = account.NewManager("data/profiles.json", logger)
		if err != nil {
			logger.Warn("account manager initialization failed", "error", err)
		}
	}

	// Stream manager (FFmpeg)
	streamMgr := stream.NewManager(logger)

	// Get active platform (default: twitch)
	platformName := "twitch"
	if accountMgr != nil {
		if active := accountMgr.GetActive(); active != nil {
			platformName = active.Platform
		}
	}
	activePlatform, err := registry.Get(platformName)
	if err != nil {
		logger.Error("platform not found", "platform", platformName, "error", err)
		os.Exit(1)
	}

	// Load .env file (optional, doesn't override existing env vars)
	config.LoadDotEnv(".env")
	config.ApplyEnvOverrides(cfg, logger)

	// Create single-channel engine
	eng := engine.New(activePlatform, proxyMgr, tokenMgr.GetValidValues(), uaPool, cfg.GetEngine(), logger)

	// Create multi-channel engine
	multiEng := engine.NewMultiEngine(activePlatform, proxyMgr, tokenMgr.GetValidValues(), uaPool, cfg.GetEngine(), logger)

	// Create scheduler and load enabled rules from config
	scheduler := engine.NewScheduler(multiEng, activePlatform, logger)
	schedCfg := cfg.GetSchedulerConfig()
	if schedCfg.Enabled {
		for _, rule := range schedCfg.Rules {
			if rule.Enabled {
				scheduler.AddRule(engine.ScheduleRule{
					ID:       rule.ID,
					Name:     rule.Name,
					Channel:  rule.Channel,
					Platform: rule.Platform,
					Trigger:  engine.TriggerType(rule.Trigger),
					Workers:  rule.Workers,
					Enabled:  rule.Enabled,
				})
			}
		}
	}
	scheduler.Start(ctx)

	// Setup stream monitor with event broadcasting
	monitor := engine.NewStreamMonitor(activePlatform, logger, 30*time.Second)
	monitor.OnEvent(func(event engine.StreamEvent) {
		logger.Info("stream event",
			"channel", event.Channel,
			"status", event.Status.String(),
		)
	})
	_ = monitor // Available for channel watching via scheduler

	// Setup webhook manager for Discord/Telegram/HTTP notifications
	webhookMgr := engine.NewWebhookManager(logger)
	webhookMgr.SetFilePath("data/webhooks.json")
	if err := webhookMgr.Load(); err != nil {
		logger.Warn("failed to load webhooks", "error", err)
	}
	api.SetWebhookManager(webhookMgr)

	// Wire stream monitor to send webhook notifications
	monitor.OnEvent(func(event engine.StreamEvent) {
		eventType := "stream.offline"
		title := fmt.Sprintf("%s went offline", event.Channel)
		if event.Status.String() == "online" {
			eventType = "stream.online"
			title = fmt.Sprintf("%s is LIVE!", event.Channel)
		}
		webhookMgr.Notify(ctx, eventType, title, "", map[string]string{
			"Channel":  event.Channel,
			"Platform": event.Platform,
		})
	})

	// Setup metrics persistence (saves snapshots to SQLite every 30s)
	var persister *engine.MetricsPersister
	if db != nil {
		persister = engine.NewMetricsPersister(db, eng, logger, 30*time.Second)
		// Auto-start session tracking for the default profile
		if profile := cfg.GetActiveProfile(); profile != nil {
			persister.StartSession(ctx, profile.ID, profile.Channel, profile.Platform)
		}
	}

	// Set extended API deps
	api.SetExtendedDeps(&api.ExtendedDeps{
		AccountMgr:   accountMgr,
		TokenMgr:     tokenMgr,
		StreamMgr:    streamMgr,

		ProxyMgr:     proxyMgr,
		Platform:     activePlatform,
		Logger:       logger,
		Storage:      db,
	})

	// Set scheduler API deps
	api.SetSchedulerDeps(&api.SchedulerDeps{
		MultiEngine: multiEng,
		Scheduler:   scheduler,
	})

	switch *mode {
	case "headless":
		runHeadless(ctx, eng, persister, cfg, logger, *channel, *workers, sigCh, cancel)
	default:
		runSidecar(ctx, eng, persister, proxyMgr, cfg, logRing, logger, sigCh, cancel)
	}
}

// runSidecar starts the API server for Tauri frontend communication.
func runSidecar(ctx context.Context, eng *engine.Engine, persister *engine.MetricsPersister, proxyMgr *proxy.Manager, cfg *config.AppConfig, logRing *logpkg.RingBuffer, logger *slog.Logger, sigCh chan os.Signal, cancel context.CancelFunc) {
	srv := api.NewServer(cfg.API, eng, proxyMgr, cfg, logRing, logger)

	go func() {
		if err := srv.Start(ctx); err != nil {
			logger.Error("API server error", "error", err)
			cancel()
		}
	}()

	logger.Info("sidecar mode: API server listening",
		"addr", fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
	)

	select {
	case sig := <-sigCh:
		logger.Info("received signal, shutting down", "signal", sig)
	case <-ctx.Done():
	}

	eng.Stop()
	// End metrics session on shutdown
	if persister != nil {
		persister.EndSession(ctx, "shutdown")
	}
	logger.Info("shutdown complete")
}

// runHeadless starts the engine directly without UI.
func runHeadless(ctx context.Context, eng *engine.Engine, persister *engine.MetricsPersister, cfg *config.AppConfig, logger *slog.Logger, channel string, workers int, sigCh chan os.Signal, cancel context.CancelFunc) {
	if channel == "" {
		if p := cfg.GetActiveProfile(); p != nil {
			channel = p.Channel
		}
		if channel == "" {
			logger.Error("no channel specified. Use --channel flag or set an active profile")
			os.Exit(1)
		}
	}
	if workers <= 0 {
		workers = cfg.GetEngine().MaxWorkers
	}

	logger.Info("headless mode: starting engine", "channel", channel, "workers", workers)

	if err := eng.Start(ctx, channel, workers); err != nil {
		logger.Error("engine start failed", "error", err)
		os.Exit(1)
	}

	select {
	case sig := <-sigCh:
		logger.Info("received signal, shutting down", "signal", sig)
	case <-ctx.Done():
	}

	eng.Stop()
	// End metrics session on shutdown
	if persister != nil {
		persister.EndSession(ctx, "shutdown")
	}
	cancel()
	logger.Info("shutdown complete")
}

func loadTokensFromFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var tokens []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			tokens = append(tokens, line)
		}
	}
	return tokens
}


