// Package proxy - concurrent proxy health checking with latency measurement.
package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// CheckResult holds the outcome of a single proxy health check.
type CheckResult struct {
	Proxy   *Proxy
	Health  HealthStatus
	Latency time.Duration
	IP      string
	Error   error
}

// Checker performs concurrent proxy health validation.
type Checker struct {
	logger      *slog.Logger
	concurrency int
	timeout     time.Duration
	testURL     string
}

// NewChecker creates a proxy health checker.
func NewChecker(logger *slog.Logger) *Checker {
	return &Checker{
		logger:      logger.With("component", "proxy-checker"),
		concurrency: 20,
		timeout:     15 * time.Second,
		testURL:     "https://api.ipify.org",
	}
}

// CheckAll validates all proxies in the manager concurrently.
// Returns results via the callback as they complete.
func (c *Checker) CheckAll(ctx context.Context, mgr *Manager, onResult func(CheckResult)) {
	proxies := mgr.All()
	c.logger.Info("starting proxy health check", "count", len(proxies))

	var wg sync.WaitGroup
	sem := make(chan struct{}, c.concurrency)

	for _, p := range proxies {
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(proxy *Proxy) {
			defer func() {
				<-sem
				wg.Done()
			}()

			result := c.checkOne(ctx, proxy)
			mgr.UpdateHealth(proxy, result.Health, result.Latency)

			if onResult != nil {
				onResult(result)
			}
		}(p)
	}

	wg.Wait()
	c.logger.Info("proxy health check complete")
}

// checkOne tests a single proxy by making a request through it.
func (c *Checker) checkOne(ctx context.Context, p *Proxy) CheckResult {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	transport := &http.Transport{
		Proxy: http.ProxyURL(p.URLParsed()),
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   c.timeout,
	}

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", c.testURL, nil)
	if err != nil {
		return CheckResult{Proxy: p, Health: HealthDead, Error: err}
	}

	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			Proxy:   p,
			Health:  HealthDead,
			Latency: latency,
			Error:   fmt.Errorf("connection failed: %w", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return CheckResult{
			Proxy:   p,
			Health:  HealthDead,
			Latency: latency,
			Error:   fmt.Errorf("status %d", resp.StatusCode),
		}
	}

	// Read IP for verification
	buf := make([]byte, 64)
	n, _ := resp.Body.Read(buf)
	ip := string(buf[:n])

	// Classify health based on latency
	health := HealthGood
	if latency > 5*time.Second {
		health = HealthSlow
	}

	return CheckResult{
		Proxy:   p,
		Health:  health,
		Latency: latency,
		IP:      ip,
	}
}

// CheckSingle validates a single proxy URL string.
func (c *Checker) CheckSingle(ctx context.Context, proxyURL string) (*CheckResult, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	p := &Proxy{
		Host: u.Hostname(),
		Port: u.Port(),
	}
	if u.User != nil {
		p.Username = u.User.Username()
		p.Password, _ = u.User.Password()
	}

	result := c.checkOne(ctx, p)
	return &result, nil
}
