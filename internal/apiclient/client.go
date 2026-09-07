// Package apiclient posts awards to the RCLootCouncilApi ingest endpoint.
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

type bulkRequest struct {
	Records []model.Award `json:"records"`
}

// PostAwards sends a batch of awards to POST /api/loot. Cloudflare Pages
// Functions accept either a single record or {"records": [...]}  — we
// always use the bulk shape here since the caller decides batching.
func (c *Client) PostAwards(awards []model.Award) error {
	if len(awards) == 0 {
		return nil
	}
	body, err := json.Marshal(bulkRequest{Records: awards})
	if err != nil {
		return fmt.Errorf("marshal awards: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/loot", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("API returned %s", resp.Status)
	}
	return nil
}
