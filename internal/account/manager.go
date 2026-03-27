// Package account - profile CRUD operations with JSON persistence.
package account

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Manager handles CRUD operations for profiles.
type Manager struct {
	mu       sync.RWMutex
	profiles []*Profile
	filePath string
	logger   *slog.Logger
}

// NewManager creates a profile manager that persists to the given file.
func NewManager(filePath string, logger *slog.Logger) (*Manager, error) {
	m := &Manager{
		profiles: make([]*Profile, 0),
		filePath: filePath,
		logger:   logger.With("component", "account-manager"),
	}

	if err := m.load(); err != nil {
		return nil, err
	}

	return m, nil
}

// Create adds a new profile.
func (m *Manager) Create(name, platform, channel string) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate channel on same platform
	for _, p := range m.profiles {
		if p.Platform == platform && p.Channel == channel {
			return nil, fmt.Errorf("profile already exists for %s/%s", platform, channel)
		}
	}

	profile := NewProfile(name, platform, channel)
	m.profiles = append(m.profiles, profile)

	if err := m.save(); err != nil {
		return nil, err
	}

	m.logger.Info("profile created", "id", profile.ID, "name", name, "channel", channel)
	return profile, nil
}

// Get returns a profile by ID.
func (m *Manager) Get(id string) *Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.profiles {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// Update modifies an existing profile.
func (m *Manager) Update(id string, fn func(*Profile)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.profiles {
		if p.ID == id {
			fn(p)
			p.UpdatedAt = p.UpdatedAt // will be set by caller
			return m.save()
		}
	}
	return fmt.Errorf("profile %q not found", id)
}

// Delete removes a profile by ID.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filtered := make([]*Profile, 0, len(m.profiles))
	found := false
	for _, p := range m.profiles {
		if p.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, p)
	}

	if !found {
		return fmt.Errorf("profile %q not found", id)
	}

	m.profiles = filtered
	m.logger.Info("profile deleted", "id", id)
	return m.save()
}

// SetActive activates one profile and deactivates all others.
func (m *Manager) SetActive(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	for _, p := range m.profiles {
		if p.ID == id {
			p.Active = true
			found = true
		} else {
			p.Active = false
		}
	}

	if !found {
		return fmt.Errorf("profile %q not found", id)
	}

	return m.save()
}

// GetActive returns the currently active profile.
func (m *Manager) GetActive() *Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.profiles {
		if p.Active {
			return p
		}
	}
	return nil
}

// List returns all profiles.
func (m *Manager) List() []*Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Profile, len(m.profiles))
	copy(result, m.profiles)
	return result
}

// Duplicate clones an existing profile with a new name.
func (m *Manager) Duplicate(id, newName string) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.profiles {
		if p.ID == id {
			clone := p.Clone(newName)
			m.profiles = append(m.profiles, clone)
			if err := m.save(); err != nil {
				return nil, err
			}
			return clone, nil
		}
	}
	return nil, fmt.Errorf("profile %q not found", id)
}

// Export serializes all profiles to JSON bytes.
func (m *Manager) Export() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return json.MarshalIndent(m.profiles, "", "  ")
}

// Import loads profiles from JSON bytes (merging with existing).
func (m *Manager) Import(data []byte) (int, error) {
	var imported []*Profile
	if err := json.Unmarshal(data, &imported); err != nil {
		return 0, fmt.Errorf("invalid profile data: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	existing := make(map[string]bool)
	for _, p := range m.profiles {
		existing[p.ID] = true
	}

	added := 0
	for _, p := range imported {
		if existing[p.ID] {
			p.ID = generateID() // Avoid conflicts
		}
		p.Active = false // Don't auto-activate imported profiles
		m.profiles = append(m.profiles, p)
		added++
	}

	if err := m.save(); err != nil {
		return 0, err
	}

	m.logger.Info("profiles imported", "added", added)
	return added, nil
}

// load reads profiles from the JSON file.
func (m *Manager) load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file yet
		}
		return fmt.Errorf("reading profiles: %w", err)
	}

	return json.Unmarshal(data, &m.profiles)
}

// save writes profiles to the JSON file.
func (m *Manager) save() error {
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.profiles, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filePath, data, 0o644)
}
