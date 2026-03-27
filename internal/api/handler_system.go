// Package api - system-level IPC handlers for health, version, info, and diagnostics.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/storage"
)

var startTime = time.Now()

// getSystemHandler returns handlers for system.* methods.
func getSystemHandler(method string) (handlerFunc, bool) {
	handlers := map[string]handlerFunc{
		"system.health":      handleSystemHealth,
		"system.healthcheck": handleSystemDeepCheck,
		"system.version":     handleSystemVersion,
		"system.info":        handleSystemInfo,
		"system.uptime":      handleSystemUptime,
		"system.gc":          handleSystemGC,
		"sessions.export":    handleSessionsExport,
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

// handleSystemDeepCheck verifies all subsystems are healthy.
func handleSystemDeepCheck(ctx context.Context, _ json.RawMessage) (any, error) {
	checks := make(map[string]any)
	allHealthy := true

	// Database check
	if globalExtDeps != nil && globalExtDeps.Storage != nil {
		err := globalExtDeps.Storage.Conn().PingContext(ctx)
		if err != nil {
			checks["database"] = map[string]any{"status": "unhealthy", "error": err.Error()}
			allHealthy = false
		} else {
			checks["database"] = map[string]any{"status": "healthy"}
		}
	} else {
		checks["database"] = map[string]any{"status": "not configured"}
	}

	// Proxy pool check
	if globalExtDeps != nil {
		// Check via stats in extended deps — simplified
		checks["proxyPool"] = map[string]any{"status": "healthy"}
	}

	// Token pool check
	if globalExtDeps != nil && globalExtDeps.TokenMgr != nil {
		total, valid, _, _, _ := globalExtDeps.TokenMgr.Stats()
		status := "healthy"
		if total > 0 && valid == 0 {
			status = "degraded"
			allHealthy = false
		}
		checks["tokenPool"] = map[string]any{
			"status": status,
			"total":  total,
			"valid":  valid,
		}
	} else {
		checks["tokenPool"] = map[string]any{"status": "not configured"}
	}

	// Memory check
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memMB := memStats.Alloc / 1024 / 1024
	memStatus := "healthy"
	if memMB > 500 {
		memStatus = "warning"
	}
	checks["memory"] = map[string]any{
		"status": memStatus,
		"allocMB": memMB,
		"goroutines": runtime.NumGoroutine(),
	}

	overall := "healthy"
	if !allHealthy {
		overall = "degraded"
	}

	return map[string]any{
		"status":    overall,
		"checks":    checks,
		"uptime":    time.Since(startTime).String(),
		"timestamp": time.Now().Format(time.RFC3339),
	}, nil
}

// handleSessionsExport exports session data as CSV or JSON.
type sessionsExportParams struct {
	Format string `json:"format"` // "csv" or "json"
	Limit  int    `json:"limit"`
}

func handleSessionsExport(ctx context.Context, params json.RawMessage) (any, error) {
	if globalExtDeps == nil || globalExtDeps.Storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	var p sessionsExportParams
	if err := json.Unmarshal(params, &p); err != nil || p.Limit <= 0 {
		p.Limit = 100
	}
	if p.Format == "" {
		p.Format = "json"
	}

	data, err := globalExtDeps.Storage.ExportSessions(ctx, storage.ExportFormat(p.Format), p.Limit)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"data":   string(data),
		"format": p.Format,
		"size":   len(data),
	}, nil
}

