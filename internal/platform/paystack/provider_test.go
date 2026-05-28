package paystack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"payflow/internal/domain"
)

func TestProviderName(t *testing.T) {
	p := NewTransferProvider(NewClient("key", "http://localhost"))
	if p.Name() != domain.ProviderPaystack {
		t.Errorf("expected %s, got %s", domain.ProviderPaystack, p.Name())
	}
}

func TestBatchLimits(t *testing.T) {
	p := NewTransferProvider(NewClient("key", "http://localhost"))
	if p.MinBatchSize() != 2 {
		t.Errorf("expected MinBatchSize 2, got %d", p.MinBatchSize())
	}
	if p.MaxBatchSize() != 100 {
		t.Errorf("expected MaxBatchSize 100, got %d", p.MaxBatchSize())
	}
}

func TestInitiateTransfer_MapsSuccessResponse(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		switch {
		case r.URL.Path == "/transferrecipient" && count == 1:
			json.NewEncoder(w).Encode(CreateRecipientResponse{
				Status: true,
				Data:   &RecipientData{RecipientCode: "RCP_test"},
			})
		case r.URL.Path == "/transfer" && count == 2:
			json.NewEncoder(w).Encode(TransferResponse{
				Status:  true,
				Message: "Transfer queued",
				Data: &TransferData{
					TransferCode: "TRF_abc",
					Reference:    "ref_001",
					Status:       "pending",
					Amount:       500000,
				},
			})
		default:
			t.Errorf("unexpected request #%d: %s %s", count, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	p := NewTransferProvider(NewClient("sk_test", server.URL))
	result, err := p.InitiateTransfer(context.Background(), &domain.SingleTransferRequest{
		Reference:     "ref_001",
		Amount:        "500000",
		BankCode:      "058",
		AccountNumber: "0123456789",
		AccountName:   "John Doe",
		Narration:     "Salary",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.Provider != domain.ProviderPaystack {
		t.Errorf("expected provider paystack, got %s", result.Provider)
	}
	if result.TransactionID != "TRF_abc" {
		t.Errorf("expected transaction ID TRF_abc, got %s", result.TransactionID)
	}
	if result.Reference != "ref_001" {
		t.Errorf("expected reference ref_001, got %s", result.Reference)
	}
}

func TestInitiateTransfer_RecipientCreationFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(CreateRecipientResponse{
			Status:  false,
			Message: "Invalid account number",
		})
	}))
	defer server.Close()

	p := NewTransferProvider(NewClient("sk_test", server.URL))
	result, err := p.InitiateTransfer(context.Background(), &domain.SingleTransferRequest{
		Reference:     "ref_002",
		Amount:        "500000",
		BankCode:      "058",
		AccountNumber: "invalid",
		AccountName:   "Test",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure when recipient creation fails")
	}
	if result.Status != "failed" {
		t.Errorf("expected status failed, got %s", result.Status)
	}
}

func TestInitiateTransfer_InvalidAmount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(CreateRecipientResponse{
			Status: true,
			Data:   &RecipientData{RecipientCode: "RCP_test"},
		})
	}))
	defer server.Close()

	p := NewTransferProvider(NewClient("sk_test", server.URL))
	_, err := p.InitiateTransfer(context.Background(), &domain.SingleTransferRequest{
		Reference: "ref_003",
		Amount:    "not_a_number",
		BankCode:  "058",
	})

	if err == nil {
		t.Error("expected error for invalid amount")
	}
}

func TestInitiateTransfer_DefaultCurrency(t *testing.T) {
	var capturedCurrency string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/transferrecipient" {
			var req CreateRecipientRequest
			json.NewDecoder(r.Body).Decode(&req)
			capturedCurrency = req.Currency
			json.NewEncoder(w).Encode(CreateRecipientResponse{
				Status: true, Data: &RecipientData{RecipientCode: "RCP_x"},
			})
		} else {
			json.NewEncoder(w).Encode(TransferResponse{
				Status: true, Data: &TransferData{TransferCode: "TRF_x"},
			})
		}
	}))
	defer server.Close()

	p := NewTransferProvider(NewClient("sk_test", server.URL))
	p.InitiateTransfer(context.Background(), &domain.SingleTransferRequest{
		Reference: "ref_004", Amount: "1000", BankCode: "058",
		AccountNumber: "0123456789", AccountName: "Test",
		// Currency intentionally empty
	})

	if capturedCurrency != "NGN" {
		t.Errorf("expected default currency NGN, got %s", capturedCurrency)
	}
}
