package platform

import (
	"context"
	"testing"
)

func TestStreamStatusString(t *testing.T) {
	tests := []struct {
		status StreamStatus
		want   string
	}{
		{StreamOffline, "offline"},
		{StreamOnline, "online"},
		{StreamUnknown, "unknown"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("StreamStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestViewerStatusString(t *testing.T) {
	tests := []struct {
		status ViewerStatus
		want   string
	}{
		{ViewerIdle, "idle"},
		{ViewerConnecting, "connecting"},
		{ViewerActive, "active"},
		{ViewerReconnecting, "reconnecting"},
		{ViewerStopped, "stopped"},
		{ViewerError, "error"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("ViewerStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	if _, err := reg.Get("twitch"); err == nil {
		t.Error("expected error for unregistered platform")
	}

	mock := &mockPlatform{name: "twitch"}
	reg.Register(mock)

	p, err := reg.Get("twitch")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if p.Name() != "twitch" {
		t.Errorf("expected name=twitch, got %s", p.Name())
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockPlatform{name: "twitch"})
	reg.Register(&mockPlatform{name: "kick"})

	names := reg.List()
	if len(names) != 2 {
		t.Errorf("expected 2 platforms, got %d", len(names))
	}
}

// mockPlatform is a minimal Platform implementation for testing.
type mockPlatform struct {
	name string
}

func (m *mockPlatform) Name() string { return m.name }
func (m *mockPlatform) Connect(_ context.Context, _ *ViewerConfig) (Viewer, error) {
	return nil, nil
}
func (m *mockPlatform) ValidateToken(_ context.Context, _ string, _ string) (TokenStatus, error) {
	return TokenValid, nil
}
func (m *mockPlatform) GetStreamStatus(_ context.Context, _ string) (StreamStatus, error) {
	return StreamUnknown, nil
}
func (m *mockPlatform) GetStreamMetadata(_ context.Context, _ string, _ string, _ string) (*StreamMetadata, error) {
	return nil, nil
}
func (m *mockPlatform) SupportedFeatures() []Feature { return nil }
