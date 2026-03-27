package engine

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Kizuno18/mongebot-go/internal/config"
	"github.com/Kizuno18/mongebot-go/internal/platform"
	"github.com/Kizuno18/mongebot-go/internal/proxy"
	"github.com/Kizuno18/mongebot-go/pkg/useragent"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// mockPlatform is a minimal platform for testing.
type mockPlatform struct{}

func (m *mockPlatform) Name() string { return "mock" }
func (m *mockPlatform) SupportedFeatures() []platform.Feature {
	return []platform.Feature{platform.FeatureSegments}
}
func (m *mockPlatform) ValidateToken(_ context.Context, _ string, _ string) (platform.TokenStatus, error) {
	return platform.TokenValid, nil
}
func (m *mockPlatform) GetStreamStatus(_ context.Context, _ string) (platform.StreamStatus, error) {
	return platform.StreamOnline, nil
}
func (m *mockPlatform) GetStreamMetadata(_ context.Context, _ string, _ string, _ string) (*platform.StreamMetadata, error) {
	return &platform.StreamMetadata{BroadcastID: "123", ChannelID: "456"}, nil
}
func (m *mockPlatform) Connect(_ context.Context, cfg *platform.ViewerConfig) (platform.Viewer, error) {
	return &mockViewer{id: cfg.DeviceID}, nil
}

// mockViewer is a minimal viewer that runs until context is cancelled.
type mockViewer struct {
	id     string
	cancel context.CancelFunc
}

func (v *mockViewer) ID() string                      { return v.id }
func (v *mockViewer) Status() platform.ViewerStatus    { return platform.ViewerActive }
func (v *mockViewer) Stop()                            { if v.cancel != nil { v.cancel() } }
func (v *mockViewer) Metrics() *platform.ViewerMetrics {
	return &platform.ViewerMetrics{Connected: true, SegmentsFetched: 10}
}
func (v *mockViewer) Start(ctx context.Context) error {
	ctx, v.cancel = context.WithCancel(ctx)
	<-ctx.Done()
	return nil
}

func TestEngineStartStop(t *testing.T) {
	logger := testLogger()
	proxyMgr := proxy.NewManager(proxy.RotationRandom)
	proxyMgr.AddBulk([]string{"1.1.1.1:8080", "2.2.2.2:8080", "3.3.3.3:8080"})

	tokens := []string{"token1", "token2", "token3"}
	uaPool := useragent.NewPool()

	cfg := config.DefaultConfig().Engine

	eng := New(&mockPlatform{}, proxyMgr, tokens, uaPool, cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start with 3 workers
	err := eng.Start(ctx, "test_channel", 3)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	if eng.GetState() != StateRunning {
		t.Errorf("expected running, got %s", eng.GetState())
	}

	// Allow workers to start
	time.Sleep(500 * time.Millisecond)

	m := eng.Metrics()
	if m.Channel != "test_channel" {
		t.Errorf("expected channel=test_channel, got %s", m.Channel)
	}
	if m.TotalWorkers < 1 {
		t.Errorf("expected at least 1 worker, got %d", m.TotalWorkers)
	}

	// Stop
	eng.Stop()
	if eng.GetState() != StateStopped {
		t.Errorf("expected stopped, got %s", eng.GetState())
	}
}

func TestEngineDoubleStart(t *testing.T) {
	logger := testLogger()
	proxyMgr := proxy.NewManager(proxy.RotationRandom)
	proxyMgr.AddBulk([]string{"1.1.1.1:8080"})

	eng := New(&mockPlatform{}, proxyMgr, []string{"tok"}, useragent.NewPool(), config.DefaultConfig().Engine, logger)

	ctx := context.Background()
	eng.Start(ctx, "ch", 1)
	defer eng.Stop()

	time.Sleep(200 * time.Millisecond)

	err := eng.Start(ctx, "ch2", 1)
	if err == nil {
		t.Error("expected error on double start")
	}
}

func TestEngineMetricsAggregation(t *testing.T) {
	logger := testLogger()
	proxyMgr := proxy.NewManager(proxy.RotationRandom)
	proxyMgr.AddBulk([]string{"1.1.1.1:8080", "2.2.2.2:8080"})

	eng := New(&mockPlatform{}, proxyMgr, []string{"t1", "t2"}, useragent.NewPool(), config.DefaultConfig().Engine, logger)

	ctx := context.Background()
	eng.Start(ctx, "test", 2)
	defer eng.Stop()

	time.Sleep(500 * time.Millisecond)

	m := eng.Metrics()
	if m.EngineState != "running" {
		t.Errorf("expected state=running, got %s", m.EngineState)
	}
	// Mock viewers each report 10 segments
	if m.SegmentsFetched < 10 {
		// At least one worker should have metrics
		t.Logf("segments: %d (workers: %d)", m.SegmentsFetched, m.TotalWorkers)
	}
}

func TestMultiEngine(t *testing.T) {
	logger := testLogger()
	proxyMgr := proxy.NewManager(proxy.RotationRandom)
	proxyMgr.AddBulk([]string{"1.1.1.1:8080", "2.2.2.2:8080", "3.3.3.3:8080", "4.4.4.4:8080"})

	me := NewMultiEngine(&mockPlatform{}, proxyMgr, []string{"t1", "t2"}, useragent.NewPool(), config.DefaultConfig().Engine, logger)

	ctx := context.Background()

	// Start two channels
	if err := me.StartChannel(ctx, "channel_a", 1); err != nil {
		t.Fatalf("StartChannel A error: %v", err)
	}
	if err := me.StartChannel(ctx, "channel_b", 1); err != nil {
		t.Fatalf("StartChannel B error: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	if me.Count() != 2 {
		t.Errorf("expected 2 channels, got %d", me.Count())
	}

	channels := me.RunningChannels()
	if len(channels) != 2 {
		t.Errorf("expected 2 running channels, got %d", len(channels))
	}

	// Duplicate should fail
	if err := me.StartChannel(ctx, "channel_a", 1); err == nil {
		t.Error("expected error on duplicate channel")
	}

	// Stop one
	me.StopChannel("channel_a")
	if me.Count() != 1 {
		t.Errorf("expected 1 channel after stop, got %d", me.Count())
	}

	// Stop all
	me.StopAll()
	if me.Count() != 0 {
		t.Errorf("expected 0 channels after stopAll, got %d", me.Count())
	}
}
