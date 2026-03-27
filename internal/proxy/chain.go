// Package proxy - proxy chain support for multi-layer anonymization.
// Routes traffic through a chain of proxies (proxy → proxy → target).
package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Chain represents an ordered list of proxies to route traffic through.
type Chain struct {
	Name    string   `json:"name"`
	Proxies []*Proxy `json:"proxies"`
	Enabled bool     `json:"enabled"`
}

// NewChain creates a proxy chain with the given proxies.
func NewChain(name string, proxies ...*Proxy) *Chain {
	return &Chain{
		Name:    name,
		Proxies: proxies,
		Enabled: true,
	}
}

// Length returns the number of hops in the chain.
func (c *Chain) Length() int {
	return len(c.Proxies)
}

// FirstProxy returns the entry proxy (the one we connect to directly).
// In a chain A→B→C→target, we connect to A, which forwards to B, etc.
// Note: true proxy chaining requires each proxy to support CONNECT tunneling.
// For simple HTTP proxies, only the first hop is usable without special software.
func (c *Chain) FirstProxy() *Proxy {
	if len(c.Proxies) == 0 {
		return nil
	}
	return c.Proxies[0]
}

// LastProxy returns the exit proxy (closest to the target).
func (c *Chain) LastProxy() *Proxy {
	if len(c.Proxies) == 0 {
		return nil
	}
	return c.Proxies[len(c.Proxies)-1]
}

// Transport creates an http.Transport configured for the first proxy in the chain.
// Note: Go's standard library only supports a single proxy hop natively.
// For true multi-hop chaining, use SOCKS5 or a tunneling solution.
func (c *Chain) Transport() *http.Transport {
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		MaxConnsPerHost:     10,
	}

	first := c.FirstProxy()
	if first != nil {
		transport.Proxy = http.ProxyURL(first.URLParsed())
	}

	return transport
}

// Client creates an http.Client using this proxy chain.
func (c *Chain) Client(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: c.Transport(),
		Timeout:   timeout,
	}
}

// String returns a human-readable representation of the chain.
func (c *Chain) String() string {
	if len(c.Proxies) == 0 {
		return fmt.Sprintf("%s: (direct)", c.Name)
	}

	hops := make([]string, len(c.Proxies))
	for i, p := range c.Proxies {
		hops[i] = fmt.Sprintf("%s:%s", p.Host, p.Port)
	}

	result := c.Name + ": "
	for i, hop := range hops {
		if i > 0 {
			result += " → "
		}
		result += hop
	}
	result += " → target"
	return result
}

// ChainManager manages multiple proxy chains.
type ChainManager struct {
	chains map[string]*Chain
}

// NewChainManager creates a chain manager.
func NewChainManager() *ChainManager {
	return &ChainManager{
		chains: make(map[string]*Chain),
	}
}

// Add registers a proxy chain.
func (cm *ChainManager) Add(chain *Chain) {
	cm.chains[chain.Name] = chain
}

// Get returns a chain by name.
func (cm *ChainManager) Get(name string) *Chain {
	return cm.chains[name]
}

// Remove deletes a chain by name.
func (cm *ChainManager) Remove(name string) {
	delete(cm.chains, name)
}

// List returns all chain names.
func (cm *ChainManager) List() []string {
	names := make([]string, 0, len(cm.chains))
	for name := range cm.chains {
		names = append(names, name)
	}
	return names
}

// BuildChain creates a chain from raw proxy strings.
func BuildChain(name string, rawProxies ...string) (*Chain, error) {
	var proxies []*Proxy
	for _, raw := range rawProxies {
		p, err := ParseProxy(raw)
		if err != nil {
			return nil, fmt.Errorf("parsing proxy %q: %w", raw, err)
		}
		proxies = append(proxies, p)
	}
	return NewChain(name, proxies...), nil
}

// ProxyURL returns the chain's entry URL for use with http.Transport.Proxy.
func (c *Chain) ProxyURL() *url.URL {
	first := c.FirstProxy()
	if first == nil {
		return nil
	}
	return first.URLParsed()
}
