package account

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewProfile(t *testing.T) {
	p := NewProfile("Test", "twitch", "streamer")
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
	if p.Name != "Test" || p.Channel != "streamer" || p.Platform != "twitch" {
		t.Errorf("unexpected profile values: %+v", p)
	}
}

func TestProfileClone(t *testing.T) {
	p := NewProfile("Original", "twitch", "ch1")
	p.MaxWorkers = intPtr(100)

	clone := p.Clone("Copy")
	if clone.ID == p.ID {
		t.Error("clone should have different ID")
	}
	if clone.Name != "Copy" {
		t.Errorf("expected name=Copy, got %s", clone.Name)
	}
	if *clone.MaxWorkers != 100 {
		t.Error("clone should preserve MaxWorkers")
	}
	if clone.Active {
		t.Error("clone should not be active")
	}
}

func TestEffectiveWorkers(t *testing.T) {
	p := &Profile{}
	if p.EffectiveWorkers(50) != 50 {
		t.Error("should return global default when no override")
	}

	p.MaxWorkers = intPtr(25)
	if p.EffectiveWorkers(50) != 25 {
		t.Error("should return profile override")
	}
}

func TestManagerCRUD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	mgr, err := NewManager(path, testLogger())
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	// Create
	p, err := mgr.Create("Test1", "twitch", "ch1")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if p.Name != "Test1" {
		t.Errorf("expected name=Test1, got %s", p.Name)
	}

	// List
	profiles := mgr.List()
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(profiles))
	}

	// Duplicate channel should fail
	_, err = mgr.Create("Test2", "twitch", "ch1")
	if err == nil {
		t.Error("expected error for duplicate channel")
	}

	// Activate
	if err := mgr.SetActive(p.ID); err != nil {
		t.Fatalf("SetActive error: %v", err)
	}
	active := mgr.GetActive()
	if active == nil || active.ID != p.ID {
		t.Error("expected profile to be active")
	}

	// Duplicate
	dup, err := mgr.Duplicate(p.ID, "Duplicated")
	if err != nil {
		t.Fatalf("Duplicate error: %v", err)
	}
	if dup.Name != "Duplicated" || dup.Active {
		t.Error("duplicate should be inactive with new name")
	}

	// Delete
	if err := mgr.Delete(p.ID); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if len(mgr.List()) != 1 {
		t.Error("expected 1 profile after delete")
	}

	// Persistence: reload from file
	mgr2, err := NewManager(path, testLogger())
	if err != nil {
		t.Fatalf("reload error: %v", err)
	}
	if len(mgr2.List()) != 1 {
		t.Errorf("expected 1 profile after reload, got %d", len(mgr2.List()))
	}
}

func TestExportImport(t *testing.T) {
	dir := t.TempDir()

	mgr1, _ := NewManager(filepath.Join(dir, "p1.json"), testLogger())
	mgr1.Create("Profile1", "twitch", "ch1")
	mgr1.Create("Profile2", "kick", "ch2")

	data, err := mgr1.Export()
	if err != nil {
		t.Fatalf("Export error: %v", err)
	}

	mgr2, _ := NewManager(filepath.Join(dir, "p2.json"), testLogger())
	added, err := mgr2.Import(data)
	if err != nil {
		t.Fatalf("Import error: %v", err)
	}
	if added != 2 {
		t.Errorf("expected 2 imported, got %d", added)
	}
}

func intPtr(n int) *int { return &n }
