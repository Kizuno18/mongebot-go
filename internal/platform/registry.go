// Package platform - registry for dynamically registering platform providers.
package platform

import (
	"fmt"
	"sync"
)

// Registry holds all registered platform providers.
type Registry struct {
	mu        sync.RWMutex
	platforms map[string]Platform
}

// NewRegistry creates an empty platform registry.
func NewRegistry() *Registry {
	return &Registry{
		platforms: make(map[string]Platform),
	}
}

// Register adds a platform provider to the registry.
func (r *Registry) Register(p Platform) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.platforms[p.Name()] = p
}

// Get returns a platform by name.
func (r *Registry) Get(name string) (Platform, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.platforms[name]
	if !ok {
		return nil, fmt.Errorf("platform %q not registered", name)
	}
	return p, nil
}

// List returns all registered platform names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.platforms))
	for name := range r.platforms {
		names = append(names, name)
	}
	return names
}
