// Package proxy - pool management and rotation logic.
package proxy

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Manager manages a pool of proxies with rotation and health tracking.
type Manager struct {
	mu           sync.RWMutex
	proxies      []*Proxy
	inUse        map[string]bool // tracks proxies currently assigned to workers
	strategy     RotationStrategy
	rrIndex      atomic.Int64 // round-robin counter
	chainManager *ChainManager // optional proxy chain manager
}

// NewManager creates a new proxy manager with the given rotation strategy.
func NewManager(strategy RotationStrategy) *Manager {
	return &Manager{
		proxies:      make([]*Proxy, 0),
		inUse:        make(map[string]bool),
		strategy:     strategy,
		chainManager: NewChainManager(),
	}
}

// SetChainManager sets a custom chain manager.
func (m *Manager) SetChainManager(cm *ChainManager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chainManager = cm
}

// GetChainManager returns the chain manager.
func (m *Manager) GetChainManager() *ChainManager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.chainManager
}

// AddChain adds a proxy chain to the manager.
func (m *Manager) AddChain(name string, proxyURLs []string) error {
	chain, err := BuildChain(name, proxyURLs...)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chainManager.Add(chain)
	return nil
}

// GetChain returns a proxy chain by name.
func (m *Manager) GetChain(name string) *Chain {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.chainManager.Get(name)
}

// AcquireChain gets the entry proxy from a named chain.
func (m *Manager) AcquireChain(name string) *Chain {
	m.mu.RLock()
	defer m.mu.RUnlock()
	chain := m.chainManager.Get(name)
	if chain == nil {
		return nil
	}
	return chain
}

// LoadFromFile reads proxies from a text file (one per line).
func (m *Manager) LoadFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No proxy file is not an error
		}
		return fmt.Errorf("opening proxy file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var proxies []*Proxy
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, err := ParseProxy(line)
		if err != nil {
			continue // Skip invalid lines
		}
		proxies = append(proxies, p)
	}

	m.mu.Lock()
	m.proxies = append(m.proxies, proxies...)
	m.mu.Unlock()

	return scanner.Err()
}

// AddBulk adds multiple proxies from raw strings.
func (m *Manager) AddBulk(rawProxies []string) (added int, errors []string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing := make(map[string]bool)
	for _, p := range m.proxies {
		existing[p.Raw()] = true
	}

	for _, raw := range rawProxies {
		p, err := ParseProxy(raw)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", raw, err))
			continue
		}
		if existing[p.Raw()] {
			continue // Skip duplicates
		}
		m.proxies = append(m.proxies, p)
		existing[p.Raw()] = true
		added++
	}
	return
}

// Acquire gets the next available proxy based on the rotation strategy.
// Returns nil if no proxies are available.
func (m *Manager) Acquire() *Proxy {
	m.mu.Lock()
	defer m.mu.Unlock()

	available := m.getAvailable()
	if len(available) == 0 {
		return nil
	}

	var selected *Proxy
	switch m.strategy {
	case RotationRandom:
		selected = available[rand.IntN(len(available))]
	case RotationLeastUsed:
		sort.Slice(available, func(i, j int) bool {
			return available[i].UseCount < available[j].UseCount
		})
		selected = available[0]
	case RotationFastest:
		sort.Slice(available, func(i, j int) bool {
			return available[i].Latency < available[j].Latency
		})
		selected = available[0]
	default: // RoundRobin
		idx := m.rrIndex.Add(1) - 1
		selected = available[idx%int64(len(available))]
	}

	selected.UseCount++
	selected.LastUsed = time.Now()
	m.inUse[selected.Raw()] = true
	return selected
}

// Release marks a proxy as no longer in use.
func (m *Manager) Release(p *Proxy) {
	if p == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.inUse, p.Raw())
}

// UpdateHealth updates the health status of a proxy.
func (m *Manager) UpdateHealth(p *Proxy, health HealthStatus, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.Health = health
	p.Latency = latency
}

// Count returns total and available proxy counts.
func (m *Manager) Count() (total, available, inUse int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total = len(m.proxies)
	inUse = len(m.inUse)
	for _, p := range m.proxies {
		if p.Health != HealthDead && !m.inUse[p.Raw()] {
			available++
		}
	}
	return
}

// All returns a copy of all proxies.
func (m *Manager) All() []*Proxy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Proxy, len(m.proxies))
	copy(result, m.proxies)
	return result
}

// SetStrategy changes the rotation strategy.
func (m *Manager) SetStrategy(s RotationStrategy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strategy = s
}

// getAvailable returns proxies that are not dead and not in use. Must hold mu.
func (m *Manager) getAvailable() []*Proxy {
	var result []*Proxy
	for _, p := range m.proxies {
		if p.Health != HealthDead && !m.inUse[p.Raw()] {
			result = append(result, p)
		}
	}
	return result
}
