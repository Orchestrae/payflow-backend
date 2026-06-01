// internal/service/payroll_processing.go
package service

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// ProcessApprovedPayroll implements the method required by the scheduler's interface.
// This is called by the scheduler when a scheduled payroll run is due.
func (s *payrollService) ProcessApprovedPayroll(ctx context.Context, runID uint) error {
	log.Ctx(ctx).Info().Uint("run_id", runID).Msg("Processing approved payroll (scheduled)")

	run, err := s.payrollRepo.FindByID(ctx, runID, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	if run.Status != domain.StatusApproved {
		log.Ctx(ctx).Warn().Str("status", string(run.Status)).Msg("Skipping payroll processing, not in approved state")
		return nil
	}

	_, err = s.ProcessPayrollRunInstantly(ctx, runID, run.BusinessID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to process approved payroll")
		return fmt.Errorf("failed to process approved payroll: %w", err)
	}

	log.Ctx(ctx).Info().Msg("Payroll processing completed successfully")
	return nil
}

// ProcessPayrollRunInstantly processes a payroll run immediately, bypassing the scheduler.
func (s *payrollService) ProcessPayrollRunInstantly(ctx context.Context, runID, businessID uint) (*domain.PayrollRun, error) {
	log.Info().Uint("run_id", runID).Uint("business_id", businessID).Msg("Processing payroll run instantly")

	run, err := s.payrollRepo.FindByID(ctx, runID, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	if run.Status != domain.StatusApproved && run.Status != domain.StatusDraft {
		return nil, fmt.Errorf("%w: can only process approved or draft payroll runs", domain.ErrValidationFailed)
	}

	run.Status = domain.StatusProcessing
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to update payroll run status: %w", err)
	}

	// Batch-load employees for entries that don't have employee data
	s.batchLoadEmployeesForEntries(ctx, run, businessID)

	// Convert payroll entries to bulk transfer requests
	transfers := s.buildTransferRequests(run)
	if len(transfers) == 0 {
		return nil, fmt.Errorf("no valid transfers to process")
	}

	// Execute bulk transfer
	bulkReq := &domain.BulkTransferRequest{
		BatchReference: fmt.Sprintf("PAYROLL-RUN-%d", run.ID),
		Description:    fmt.Sprintf("Payroll for %s", run.Period.Format("January 2006")),
		Currency:       "NGN",
		Transfers:      transfers,
		BusinessID:     businessID,
	}

	bulkResp, err := s.transferSvc.ExecuteBatchTransfer(ctx, businessID, bulkReq)
	if err != nil {
		run.Status = domain.StatusFailed
		s.payrollRepo.Update(ctx, run)
		return nil, fmt.Errorf("bulk transfer failed: %w", err)
	}

	// Verify transfers in database
	verificationResult := s.verifyTransfersInDatabase(ctx, businessID, run.ID, len(transfers))

	// Update payroll run status based on transfer results
	if bulkResp.SuccessfulTransfers == len(transfers) || verificationResult.AllVerified {
		run.Status = domain.StatusCompleted
		run.PaymentReference = bulkReq.BatchReference
		now := time.Now()
		run.ProcessedAt = &now
		log.Info().Uint("run_id", runID).Int("total", len(transfers)).Int("api_ok", bulkResp.SuccessfulTransfers).Int("db_verified", verificationResult.VerifiedCount).Msg("Payroll run completed")
	} else {
		run.Status = domain.StatusFailed
		log.Warn().Uint("run_id", runID).Int("successful", bulkResp.SuccessfulTransfers).Int("total", len(transfers)).Msg("Payroll run failed")
	}

	if err := s.payrollRepo.Update(ctx, run); err != nil {
		log.Error().Err(err).Msg("Failed to update payroll run status")
		return nil, fmt.Errorf("failed to update payroll run: %w", err)
	}

	// Update loan balances on successful payroll
	if run.Status == domain.StatusCompleted {
		s.updateLoanBalances(ctx, run)
	}

	// Send payslip notifications
	if run.Status == domain.StatusCompleted && s.notificationSvc != nil {
		go func() {
			period := run.Period.Format("January 2006")
			for _, entry := range run.Entries {
				if entry.Employee != nil && entry.Employee.Email != "" {
					s.notificationSvc.SendEmail(context.Background(), entry.Employee.Email,
						"Your Payslip is Ready: "+period,
						fmt.Sprintf("Hi %s,\n\nYour payslip for %s is ready.\nNet Pay: NGN %s\n\nLog in to PayFlow to download your full payslip.",
							entry.Employee.FullName, period, formatKoboAsNGN(entry.NetPay)))
				}
			}
		}()
	}

	return run, nil
}

// processPayrollRunInstantly is an internal helper called by SubmitForApproval and ApprovePayrollRun
func (s *payrollService) processPayrollRunInstantly(ctx context.Context, run *domain.PayrollRun, businessID uint) (*domain.PayrollRun, error) {
	fullRun, err := s.payrollRepo.FindByID(ctx, run.ID, businessID)
	if err != nil {
		return nil, err
	}
	return s.ProcessPayrollRunInstantly(ctx, fullRun.ID, businessID)
}

// batchLoadEmployeesForEntries loads employee data for entries missing it.
func (s *payrollService) batchLoadEmployeesForEntries(ctx context.Context, run *domain.PayrollRun, businessID uint) {
	var missingEmpIDs []uint
	for i := range run.Entries {
		if run.Entries[i].Employee == nil {
			missingEmpIDs = append(missingEmpIDs, run.Entries[i].EmployeeID)
		}
	}
	if len(missingEmpIDs) == 0 {
		return
	}

	loadedEmployees, err := s.employeeRepo.FindByIDs(ctx, missingEmpIDs, businessID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to batch-load employees for payroll entries")
		return
	}

	empMap := make(map[uint]*domain.Employee, len(loadedEmployees))
	for _, emp := range loadedEmployees {
		empMap[emp.ID] = emp
	}
	for i := range run.Entries {
		if run.Entries[i].Employee == nil {
			if emp, ok := empMap[run.Entries[i].EmployeeID]; ok {
				run.Entries[i].Employee = emp
			} else {
				log.Warn().Uint("employee_id", run.Entries[i].EmployeeID).Msg("Failed to load employee for entry")
			}
		}
	}
}

// buildTransferRequests converts payroll entries to transfer requests.
func (s *payrollService) buildTransferRequests(run *domain.PayrollRun) []domain.SingleTransferRequest {
	transfers := make([]domain.SingleTransferRequest, 0, len(run.Entries))
	for _, entry := range run.Entries {
		if entry.Employee == nil {
			log.Warn().Uint("entry_id", entry.ID).Msg("Skipping entry with nil employee")
			continue
		}

		bankCode, err := s.mapBankNameToCode(entry.Employee.BankName)
		if err != nil {
			log.Warn().Err(err).Uint("employee_id", entry.EmployeeID).Str("bank_name", entry.Employee.BankName).Msg("Skipping entry: could not map bank name to code")
			continue
		}

		accountNumber := entry.Employee.BankAccountNumber
		if len(accountNumber) > 10 {
			accountNumber = accountNumber[len(accountNumber)-10:]
		} else if len(accountNumber) < 10 {
			accountNumber = fmt.Sprintf("%010s", accountNumber)
		}

		transfers = append(transfers, domain.SingleTransferRequest{
			Reference:     fmt.Sprintf("PAYROLL-%d-EMP-%d", run.ID, entry.EmployeeID),
			Amount:        fmt.Sprintf("%d", entry.NetPay),
			BankCode:      bankCode,
			AccountNumber: accountNumber,
			AccountName:   entry.Employee.FullName,
			Narration:     fmt.Sprintf("Salary payment for %s", run.Period.Format("January 2006")),
			Currency:      "NGN",
		})
	}
	return transfers
}

// TransferVerificationResult holds the result of transfer verification
type TransferVerificationResult struct {
	AllVerified   bool
	VerifiedCount int
	TotalCount    int
	Transfers     []*domain.Transfer
}

// verifyTransfersInDatabase verifies that all transfers were created in the database
func (s *payrollService) verifyTransfersInDatabase(ctx context.Context, businessID uint, payrollRunID uint, expectedCount int) TransferVerificationResult {
	transfers, total, err := s.transferRepo.FindByBusinessID(ctx, businessID, 1, 200)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch transfers for verification")
		return TransferVerificationResult{AllVerified: false, TotalCount: expectedCount}
	}

	verifiedCount := 0
	var matchingTransfers []*domain.Transfer
	recentTime := time.Now().Add(-2 * time.Minute)
	expectedPattern := fmt.Sprintf("PAYROLL-%d-EMP-", payrollRunID)

	for _, transfer := range transfers {
		if transfer.CreatedAt.After(recentTime) && transfer.Reference != "" {
			if len(transfer.Reference) >= len(expectedPattern) && transfer.Reference[:len(expectedPattern)] == expectedPattern {
				verifiedCount++
				matchingTransfers = append(matchingTransfers, transfer)
			}
		}
	}

	allVerified := verifiedCount >= expectedCount
	log.Info().Uint("payroll_run_id", payrollRunID).Int("expected", expectedCount).Int("verified", verifiedCount).Int("total_fetched", total).Bool("all_verified", allVerified).Msg("Transfer verification completed")

	return TransferVerificationResult{
		AllVerified:   allVerified,
		VerifiedCount: verifiedCount,
		TotalCount:    expectedCount,
		Transfers:     matchingTransfers,
	}
}

