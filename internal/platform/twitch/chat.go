// Package twitch - IRC Chat WebSocket connection for channel presence.
package twitch

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// ChatClient manages an IRC WebSocket connection to Twitch Chat.
type ChatClient struct {
	conn    *websocket.Conn
	token   string
	channel string
	nick    string
	logger  *slog.Logger
}

// NewChatClient creates a new Twitch IRC chat client.
func NewChatClient(token, channel string, logger *slog.Logger) *ChatClient {
	return &ChatClient{
		token:   token,
		channel: channel,
		nick:    "justinfan" + fmt.Sprintf("%d", time.Now().UnixMilli()%99999),
		logger:  logger.With("subsystem", "chat"),
	}
}

// Connect establishes the IRC WebSocket connection and joins the channel.
func (c *ChatClient) Connect(ctx context.Context) error {
	opts := &websocket.DialOptions{
		HTTPHeader: http.Header{
			"User-Agent": {"Mozilla/5.0"},
		},
	}

	conn, _, err := websocket.Dial(ctx, ChatURL, opts)
	if err != nil {
		return fmt.Errorf("chat dial failed: %w", err)
	}
	c.conn = conn

	// IRC handshake
	commands := []string{
		"CAP REQ :twitch.tv/tags twitch.tv/commands",
		fmt.Sprintf("PASS oauth:%s", c.token),
		fmt.Sprintf("NICK %s", c.nick),
		fmt.Sprintf("JOIN #%s", c.channel),
	}

	for _, cmd := range commands {
		if err := conn.Write(ctx, websocket.MessageText, []byte(cmd)); err != nil {
			conn.CloseNow()
			return fmt.Errorf("chat command failed: %w", err)
		}
	}

	c.logger.Debug("joined chat channel", "channel", c.channel, "nick", c.nick)

	// Start ping loop
	go c.pingLoop(ctx)

	// Message read loop
	return c.readLoop(ctx)
}

// readLoop reads incoming IRC messages.
func (c *ChatClient) readLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			c.conn.Close(websocket.StatusNormalClosure, "shutdown")
			return nil
		default:
		}

		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("chat read error: %w", err)
		}

		message := string(data)

		// Respond to PING to keep connection alive
		if strings.HasPrefix(message, "PING") {
			pong := strings.Replace(message, "PING", "PONG", 1)
			if err := c.conn.Write(ctx, websocket.MessageText, []byte(pong)); err != nil {
				return fmt.Errorf("PONG failed: %w", err)
			}
			continue
		}

		// Log interesting messages in debug mode
		if strings.Contains(message, "PRIVMSG") {
			c.logger.Debug("chat message received")
		}
	}
}

// pingLoop sends periodic PINGs to keep the IRC connection alive.
func (c *ChatClient) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(PingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.conn.Write(ctx, websocket.MessageText, []byte("PING")); err != nil {
				c.logger.Error("chat PING failed", "error", err)
				return
			}
		}
	}
}

// SendMessage sends a chat message to the channel (use sparingly).
func (c *ChatClient) SendMessage(ctx context.Context, message string) error {
	msg := fmt.Sprintf("PRIVMSG #%s :%s", c.channel, message)
	return c.conn.Write(ctx, websocket.MessageText, []byte(msg))
}
