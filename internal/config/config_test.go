package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Engine.MaxWorkers != 50 {
		t.Errorf("expected MaxWorkers=50, got %d", cfg.Engine.MaxWorkers)
	}
	if cfg.Engine.RestartInterval.Duration != 10*time.Second {
		t.Errorf("expected RestartInterval=10s, got %v", cfg.Engine.RestartInterval)
	}
	if !cfg.Engine.Features.Ads {
		t.Error("expected Ads=true by default")
	}
	if cfg.API.Port != 9800 {
		t.Errorf("expected API.Port=9800, got %d", cfg.API.Port)
	}
}

func TestLoadCreatesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Engine.MaxWorkers != 50 {
		t.Errorf("default MaxWorkers should be 50, got %d", cfg.Engine.MaxWorkers)
	}

	// File should exist now
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("config file should have been created")
	}
}

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Modify and save
	cfg.Update(func(c *AppConfig) {
		c.Engine.MaxWorkers = 100
	})

	// Reload
	cfg2, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after save error: %v", err)
	}

	if cfg2.Engine.MaxWorkers != 100 {
		t.Errorf("expected MaxWorkers=100 after reload, got %d", cfg2.Engine.MaxWorkers)
	}
}

func TestDurationJSON(t *testing.T) {
	d := Duration{5 * time.Minute}

	data, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}
	if string(data) != `"5m0s"` {
		t.Errorf("expected \"5m0s\", got %s", string(data))
	}

	var d2 Duration
	if err := d2.UnmarshalJSON(data); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if d2.Duration != 5*time.Minute {
		t.Errorf("expected 5m, got %v", d2.Duration)
	}
}

func TestGetActiveProfile(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Profiles = []ProfileConfig{
		{ID: "1", Name: "Test1", Channel: "ch1", Active: false},
		{ID: "2", Name: "Test2", Channel: "ch2", Active: true},
	}

	active := cfg.GetActiveProfile()
	if active == nil {
		t.Fatal("expected active profile, got nil")
	}
	if active.ID != "2" {
		t.Errorf("expected active profile ID=2, got %s", active.ID)
	}
}