// GetPayrollRunForDisbursement fetches a payroll run for disbursement (implements domain.PayrollService)
func (s *payrollService) GetPayrollRunForDisbursement(ctx context.Context, runID uint) (*domain.PayrollRun, error) {
	return s.payrollRepo.FindByID(ctx, runID, 0)
}

// UpdateRunStatus updates the status of a payroll run (implements domain.PayrollService)
func (s *payrollService) UpdateRunStatus(ctx context.Context, runID uint, status domain.PayrollStatus) error {
	run, err := s.payrollRepo.FindByID(ctx, runID, 0)
	if err != nil {
		return err
	}
	run.Status = status
	return s.payrollRepo.Update(ctx, run)
}

// MarkRunAsFailed marks a payroll run as failed (implements domain.PayrollService)
func (s *payrollService) MarkRunAsFailed(ctx context.Context, runID uint, reason string) error {
	run, err := s.payrollRepo.FindByID(ctx, runID, 0)
	if err != nil {
		return err
	}
	run.Status = domain.StatusFailed
	run.RejectionReason = reason
	return s.payrollRepo.Update(ctx, run)
}

// MarkRunAsCompleted marks a payroll run as completed (implements domain.PayrollService)
func (s *payrollService) MarkRunAsCompleted(ctx context.Context, runID uint, reference string) error {
	run, err := s.payrollRepo.FindByID(ctx, runID, 0)
	if err != nil {
		return err
	}
	run.Status = domain.StatusCompleted
	run.PaymentReference = reference
	now := time.Now()
	run.ProcessedAt = &now
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return err
	}
	s.updateLoanBalances(ctx, run)
	return nil
}

