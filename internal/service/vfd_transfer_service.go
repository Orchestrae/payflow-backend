package service

import (
	"context"
	"crypto/sha512"
	"fmt"
	"log/slog"
	"payflow/internal/domain"
	"payflow/internal/platform/vfd"
	"payflow/internal/repository"
	"time"

	"gorm.io/gorm"
)

type vfdTransferService struct {
	transferRepo repository.VFDTransferRepository
	vfdService   vfd.VFDService
	txer         repository.Transactioner
}

func NewVFDTransferService(
	transferRepo repository.VFDTransferRepository,
	vfdService vfd.VFDService,
	txer repository.Transactioner,
) VFDTransferService {
	return &vfdTransferService{
		transferRepo: transferRepo,
		vfdService:   vfdService,
		txer:         txer,
	}
}

func (s *vfdTransferService) AccountEnquiry(ctx context.Context, accountNumber string) (*domain.AccountEnquiryResponse, error) {
	slog.Info("Performing account enquiry", "account_number", accountNumber)

	response, err := s.vfdService.AccountEnquiry(ctx, accountNumber)
	if err != nil {
		slog.Error("Failed to perform account enquiry", "error", err, "account_number", accountNumber)
		return nil, fmt.Errorf("failed to perform account enquiry: %w", err)
	}

	slog.Info("Account enquiry successful", "account_number", accountNumber, "status", response.Status)
	return response, nil
}

func (s *vfdTransferService) BeneficiaryEnquiry(ctx context.Context, accountNo, bank, transferType string) (*domain.BeneficiaryEnquiryResponse, error) {
	slog.Info("Performing beneficiary enquiry",
		"account_no", accountNo,
		"bank", bank,
		"transfer_type", transferType,
	)

	response, err := s.vfdService.BeneficiaryEnquiry(ctx, accountNo, bank, transferType)
	if err != nil {
		slog.Error("Failed to perform beneficiary enquiry", "error", err, "account_no", accountNo)
		return nil, fmt.Errorf("failed to perform beneficiary enquiry: %w", err)
	}

	slog.Info("Beneficiary enquiry successful", "account_no", accountNo, "status", response.Status)
	return response, nil
}

func (s *vfdTransferService) GetBankList(ctx context.Context) (*domain.BankListResponse, error) {
	slog.Info("Fetching bank list")

	response, err := s.vfdService.GetBankList(ctx)
	if err != nil {
		slog.Error("Failed to fetch bank list", "error", err)
		return nil, fmt.Errorf("failed to fetch bank list: %w", err)
	}

	slog.Info("Bank list fetched successfully", "count", len(response.Data))
	return response, nil
}

func (s *vfdTransferService) InitiateTransfer(ctx context.Context, businessID uint, req *domain.TransferRequest) (*domain.TransferResponse, error) {
	slog.Info("Initiating transfer",
		"business_id", businessID,
		"from_account", req.FromAccount,
		"to_account", req.ToAccount,
		"amount", req.Amount,
		"transfer_type", req.TransferType,
		"reference", req.Reference,
	)

	// Start transaction
	tx := s.txer.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			s.txer.Rollback(tx)
			panic(r)
		}
	}()

	// Create transfer record
	transfer := &domain.TransferRecord{
		BusinessID:    businessID,
		FromAccount:   req.FromAccount,
		FromClientId:  req.FromClientId,
		FromClient:    req.FromClient,
		FromSavingsId: req.FromSavingsId,
		FromBvn:       &req.FromBvn,
		ToClientId:    req.ToClientId,
		ToClient:      req.ToClient,
		ToSavingsId:   req.ToSavingsId,
		ToSession:     &req.ToSession,
		ToBvn:         &req.ToBvn,
		ToAccount:     req.ToAccount,
		ToBank:        req.ToBank,
		Amount:        req.Amount,
		Remark:        req.Remark,
		TransferType:  req.TransferType,
		Reference:     req.Reference,
		Status:        string(domain.TransferStatusPending),
	}

	// Save transfer record
	var transferRepoTx repository.VFDTransferRepository
	if gormTx, ok := tx.(*gorm.DB); ok {
		transferRepoTx = s.transferRepo.WithTx(gormTx)
		if err := transferRepoTx.Create(ctx, transfer); err != nil {
			s.txer.Rollback(tx)
			return nil, fmt.Errorf("failed to save transfer record: %w", err)
		}
	} else {
		s.txer.Rollback(tx)
		return nil, fmt.Errorf("invalid transaction type")
	}

	// Generate signature
	signature := s.generateSignature(req.FromAccount, req.ToAccount)
	req.Signature = signature

	// Call VFD API to initiate transfer
	response, err := s.vfdService.InitiateTransfer(ctx, req)
	if err != nil {
		// Update transfer record with error
		transfer.Status = string(domain.TransferStatusFailed)
		transfer.VFDStatus = "99"
		transfer.VFDMessage = "Transfer failed"
		errorMsg := err.Error()
		transfer.ProcessingError = &errorMsg
		now := time.Now()
		transfer.ProcessedAt = &now

		if updateErr := transferRepoTx.Update(ctx, transfer); updateErr != nil {
			slog.Error("Failed to update transfer status to failed", "error", updateErr)
		}

		s.txer.Rollback(tx)
		return nil, fmt.Errorf("failed to initiate transfer: %w", err)
	}

	// Update transfer record with success
	transfer.Status = string(domain.TransferStatusSuccess)
	transfer.VFDStatus = response.Status
	transfer.VFDMessage = response.Message
	if response.Data != nil {
		transfer.TxnId = &response.Data.TxnId
		if response.Data.SessionId != "" {
			transfer.SessionId = &response.Data.SessionId
		}
	}
	now := time.Now()
	transfer.ProcessedAt = &now

	if err := transferRepoTx.Update(ctx, transfer); err != nil {
		s.txer.Rollback(tx)
		return nil, fmt.Errorf("failed to update transfer record: %w", err)
	}

	if err := s.txer.Commit(tx); err != nil {
		s.txer.Rollback(tx)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("Transfer initiated successfully",
		"transfer_id", transfer.ID,
		"reference", req.Reference,
		"vfd_status", response.Status,
	)

	return response, nil
}

func (s *vfdTransferService) ListTransfers(ctx context.Context, businessID uint, page, limit int) ([]*domain.TransferRecord, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	transfers, total, err := s.transferRepo.FindByBusinessID(ctx, businessID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list transfers: %w", err)
	}

	return transfers, total, nil
}

func (s *vfdTransferService) GetTransferByID(ctx context.Context, id uint) (*domain.TransferRecord, error) {
	transfer, err := s.transferRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transfer: %w", err)
	}

	return transfer, nil
}

func (s *vfdTransferService) GetTransfersByFromAccount(ctx context.Context, fromAccount string, page, limit int) ([]*domain.TransferRecord, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	transfers, total, err := s.transferRepo.FindByFromAccount(ctx, fromAccount, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get transfers by from account: %w", err)
	}

	return transfers, total, nil
}

func (s *vfdTransferService) GetTransfersByToAccount(ctx context.Context, toAccount string, page, limit int) ([]*domain.TransferRecord, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	transfers, total, err := s.transferRepo.FindByToAccount(ctx, toAccount, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get transfers by to account: %w", err)
	}

	return transfers, total, nil
}

// Helper methods

func (s *vfdTransferService) generateSignature(fromAccount, toAccount string) string {
	// Generate signature using SHA512(fromAccount + toAccount)
	data := fromAccount + toAccount
	hash := sha512.Sum512([]byte(data))
	return fmt.Sprintf("%x", hash)
}
