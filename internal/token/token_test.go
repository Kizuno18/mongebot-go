package token

import (
	"log/slog"
	"os"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestManagerAddBulk(t *testing.T) {
	mgr := NewManager(testLogger())

	added := mgr.AddBulk([]string{"token1", "token2", "token3", "token1"}, "twitch")
	if added != 3 {
		t.Errorf("expected 3 added (deduplicated), got %d", added)
	}

	total, valid, _, _, _ := mgr.Stats()
	if total != 3 || valid != 3 {
		t.Errorf("expected total=3 valid=3, got total=%d valid=%d", total, valid)
	}
}

func TestManagerAcquireRelease(t *testing.T) {
	mgr := NewManager(testLogger())
	mgr.AddBulk([]string{"t1", "t2", "t3"}, "twitch")

	tok := mgr.Acquire()
	if tok == nil {
		t.Fatal("expected token, got nil")
	}

	_, _, _, _, inUse := mgr.Stats()
	if inUse != 1 {
		t.Errorf("expected inUse=1, got %d", inUse)
	}

	mgr.Release(tok)
	_, _, _, _, inUse2 := mgr.Stats()
	if inUse2 != 0 {
		t.Errorf("expected inUse=0 after release, got %d", inUse2)
	}
}

func TestManagerQuarantine(t *testing.T) {
	mgr := NewManager(testLogger())
	mgr.AddBulk([]string{"t1"}, "twitch")

	tok := mgr.Acquire()
	mgr.Quarantine(tok)
	mgr.Release(tok)

	_, valid, _, quarantined, _ := mgr.Stats()
	if valid != 0 || quarantined != 1 {
		t.Errorf("expected valid=0 quarantined=1, got valid=%d quarantined=%d", valid, quarantined)
	}

	// Should not be acquirable
	tok2 := mgr.Acquire()
	if tok2 != nil {
		t.Error("quarantined token should not be acquirable")
	}
}

func TestManagerAutoQuarantine(t *testing.T) {
	mgr := NewManager(testLogger())
	mgr.AddBulk([]string{"t1"}, "twitch")

	tok := mgr.Acquire()
	mgr.Release(tok)

	// Report 3 errors to trigger auto-quarantine
	mgr.ReportError(tok)
	mgr.ReportError(tok)
	mgr.ReportError(tok)

	if tok.State != StateQuarantined {
		t.Errorf("expected quarantined after 3 errors, got %v", tok.State)
	}
}

func TestManagedTokenMasked(t *testing.T) {
	tok := &ManagedToken{Value: "abcdefghijklmnopqrst"}
	masked := tok.Masked()
	if masked != "abcdef...qrst" {
		t.Errorf("expected abcdef...qrst, got %s", masked)
	}

	short := &ManagedToken{Value: "short"}
	if short.Masked() != "****" {
		t.Errorf("expected ****, got %s", short.Masked())
	}
}

func TestGetValidValues(t *testing.T) {
	mgr := NewManager(testLogger())
	mgr.AddBulk([]string{"valid1", "valid2", "expire"}, "twitch")

	// Quarantine one
	all := mgr.All()
	all[2].State = StateExpired

	values := mgr.GetValidValues()
	if len(values) != 2 {
		t.Errorf("expected 2 valid values, got %d", len(values))
	}
}