// updateLoanBalances deducts monthly repayments from active loans after payroll completion.
func (s *payrollService) updateLoanBalances(ctx context.Context, run *domain.PayrollRun) {
	if s.loanRepo == nil {
		return
	}
	for _, entry := range run.Entries {
		if entry.Employee == nil {
			continue
		}
		activeLoans, err := s.loanRepo.FindActiveByEmployeeID(ctx, entry.EmployeeID)
		if err != nil {
			log.Warn().Err(err).Uint("employee_id", entry.EmployeeID).Msg("Failed to load loans for balance update")
			continue
		}
		for _, loan := range activeLoans {
			if loan.RemainingBalance <= 0 {
				continue
			}
			deductionAmt := loan.MonthlyDeduction
			if deductionAmt > loan.RemainingBalance {
				deductionAmt = loan.RemainingBalance
			}
			loan.TotalRepaid += deductionAmt
			loan.RemainingBalance -= deductionAmt
			if loan.RemainingBalance <= 0 {
				loan.Status = "completed"
			}
			if err := s.loanRepo.Update(ctx, loan); err != nil {
				log.Error().Err(err).Uint("loan_id", loan.ID).Msg("Failed to update loan balance after payroll")
			}
		}
	}
}

// formatKoboAsNGN converts kobo to NGN string.
func formatKoboAsNGN(kobo int64) string {
	return fmt.Sprintf("%.2f", float64(kobo)/100.0)
}

// mapBankNameToCode maps bank names to bank codes.
func (s *payrollService) mapBankNameToCode(bankName string) (string, error) {
	bankMap := map[string]string{
		"access bank":            "044",
		"access":                 "044",
		"united bank for africa": "033",
		"uba":                    "033",
		"guaranty trust bank":    "058",
		"gtb":                    "058",
		"gtbank":                 "058",
		"first bank":             "011",
		"zenith bank":            "057",
		"zenith":                 "057",
		"fidelity bank":          "070",
		"union bank":             "032",
		"stanbic ibtc":           "221",
		"ecobank":                "050",
	}

	normalizedName := strings.ToLower(strings.TrimSpace(bankName))
	if code, ok := bankMap[normalizedName]; ok {
		return code, nil
	}

	for name, code := range bankMap {
		if strings.HasPrefix(normalizedName, name) {
			return code, nil
		}
	}

	return "", fmt.Errorf("unsupported bank name: %q", bankName)
}
