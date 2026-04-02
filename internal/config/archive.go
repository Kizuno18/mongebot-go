// Package config - encrypted configuration archive for export/import across machines.
// Bundles profiles, config, and proxy lists into a single encrypted JSON file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Archive is the portable export format containing all user data.
type Archive struct {
	Version   int             `json:"version"`
	CreatedAt time.Time       `json:"createdAt"`
	Config    *AppConfig      `json:"config"`
	Profiles  json.RawMessage `json:"profiles,omitempty"`
	Proxies   []string        `json:"proxies,omitempty"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
}

// ExportArchive creates a plain JSON archive of the current configuration.
func ExportArchive(cfg *AppConfig, profiles json.RawMessage, proxies []string) ([]byte, error) {
	archive := Archive{
		Version:   1,
		CreatedAt: time.Now(),
		Config:    cfg,
		Profiles:  profiles,
		Proxies:   proxies,
		Metadata: map[string]any{
			"app":     "mongebot",
			"version": "2.0.0",
		},
	}

	return json.MarshalIndent(archive, "", "  ")
}

// ImportArchive parses an archive file.
func ImportArchive(data []byte) (*Archive, error) {
	var archive Archive
	if err := json.Unmarshal(data, &archive); err != nil {
		return nil, fmt.Errorf("parsing archive: %w", err)
	}
	return &archive, nil
}

// ExportToFile creates an archive and writes it to a file.
func ExportToFile(path string, cfg *AppConfig, profiles json.RawMessage, proxies []string) error {
	data, err := ExportArchive(cfg, profiles, proxies)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ImportFromFile reads and parses an archive from a file.
func ImportFromFile(path string) (*Archive, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading archive: %w", err)
	}
	return ImportArchive(data)
}


