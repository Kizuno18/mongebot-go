package engine

import (
	"testing"

	"github.com/Kizuno18/mongebot-go/internal/platform"
)

func TestFSM_InitialState(t *testing.T) {
	fsm := NewViewerFSM()
	if fsm.State() != platform.ViewerIdle {
		t.Errorf("expected idle, got %s", fsm.State())
	}
}

func TestFSM_ValidTransitions(t *testing.T) {
	fsm := NewViewerFSM()

	// idle -> connecting
	if err := fsm.Transition(platform.ViewerConnecting); err != nil {
		t.Fatalf("idle->connecting should be valid: %v", err)
	}

	// connecting -> active
	if err := fsm.Transition(platform.ViewerActive); err != nil {
		t.Fatalf("connecting->active should be valid: %v", err)
	}

	// active -> reconnecting
	if err := fsm.Transition(platform.ViewerReconnecting); err != nil {
		t.Fatalf("active->reconnecting should be valid: %v", err)
	}

	// reconnecting -> connecting (retry)
	if err := fsm.Transition(platform.ViewerConnecting); err != nil {
		t.Fatalf("reconnecting->connecting should be valid: %v", err)
	}

	// connecting -> stopped
	if err := fsm.Transition(platform.ViewerStopped); err != nil {
		t.Fatalf("connecting->stopped should be valid: %v", err)
	}
}

func TestFSM_InvalidTransitions(t *testing.T) {
	fsm := NewViewerFSM()

	// idle -> active (must go through connecting first)
	if err := fsm.Transition(platform.ViewerActive); err == nil {
		t.Error("idle->active should be invalid")
	}

	// idle -> reconnecting (can't reconnect from idle)
	if err := fsm.Transition(platform.ViewerReconnecting); err == nil {
		t.Error("idle->reconnecting should be invalid")
	}
}

func TestFSM_Hooks(t *testing.T) {
	fsm := NewViewerFSM()
	hookFired := false

	fsm.OnEnter(platform.ViewerActive, func(from, to platform.ViewerStatus) {
		hookFired = true
		if from != platform.ViewerConnecting {
			t.Errorf("expected from=connecting, got %s", from)
		}
	})

	fsm.Transition(platform.ViewerConnecting)
	fsm.Transition(platform.ViewerActive)

	if !hookFired {
		t.Error("hook should have fired on entering active state")
	}
}

func TestFSM_CanTransition(t *testing.T) {
	fsm := NewViewerFSM()

	if !fsm.CanTransition(platform.ViewerConnecting) {
		t.Error("should be able to transition to connecting from idle")
	}
	if fsm.CanTransition(platform.ViewerActive) {
		t.Error("should NOT be able to transition to active from idle")
	}
}

func TestFSM_IsActive(t *testing.T) {
	fsm := NewViewerFSM()

	if fsm.IsActive() {
		t.Error("should not be active in idle state")
	}

	fsm.Transition(platform.ViewerConnecting)
	fsm.Transition(platform.ViewerActive)

	if !fsm.IsActive() {
		t.Error("should be active")
	}
}

func TestFSM_IsTerminal(t *testing.T) {
	fsm := NewViewerFSM()
	fsm.Transition(platform.ViewerConnecting)
	fsm.Transition(platform.ViewerStopped)

	if !fsm.IsTerminal() {
		t.Error("stopped should be terminal")
	}
}

func TestFSM_ForceState(t *testing.T) {
	fsm := NewViewerFSM()
	fsm.ForceState(platform.ViewerError)

	if fsm.State() != platform.ViewerError {
		t.Error("force state should bypass validation")
	}
}

func TestFSM_TransitionMap(t *testing.T) {
	fsm := NewViewerFSM()
	transitions := fsm.TransitionMap()

	if len(transitions) != 1 || transitions[0] != platform.ViewerConnecting {
		t.Errorf("idle should only transition to connecting, got %v", transitions)
	}
}
