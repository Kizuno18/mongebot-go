// Package twitch - auto-claim channel points bonus via GQL mutation.
// Detects when the bonus (click-to-claim) is available and automatically claims it.
package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// PointsAutoClaimConfig controls the auto-claim behavior.
type PointsAutoClaimConfig struct {
	Enabled  bool          `json:"enabled"`
	Interval time.Duration `json:"interval"` // How often to check for bonus
}

// PointsClaimer monitors and auto-claims channel points bonuses.
type PointsClaimer struct {
	client    *http.Client
	token     string
	channelID string
	logger    *slog.Logger
	config    PointsAutoClaimConfig
}

// NewPointsClaimer creates a channel points auto-claimer.
func NewPointsClaimer(client *http.Client, token, channelID string, cfg PointsAutoClaimConfig, logger *slog.Logger) *PointsClaimer {
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Minute
	}
	return &PointsClaimer{
		client:    client,
		token:     token,
		channelID: channelID,
		logger:    logger.With("subsystem", "points-claimer"),
		config:    cfg,
	}
}

// Run starts the auto-claim loop. Blocks until context is cancelled.
func (pc *PointsClaimer) Run(ctx context.Context) error {
	if !pc.config.Enabled {
		return nil
	}

	pc.logger.Info("auto-claim started", "channelId", pc.channelID, "interval", pc.config.Interval)

	ticker := time.NewTicker(pc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			pc.checkAndClaim(ctx)
		}
	}
}

// checkAndClaim checks for available bonus and claims it.
func (pc *PointsClaimer) checkAndClaim(ctx context.Context) {
	claimID, err := pc.getAvailableBonus(ctx)
	if err != nil {
		pc.logger.Debug("bonus check failed", "error", err)
		return
	}

	if claimID == "" {
		return // No bonus available
	}

	if err := pc.claimBonus(ctx, claimID); err != nil {
		pc.logger.Warn("bonus claim failed", "claimId", claimID, "error", err)
		return
	}

	pc.logger.Info("channel points bonus claimed!", "claimId", claimID)
}

// getAvailableBonus checks if a click-to-claim bonus is available.
func (pc *PointsClaimer) getAvailableBonus(ctx context.Context) (string, error) {
	gql := newGQLClient(pc.client, pc.token, "", "")

	query := fmt.Sprintf(`query {
		community {
			channel(id: "%s") {
				self {
					communityPoints {
						availableClaim {
							id
						}
					}
				}
			}
		}
	}`, pc.channelID)

	body, status, err := gql.doRequest(ctx, map[string]string{"query": query})
	if err != nil {
		return "", err
	}
	if status != 200 {
		return "", fmt.Errorf("GQL returned %d", status)
	}

	var result struct {
		Data struct {
			Community struct {
				Channel struct {
					Self struct {
						CommunityPoints struct {
							AvailableClaim *struct {
								ID string `json:"id"`
							} `json:"availableClaim"`
						} `json:"communityPoints"`
					} `json:"self"`
				} `json:"channel"`
			} `json:"community"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	claim := result.Data.Community.Channel.Self.CommunityPoints.AvailableClaim
	if claim == nil {
		return "", nil
	}
	return claim.ID, nil
}

// claimBonus executes the GQL mutation to claim the bonus.
func (pc *PointsClaimer) claimBonus(ctx context.Context, claimID string) error {
	gql := newGQLClient(pc.client, pc.token, "", "")

	payload := map[string]any{
		"operationName": "ClaimCommunityPoints",
		"variables": map[string]any{
			"input": map[string]any{
				"channelID": pc.channelID,
				"claimID":   claimID,
			},
		},
		"extensions": map[string]any{
			"persistedQuery": map[string]any{
				"version":    1,
				"sha256Hash": "46aaeebe02c99afdf4fc97c7c0cba964124bf6b0af229f7571a3d6d0010e4f20",
			},
		},
	}

	_, status, err := gql.doRequest(ctx, payload)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("claim returned %d", status)
	}
	return nil
}
