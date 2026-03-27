// Package main provides a headless CLI with an interactive TUI dashboard.
// Uses charmbracelet/bubbletea v2 for terminal UI rendering.
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

	"github.com/Kizuno18/mongebot-go/internal/config"
	"github.com/Kizuno18/mongebot-go/internal/engine"
	logpkg "github.com/Kizuno18/mongebot-go/internal/logger"
	"github.com/Kizuno18/mongebot-go/internal/platform"
	"github.com/Kizuno18/mongebot-go/internal/platform/twitch"
	"github.com/Kizuno18/mongebot-go/internal/proxy"
	"github.com/Kizuno18/mongebot-go/pkg/useragent"
)

func main() {
	channel := flag.String("channel", "", "Target channel name (required)")
	workers := flag.Int("workers", 0, "Number of workers (0 = use config default)")
	configPath := flag.String("config", "data/config.json", "Config file path")
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	flag.Parse()

	if *channel == "" {
		fmt.Fprintln(os.Stderr, "Usage: mongebot-cli --channel <name> [--workers N]")
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}
	if *workers > 0 {
		cfg.Engine.MaxWorkers = *workers
	}

	// Logger (file only for CLI — TUI owns stdout)
	logger, _ := logpkg.Setup(*logLevel, cfg.Logging.File)

	// Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Setup platform
	registry := platform.NewRegistry()
	registry.Register(twitch.NewProvider(logger))

	activePlatform, _ := registry.Get("twitch")

	// Setup resources
	proxyMgr := proxy.NewManager(proxy.RotationRandom)
	proxyMgr.LoadFromFile("data/proxies.txt")

	uaPool := useragent.NewPool()
	uaPool.LoadFromFile("data/user-agents.txt")

	tokens := loadTokens("data/tokens.txt")

	// Create engine
	eng := engine.New(activePlatform, proxyMgr, tokens, uaPool, cfg.GetEngine(), logger)

	// Print TUI header
	printHeader(*channel, cfg.Engine.MaxWorkers)

	// Start engine
	if err := eng.Start(ctx, *channel, cfg.Engine.MaxWorkers); err != nil {
		fmt.Fprintf(os.Stderr, "Engine start failed: %v\n", err)
		os.Exit(1)
	}

	// Simple TUI refresh loop (non-bubbletea fallback for environments without bubbletea)
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m := eng.Metrics()
				printMetrics(m)
			}
		}
	}()

	// Wait for signal
	sig := <-sigCh
	fmt.Printf("\n\033[33m⚡ Received %s, shutting down...\033[0m\n", sig)
	eng.Stop()
	cancel()
	fmt.Println("\033[32m✓ Shutdown complete.\033[0m")
}

func printHeader(channel string, workers int) {
	fmt.Println("\033[1;36m")
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║       MongeBot CLI v2.0 (Go)         ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Printf("  Channel: \033[1;33m%s\033[1;36m | Workers: \033[1;33m%d\033[0m\n\n", channel, workers)
}

func printMetrics(m *engine.AggregatedMetrics) {
	// Move cursor up and overwrite
	fmt.Print("\033[4A\033[K") // Clear 4 lines
	fmt.Printf("  \033[1;32m● Active Viewers: %d/%d\033[0m", m.ActiveViewers, m.TotalWorkers)
	fmt.Printf("  |  Segments: %d", m.SegmentsFetched)
	fmt.Printf("  |  Heartbeats: %d", m.HeartbeatsSent)
	fmt.Printf("  |  Ads: %d\n", m.AdsWatched)

	fmt.Printf("  Data: %s", formatBytes(m.BytesReceived))
	fmt.Printf("  |  Uptime: %s", formatDuration(m.Uptime))
	fmt.Printf("  |  State: %s\n", m.EngineState)

	// Worker bar
	bar := renderBar(m.ActiveViewers, m.TotalWorkers, 40)
	fmt.Printf("  Workers: %s\n", bar)
	fmt.Println("  \033[90mPress Ctrl+C to stop.\033[0m")
}

func renderBar(active, total, width int) string {
	if total == 0 {
		return strings.Repeat("░", width)
	}
	filled := (active * width) / total
	if filled > width {
		filled = width
	}
	return "\033[32m" + strings.Repeat("█", filled) + "\033[90m" + strings.Repeat("░", width-filled) + "\033[0m"
}

func formatBytes(b int64) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%d B", b)
	case b < 1048576:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	case b < 1073741824:
		return fmt.Sprintf("%.1f MB", float64(b)/1048576)
	default:
		return fmt.Sprintf("%.2f GB", float64(b)/1073741824)
	}
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func loadTokens(path string) []string {
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
