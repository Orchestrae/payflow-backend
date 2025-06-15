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

	"github.com/rs/zerolog/log"
)

// Client handles communication with the KoraPay API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string // This is the permanent key from our .env
	authToken  string // This is the temporary token from their /auth endpoint
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

// authenticate gets a temporary auth token from KoraPay.
// A senior engineer makes this thread-safe using a mutex.
func (c *Client) authenticate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// In a real high-throughput system, we would store the token and its expiry
	// to avoid re-authenticating on every single call. For MVP, this is safe and simple.

	authReq := AuthRequest{SecretKey: c.apiKey}
	reqBody, _ := json.Marshal(authReq)

	resp, err := c.httpClient.Post(c.baseURL+"/auth", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("korapay auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("korapay auth failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode korapay auth response: %w", err)
	}

	if authResp.Status != "success" {
		return fmt.Errorf("korapay auth was not successful: %s", authResp.Message)
	}

	c.authToken = authResp.Data.SecretKey
	log.Info().Msg("Successfully authenticated with KoraPay")
	return nil
}

// SendBulkPayout makes the final API call to disburse funds.
func (c *Client) SendBulkPayout(request BulkPayoutRequest) (*BulkPayoutResponse, error) {
	if err := c.authenticate(); err != nil {
		return nil, err
	}

	reqBody, _ := json.Marshal(request)

	req, err := http.NewRequest("POST", c.baseURL+"/payout/bulk", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("korapay bulk payout request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("korapay bulk payout failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var payoutResp BulkPayoutResponse
	if err := json.Unmarshal(bodyBytes, &payoutResp); err != nil {
		return nil, fmt.Errorf("failed to decode korapay payout response: %w", err)
	}

	return &payoutResp, nil
}
