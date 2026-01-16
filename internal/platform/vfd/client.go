// internal/platform/vfd/client.go
package vfd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Client is a low-level client for interacting with the VFD Bank API.
type Client struct {
	httpClient     *http.Client
	baseURL        string
	consumerKey    string
	consumerSecret string

	mu          sync.RWMutex
	accessToken string
}

// NewClient creates and configures a new VFD API client.
func NewClient(baseURL, consumerKey, consumerSecret string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:        baseURL,
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
	}
}

// ============================================================================
// Authentication
// ============================================================================

// GetAccessToken retrieves a valid access token, fetching a new one if needed.
func (c *Client) GetAccessToken(ctx context.Context) (string, error) {
	// Try reading with read lock first (optimistic path)
	c.mu.RLock()
	if c.accessToken != "" {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	// Need to fetch new token
	return c.fetchNewToken(ctx)
}

// fetchNewToken fetches a new access token from VFD.
func (c *Client) fetchNewToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.accessToken != "" {
		return c.accessToken, nil
	}

	reqBody := TokenRequest{
		ConsumerKey:    c.consumerKey,
		ConsumerSecret: c.consumerSecret,
		ValidityTime:   "-1", // Non-expiring token
	}

	var tokenData TokenResponseData
	err := c.doRequest(ctx, http.MethodPost, "/baasauth/token", nil, reqBody, &tokenData)
	if err != nil {
		return "", fmt.Errorf("failed to get VFD access token: %w", err)
	}

	c.accessToken = tokenData.AccessToken
	slog.Info("VFD access token obtained successfully")
	return c.accessToken, nil
}

// ============================================================================
// API Request Helper
// ============================================================================

// doRequest executes an HTTP request against the VFD API.
func (c *Client) doRequest(ctx context.Context, method, path string, headers map[string]string, body, result interface{}) error {
	url := c.baseURL + path

	// Marshal request body if provided
	var reqBody io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(bodyBytes)
		slog.Debug("VFD API request", "method", method, "path", path, "body", string(bodyBytes))
	} else {
		slog.Debug("VFD API request", "method", method, "path", path)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	slog.Debug("VFD API response",
		"status_code", resp.StatusCode,
		"body_length", len(respBody),
		"body", string(respBody),
	)

	// Handle 202 Accepted - VFD returns this when credentials don't have proper access
	if resp.StatusCode == 202 {
		if len(respBody) == 0 {
			return fmt.Errorf("VFD API returned 202 Accepted with no response body - this typically means the API credentials don't have access to this endpoint or no wallet is configured")
		}
	}

	// Handle non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("VFD API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Handle empty response
	if len(respBody) == 0 {
		return fmt.Errorf("VFD API returned empty response - check API credentials and wallet configuration")
	}

	// Parse response
	var vfdResp VFDResponse
	vfdResp.Data = result
	if err := json.Unmarshal(respBody, &vfdResp); err != nil {
		return fmt.Errorf("failed to parse VFD response: %w", err)
	}

	// Check VFD business-level status
	if err := c.checkVFDStatus(vfdResp, path); err != nil {
		return err
	}

	return nil
}

// checkVFDStatus checks the VFD response status and returns appropriate errors.
func (c *Client) checkVFDStatus(resp VFDResponse, path string) error {
	// Success status codes
	if resp.Status == "00" || resp.Status == "success" {
		return nil
	}

	// Map VFD error codes to our errors
	switch resp.Status {
	case "02":
		return fmt.Errorf("VFD signature mismatch")
	case "98":
		if resp.Message == "Transaction Exist" {
			return fmt.Errorf("VFD transaction already exists")
		}
		if resp.Message == "Invalid uniqueSenderAccountId" {
			return fmt.Errorf("VFD invalid sender account ID")
		}
		return fmt.Errorf("VFD error (98): %s", resp.Message)
	case "99":
		return fmt.Errorf("VFD transaction failed: %s", resp.Message)
	case "104":
		return fmt.Errorf("VFD account not found")
	case "108":
		return fmt.Errorf("VFD no transaction found")
	case "199":
		return fmt.Errorf("VFD transaction/session ID is mandatory")
	case "500":
		return fmt.Errorf("VFD internal server error")
	default:
		return fmt.Errorf("VFD error (status %s): %s", resp.Status, resp.Message)
	}
}

// ============================================================================
// Authenticated Request Helper
// ============================================================================

// doAuthenticatedRequest executes an authenticated request (includes AccessToken header).
func (c *Client) doAuthenticatedRequest(ctx context.Context, method, path string, body, result interface{}) error {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return err
	}

	headers := map[string]string{
		"AccessToken": token,
	}

	return c.doRequest(ctx, method, path, headers, body, result)
}
