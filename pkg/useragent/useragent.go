// Package useragent provides user-agent string management and rotation.
package useragent

import (
	"bufio"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
)

// Pool manages a collection of user-agent strings.
type Pool struct {
	mu     sync.RWMutex
	agents []string
}

// NewPool creates a new user-agent pool.
func NewPool() *Pool {
	return &Pool{
		agents: defaultAgents(),
	}
}

// LoadFromFile loads user agents from a text file (one per line).
func (p *Pool) LoadFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	var agents []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			agents = append(agents, line)
		}
	}

	if len(agents) > 0 {
		p.mu.Lock()
		p.agents = agents
		p.mu.Unlock()
	}

	return scanner.Err()
}

// Random returns a random user-agent string.
func (p *Pool) Random() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.agents) == 0 {
		return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	}
	return p.agents[rand.IntN(len(p.agents))]
}

// Count returns the number of user agents in the pool.
func (p *Pool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.agents)
}

// defaultAgents returns a set of common modern user-agent strings.
func defaultAgents() []string {
	return []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:134.0) Gecko/20100101 Firefox/134.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:134.0) Gecko/20100101 Firefox/134.0",
		"Mozilla/5.0 (X11; Linux x86_64; rv:134.0) Gecko/20100101 Firefox/134.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
	}
}
