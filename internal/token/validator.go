// Package token - concurrent batch token validation with throttling and progress reporting.
package token

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
	"github.com/Kizuno18/mongebot-go/internal/proxy"
)

// ValidationResult holds the outcome of validating a single token.
type ValidationResult struct {
	Token  *ManagedToken       `json:"token"`
	Status platform.TokenStatus `json:"status"`
	Error  error               `json:"error,omitempty"`
}

// ValidationProgress reports batch validation progress.
type ValidationProgress struct {
	Total     int `json:"total"`
	Checked   int `json:"checked"`
	Valid     int `json:"valid"`
	Invalid   int `json:"invalid"`
	Errors    int `json:"errors"`
	Remaining int `json:"remaining"`
}

// Validator performs concurrent token validation against a platform API.
type Validator struct {
	manager     *Manager
	platform    platform.Platform
	logger      *slog.Logger
	concurrency int
	throttle    time.Duration
	onProgress  func(ValidationProgress)
	onResult    func(ValidationResult)
}

// NewValidator creates a token validator.
func NewValidator(mgr *Manager, p platform.Platform, logger *slog.Logger) *Validator {
	return &Validator{
		manager:     mgr,
		platform:    p,
		logger:      logger.With("component", "token-validator"),
		concurrency: 5,
		throttle:    300 * time.Millisecond,
	}
}

// SetConcurrency sets the maximum concurrent validation requests.
func (v *Validator) SetConcurrency(n int) {
	if n > 0 && n <= 20 {
		v.concurrency = n
	}
}

// OnProgress sets a callback for progress updates.
func (v *Validator) OnProgress(fn func(ValidationProgress)) {
	v.onProgress = fn
}

// OnResult sets a callback for individual token results.
func (v *Validator) OnResult(fn func(ValidationResult)) {
	v.onResult = fn
}

// ValidateAll checks all tokens in the manager concurrently.
func (v *Validator) ValidateAll(ctx context.Context, proxyMgr *proxy.Manager) ValidationProgress {
	tokens := v.manager.All()
	total := len(tokens)

	v.logger.Info("starting batch token validation", "total", total, "concurrency", v.concurrency)

	var (
		wg      sync.WaitGroup
		sem     = make(chan struct{}, v.concurrency)
		valid   atomic.Int32
		invalid atomic.Int32
		errs    atomic.Int32
		checked atomic.Int32
	)

	for _, tok := range tokens {
		select {
		case <-ctx.Done():
			break
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(t *ManagedToken) {
			defer func() {
				<-sem
				wg.Done()
			}()

			var pURL string
			var p *proxy.Proxy
			if proxyMgr != nil {
				p = proxyMgr.Acquire()
				if p != nil {
					pURL = p.URL()
					defer proxyMgr.Release(p)
				}
			}

			result := v.validateOne(ctx, t, pURL)

			switch result.Status {
			case platform.TokenValid:
				valid.Add(1)
			case platform.TokenExpired, platform.TokenInvalid:
				invalid.Add(1)
			}
			if result.Error != nil {
				errs.Add(1)
			}

			n := int(checked.Add(1))

			if v.onResult != nil {
				v.onResult(result)
			}
			if v.onProgress != nil {
				v.onProgress(ValidationProgress{
					Total:     total,
					Checked:   n,
					Valid:     int(valid.Load()),
					Invalid:   int(invalid.Load()),
					Errors:    int(errs.Load()),
					Remaining: total - n,
				})
			}

			// Throttle to avoid rate limiting
			time.Sleep(v.throttle)
		}(tok)
	}

	wg.Wait()

	finalProgress := ValidationProgress{
		Total:   total,
		Checked: total,
		Valid:   int(valid.Load()),
		Invalid: int(invalid.Load()),
		Errors:  int(errs.Load()),
	}

	v.logger.Info("batch validation complete",
		"valid", finalProgress.Valid,
		"invalid", finalProgress.Invalid,
		"errors", finalProgress.Errors,
	)

	return finalProgress
}

// validateOne checks a single token and updates its state.
func (v *Validator) validateOne(ctx context.Context, tok *ManagedToken, proxyURL string) ValidationResult {
	status, err := v.platform.ValidateToken(ctx, tok.Value, proxyURL)
	tok.LastChecked = time.Now()

	v.manager.mu.Lock()
	switch status {
	case platform.TokenValid:
		tok.State = StateValid
		tok.ErrorCount = 0
	case platform.TokenExpired:
		tok.State = StateExpired
	case platform.TokenInvalid:
		tok.State = StateQuarantined
	case platform.TokenRateLimited:
		// Don't change state on rate limit — try again later
	}
	v.manager.mu.Unlock()

	return ValidationResult{
		Token:  tok,
		Status: status,
		Error:  err,
	}
}

// ValidateSingle checks a single token value without adding it to the pool.
func (v *Validator) ValidateSingle(ctx context.Context, tokenValue string, proxyURL string) (platform.TokenStatus, error) {
	return v.platform.ValidateToken(ctx, tokenValue, proxyURL)
}
