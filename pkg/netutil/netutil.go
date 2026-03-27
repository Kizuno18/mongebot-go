// Package netutil provides reusable network utilities: retry with exponential
// backoff, jitter, IP geolocation, and connection helpers.
package netutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"time"
)

// RetryConfig controls retry behavior with exponential backoff.
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Jitter      bool // Add random jitter to prevent thundering herd
}

// DefaultRetry returns a sensible default retry config.
func DefaultRetry() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Jitter:      true,
	}
}

// Retry executes fn up to MaxAttempts times with exponential backoff.
// Returns the first successful result or the last error.
func Retry[T any](ctx context.Context, cfg RetryConfig, fn func(ctx context.Context, attempt int) (T, error)) (T, error) {
	var lastErr error
	var zero T

	for attempt := range cfg.MaxAttempts {
		result, err := fn(ctx, attempt)
		if err == nil {
			return result, nil
		}
		lastErr = err

		// Don't sleep after the last attempt
		if attempt >= cfg.MaxAttempts-1 {
			break
		}

		delay := backoffDelay(attempt, cfg.BaseDelay, cfg.MaxDelay, cfg.Jitter)

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
	}

	return zero, fmt.Errorf("after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// RetryVoid is like Retry but for functions that don't return a value.
func RetryVoid(ctx context.Context, cfg RetryConfig, fn func(ctx context.Context, attempt int) error) error {
	_, err := Retry(ctx, cfg, func(ctx context.Context, attempt int) (struct{}, error) {
		return struct{}{}, fn(ctx, attempt)
	})
	return err
}

// backoffDelay calculates exponential backoff with optional jitter.
func backoffDelay(attempt int, base, max time.Duration, jitter bool) time.Duration {
	delay := time.Duration(float64(base) * math.Pow(2, float64(attempt)))
	if delay > max {
		delay = max
	}
	if jitter {
		// Add up to 25% random jitter
		jitterAmount := time.Duration(rand.Int64N(int64(delay) / 4))
		delay += jitterAmount
	}
	return delay
}

// IPInfo holds geolocation data for an IP address.
type IPInfo struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	Region  string `json:"regionName"`
	City    string `json:"city"`
	ISP     string `json:"isp"`
	Org     string `json:"org"`
}

// GetPublicIP returns the public IP of the current connection (or through a proxy).
func GetPublicIP(ctx context.Context, client *http.Client) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org", nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// GetIPInfo returns geolocation data for the given IP address.
func GetIPInfo(ctx context.Context, ip string) (*IPInfo, error) {
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=ip,country,regionName,city,isp,org", ip)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// RandomDuration returns a random duration between min and max.
func RandomDuration(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}
	return min + time.Duration(rand.Int64N(int64(max-min)))
}

// IsTemporaryError checks if an HTTP status code indicates a temporary/retryable error.
func IsTemporaryError(statusCode int) bool {
	switch statusCode {
	case 408, 425, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}
