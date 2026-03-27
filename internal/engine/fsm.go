// Package engine - formal finite state machine for viewer lifecycle management.
// Defines valid state transitions and guards to prevent illegal states.
package engine

import (
	"fmt"
	"sync"

	"github.com/Kizuno18/mongebot-go/internal/platform"
)

// ViewerFSM manages the lifecycle states of a viewer with guarded transitions.
type ViewerFSM struct {
	mu    sync.RWMutex
	state platform.ViewerStatus
	hooks map[platform.ViewerStatus][]TransitionHook
}

// TransitionHook is called when entering a new state.
type TransitionHook func(from, to platform.ViewerStatus)

// Valid state transitions
var validTransitions = map[platform.ViewerStatus][]platform.ViewerStatus{
	platform.ViewerIdle:         {platform.ViewerConnecting},
	platform.ViewerConnecting:   {platform.ViewerActive, platform.ViewerError, platform.ViewerStopped},
	platform.ViewerActive:       {platform.ViewerReconnecting, platform.ViewerStopped, platform.ViewerError},
	platform.ViewerReconnecting: {platform.ViewerConnecting, platform.ViewerStopped, platform.ViewerError},
	platform.ViewerStopped:      {platform.ViewerIdle}, // Can restart
	platform.ViewerError:        {platform.ViewerIdle, platform.ViewerStopped}, // Can retry or give up
}

// NewViewerFSM creates a new FSM starting in Idle state.
func NewViewerFSM() *ViewerFSM {
	return &ViewerFSM{
		state: platform.ViewerIdle,
		hooks: make(map[platform.ViewerStatus][]TransitionHook),
	}
}

// State returns the current state.
func (fsm *ViewerFSM) State() platform.ViewerStatus {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.state
}

// Transition attempts to move to a new state.
// Returns error if the transition is not valid.
func (fsm *ViewerFSM) Transition(to platform.ViewerStatus) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	if !fsm.isValid(fsm.state, to) {
		return fmt.Errorf("invalid transition: %s -> %s", fsm.state, to)
	}

	from := fsm.state
	fsm.state = to

	// Fire hooks (after releasing write lock would be better, but keep simple)
	if hooks, ok := fsm.hooks[to]; ok {
		for _, hook := range hooks {
			hook(from, to)
		}
	}

	return nil
}

// ForceState sets the state without transition validation (escape hatch).
func (fsm *ViewerFSM) ForceState(state platform.ViewerStatus) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	fsm.state = state
}

// OnEnter registers a hook that fires when entering the given state.
func (fsm *ViewerFSM) OnEnter(state platform.ViewerStatus, hook TransitionHook) {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	fsm.hooks[state] = append(fsm.hooks[state], hook)
}

// CanTransition checks if a transition is valid without performing it.
func (fsm *ViewerFSM) CanTransition(to platform.ViewerStatus) bool {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.isValid(fsm.state, to)
}

// isValid checks if a transition from -> to is allowed. Must hold mu.
func (fsm *ViewerFSM) isValid(from, to platform.ViewerStatus) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// IsTerminal returns true if the current state is a terminal state.
func (fsm *ViewerFSM) IsTerminal() bool {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.state == platform.ViewerStopped || fsm.state == platform.ViewerError
}

// IsActive returns true if the viewer is currently running.
func (fsm *ViewerFSM) IsActive() bool {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.state == platform.ViewerActive || fsm.state == platform.ViewerReconnecting
}

// TransitionMap returns all valid transitions from the current state.
func (fsm *ViewerFSM) TransitionMap() []platform.ViewerStatus {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	if allowed, ok := validTransitions[fsm.state]; ok {
		result := make([]platform.ViewerStatus, len(allowed))
		copy(result, allowed)
		return result
	}
	return nil
}
