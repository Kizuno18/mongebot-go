// Package stream manages FFmpeg-based RTMP restreaming to Twitch/Kick/YouTube.
package stream

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
)

// State represents the stream process state.
type State int

const (
	StreamStopped State = iota
	StreamStarting
	StreamLive
	StreamError
)

// String returns a human-readable state.
func (s State) String() string {
	names := [...]string{"stopped", "starting", "live", "error"}
	if int(s) < len(names) {
		return names[s]
	}
	return "unknown"
}

// QualityPreset defines video encoding parameters.
type QualityPreset struct {
	Name       string `json:"name"`
	Resolution string `json:"resolution"` // e.g., "1920x1080"
	Bitrate    string `json:"bitrate"`    // e.g., "4500k"
	MaxRate    string `json:"maxRate"`    // e.g., "5000k"
	BufSize    string `json:"bufSize"`    // e.g., "10000k"
	FPS        int    `json:"fps"`
	Preset     string `json:"preset"`     // ultrafast, veryfast, fast, medium
	AudioRate  string `json:"audioRate"`  // e.g., "128k"
}

// Presets is the built-in quality presets.
var Presets = map[string]QualityPreset{
	"potato": {
		Name: "Potato (Low CPU)", Resolution: "320x180",
		Bitrate: "100k", MaxRate: "100k", BufSize: "200k",
		FPS: 10, Preset: "ultrafast", AudioRate: "32k",
	},
	"low": {
		Name: "Low", Resolution: "640x360",
		Bitrate: "800k", MaxRate: "900k", BufSize: "1800k",
		FPS: 24, Preset: "veryfast", AudioRate: "64k",
	},
	"medium": {
		Name: "Medium", Resolution: "1280x720",
		Bitrate: "2500k", MaxRate: "3000k", BufSize: "6000k",
		FPS: 30, Preset: "fast", AudioRate: "128k",
	},
	"high": {
		Name: "High", Resolution: "1920x1080",
		Bitrate: "4500k", MaxRate: "5000k", BufSize: "10000k",
		FPS: 30, Preset: "fast", AudioRate: "160k",
	},
	"ultra": {
		Name: "Ultra", Resolution: "1920x1080",
		Bitrate: "6000k", MaxRate: "7000k", BufSize: "14000k",
		FPS: 60, Preset: "medium", AudioRate: "192k",
	},
}

// Config holds all parameters for a restream session.
type Config struct {
	InputFile  string        `json:"inputFile"`
	StreamKey  string        `json:"streamKey"`
	RTMPURL    string        `json:"rtmpUrl"`    // e.g., "rtmp://live.twitch.tv/app"
	Quality    QualityPreset `json:"quality"`
	Loop       bool          `json:"loop"`
	ProxyURL   string        `json:"proxyUrl,omitempty"`
}

// Manager handles FFmpeg restream processes.
type Manager struct {
	mu     sync.Mutex
	state  State
	cmd    *exec.Cmd
	cancel context.CancelFunc
	logger *slog.Logger
}

// NewManager creates a new stream manager.
func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		state:  StreamStopped,
		logger: logger.With("component", "stream"),
	}
}

// Start begins an FFmpeg restream with the given configuration.
func (m *Manager) Start(ctx context.Context, cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StreamLive {
		return fmt.Errorf("stream already running")
	}

	if cfg.RTMPURL == "" {
		cfg.RTMPURL = "rtmp://live.twitch.tv/app"
	}

	args := m.buildFFmpegArgs(cfg)
	ctx, m.cancel = context.WithCancel(ctx)

	m.cmd = exec.CommandContext(ctx, "ffmpeg", args...)
	m.state = StreamStarting

	m.logger.Info("starting restream",
		"input", cfg.InputFile,
		"quality", cfg.Quality.Name,
		"rtmpUrl", cfg.RTMPURL,
	)

	go func() {
		err := m.cmd.Run()
		m.mu.Lock()
		defer m.mu.Unlock()

		if err != nil {
			m.logger.Error("ffmpeg exited", "error", err)
			m.state = StreamError
		} else {
			m.state = StreamStopped
		}

		// Auto-restart if loop is enabled and context isn't cancelled
		if cfg.Loop && ctx.Err() == nil {
			m.logger.Info("restarting stream (loop enabled)")
			m.state = StreamStopped
			go m.Start(ctx, cfg)
		}
	}()

	m.state = StreamLive
	return nil
}

// Stop stops the current restream.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
	}
	m.state = StreamStopped
	m.logger.Info("stream stopped")
}

// GetState returns the current stream state.
func (m *Manager) GetState() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// buildFFmpegArgs constructs the FFmpeg command arguments.
func (m *Manager) buildFFmpegArgs(cfg Config) []string {
	q := cfg.Quality

	args := []string{
		"-re",
	}

	// Input looping
	if cfg.Loop {
		args = append(args, "-stream_loop", "-1")
	}

	args = append(args,
		"-i", cfg.InputFile,
		"-c:v", "libx264",
		"-preset", q.Preset,
		"-b:v", q.Bitrate,
		"-maxrate", q.MaxRate,
		"-bufsize", q.BufSize,
		"-s", q.Resolution,
		"-r", fmt.Sprintf("%d", q.FPS),
		"-c:a", "aac",
		"-b:a", q.AudioRate,
		"-ar", "44100",
		"-ac", "2",
		"-f", "flv",
	)

	// Proxy support
	if cfg.ProxyURL != "" {
		args = append(args, "-http_proxy", cfg.ProxyURL)
	}

	// Output
	args = append(args, fmt.Sprintf("%s/%s", cfg.RTMPURL, cfg.StreamKey))

	return args
}

// GetPresets returns available quality presets.
func GetPresets() map[string]QualityPreset {
	result := make(map[string]QualityPreset)
	for k, v := range Presets {
		result[k] = v
	}
	return result
}
