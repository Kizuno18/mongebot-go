// Package twitch - GraphQL API client for Twitch GQL endpoint.
package twitch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
)

// gqlClient handles communication with the Twitch GQL API.
type gqlClient struct {
	client    *http.Client
	token     string
	userAgent string
	deviceID  string
	cb        CircuitBreakerExecutor // Optional circuit breaker
}

// CircuitBreakerExecutor is the interface for circuit breaker execution.
type CircuitBreakerExecutor interface {
	Execute(fn func() error) error
}

// newGQLClient creates a GQL client.
func newGQLClient(client *http.Client, token, userAgent, deviceID string) *gqlClient {
	return &gqlClient{
		client:    client,
		token:     token,
		userAgent: userAgent,
		deviceID:  deviceID,
	}
}

// setCircuitBreaker sets the circuit breaker for GQL requests.
func (g *gqlClient) setCircuitBreaker(cb CircuitBreakerExecutor) {
	g.cb = cb
}

// getStreamToken fetches the stream playback access token and signature.
func (g *gqlClient) getStreamToken(ctx context.Context, channel string) (token, signature string, status int, err error) {
	query := fmt.Sprintf(`query {
		streamPlaybackAccessToken(channelName: "%s",
		params: { platform: "web", playerBackend: "%s", playerType: "%s" })
		{ value, signature }
	}`, channel, PlayerBackend, PlayerType)

	payload := map[string]string{"query": query}
	body, statusCode, err := g.doRequest(ctx, payload)
	if err != nil {
		return "", "", statusCode, err
	}

	var result struct {
		Data struct {
			Token struct {
				Value     string `json:"value"`
				Signature string `json:"signature"`
			} `json:"streamPlaybackAccessToken"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", statusCode, fmt.Errorf("parsing stream token response: %w", err)
	}

	return result.Data.Token.Value, result.Data.Token.Signature, statusCode, nil
}

// getStreamMetadata fetches broadcast ID and channel ID.
func (g *gqlClient) getStreamMetadata(ctx context.Context, channel string) (broadcastID, channelID string, err error) {
	query := fmt.Sprintf(`query {
		user(login: "%s") {
			id
			stream { id }
		}
	}`, channel)

	payload := map[string]string{"query": query}
	body, status, err := g.doRequest(ctx, payload)
	if err != nil {
		return "", "", err
	}
	if status != 200 {
		return "", "", fmt.Errorf("GQL returned status %d", status)
	}

	var result struct {
		Data struct {
			User *struct {
				ID     string `json:"id"`
				Stream *struct {
					ID string `json:"id"`
				} `json:"stream"`
			} `json:"user"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("parsing metadata: %w", err)
	}

	if result.Data.User == nil {
		return "", "", fmt.Errorf("user %q not found", channel)
	}

	channelID = result.Data.User.ID
	if result.Data.User.Stream != nil {
		broadcastID = result.Data.User.Stream.ID
	}
	return broadcastID, channelID, nil
}

// getAuthenticatedUserID fetches the user ID of the authenticated account.
func (g *gqlClient) getAuthenticatedUserID(ctx context.Context) (string, int, error) {
	payload := map[string]string{"query": "query { currentUser { id } }"}
	body, status, err := g.doRequest(ctx, payload)
	if err != nil {
		return "", status, err
	}

	var result struct {
		Data struct {
			CurrentUser *struct {
				ID string `json:"id"`
			} `json:"currentUser"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", status, err
	}

	if result.Data.CurrentUser == nil {
		return "", status, nil
	}
	return result.Data.CurrentUser.ID, status, nil
}

// sendWatchTrackQuery sends a GQL heartbeat pulse.
func (g *gqlClient) sendWatchTrackQuery(ctx context.Context, channel string) error {
	payload := map[string]any{
		"operationName": "WatchTrackQuery",
		"variables": map[string]any{
			"channelLogin": channel,
			"videoID":      nil,
			"hasVideoID":   false,
		},
		"extensions": map[string]any{
			"persistedQuery": map[string]any{
				"version":    1,
				"sha256Hash": gqlOperations["WatchTrackQuery"],
			},
		},
	}

	_, status, err := g.doRequest(ctx, payload)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("WatchTrackQuery returned status %d", status)
	}
	return nil
}

// doRequest executes a GQL request and returns the response body.
func (g *gqlClient) doRequest(ctx context.Context, payload any) ([]byte, int, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("marshaling GQL payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", GQLURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, 0, fmt.Errorf("creating GQL request: %w", err)
	}

	setGQLHeaders(req, g.token, g.userAgent, g.deviceID)

	var resp *http.Response
	var body []byte
	var statusCode int

	doReq := func() error {
		var reqErr error
		resp, reqErr = g.client.Do(req)
		if reqErr != nil {
			return reqErr
		}
		defer resp.Body.Close()

		body, reqErr = io.ReadAll(resp.Body)
		if reqErr != nil {
			return fmt.Errorf("reading GQL response: %w", reqErr)
		}
		statusCode = resp.StatusCode
		return nil
	}

	// Use circuit breaker if available
	if g.cb != nil {
		if err := g.cb.Execute(doReq); err != nil {
			return nil, 0, err
		}
	} else {
		if err := doReq(); err != nil {
			return nil, 0, err
		}
	}

	return body, statusCode, nil
}

// setGQLHeaders sets the standard Twitch GQL request headers.
func setGQLHeaders(req *http.Request, token, userAgent, deviceID string) {
	req.Header.Set("Client-Id", PCClientID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/plain")

	if token != "" {
		req.Header.Set("Authorization", "OAuth "+token)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	if deviceID != "" {
		req.Header.Set("X-Device-Id", deviceID)
	}

	req.Header.Set("Origin", origins[rand.IntN(len(origins))])
	req.Header.Set("DNT", "1")
	req.Header.Set("Cache-Control", "no-cache")
}

// stringReader creates a strings.Reader for request bodies.
func stringReader(s string) *strings.Reader {
	return strings.NewReader(s)
}
