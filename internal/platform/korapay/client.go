// internal/platform/korapay/client.go
package korapay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client handles communication with the KoraPay API.
// The API key is used directly as a Bearer token (no auth endpoint needed).
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string // This is the permanent key from our .env, used directly as Bearer token
	mu         sync.Mutex
}

// NewClient creates a new KoraPay client.
func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		apiKey:     apiKey,
	}
}

// makeRequest is a helper method to make HTTP requests with Bearer token authentication.
// This centralizes the common HTTP request logic to avoid duplication (DRY).
func (c *Client) makeRequest(method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := http.NewRequest(method, c.baseURL+endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("korapay request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("korapay request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return bodyBytes, nil
}

// SendSingleDisbursement makes a single disbursement API call.
// Endpoint: POST /merchant/api/v1/transactions/disburse
func (c *Client) SendSingleDisbursement(request SingleDisbursementRequest) (*SingleDisbursementResponse, error) {
	bodyBytes, err := c.makeRequest("POST", "/merchant/api/v1/transactions/disburse", request)
	if err != nil {
		return nil, fmt.Errorf("korapay single disbursement request failed: %w", err)
	}

	var response SingleDisbursementResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay single disbursement response: %w", err)
	}

	return &response, nil
}

// SendBulkPayout makes a bulk disbursement API call.
// Endpoint: POST /api/v1/transactions/disburse/bulk
func (c *Client) SendBulkPayout(request BulkPayoutRequest) (*BulkPayoutResponse, error) {
	bodyBytes, err := c.makeRequest("POST", "/api/v1/transactions/disburse/bulk", request)
	if err != nil {
		return nil, fmt.Errorf("korapay bulk payout request failed: %w", err)
	}

	var response BulkPayoutResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay bulk payout response: %w", err)
	}

	return &response, nil
}
