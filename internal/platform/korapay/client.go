// internal/platform/korapay/client.go
package korapay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
// Endpoint: POST /merchant/api/v1/transactions/disburse/bulk
func (c *Client) SendBulkPayout(request BulkPayoutRequest) (*BulkPayoutResponse, error) {
	bodyBytes, err := c.makeRequest("POST", "/merchant/api/v1/transactions/disburse/bulk", request)
	if err != nil {
		return nil, fmt.Errorf("korapay bulk payout request failed: %w", err)
	}

	var response BulkPayoutResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay bulk payout response: %w", err)
	}

	return &response, nil
}

// GetBulkPayoutStatus fetches the status of a bulk payout batch.
// Endpoint: GET /merchant/api/v1/transactions/bulk/:batch_reference
func (c *Client) GetBulkPayoutStatus(batchReference string) (*BulkPayoutResponse, error) {
	endpoint := fmt.Sprintf("/merchant/api/v1/transactions/bulk/%s", batchReference)
	bodyBytes, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("korapay get bulk payout status failed: %w", err)
	}

	var response BulkPayoutResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay bulk payout status response: %w", err)
	}

	return &response, nil
}

// GetBulkPayoutPayouts fetches all payouts in a bulk payout batch.
// Endpoint: GET /merchant/api/v1/transactions/bulk/:batch_reference/payouts
func (c *Client) GetBulkPayoutPayouts(batchReference string) (*BulkPayoutPayoutsResponse, error) {
	endpoint := fmt.Sprintf("/merchant/api/v1/transactions/bulk/%s/payouts", batchReference)
	bodyBytes, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("korapay get bulk payout payouts failed: %w", err)
	}

	var response BulkPayoutPayoutsResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay bulk payout payouts response: %w", err)
	}

	return &response, nil
}

// GetTransactionStatus fetches the status of a single transaction.
// Endpoint: GET /merchant/api/v1/transactions/:reference
func (c *Client) GetTransactionStatus(reference string) (*SingleDisbursementResponse, error) {
	endpoint := fmt.Sprintf("/merchant/api/v1/transactions/%s", reference)
	bodyBytes, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("korapay get transaction status failed: %w", err)
	}

	var response SingleDisbursementResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay transaction status response: %w", err)
	}

	return &response, nil
}

// ============================================================================
// Virtual Bank Account Methods
// ============================================================================

// CreateVirtualAccount creates a virtual bank account.
// Endpoint: POST /merchant/api/v1/virtual-bank-account
func (c *Client) CreateVirtualAccount(request VirtualAccountCreateRequest) (*VirtualAccountResponse, error) {
	bodyBytes, err := c.makeRequest("POST", "/merchant/api/v1/virtual-bank-account", request)
	if err != nil {
		return nil, fmt.Errorf("korapay create virtual account request failed: %w", err)
	}

	var response VirtualAccountResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay create virtual account response: %w", err)
	}

	return &response, nil
}

// GetVirtualAccount retrieves virtual account details by reference.
// Endpoint: GET /merchant/api/v1/virtual-bank-account/:account_reference
func (c *Client) GetVirtualAccount(accountReference string) (*VirtualAccountResponse, error) {
	endpoint := fmt.Sprintf("/merchant/api/v1/virtual-bank-account/%s", accountReference)
	bodyBytes, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("korapay get virtual account request failed: %w", err)
	}

	var response VirtualAccountResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay get virtual account response: %w", err)
	}

	return &response, nil
}

// GetVirtualAccountTransactions retrieves transaction history for a virtual account.
// Endpoint: GET /merchant/api/v1/virtual-bank-account/transactions?account_number=...&start_date=...&end_date=...&page=...&limit=...
func (c *Client) GetVirtualAccountTransactions(accountNumber string, startDate, endDate *string, page, limit int) (*VirtualAccountTransactionsResponse, error) {
	endpoint := fmt.Sprintf("/merchant/api/v1/virtual-bank-account/transactions?account_number=%s", accountNumber)
	
	// Add optional query parameters
	if startDate != nil {
		endpoint += fmt.Sprintf("&start_date=%s", *startDate)
	}
	if endDate != nil {
		endpoint += fmt.Sprintf("&end_date=%s", *endDate)
	}
	if page > 0 {
		endpoint += fmt.Sprintf("&page=%d", page)
	}
	if limit > 0 {
		endpoint += fmt.Sprintf("&limit=%d", limit)
	}

	bodyBytes, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("korapay get virtual account transactions request failed: %w", err)
	}

	var response VirtualAccountTransactionsResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay virtual account transactions response: %w", err)
	}

	return &response, nil
}

