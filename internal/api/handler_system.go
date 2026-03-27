// Package api - system-level IPC handlers for health, version, info, and diagnostics.
package api

import (
	"context"
	"encoding/json"
	"runtime"
	"time"
)

var startTime = time.Now()

// getSystemHandler returns handlers for system.* methods.
func getSystemHandler(method string) (handlerFunc, bool) {
	handlers := map[string]handlerFunc{
		"system.health":  handleSystemHealth,
		"system.version": handleSystemVersion,
		"system.info":    handleSystemInfo,
		"system.uptime":  handleSystemUptime,
		"system.gc":      handleSystemGC,
	}
	h, ok := handlers[method]
	return h, ok
}

func handleSystemHealth(_ context.Context, _ json.RawMessage) (any, error) {
	return map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(startTime).String(),
	}, nil
}

func handleSystemVersion(_ context.Context, _ json.RawMessage) (any, error) {
	return map[string]any{
		"version":   "2.0.0",
		"goVersion": runtime.Version(),
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"compiler":  runtime.Compiler,
	}, nil
}

func handleSystemInfo(_ context.Context, _ json.RawMessage) (any, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]any{
		"version":     "2.0.0",
		"goVersion":   runtime.Version(),
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"cpus":        runtime.NumCPU(),
		"goroutines":  runtime.NumGoroutine(),
		"uptime":      time.Since(startTime).String(),
		"uptimeNs":    time.Since(startTime).Nanoseconds(),
		"memory": map[string]any{
			"allocMB":    memStats.Alloc / 1024 / 1024,
			"totalMB":    memStats.TotalAlloc / 1024 / 1024,
			"sysMB":      memStats.Sys / 1024 / 1024,
			"numGC":      memStats.NumGC,
			"gcPauseMs":  memStats.PauseTotalNs / 1e6,
		},
		"platforms": []string{"twitch", "kick", "youtube"},
	}, nil
}

func handleSystemUptime(_ context.Context, _ json.RawMessage) (any, error) {
	return map[string]any{
		"uptime":   time.Since(startTime).String(),
		"uptimeNs": time.Since(startTime).Nanoseconds(),
		"started":  startTime.Format(time.RFC3339),
	}, nil
}

func handleSystemGC(_ context.Context, _ json.RawMessage) (any, error) {
	runtime.GC()
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return map[string]any{
		"status":  "gc triggered",
		"allocMB": memStats.Alloc / 1024 / 1024,
		"numGC":   memStats.NumGC,
	}, nil
}
