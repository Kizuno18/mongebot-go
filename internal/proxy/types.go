// Package proxy manages proxy pools, rotation strategies, and health checking.
package proxy

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// ProxyType represents the protocol of a proxy.
type ProxyType int

const (
	ProxyHTTP ProxyType = iota
	ProxySOCKS4
	ProxySOCKS5
)

// String returns the scheme for a proxy type.
func (t ProxyType) String() string {
	switch t {
	case ProxySOCKS4:
		return "socks4"
	case ProxySOCKS5:
		return "socks5"
	default:
		return "http"
	}
}

// HealthStatus represents the current health of a proxy.
type HealthStatus int

const (
	HealthUnknown HealthStatus = iota
	HealthGood
	HealthSlow
	HealthDead
)

// String returns a human-readable health status.
func (s HealthStatus) String() string {
	switch s {
	case HealthGood:
		return "good"
	case HealthSlow:
		return "slow"
	case HealthDead:
		return "dead"
	default:
		return "unknown"
	}
}

// Proxy represents a single proxy endpoint.
type Proxy struct {
	Host     string       `json:"host"`
	Port     string       `json:"port"`
	Username string       `json:"username,omitempty"`
	Password string       `json:"password,omitempty"`
	Type     ProxyType    `json:"type"`
	Health   HealthStatus `json:"health"`
	Latency  time.Duration `json:"latency"`
	LastUsed time.Time    `json:"lastUsed"`
	UseCount int64        `json:"useCount"`
	Country  string       `json:"country,omitempty"`
}

// URL returns the proxy as a URL string (e.g., "http://user:pass@host:port").
func (p *Proxy) URL() string {
	scheme := p.Type.String()
	if p.Username != "" && p.Password != "" {
		return fmt.Sprintf("%s://%s:%s@%s:%s", scheme, p.Username, p.Password, p.Host, p.Port)
	}
	return fmt.Sprintf("%s://%s:%s", scheme, p.Host, p.Port)
}

// URLParsed returns the proxy as a *url.URL.
func (p *Proxy) URLParsed() *url.URL {
	u, _ := url.Parse(p.URL())
	return u
}

// Raw returns the proxy in ip:port:user:pass format.
func (p *Proxy) Raw() string {
	if p.Username != "" && p.Password != "" {
		return fmt.Sprintf("%s:%s:%s:%s", p.Host, p.Port, p.Username, p.Password)
	}
	return fmt.Sprintf("%s:%s", p.Host, p.Port)
}

// RotationStrategy defines how proxies are selected from the pool.
type RotationStrategy int

const (
	RotationRoundRobin RotationStrategy = iota
	RotationRandom
	RotationLeastUsed
	RotationFastest
)

// String returns the strategy name.
func (s RotationStrategy) String() string {
	switch s {
	case RotationRandom:
		return "random"
	case RotationLeastUsed:
		return "least-used"
	case RotationFastest:
		return "fastest"
	default:
		return "round-robin"
	}
}

// ParseProxy parses various proxy string formats into a Proxy struct.
// Supported formats:
//   - ip:port
//   - ip:port:user:pass
//   - scheme://ip:port
//   - scheme://user:pass@ip:port
func ParseProxy(raw string) (*Proxy, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty proxy string")
	}

	// Try URL format first
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		p := &Proxy{
			Host:   u.Hostname(),
			Port:   u.Port(),
			Health: HealthUnknown,
		}
		if u.User != nil {
			p.Username = u.User.Username()
			p.Password, _ = u.User.Password()
		}
		switch u.Scheme {
		case "socks4":
			p.Type = ProxySOCKS4
		case "socks5":
			p.Type = ProxySOCKS5
		default:
			p.Type = ProxyHTTP
		}
		return p, nil
	}

	// Try ip:port:user:pass format
	parts := strings.SplitN(raw, ":", 4)
	switch len(parts) {
	case 2:
		return &Proxy{Host: parts[0], Port: parts[1], Type: ProxyHTTP, Health: HealthUnknown}, nil
	case 4:
		return &Proxy{
			Host: parts[0], Port: parts[1],
			Username: parts[2], Password: parts[3],
			Type: ProxyHTTP, Health: HealthUnknown,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported proxy format: %q", raw)
	}
}