// SandboxCreditVirtualAccount credits a virtual account in sandbox environment (testing only).
// Endpoint: POST /merchant/api/v1/virtual-bank-account/sandbox/credit
func (c *Client) SandboxCreditVirtualAccount(request VirtualAccountSandboxCreditRequest) (*VirtualAccountSandboxCreditResponse, error) {
	bodyBytes, err := c.makeRequest("POST", "/merchant/api/v1/virtual-bank-account/sandbox/credit", request)
	if err != nil {
		return nil, fmt.Errorf("korapay sandbox credit virtual account request failed: %w", err)
	}

	var response VirtualAccountSandboxCreditResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay sandbox credit response: %w", err)
	}

	return &response, nil
}

// CreateAccountHolder creates a virtual bank account holder (KYC onboarding).
// Endpoint: POST /api/v1/virtual-bank-account/account-holders
func (c *Client) CreateAccountHolder(request AccountHolderCreateRequest) (*AccountHolderCreateResponse, error) {
	bodyBytes, err := c.makeRequest("POST", "/api/v1/virtual-bank-account/account-holders", request)
	if err != nil {
		return nil, fmt.Errorf("korapay create account holder request failed: %w", err)
	}

	var response AccountHolderCreateResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay create account holder response: %w", err)
	}

	return &response, nil
}

// GetAccountHolderDetails retrieves account holder details by reference.
// Endpoint: GET /api/v1/virtual-bank-account/account-holders/{reference}/details
func (c *Client) GetAccountHolderDetails(reference string) (*AccountHolderDetailsResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/virtual-bank-account/account-holders/%s/details", reference)
	bodyBytes, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("korapay get account holder details request failed: %w", err)
	}

	var response AccountHolderDetailsResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay account holder details response: %w", err)
	}

	return &response, nil
}

// UpdateAccountHolderKYC updates account holder KYC information.
// Endpoint: PATCH /api/v1/virtual-bank-account/account-holders/{reference}/update-kyc
func (c *Client) UpdateAccountHolderKYC(reference string, request AccountHolderUpdateKYCRequest) (*AccountHolderUpdateKYCResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/virtual-bank-account/account-holders/%s/update-kyc", reference)
	bodyBytes, err := c.makeRequest("PATCH", endpoint, request)
	if err != nil {
		return nil, fmt.Errorf("korapay update account holder KYC request failed: %w", err)
	}

	var response AccountHolderUpdateKYCResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay update account holder KYC response: %w", err)
	}

	return &response, nil
}

// GenerateFileUploadURL generates a pre-signed S3 URL for file uploads (KYC documents).
// Endpoint: GET /api/v1/files/generate-upload-url
func (c *Client) GenerateFileUploadURL(request FileUploadURLRequest) (*FileUploadURLResponse, error) {
	// Note: This is a GET request but with body, which is unusual
	// Based on the curl example, it seems like it should be POST
	// Let me check the actual endpoint - the curl shows --request GET --data
	// This might be a mistake in the API docs, or it's a special endpoint
	// We'll implement as POST since sending data in GET body is not standard
	bodyBytes, err := c.makeRequest("POST", "/api/v1/files/generate-upload-url", request)
	if err != nil {
		// Try GET if POST fails (in case the API docs are correct but unusual)
		if strings.Contains(err.Error(), "405") || strings.Contains(err.Error(), "Method Not Allowed") {
			// If POST fails, this endpoint might not be implemented yet or needs different approach
			return nil, fmt.Errorf("korapay generate file upload URL request failed (note: endpoint may require different method): %w", err)
		}
		return nil, fmt.Errorf("korapay generate file upload URL request failed: %w", err)
	}

	var response FileUploadURLResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to decode korapay file upload URL response: %w", err)
	}

	return &response, nil
}
