// Package twitch - channel points and drops progress tracking via GQL.
package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ChannelPointsBalance holds the viewer's current channel points.
type ChannelPointsBalance struct {
	ChannelID string `json:"channelId"`
	Balance   int    `json:"balance"`
}

// DropsCampaign represents an active Twitch drops campaign.
type DropsCampaign struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	GameName    string    `json:"gameName"`
	Status      string    `json:"status"`
	StartDate   time.Time `json:"startDate"`
	EndDate     time.Time `json:"endDate"`
	Rewards     []DropReward `json:"rewards"`
}

// DropReward represents a single drop reward within a campaign.
type DropReward struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	MinutesRequired int     `json:"minutesRequired"`
	MinutesWatched  int     `json:"minutesWatched"`
	Progress        float64 `json:"progress"` // 0.0 - 1.0
	Claimed         bool    `json:"claimed"`
}

// DropsTracker monitors drops progress and channel points.
type DropsTracker struct {
	client *http.Client
	token  string
	logger interface{ Debug(string, ...any) }
}

// NewDropsTracker creates a drops/points tracker.
func NewDropsTracker(client *http.Client, token string) *DropsTracker {
	return &DropsTracker{client: client, token: token}
}

// GetChannelPoints fetches the current channel points balance for a channel.
func (dt *DropsTracker) GetChannelPoints(ctx context.Context, channelID string) (*ChannelPointsBalance, error) {
	gql := newGQLClient(dt.client, dt.token, "", "")

	query := fmt.Sprintf(`query {
		community {
			channel(id: "%s") {
				self {
					communityPoints {
						balance
					}
				}
			}
		}
	}`, channelID)

	body, status, err := gql.doRequest(ctx, map[string]string{"query": query})
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("GQL returned %d", status)
	}

	var result struct {
		Data struct {
			Community struct {
				Channel struct {
					Self struct {
						CommunityPoints struct {
							Balance int `json:"balance"`
						} `json:"communityPoints"`
					} `json:"self"`
				} `json:"channel"`
			} `json:"community"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing points: %w", err)
	}

	return &ChannelPointsBalance{
		ChannelID: channelID,
		Balance:   result.Data.Community.Channel.Self.CommunityPoints.Balance,
	}, nil
}

// GetDropsProgress fetches active drops campaigns and their progress.
func (dt *DropsTracker) GetDropsProgress(ctx context.Context) ([]DropsCampaign, error) {
	gql := newGQLClient(dt.client, dt.token, "", "")

	query := `query {
		currentUser {
			dropCampaigns {
				id
				name
				game { displayName }
				status
				startAt
				endAt
				timeBasedDrops {
					id
					name
					requiredMinutesWatched
					self {
						currentMinutesWatched
						dropInstanceID
						isClaimed
					}
				}
			}
		}
	}`

	body, status, err := gql.doRequest(ctx, map[string]string{"query": query})
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("GQL returned %d", status)
	}

	var result struct {
		Data struct {
			CurrentUser struct {
				DropCampaigns []struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Game   *struct {
						DisplayName string `json:"displayName"`
					} `json:"game"`
					Status  string `json:"status"`
					StartAt string `json:"startAt"`
					EndAt   string `json:"endAt"`
					TimeBasedDrops []struct {
						ID                     string `json:"id"`
						Name                   string `json:"name"`
						RequiredMinutesWatched int    `json:"requiredMinutesWatched"`
						Self                   *struct {
							CurrentMinutesWatched int    `json:"currentMinutesWatched"`
							DropInstanceID        string `json:"dropInstanceID"`
							IsClaimed             bool   `json:"isClaimed"`
						} `json:"self"`
					} `json:"timeBasedDrops"`
				} `json:"dropCampaigns"`
			} `json:"currentUser"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing drops: %w", err)
	}

	var campaigns []DropsCampaign
	for _, c := range result.Data.CurrentUser.DropCampaigns {
		campaign := DropsCampaign{
			ID:     c.ID,
			Name:   c.Name,
			Status: c.Status,
		}
		if c.Game != nil {
			campaign.GameName = c.Game.DisplayName
		}
		campaign.StartDate, _ = time.Parse(time.RFC3339, c.StartAt)
		campaign.EndDate, _ = time.Parse(time.RFC3339, c.EndAt)

		for _, d := range c.TimeBasedDrops {
			reward := DropReward{
				ID:              d.ID,
				Name:            d.Name,
				MinutesRequired: d.RequiredMinutesWatched,
			}
			if d.Self != nil {
				reward.MinutesWatched = d.Self.CurrentMinutesWatched
				reward.Claimed = d.Self.IsClaimed
				if d.RequiredMinutesWatched > 0 {
					reward.Progress = float64(d.Self.CurrentMinutesWatched) / float64(d.RequiredMinutesWatched)
					if reward.Progress > 1.0 {
						reward.Progress = 1.0
					}
				}
			}
			campaign.Rewards = append(campaign.Rewards, reward)
		}

		campaigns = append(campaigns, campaign)
	}

	return campaigns, nil
}

// ClaimDrop attempts to claim a completed drop reward.
func (dt *DropsTracker) ClaimDrop(ctx context.Context, dropInstanceID string) error {
	gql := newGQLClient(dt.client, dt.token, "", "")

	payload := map[string]any{
		"operationName": "DropsPage_ClaimDropRewards",
		"variables": map[string]any{
			"input": map[string]any{
				"dropInstanceID": dropInstanceID,
			},
		},
		"extensions": map[string]any{
			"persistedQuery": map[string]any{
				"version":    1,
				"sha256Hash": "a455deea71bdc9015f7c9571d4b98e04e0deee1c4fa8a3e69ef2115e20c3ff47",
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
