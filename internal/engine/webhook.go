// Package engine - webhook notifications for Discord, Telegram, and generic HTTP.
// Fires on stream events, session lifecycle, and errors.
package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// WebhookType identifies the notification service.
type WebhookType string

const (
	WebhookDiscord  WebhookType = "discord"
	WebhookTelegram WebhookType = "telegram"
	WebhookGeneric  WebhookType = "generic" // Any HTTP POST endpoint
)

// WebhookConfig defines a notification endpoint.
type WebhookConfig struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Type    WebhookType `json:"type"`
	URL     string      `json:"url"`
	Enabled bool        `json:"enabled"`

	// Telegram-specific
	ChatID string `json:"chatId,omitempty"`
	BotToken string `json:"botToken,omitempty"`

	// Event filters — which events trigger this webhook
	Events []string `json:"events"` // "stream.online", "stream.offline", "session.end", "error"
}

// WebhookManager handles sending notifications to configured webhooks.
type WebhookManager struct {
	webhooks []WebhookConfig
	client   *http.Client
	logger   *slog.Logger
}

// NewWebhookManager creates a webhook manager.
func NewWebhookManager(logger *slog.Logger) *WebhookManager {
	return &WebhookManager{
		webhooks: make([]WebhookConfig, 0),
		client:   &http.Client{Timeout: 10 * time.Second},
		logger:   logger.With("component", "webhook"),
	}
}

// AddWebhook registers a new webhook endpoint.
func (wm *WebhookManager) AddWebhook(cfg WebhookConfig) {
	wm.webhooks = append(wm.webhooks, cfg)
}

// RemoveWebhook removes a webhook by ID.
func (wm *WebhookManager) RemoveWebhook(id string) {
	filtered := make([]WebhookConfig, 0, len(wm.webhooks))
	for _, w := range wm.webhooks {
		if w.ID != id {
			filtered = append(filtered, w)
		}
	}
	wm.webhooks = filtered
}

// ListWebhooks returns all configured webhooks.
func (wm *WebhookManager) ListWebhooks() []WebhookConfig {
	result := make([]WebhookConfig, len(wm.webhooks))
	copy(result, wm.webhooks)
	return result
}

// Notify sends a notification to all webhooks that match the event type.
func (wm *WebhookManager) Notify(ctx context.Context, eventType string, title string, message string, fields map[string]string) {
	for _, webhook := range wm.webhooks {
		if !webhook.Enabled || !wm.matchesEvent(webhook, eventType) {
			continue
		}

		go func(w WebhookConfig) {
			var err error
			switch w.Type {
			case WebhookDiscord:
				err = wm.sendDiscord(ctx, w, title, message, fields)
			case WebhookTelegram:
				err = wm.sendTelegram(ctx, w, title, message, fields)
			case WebhookGeneric:
				err = wm.sendGeneric(ctx, w, eventType, title, message, fields)
			}
			if err != nil {
				wm.logger.Warn("webhook failed", "name", w.Name, "type", w.Type, "error", err)
			}
		}(webhook)
	}
}

// sendDiscord sends a rich embed to a Discord webhook URL.
func (wm *WebhookManager) sendDiscord(ctx context.Context, cfg WebhookConfig, title, message string, fields map[string]string) error {
	// Build Discord embed
	embedFields := make([]map[string]any, 0, len(fields))
	for k, v := range fields {
		embedFields = append(embedFields, map[string]any{
			"name":   k,
			"value":  v,
			"inline": true,
		})
	}

	color := 3447003 // Blue
	if strings.Contains(title, "offline") || strings.Contains(title, "error") {
		color = 15158332 // Red
	} else if strings.Contains(title, "online") || strings.Contains(title, "started") {
		color = 3066993 // Green
	}

	payload := map[string]any{
		"embeds": []map[string]any{
			{
				"title":       fmt.Sprintf("⚡ %s", title),
				"description": message,
				"color":       color,
				"fields":      embedFields,
				"footer": map[string]string{
					"text": "MongeBot v2.0",
				},
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	}

	return wm.postJSON(ctx, cfg.URL, payload)
}

// sendTelegram sends a formatted message via the Telegram Bot API.
func (wm *WebhookManager) sendTelegram(ctx context.Context, cfg WebhookConfig, title, message string, fields map[string]string) error {
	// Build Telegram message with HTML formatting
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>⚡ %s</b>\n", title))
	sb.WriteString(fmt.Sprintf("%s\n", message))

	if len(fields) > 0 {
		sb.WriteString("\n")
		for k, v := range fields {
			sb.WriteString(fmt.Sprintf("• <b>%s:</b> %s\n", k, v))
		}
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	payload := map[string]any{
		"chat_id":    cfg.ChatID,
		"text":       sb.String(),
		"parse_mode": "HTML",
	}

	return wm.postJSON(ctx, apiURL, payload)
}

// sendGeneric sends a JSON POST to any HTTP endpoint.
func (wm *WebhookManager) sendGeneric(ctx context.Context, cfg WebhookConfig, eventType, title, message string, fields map[string]string) error {
	payload := map[string]any{
		"event":     eventType,
		"title":     title,
		"message":   message,
		"fields":    fields,
		"timestamp": time.Now().Format(time.RFC3339),
		"source":    "mongebot",
	}

	return wm.postJSON(ctx, cfg.URL, payload)
}

// postJSON sends a JSON payload via HTTP POST.
func (wm *WebhookManager) postJSON(ctx context.Context, url string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := wm.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

// matchesEvent checks if a webhook is configured to receive the given event type.
func (wm *WebhookManager) matchesEvent(cfg WebhookConfig, eventType string) bool {
	if len(cfg.Events) == 0 {
		return true // No filter = receive all events
	}
	for _, e := range cfg.Events {
		if e == eventType || e == "*" {
			return true
		}
	}
	return false
}
