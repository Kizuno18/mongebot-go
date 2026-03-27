// Package twitch - channel search via GQL for autocomplete.
package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ChannelResult represents a search result for channel lookup.
type ChannelResult struct {
	Login       string `json:"login"`
	DisplayName string `json:"displayName"`
	ID          string `json:"id"`
	IsLive      bool   `json:"isLive"`
	GameName    string `json:"gameName,omitempty"`
	ViewerCount int    `json:"viewerCount"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
}

// SearchChannels queries Twitch GQL to find channels matching a query string.
func SearchChannels(ctx context.Context, client *http.Client, query string, token string, limit int) ([]ChannelResult, error) {
	if limit <= 0 || limit > 25 {
		limit = 10
	}

	gqlQuery := fmt.Sprintf(`query {
		searchFor(userQuery: "%s", options: { targets: [{index: CHANNEL}] }) {
			channels {
				items {
					id
					login
					displayName
					profileImageURL(width: 70)
					stream {
						id
						game { name }
						viewersCount
					}
				}
			}
		}
	}`, query)

	payload := map[string]string{"query": gqlQuery}
	gql := newGQLClient(client, token, "", "")

	body, status, err := gql.doRequest(ctx, payload)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("search returned status %d", status)
	}

	var result struct {
		Data struct {
			SearchFor struct {
				Channels struct {
					Items []struct {
						ID              string `json:"id"`
						Login           string `json:"login"`
						DisplayName     string `json:"displayName"`
						ProfileImageURL string `json:"profileImageURL"`
						Stream          *struct {
							ID           string `json:"id"`
							ViewersCount int    `json:"viewersCount"`
							Game         *struct {
								Name string `json:"name"`
							} `json:"game"`
						} `json:"stream"`
					} `json:"items"`
				} `json:"channels"`
			} `json:"searchFor"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}

	var channels []ChannelResult
	for _, item := range result.Data.SearchFor.Channels.Items {
		ch := ChannelResult{
			Login:       item.Login,
			DisplayName: item.DisplayName,
			ID:          item.ID,
			AvatarURL:   item.ProfileImageURL,
		}
		if item.Stream != nil {
			ch.IsLive = true
			ch.ViewerCount = item.Stream.ViewersCount
			if item.Stream.Game != nil {
				ch.GameName = item.Stream.Game.Name
			}
		}
		channels = append(channels, ch)
		if len(channels) >= limit {
			break
		}
	}

	return channels, nil
}
