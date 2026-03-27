// Package twitch - PubSub WebSocket connection for stream events and ad detection.
package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// pubsubMessage represents a PubSub protocol message.
type pubsubMessage struct {
	Type  string          `json:"type"`
	Nonce string          `json:"nonce,omitempty"`
	Data  *pubsubData     `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

type pubsubData struct {
	Topics    []string `json:"topics,omitempty"`
	AuthToken string   `json:"auth_token,omitempty"`
	Topic     string   `json:"topic,omitempty"`
	Message   string   `json:"message,omitempty"`
}

// PubSubClient manages a WebSocket connection to Twitch PubSub.
type PubSubClient struct {
	conn      *websocket.Conn
	token     string
	channelID string
	logger    *slog.Logger
	onAd      func() // Callback when ad is detected
	onStream  func(string, int) // Callback for stream events (type, viewers)
}

// NewPubSubClient creates a PubSub client for the given channel.
func NewPubSubClient(token, channelID string, logger *slog.Logger) *PubSubClient {
	return &PubSubClient{
		token:     token,
		channelID: channelID,
		logger:    logger.With("subsystem", "pubsub"),
	}
}

// OnAd sets the callback for ad detection events.
func (p *PubSubClient) OnAd(fn func()) {
	p.onAd = fn
}

// OnStreamEvent sets the callback for stream status changes.
func (p *PubSubClient) OnStreamEvent(fn func(eventType string, viewers int)) {
	p.onStream = fn
}

// Connect establishes the PubSub WebSocket connection and subscribes to topics.
func (p *PubSubClient) Connect(ctx context.Context, proxyURL string) error {
	opts := &websocket.DialOptions{
		HTTPHeader: http.Header{
			"User-Agent": {"Mozilla/5.0"},
		},
	}

	conn, _, err := websocket.Dial(ctx, PubSubURL, opts)
	if err != nil {
		return fmt.Errorf("PubSub dial failed: %w", err)
	}
	p.conn = conn

	// Subscribe to ad events
	listenMsg := pubsubMessage{
		Type:  "LISTEN",
		Nonce: "mongebot-ads",
		Data: &pubsubData{
			Topics:    []string{fmt.Sprintf("ads.%s", p.channelID)},
			AuthToken: fmt.Sprintf("Bearer %s", p.token),
		},
	}

	if err := wsjson.Write(ctx, conn, listenMsg); err != nil {
		conn.CloseNow()
		return fmt.Errorf("PubSub subscribe failed: %w", err)
	}

	p.logger.Debug("subscribed to PubSub topics", "channelId", p.channelID)

	// Start ping loop
	go p.pingLoop(ctx)

	// Message read loop
	return p.readLoop(ctx)
}

// readLoop processes incoming PubSub messages.
func (p *PubSubClient) readLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			p.conn.Close(websocket.StatusNormalClosure, "shutdown")
			return nil
		default:
		}

		var msg pubsubMessage
		if err := wsjson.Read(ctx, p.conn, &msg); err != nil {
			return fmt.Errorf("PubSub read error: %w", err)
		}

		switch msg.Type {
		case "MESSAGE":
			p.handleMessage(msg)
		case "PONG":
			p.logger.Debug("PubSub PONG received")
		case "RECONNECT":
			p.logger.Warn("PubSub RECONNECT requested")
			return fmt.Errorf("server requested reconnect")
		case "RESPONSE":
			if msg.Error != "" {
				p.logger.Error("PubSub subscription error", "error", msg.Error)
			}
		}
	}
}

// handleMessage processes a PubSub MESSAGE event.
func (p *PubSubClient) handleMessage(msg pubsubMessage) {
	if msg.Data == nil {
		return
	}

	topic := msg.Data.Topic

	// Ad events
	if topic == fmt.Sprintf("ads.%s", p.channelID) {
		p.logger.Debug("ad event detected")
		if p.onAd != nil {
			p.onAd()
		}
		return
	}

	// Stream playback events
	if msg.Data.Message != "" {
		var playback struct {
			Type    string `json:"type"`
			Viewers int    `json:"viewers"`
		}
		if err := json.Unmarshal([]byte(msg.Data.Message), &playback); err == nil {
			if p.onStream != nil {
				p.onStream(playback.Type, playback.Viewers)
			}
		}
	}
}

// pingLoop sends PING messages every 4 minutes to keep the connection alive.
func (p *PubSubClient) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(PingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ping := pubsubMessage{Type: "PING"}
			if err := wsjson.Write(ctx, p.conn, ping); err != nil {
				p.logger.Error("PubSub PING failed", "error", err)
				return
			}
			p.logger.Debug("PubSub PING sent")
		}
	}
}
