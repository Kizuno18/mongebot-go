// Package api - IPC handlers for webhook CRUD and testing.
package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Kizuno18/mongebot-go/internal/engine"
)

var globalWebhookMgr *engine.WebhookManager

// SetWebhookManager sets the global webhook manager.
func SetWebhookManager(wm *engine.WebhookManager) {
	globalWebhookMgr = wm
}

// getWebhookHandler returns handlers for webhook.* methods.
func getWebhookHandler(method string) (handlerFunc, bool) {
	if globalWebhookMgr == nil {
		return nil, false
	}

	handlers := map[string]handlerFunc{
		"webhook.list":   handleWebhookList,
		"webhook.add":    handleWebhookAdd,
		"webhook.remove": handleWebhookRemove,
		"webhook.test":   handleWebhookTest,
	}

	h, ok := handlers[method]
	return h, ok
}

func handleWebhookList(_ context.Context, _ json.RawMessage) (any, error) {
	webhooks := globalWebhookMgr.ListWebhooks()
	// Mask sensitive fields
	type safeWebhook struct {
		ID      string              `json:"id"`
		Name    string              `json:"name"`
		Type    engine.WebhookType  `json:"type"`
		Enabled bool                `json:"enabled"`
		Events  []string            `json:"events"`
		HasURL  bool                `json:"hasUrl"`
	}

	var safe []safeWebhook
	for _, w := range webhooks {
		safe = append(safe, safeWebhook{
			ID:      w.ID,
			Name:    w.Name,
			Type:    w.Type,
			Enabled: w.Enabled,
			Events:  w.Events,
			HasURL:  w.URL != "",
		})
	}
	return safe, nil
}

func handleWebhookAdd(_ context.Context, params json.RawMessage) (any, error) {
	var cfg engine.WebhookConfig
	if err := json.Unmarshal(params, &cfg); err != nil {
		return nil, fmt.Errorf("invalid webhook config: %w", err)
	}
	if cfg.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if cfg.URL == "" && cfg.BotToken == "" {
		return nil, fmt.Errorf("URL or botToken is required")
	}
	if cfg.ID == "" {
		cfg.ID = fmt.Sprintf("wh-%d", len(globalWebhookMgr.ListWebhooks())+1)
	}

	globalWebhookMgr.AddWebhook(cfg)
	return map[string]string{"status": "added", "id": cfg.ID}, nil
}

type webhookRemoveParams struct {
	ID string `json:"id"`
}

func handleWebhookRemove(_ context.Context, params json.RawMessage) (any, error) {
	var p webhookRemoveParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	globalWebhookMgr.RemoveWebhook(p.ID)
	return map[string]string{"status": "removed"}, nil
}

type webhookTestParams struct {
	ID string `json:"id"`
}

func handleWebhookTest(ctx context.Context, params json.RawMessage) (any, error) {
	var p webhookTestParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// Send a test notification
	globalWebhookMgr.Notify(ctx, "test",
		"Test Notification",
		"This is a test message from MongeBot.",
		map[string]string{
			"Source": "MongeBot v2.0",
			"Type":   "Test",
		},
	)

	return map[string]string{"status": "test sent"}, nil
}
