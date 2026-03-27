// Package api - HTTP middleware for rate limiting, CORS, request logging, and validation.
package api

import (
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter per client IP.
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*bucket
	rate     int           // Tokens per interval
	interval time.Duration // Refill interval
	burst    int           // Max burst size
}

type bucket struct {
	tokens    int
	lastCheck time.Time
}

// NewRateLimiter creates a rate limiter.
// rate = tokens/interval, burst = max tokens that can accumulate.
func NewRateLimiter(rate int, interval time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*bucket),
		rate:     rate,
		interval: interval,
		burst:    burst,
	}
	// Cleanup stale entries every minute
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
	return rl
}

// Allow checks if a request from the given key should be allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.clients[key]
	if !exists {
		rl.clients[key] = &bucket{tokens: rl.burst - 1, lastCheck: time.Now()}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := time.Since(b.lastCheck)
	refill := int(elapsed / rl.interval) * rl.rate
	b.tokens = min(b.tokens+refill, rl.burst)
	b.lastCheck = time.Now()

	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}

// cleanup removes stale entries older than 5 minutes.
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-5 * time.Minute)
	for key, b := range rl.clients {
		if b.lastCheck.Before(cutoff) {
			delete(rl.clients, key)
		}
	}
}

// CORSMiddleware adds CORS headers for Tauri localhost access.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow Tauri and localhost origins
		allowedOrigins := map[string]bool{
			"tauri://localhost":    true,
			"http://localhost":     true,
			"https://tauri.localhost": true,
			"http://localhost:1420": true,
		}

		if allowedOrigins[origin] || origin == "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs HTTP requests with timing.
func LoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &statusWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(wrapped, r)

		logger.Debug("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", time.Since(start).String(),
			"remote", r.RemoteAddr,
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// RateLimitMiddleware applies rate limiting per remote address.
func RateLimitMiddleware(rl *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(r.RemoteAddr) {
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders adds security headers to responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

// Chain applies multiple middleware in order.
func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
