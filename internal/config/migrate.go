// Package config - versioned configuration migration system.
// Automatically upgrades config files from older versions on load.
package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// CurrentVersion is the latest config version.
const CurrentVersion = 2

// Migrator handles version-to-version config upgrades.
type Migrator struct {
	logger *slog.Logger
}

// NewMigrator creates a config migrator.
func NewMigrator(logger *slog.Logger) *Migrator {
	return &Migrator{logger: logger}
}

// MigrateIfNeeded checks the config version and applies migrations.
// Returns the migrated config data as JSON bytes.
func (m *Migrator) MigrateIfNeeded(data []byte) ([]byte, bool, error) {
	// Parse version only
	var versionCheck struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &versionCheck); err != nil {
		return data, false, fmt.Errorf("reading config version: %w", err)
	}

	if versionCheck.Version >= CurrentVersion {
		return data, false, nil // Already current
	}

	m.logger.Info("config migration needed",
		"from", versionCheck.Version,
		"to", CurrentVersion,
	)

	// Apply migrations sequentially
	var err error
	migrated := false
	currentData := data

	for v := versionCheck.Version; v < CurrentVersion; v++ {
		migrateFn, exists := migrations[v]
		if !exists {
			return data, false, fmt.Errorf("no migration path from version %d", v)
		}

		currentData, err = migrateFn(currentData)
		if err != nil {
			return data, false, fmt.Errorf("migration v%d -> v%d failed: %w", v, v+1, err)
		}
		migrated = true
		m.logger.Info("config migrated", "from", v, "to", v+1)
	}

	return currentData, migrated, nil
}

// migrationFunc transforms config JSON from version N to N+1.
type migrationFunc func(data []byte) ([]byte, error)

// migrations maps source version to its migration function.
var migrations = map[int]migrationFunc{
	0: migrateV0toV1,
	1: migrateV1toV2,
}

// migrateV0toV1 handles initial version (no version field) to v1.
func migrateV0toV1(data []byte) ([]byte, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	raw["version"] = 1

	// Ensure engine section exists
	if _, ok := raw["engine"]; !ok {
		defaults := DefaultConfig()
		engineJSON, _ := json.Marshal(defaults.Engine)
		var engineMap map[string]any
		json.Unmarshal(engineJSON, &engineMap)
		raw["engine"] = engineMap
	}

	return json.MarshalIndent(raw, "", "  ")
}

// migrateV1toV2 adds new v2 fields: scheduler rules, multi-channel support, UI prefs.
func migrateV1toV2(data []byte) ([]byte, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	raw["version"] = 2

	// Add scheduler section if missing
	if _, ok := raw["scheduler"]; !ok {
		raw["scheduler"] = map[string]any{
			"enabled": false,
			"rules":   []any{},
		}
	}

	// Add UI preferences section if missing
	if _, ok := raw["ui"]; !ok {
		raw["ui"] = map[string]any{
			"theme":       "dark",
			"accentColor": "blue",
			"compactMode": false,
			"showCharts":  true,
		}
	}

	// Add multi-channel flag to engine
	if engine, ok := raw["engine"].(map[string]any); ok {
		if _, ok := engine["multiChannel"]; !ok {
			engine["multiChannel"] = false
		}
	}

	return json.MarshalIndent(raw, "", "  ")
}

// RegisterMigration allows adding custom migrations (useful for plugins).
func RegisterMigration(fromVersion int, fn migrationFunc) {
	migrations[fromVersion] = fn
}
