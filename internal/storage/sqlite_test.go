package storage

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path, testLogger())
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenAndMigrate(t *testing.T) {
	db := testDB(t)

	// Tables should exist after migration
	var count int
	err := db.Conn().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&count)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if count != 1 {
		t.Error("sessions table should exist")
	}
}

func createTestProfile(t *testing.T, db *DB, id string) {
	t.Helper()
	repo := NewProfileRepo(db)
	repo.Create(context.Background(), &ProfileRow{ID: id, Name: "Test", Platform: "twitch", Channel: "ch"})
}

func TestSessionLifecycle(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createTestProfile(t, db, "profile-1")

	// Insert session
	sessionID, err := db.InsertSession(ctx, "profile-1", "streamer_name", "twitch")
	if err != nil {
		t.Fatalf("InsertSession error: %v", err)
	}
	if sessionID <= 0 {
		t.Errorf("expected positive session ID, got %d", sessionID)
	}

	// Insert metrics snapshot
	err = db.InsertMetricsSnapshot(ctx, sessionID, 42, 50, 1000, 5242880, 120, 3)
	if err != nil {
		t.Fatalf("InsertMetricsSnapshot error: %v", err)
	}

	// End session
	err = db.EndSession(ctx, sessionID, "manual_stop", 45, 2000, 10485760, 5, 200)
	if err != nil {
		t.Fatalf("EndSession error: %v", err)
	}

	// Get recent sessions
	sessions, err := db.GetRecentSessions(ctx, 10)
	if err != nil {
		t.Fatalf("GetRecentSessions error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.Channel != "streamer_name" {
		t.Errorf("expected channel=streamer_name, got %s", s.Channel)
	}
	if s.MaxViewers != 45 {
		t.Errorf("expected maxViewers=45, got %d", s.MaxViewers)
	}
	if s.EndReason == nil || *s.EndReason != "manual_stop" {
		t.Error("expected endReason=manual_stop")
	}
}

func TestMetricsTimeline(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	createTestProfile(t, db, "p1")
	sessionID, _ := db.InsertSession(ctx, "p1", "ch1", "twitch")

	// Insert 3 snapshots
	for i := range 3 {
		db.InsertMetricsSnapshot(ctx, sessionID, 10+i, 50, i*100, int64(i)*1024, i*10, i)
	}

	timeline, err := db.GetMetricsTimeline(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetMetricsTimeline error: %v", err)
	}
	if len(timeline) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(timeline))
	}
}

func TestSessionStats(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	createTestProfile(t, db, "p1")

	// Create 2 sessions with different metrics
	id1, _ := db.InsertSession(ctx, "p1", "ch1", "twitch")
	db.EndSession(ctx, id1, "done", 30, 500, 1048576, 2, 100)

	id2, _ := db.InsertSession(ctx, "p1", "ch2", "twitch")
	db.EndSession(ctx, id2, "done", 50, 800, 2097152, 3, 150)

	stats, err := db.GetSessionStats(ctx)
	if err != nil {
		t.Fatalf("GetSessionStats error: %v", err)
	}
	if stats.TotalSessions != 2 {
		t.Errorf("expected 2 sessions, got %d", stats.TotalSessions)
	}
	if stats.PeakViewers != 50 {
		t.Errorf("expected peak=50, got %d", stats.PeakViewers)
	}
	if stats.TotalAds != 5 {
		t.Errorf("expected totalAds=5, got %d", stats.TotalAds)
	}
}

func TestProfileRepo(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	repo := NewProfileRepo(db)

	// Create
	profile := &ProfileRow{
		ID:       "test-1",
		Name:     "Test Profile",
		Platform: "twitch",
		Channel:  "streamer",
	}
	if err := repo.Create(ctx, profile); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// Get by ID
	found, err := repo.GetByID(ctx, "test-1")
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if found.Name != "Test Profile" {
		t.Errorf("expected name=Test Profile, got %s", found.Name)
	}

	// List
	all, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 profile, got %d", len(all))
	}

	// Set active
	if err := repo.SetActive(ctx, "test-1"); err != nil {
		t.Fatalf("SetActive error: %v", err)
	}

	active, err := repo.GetActive(ctx)
	if err != nil {
		t.Fatalf("GetActive error: %v", err)
	}
	if active == nil || active.ID != "test-1" {
		t.Error("expected active profile")
	}

	// Check exists by channel
	exists, err := repo.ExistsByChannel(ctx, "twitch", "streamer")
	if err != nil {
		t.Fatalf("ExistsByChannel error: %v", err)
	}
	if !exists {
		t.Error("expected channel to exist")
	}

	// Delete
	if err := repo.Delete(ctx, "test-1"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	all2, _ := repo.List(ctx)
	if len(all2) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(all2))
	}
}
