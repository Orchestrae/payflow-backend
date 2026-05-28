package paystack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	client := NewClient("sk_test_xxx", server.URL)
	return client, server
}

func TestCreateTransferRecipient_Success(t *testing.T) {
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/transferrecipient" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk_test_xxx" {
			t.Error("missing or incorrect authorization header")
		}

		var req CreateRecipientRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.AccountNumber != "0123456789" {
			t.Errorf("expected account 0123456789, got %s", req.AccountNumber)
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(CreateRecipientResponse{
			Status:  true,
			Message: "Transfer recipient created",
			Data:    &RecipientData{RecipientCode: "RCP_test123", Name: "John Doe"},
		})
	})
	defer server.Close()

	resp, err := client.CreateTransferRecipient(context.Background(), CreateRecipientRequest{
		Type:          "nuban",
		Name:          "John Doe",
		AccountNumber: "0123456789",
		BankCode:      "058",
		Currency:      "NGN",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Status {
		t.Error("expected status true")
	}
	if resp.Data.RecipientCode != "RCP_test123" {
		t.Errorf("expected recipient code RCP_test123, got %s", resp.Data.RecipientCode)
	}
}

func TestCreateTransferRecipient_Error(t *testing.T) {
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"status":false,"message":"Invalid bank code"}`))
	})
	defer server.Close()

	_, err := client.CreateTransferRecipient(context.Background(), CreateRecipientRequest{
		Type: "nuban", BankCode: "999",
	})
	if err == nil {
		t.Error("expected error for 400 response")
	}
}

func TestInitiateTransfer_Success(t *testing.T) {
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transfer" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req TransferRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Amount != 500000 {
			t.Errorf("expected amount 500000, got %d", req.Amount)
		}
		if req.Reference != "ref_001" {
			t.Errorf("expected reference ref_001, got %s", req.Reference)
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(TransferResponse{
			Status:  true,
			Message: "Transfer has been queued",
			Data: &TransferData{
				TransferCode: "TRF_abc123",
				Reference:    "ref_001",
				Status:       "pending",
				Amount:       500000,
				Currency:     "NGN",
			},
		})
	})
	defer server.Close()

	resp, err := client.InitiateTransfer(context.Background(), TransferRequest{
		Source:    "balance",
		Amount:    500000,
		Recipient: "RCP_test123",
		Reference: "ref_001",
		Reason:    "Salary",
		Currency:  "NGN",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Status {
		t.Error("expected status true")
	}
	if resp.Data.TransferCode != "TRF_abc123" {
		t.Errorf("expected transfer code TRF_abc123, got %s", resp.Data.TransferCode)
	}
}

func TestInitiateTransfer_Error(t *testing.T) {
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"status":false,"message":"Insufficient balance"}`))
	})
	defer server.Close()

	_, err := client.InitiateTransfer(context.Background(), TransferRequest{
		Source: "balance", Amount: 999999999,
	})
	if err == nil {
		t.Error("expected error for insufficient balance")
	}
}

func TestInitiateBulkTransfer_Success(t *testing.T) {
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/transfer/bulk" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req BulkTransferRequest
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Transfers) != 2 {
			t.Errorf("expected 2 transfers, got %d", len(req.Transfers))
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(BulkTransferResponse{
			Status:  true,
			Message: "Bulk transfer queued",
		})
	})
	defer server.Close()

	resp, err := client.InitiateBulkTransfer(context.Background(), BulkTransferRequest{
		Source:   "balance",
		Currency: "NGN",
		Transfers: []BulkTransferItem{
			{Amount: 100000, Recipient: "RCP_1", Reference: "ref_1"},
			{Amount: 200000, Recipient: "RCP_2", Reference: "ref_2"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Status {
		t.Error("expected status true")
	}
}

func TestResolveAccountNumber_Success(t *testing.T) {
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("account_number") != "0123456789" {
			t.Errorf("expected account_number query param")
		}

		w.WriteHeader(200)
		json.NewEncoder(w).Encode(ResolveAccountResponse{
			Status: true,
			Data: &ResolveAccountData{
				AccountNumber: "0123456789",
				AccountName:   "JOHN DOE",
			},
		})
	})
	defer server.Close()

	resp, err := client.ResolveAccountNumber(context.Background(), "0123456789", "058")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Data.AccountName != "JOHN DOE" {
		t.Errorf("expected account name JOHN DOE, got %s", resp.Data.AccountName)
	}
}

func TestMakeRequest_AuthHeader(t *testing.T) {
	var capturedAuth string
	client, server := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":true}`))
	})
	defer server.Close()

	client.makeRequest(context.Background(), "GET", "/test", nil)
	if capturedAuth != "Bearer sk_test_xxx" {
		t.Errorf("expected 'Bearer sk_test_xxx', got '%s'", capturedAuth)
	}
}
