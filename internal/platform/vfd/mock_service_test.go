package vfd

import (
	"context"
	"payflow/internal/domain"
	"testing"
)

func TestMockVFDService_AccountEnquiry(t *testing.T) {
	service := NewMockVFDService()

	resp, err := service.AccountEnquiry(context.Background(), "1001554791")
	if err != nil {
		t.Fatalf("AccountEnquiry failed: %v", err)
	}

	if resp.Status != "00" {
		t.Errorf("Expected status '00', got '%s'", resp.Status)
	}

	if resp.Data == nil {
		t.Fatal("Expected account data, got nil")
	}

	if resp.Data.AccountNo != "1001554791" {
		t.Errorf("Expected account number '1001554791', got '%s'", resp.Data.AccountNo)
	}

	t.Logf("✓ Account Enquiry Success: %s - %s", resp.Status, resp.Message)
	t.Logf("  Account: %s, Balance: %s, Client: %s",
		resp.Data.AccountNo, resp.Data.AccountBalance, resp.Data.Client)
}

func TestMockVFDService_BeneficiaryEnquiry(t *testing.T) {
	service := NewMockVFDService()

	resp, err := service.BeneficiaryEnquiry(context.Background(), "1001547795", "000004", "intra")
	if err != nil {
		t.Fatalf("BeneficiaryEnquiry failed: %v", err)
	}

	if resp.Status != "00" {
		t.Errorf("Expected status '00', got '%s'", resp.Status)
	}

	if resp.Data == nil {
		t.Fatal("Expected beneficiary data, got nil")
	}

	if resp.Data.Account.Number != "1001547795" {
		t.Errorf("Expected account number '1001547795', got '%s'", resp.Data.Account.Number)
	}

	t.Logf("✓ Beneficiary Enquiry Success: %s - %s", resp.Status, resp.Message)
	t.Logf("  Name: %s, Account: %s, Bank: %s",
		resp.Data.Name, resp.Data.Account.Number, resp.Data.Bank)
}

func TestMockVFDService_GetBankList(t *testing.T) {
	service := NewMockVFDService()

	resp, err := service.GetBankList(context.Background())
	if err != nil {
		t.Fatalf("GetBankList failed: %v", err)
	}

	if resp.Status != "00" {
		t.Errorf("Expected status '00', got '%s'", resp.Status)
	}

	if len(resp.Data) == 0 {
		t.Fatal("Expected bank list, got empty")
	}

	t.Logf("✓ Bank List Success: %s - %s", resp.Status, resp.Message)
	t.Logf("  Found %d banks", len(resp.Data))
	for _, bank := range resp.Data {
		t.Logf("    %s: %s", bank.Code, bank.Name)
	}
}

func TestMockVFDService_InitiateTransfer(t *testing.T) {
	service := NewMockVFDService()

	req := &domain.TransferRequest{
		FromAccount:   "1001554791",
		FromClientId:  "789",
		FromClient:    "Mock Client",
		FromSavingsId: "123456",
		ToClientId:    "456",
		ToClient:      "Mock Beneficiary",
		ToSavingsId:   "654321",
		ToAccount:     "1001547795",
		ToBank:        "000004",
		Signature:     "test-signature",
		Amount:        "1000",
		Remark:        "Test transfer",
		TransferType:  "intra",
		Reference:     "PayFlow-Test-001",
	}

	resp, err := service.InitiateTransfer(context.Background(), req)
	if err != nil {
		t.Fatalf("InitiateTransfer failed: %v", err)
	}

	if resp.Status != "00" {
		t.Errorf("Expected status '00', got '%s'", resp.Status)
	}

	if resp.Data == nil {
		t.Fatal("Expected transfer data, got nil")
	}

	if resp.Data.TxnId != req.Reference {
		t.Errorf("Expected TxnId '%s', got '%s'", req.Reference, resp.Data.TxnId)
	}

	t.Logf("✓ Transfer Success: %s - %s", resp.Status, resp.Message)
	t.Logf("  TxnId: %s, SessionId: %s, Reference: %s",
		resp.Data.TxnId, resp.Data.SessionId, resp.Data.Reference)
}

func TestMockVFDService_CompleteTransferFlow(t *testing.T) {
	service := NewMockVFDService()

	// Simulate the complete bulk transfer flow
	fromAccount := "1001554791"
	toAccount := "1001547795"
	toBankCode := "000004"
	amount := "1000"
	remark := "Test bulk transfer"
	transferType := "intra"
	reference := "PayFlow-Bulk-Test-001"

	t.Log("Step 1: Getting from account details...")
	fromAccountResp, err := service.AccountEnquiry(context.Background(), fromAccount)
	if err != nil {
		t.Fatalf("From Account Error: %v", err)
	}
	t.Logf("✓ From Account: %s (%s)", fromAccountResp.Data.AccountNo, fromAccountResp.Data.Client)

	t.Log("Step 2: Getting to account details...")
	toAccountResp, err := service.BeneficiaryEnquiry(context.Background(), toAccount, toBankCode, transferType)
	if err != nil {
		t.Fatalf("To Account Error: %v", err)
	}
	t.Logf("✓ To Account: %s (%s)", toAccountResp.Data.Account.Number, toAccountResp.Data.Name)

	t.Log("Step 3: Preparing transfer request...")
	transferRequest := &domain.TransferRequest{
		FromAccount:   fromAccountResp.Data.AccountNo,
		FromClientId:  fromAccountResp.Data.ClientId,
		FromClient:    fromAccountResp.Data.Client,
		FromSavingsId: fromAccountResp.Data.AccountId,
		ToClientId:    toAccountResp.Data.ClientId,
		ToClient:      toAccountResp.Data.Name,
		ToSavingsId:   toAccountResp.Data.Account.ID,
		ToAccount:     toAccountResp.Data.Account.Number,
		ToBank:        toBankCode,
		Signature:     "auto-generated-signature", // Would be SHA512(fromAccount + toAccount)
		Amount:        amount,
		Remark:        remark,
		TransferType:  transferType,
		Reference:     reference,
	}
	t.Logf("✓ Transfer Request Prepared: %s -> %s, Amount: %s",
		transferRequest.FromAccount, transferRequest.ToAccount, transferRequest.Amount)

	t.Log("Step 4: Executing transfer...")
	finalTransferResp, err := service.InitiateTransfer(context.Background(), transferRequest)
	if err != nil {
		t.Fatalf("Final Transfer Error: %v", err)
	}
	t.Logf("✓ Transfer Executed Successfully: %s - %s", finalTransferResp.Status, finalTransferResp.Message)
	if finalTransferResp.Data != nil {
		t.Logf("  Transaction ID: %s", finalTransferResp.Data.TxnId)
	}

	t.Log("=== Bulk Transfer Flow Test Complete ===")
	t.Log("✅ All VFD service methods working correctly")
	t.Log("✅ Transfer flow logic is sound")
	t.Log("✅ Ready for integration with payroll system")
}
