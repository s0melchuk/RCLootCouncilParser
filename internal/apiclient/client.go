// Package apiclient posts awards and roster entries to the RCLootCouncilApi
// ingest endpoints.
package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/s0melchuk/RCLootCouncilParser/internal/model"
)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type bulkAwardsRequest struct {
	Records []model.Award `json:"records"`
}

// PostAwards sends a batch of awards to POST /api/loot. Cloudflare Pages
// Functions accept either a single record or {"records": [...]}  — we
// always use the bulk shape here since the caller decides batching.
func (c *Client) PostAwards(awards []model.Award) error {
	if len(awards) == 0 {
		return nil
	}
	return c.post("/api/loot", bulkAwardsRequest{Records: awards})
}

type bulkPlayersRequest struct {
	Players []model.Player `json:"players"`
}

// PostPlayers upserts a batch of roster entries via POST /api/players.
func (c *Client) PostPlayers(players []model.Player) error {
	if len(players) == 0 {
		return nil
	}
	return c.post("/api/players", bulkPlayersRequest{Players: players})
}

func (c *Client) post(path string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("API returned %s for %s", resp.Status, path)
	}
	return nil
}
