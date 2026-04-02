// Package token manages authentication token pools with validation,
// rotation, and quarantine of expired tokens.
package token

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/platform"
)

// TokenState represents the lifecycle state of a token.
type TokenState int

const (
	StateValid TokenState = iota
	StateExpired
	StateRateLimited
	StateQuarantined
)

// String returns a human-readable token state.
func (s TokenState) String() string {
	names := [...]string{"valid", "expired", "rate_limited", "quarantined"}
	if int(s) < len(names) {
		return names[s]
	}
	return "unknown"
}

// ManagedToken wraps a raw token with metadata for pool management.
type ManagedToken struct {
	Value        string     `json:"value"`
	Platform     string     `json:"platform"`
	Label        string     `json:"label,omitempty"`
	State        TokenState `json:"state"`
	LastUsed     time.Time  `json:"lastUsed"`
	LastChecked  time.Time  `json:"lastChecked"`
	UseCount     int64      `json:"useCount"`
	ErrorCount   int        `json:"errorCount"`
	UserID       string     `json:"userId,omitempty"`
}

// Masked returns the token value with middle characters hidden.
func (t *ManagedToken) Masked() string {
	if len(t.Value) <= 12 {
		return "****"
	}
	return t.Value[:6] + "..." + t.Value[len(t.Value)-4:]
}

// Manager handles a pool of tokens with rotation, validation, and quarantine.
type Manager struct {
	mu     sync.RWMutex
	tokens []*ManagedToken
	inUse  map[string]bool
	logger *slog.Logger

	// Round-robin index
	rrIndex int
}

// NewManager creates a new token manager.
func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		tokens: make([]*ManagedToken, 0),
		inUse:  make(map[string]bool),
		logger: logger.With("component", "token-manager"),
	}
}

// AddBulk adds multiple raw token strings to the pool.
func (m *Manager) AddBulk(values []string, plat string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing := make(map[string]bool)
	for _, t := range m.tokens {
		existing[t.Value] = true
	}

	added := 0
	var addedTokens []string
	for _, val := range values {
		if val == "" || existing[val] {
			continue
		}
		m.tokens = append(m.tokens, &ManagedToken{
			Value:    val,
			Platform: plat,
			State:    StateValid,
		})
		existing[val] = true
		added++
		addedTokens = append(addedTokens, val)
	}

	if len(addedTokens) > 0 {
		go func(newTokens []string) {
			f, err := os.OpenFile("data/tokens.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				m.logger.Error("failed to open tokens.txt for append", "error", err)
				return
			}
			defer f.Close()
			for _, t := range newTokens {
				f.WriteString(t + "\n")
			}
		}(addedTokens)
	}

	m.logger.Info("tokens added", "added", added, "total", len(m.tokens))
	return added
}

// Acquire returns the next available valid token (round-robin).
func (m *Manager) Acquire() *ManagedToken {
	m.mu.Lock()
	defer m.mu.Unlock()

	valid := m.getAvailable()
	if len(valid) == 0 {
		return nil
	}

	token := valid[m.rrIndex%len(valid)]
	m.rrIndex++
	token.UseCount++
	token.LastUsed = time.Now()
	m.inUse[token.Value] = true
	return token
}

// Release marks a token as no longer actively in use.
func (m *Manager) Release(t *ManagedToken) {
	if t == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.inUse, t.Value)
}

// Quarantine marks a token as invalid (e.g., 401 response).
func (m *Manager) Quarantine(t *ManagedToken) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t.State = StateQuarantined
	t.ErrorCount++
	m.logger.Warn("token quarantined", "token", t.Masked(), "errors", t.ErrorCount)
}

// ReportError increments the error count; quarantines after threshold.
func (m *Manager) ReportError(t *ManagedToken) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t.ErrorCount++
	if t.ErrorCount >= 3 {
		t.State = StateQuarantined
		m.logger.Warn("token auto-quarantined after errors", "token", t.Masked())
	}
}

// ValidateAll checks all tokens against the platform API.
func (m *Manager) ValidateAll(ctx context.Context, p platform.Platform, proxyURL string) (valid, invalid int) {
	m.mu.RLock()
	tokens := make([]*ManagedToken, len(m.tokens))
	copy(tokens, m.tokens)
	m.mu.RUnlock()

	m.logger.Info("validating all tokens", "count", len(tokens))

	for _, t := range tokens {
		select {
		case <-ctx.Done():
			return
		default:
		}

		status, err := p.ValidateToken(ctx, t.Value, proxyURL)
		t.LastChecked = time.Now()

		if err != nil || status != platform.TokenValid {
			m.mu.Lock()
			t.State = StateExpired
			m.mu.Unlock()
			invalid++
		} else {
			m.mu.Lock()
			t.State = StateValid
			t.ErrorCount = 0
			m.mu.Unlock()
			valid++
		}

		// Throttle to avoid rate limiting
		time.Sleep(200 * time.Millisecond)
	}

	m.logger.Info("token validation complete", "valid", valid, "invalid", invalid)
	return
}

// GetValidValues returns raw token values for all valid tokens.
func (m *Manager) GetValidValues() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var values []string
	for _, t := range m.tokens {
		if t.State == StateValid {
			values = append(values, t.Value)
		}
	}
	return values
}

// All returns a copy of all managed tokens.
func (m *Manager) All() []*ManagedToken {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*ManagedToken, len(m.tokens))
	copy(result, m.tokens)
	return result
}

// Stats returns counts by state.
func (m *Manager) Stats() (total, valid, expired, quarantined, inUse int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total = len(m.tokens)
	inUse = len(m.inUse)
	for _, t := range m.tokens {
		switch t.State {
		case StateValid:
			valid++
		case StateExpired:
			expired++
		case StateQuarantined:
			quarantined++
		}
	}
	return
}

// getAvailable returns tokens that are valid and not in use. Must hold mu.
func (m *Manager) getAvailable() []*ManagedToken {
	var result []*ManagedToken
	for _, t := range m.tokens {
		if t.State == StateValid && !m.inUse[t.Value] {
			result = append(result, t)
		}
	}
	return result
}
