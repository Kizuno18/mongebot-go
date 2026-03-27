// Package api - Prometheus metrics endpoint for monitoring integration.
// Exposes engine, proxy, token, and system metrics in Prometheus format.
package api

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// PrometheusHandler returns an HTTP handler that serves metrics in Prometheus text format.
func PrometheusHandler(srv *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		var sb strings.Builder

		// Engine metrics
		if srv.engine != nil {
			m := srv.engine.Metrics()
			writeGauge(&sb, "mongebot_active_viewers", "Number of active viewer connections", float64(m.ActiveViewers))
			writeGauge(&sb, "mongebot_total_workers", "Total number of workers", float64(m.TotalWorkers))
			writeCounter(&sb, "mongebot_segments_fetched_total", "Total HLS segments fetched", float64(m.SegmentsFetched))
			writeCounter(&sb, "mongebot_bytes_received_total", "Total bytes received from streams", float64(m.BytesReceived))
			writeCounter(&sb, "mongebot_heartbeats_sent_total", "Total heartbeats sent", float64(m.HeartbeatsSent))
			writeCounter(&sb, "mongebot_ads_watched_total", "Total ads watched", float64(m.AdsWatched))
			writeGauge(&sb, "mongebot_uptime_seconds", "Engine uptime in seconds", m.Uptime.Seconds())

			// Engine state as labeled metric
			states := []string{"stopped", "starting", "running", "paused", "stopping"}
			for _, state := range states {
				val := 0.0
				if m.EngineState == state {
					val = 1.0
				}
				fmt.Fprintf(&sb, "mongebot_engine_state{state=\"%s\"} %g\n", state, val)
			}
		}

		// Proxy metrics
		if srv.proxyMgr != nil {
			total, available, inUse := srv.proxyMgr.Count()
			writeGauge(&sb, "mongebot_proxies_total", "Total proxies in pool", float64(total))
			writeGauge(&sb, "mongebot_proxies_available", "Available proxies", float64(available))
			writeGauge(&sb, "mongebot_proxies_in_use", "Proxies currently in use", float64(inUse))
		}

		// Token metrics
		if globalExtDeps != nil && globalExtDeps.TokenMgr != nil {
			total, valid, expired, quarantined, inUse := globalExtDeps.TokenMgr.Stats()
			writeGauge(&sb, "mongebot_tokens_total", "Total tokens", float64(total))
			writeGauge(&sb, "mongebot_tokens_valid", "Valid tokens", float64(valid))
			writeGauge(&sb, "mongebot_tokens_expired", "Expired tokens", float64(expired))
			writeGauge(&sb, "mongebot_tokens_quarantined", "Quarantined tokens", float64(quarantined))
			writeGauge(&sb, "mongebot_tokens_in_use", "Tokens in use", float64(inUse))
		}

		// System metrics
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		writeGauge(&sb, "mongebot_go_goroutines", "Number of goroutines", float64(runtime.NumGoroutine()))
		writeGauge(&sb, "mongebot_go_memory_alloc_bytes", "Current memory allocation", float64(memStats.Alloc))
		writeGauge(&sb, "mongebot_go_memory_sys_bytes", "Total memory from OS", float64(memStats.Sys))
		writeCounter(&sb, "mongebot_go_gc_runs_total", "Total GC cycles", float64(memStats.NumGC))
		writeGauge(&sb, "mongebot_process_uptime_seconds", "Process uptime", time.Since(startTime).Seconds())

		// Build info
		fmt.Fprintf(&sb, "mongebot_build_info{version=\"2.0.0\",go_version=\"%s\",os=\"%s\",arch=\"%s\"} 1\n",
			runtime.Version(), runtime.GOOS, runtime.GOARCH)

		w.Write([]byte(sb.String()))
	}
}

func writeGauge(sb *strings.Builder, name, help string, value float64) {
	fmt.Fprintf(sb, "# HELP %s %s\n# TYPE %s gauge\n%s %g\n", name, help, name, name, value)
}

func writeCounter(sb *strings.Builder, name, help string, value float64) {
	fmt.Fprintf(sb, "# HELP %s %s\n# TYPE %s counter\n%s %g\n", name, help, name, name, value)
}
