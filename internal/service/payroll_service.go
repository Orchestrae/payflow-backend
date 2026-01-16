// internal/service/payroll_service.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"payflow/internal/domain"
	"payflow/internal/repository"
	"time"

	"github.com/rs/zerolog/log"
)

type payrollService struct {
	payrollRepo     repository.PayrollRepository
	employeeRepo    repository.EmployeeRepository
	cadreRepo       repository.CadreRepository
	businessRepo    repository.BusinessRepository
	transferRepo    repository.TransferRepository
	txer            repository.Transactioner
	payoutSvc       PayoutService
	notificationSvc NotificationService
	scheduler       domain.Scheduler
	userRepo        repository.UserRepository
	transferSvc     TransferService // For instant run bulk transfers
}

// ... (inside the payrollService struct and NewPayrollService function)

// RejectPayrollRun handles the rejection of a payroll run.
// It can only be called on a run that is 'pending_approval'.
func (s *payrollService) RejectPayrollRun(ctx context.Context, runID, rejecterID uint, reason string) (*domain.PayrollRun, error) {
	log.Ctx(ctx).Info().Uint("run_id", runID).Uint("user_id", rejecterID).Msg("Attempting to reject payroll run")

	// 1. Basic Validation: Ensure a reason is provided.
	if reason == "" {
		return nil, fmt.Errorf("%w: rejection requires a reason", domain.ErrValidationFailed)
	}

	// 2. Fetch the rejecter to get their BusinessID.
	rejecter, err := s.userRepo.FindByID(ctx, rejecterID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Could not find rejecting user")
		return nil, domain.ErrInternalServer
	}

	// 3. Fetch the payroll run using the rejecter's BusinessID (Tenancy Check).
	run, err := s.payrollRepo.FindByID(ctx, runID, rejecter.BusinessID)
	if err != nil {
		return nil, err
	}

	// 4. Business Rule: Check for correct state.
	if run.Status != domain.StatusPendingApproval {
		log.Ctx(ctx).Warn().
			Str("current_status", string(run.Status)).
			Msg("Attempted to reject a payroll run not in 'pending_approval' state")
		return nil, fmt.Errorf("%w: cannot reject a payroll run with status '%s'", domain.ErrValidationFailed, run.Status)
	}

	// Permission check is largely handled by fetching with rejecter.BusinessID, but being explicit doesn't hurt.
	if rejecter.BusinessID != run.BusinessID {
		return nil, domain.ErrForbidden
	}

	// 5. Update the run's status and record the reason.
	run.Status = domain.StatusRejected
	run.RejectionReason = reason
	run.UpdatedAt = time.Now()

	if err := s.payrollRepo.Update(ctx, run); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to update payroll run status to rejected")
		return nil, err
	}

	log.Ctx(ctx).Info().Msg("Payroll run successfully rejected")

	// 6. Side Effect: Notify the operator(s).
	// Since FindOperatorsByBusinessID is missing, we list all users and filter.
	users, err := s.userRepo.FindByBusinessID(ctx, run.BusinessID)
	if err == nil {
		for _, user := range users {
			if user.Role == domain.RoleOperator {
				emailSubject := fmt.Sprintf("Action Required: Payroll for %s was Rejected", run.Period.Format("January 2006"))
				emailBody := fmt.Sprintf(
					"Hello %s,\n\nThe payroll run for the period %s has been rejected by an approver.\n\nReason: %s\n\nPlease log in to PayFlow to review and correct the details before resubmitting.\n\nThank you,\nThe PayFlow Team",
					user.Email,
					run.Period.Format("January 2006"),
					reason,
				)
				go s.notificationSvc.SendEmail(context.Background(), user.Email, emailSubject, emailBody)
			}
		}
	} else {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to fetch users for rejection notification")
	}

	return run, nil
}

// ... NewPayrollService ... (Keeping as is, just need valid content around)

// NewPayrollService creates a new payroll service instance.
func NewPayrollService(
	payrollRepo repository.PayrollRepository,
	employeeRepo repository.EmployeeRepository,
	cadreRepo repository.CadreRepository,
	businessRepo repository.BusinessRepository,
	transferRepo repository.TransferRepository,
	txer repository.Transactioner,
	payoutSvc PayoutService,
	notificationSvc NotificationService,
	userRepo repository.UserRepository,
	scheduler domain.Scheduler,
	transferSvc TransferService,
) PayrollService {
	return &payrollService{
		payrollRepo:     payrollRepo,
		employeeRepo:    employeeRepo,
		cadreRepo:       cadreRepo,
		businessRepo:    businessRepo,
		transferRepo:    transferRepo,
		txer:            txer,
		payoutSvc:       payoutSvc,
		notificationSvc: notificationSvc,
		userRepo:        userRepo,
		scheduler:       scheduler,
		transferSvc:     transferSvc,
	}
}

// CalculatePayrollRun performs an in-memory calculation of the payroll.
func (s *payrollService) CalculatePayrollRun(ctx context.Context, businessID uint, period time.Time, adjustments map[uint][]EmployeeAdjustment) (*domain.PayrollRun, error) {
	// 1. Fetch all employees for the business.
	allEmployees, err := s.employeeRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		slog.Error("Failed to fetch employees", "error", err)
		return nil, fmt.Errorf("failed to fetch employees: %w", err)
	}

	// Filter for active employees and load their cadres
	var employees []*domain.Employee
	for _, emp := range allEmployees {
		if emp.IsActive {
			// Load cadre with earning components and deduction rules
			cadre, err := s.cadreRepo.FindByID(ctx, emp.CadreID, businessID)
			if err != nil {
				slog.Warn("Failed to load cadre for employee", "employee_id", emp.ID, "cadre_id", emp.CadreID, "error", err)
				continue
			}
			emp.Cadre = cadre
			employees = append(employees, emp)
		}
	}

	if len(employees) == 0 {
		return nil, fmt.Errorf("no active employees found for business to run payroll")
	}

	// 2. For each employee, calculate their pay.
	var totalGross, totalDeductions, totalNet int64
	runEntries := make([]domain.PayrollRunEntry, 0, len(employees))

	for _, emp := range employees {
		if emp.Cadre == nil {
			slog.Warn("Employee without cadre", "slog_employee_id", emp.ID, "slog_employee_name", emp.FullName, "slog_business_id", businessID)
			continue
		}

		var entryGross, entryDeductions, entryBonus int64
		details := make([]domain.PayrollRunEntryDetail, 0)

		// Calculate Gross Pay
		for _, ec := range emp.Cadre.EarningComponents {
			entryGross += ec.Amount
			details = append(details, domain.PayrollRunEntryDetail{
				Type:   domain.DetailTypeEarning,
				Name:   ec.Name,
				Amount: ec.Amount,
			})
		}

		// Calculate Deductions
		for _, dr := range emp.Cadre.DeductionRules {
			var deductionAmount int64
			if dr.Type == domain.DeductionTypePercentage {
				deductionAmount = int64(float64(entryGross) * (dr.Value / 100.0))
			} else {
				deductionAmount = int64(dr.Value)
			}
			entryDeductions += deductionAmount
			details = append(details, domain.PayrollRunEntryDetail{
				Type:   domain.DetailTypeDeduction,
				Name:   dr.Name,
				Amount: deductionAmount,
			})
		}

		// Process detailed adjustments for this employee
		if employeeAdjustments, ok := adjustments[emp.ID]; ok {
			for _, adj := range employeeAdjustments {
				// Determine component type (auto-infer from amount if not provided)
				componentType := adj.ComponentType
				if componentType == "" {
					if adj.Amount >= 0 {
						componentType = "earnings"
					} else {
						componentType = "deduction"
					}
				}

				// Determine detail type
				var detailType domain.PayrollEntryDetailType
				if adj.Amount >= 0 {
					if componentType == "earnings" {
						detailType = domain.DetailTypeEarning
						entryGross += adj.Amount
					} else {
						detailType = domain.DetailTypeBonus
						entryBonus += adj.Amount
					}
				} else {
					detailType = domain.DetailTypeDeduction
					entryDeductions += -adj.Amount // Amount is negative, so negate it
				}

				// Store adjustment amount as positive for database (we track sign via type)
				adjustmentAmount := adj.Amount
				if adjustmentAmount < 0 {
					adjustmentAmount = -adjustmentAmount
				}

				details = append(details, domain.PayrollRunEntryDetail{
					Type:        detailType,
					Name:        adj.ItemName,
					Amount:      adjustmentAmount,
					Description: adj.Description,
				})
			}
		}

		entryNet := entryGross + entryBonus - entryDeductions

		runEntries = append(runEntries, domain.PayrollRunEntry{
			EmployeeID:      emp.ID,
			Employee:        emp,
			GrossPay:        entryGross,
			TotalDeductions: entryDeductions,
			Bonuses:         entryBonus,
			NetPay:          entryNet,
			Details:         details,
		})

		totalGross += entryGross
		totalDeductions += entryDeductions
		totalNet += entryNet
	}

	// 3. Assemble the final payroll run object.
	now := time.Now()
	// Normalize period to first day of the month
	normalizedPeriod := time.Date(period.Year(), period.Month(), 1, 0, 0, 0, 0, time.UTC)
	
	payrollRun := &domain.PayrollRun{
		BusinessID:      businessID,
		Period:          normalizedPeriod,
		Status:          domain.StatusDraft,
		TotalGrossPay:   totalGross,
		TotalDeductions: totalDeductions,
		TotalNetPay:     totalNet,
		ScheduledFor:    now.AddDate(0, 0, 5), // Default to 5 days from now
		Entries:         runEntries,
	}

	return payrollRun, nil
}

// CreateAndStorePayrollRun creates and stores a payroll run for a specific period.
func (s *payrollService) CreateAndStorePayrollRun(ctx context.Context, businessID uint, period time.Time, adjustments map[uint][]EmployeeAdjustment) (*domain.PayrollRun, error) {
	payrollRun, err := s.CalculatePayrollRun(ctx, businessID, period, adjustments)
	if err != nil {
		return nil, err
	}
	if err := s.payrollRepo.Create(ctx, payrollRun); err != nil {
		return nil, fmt.Errorf("failed to save payroll run: %w", err)
	}
	return payrollRun, nil
}

func (s *payrollService) ApprovePayrollRun(ctx context.Context, runID, approverID uint) (*domain.PayrollRun, error) {
	approver, err := s.userRepo.FindByID(ctx, approverID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch approver: %w", err)
	}

	run, err := s.payrollRepo.FindByID(ctx, runID, approver.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	if run.Status != domain.StatusPendingApproval {
		return nil, domain.ErrValidationFailed
	}

	// Check business configuration for auto-process
	business, err := s.businessRepo.FindByID(ctx, approver.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch business: %w", err)
	}

	run.Status = domain.StatusApproved
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, err
	}

	// If auto-process is enabled, process immediately (for testing)
	if business.PayrollAutoProcess {
		slog.Info("Auto-processing payroll run (business config: auto-process enabled)",
			"run_id", runID,
			"business_id", approver.BusinessID,
		)
		return s.processPayrollRunInstantly(ctx, run, approver.BusinessID)
	}

	// Otherwise, schedule normally
	if err := s.scheduler.SchedulePayout(*run); err != nil {
		log.Error().Err(err).Uint("runID", runID).Msg("CRITICAL: Failed to schedule payout for approved run")
		run.Status = domain.StatusPendingApproval
		_ = s.payrollRepo.Update(ctx, run)
		return nil, fmt.Errorf("could not schedule payout job: %w", err)
	}

	return run, nil
}

func (s *payrollService) SubmitForApproval(ctx context.Context, runID, userID uint) (*domain.PayrollRun, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	run, err := s.payrollRepo.FindByID(ctx, runID, user.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	if run.Status != domain.StatusDraft {
		return nil, fmt.Errorf("%w: cannot submit non-draft run", domain.ErrValidationFailed)
	}

	// Check business configuration
	business, err := s.businessRepo.FindByID(ctx, user.BusinessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch business: %w", err)
	}

	// If business doesn't require approval, auto-approve
	if !business.PayrollRequiresApproval {
		run.Status = domain.StatusApproved
		slog.Info("Auto-approving payroll run (business config: no approval required)",
			"run_id", runID,
			"business_id", user.BusinessID,
		)

		// If auto-process is enabled, process immediately
		if business.PayrollAutoProcess {
			return s.processPayrollRunInstantly(ctx, run, user.BusinessID)
		}

		// Otherwise, schedule normally
		if err := s.scheduler.SchedulePayout(*run); err != nil {
			return nil, fmt.Errorf("failed to schedule payout: %w", err)
		}
	} else {
		run.Status = domain.StatusPendingApproval
	}

	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to update payroll run status: %w", err)
	}

	return run, nil
}

// ProcessApprovedPayroll implements the method required by the scheduler's interface.
// This is called by the scheduler when a scheduled payroll run is due.
func (s *payrollService) ProcessApprovedPayroll(ctx context.Context, runID uint) error {
	log.Ctx(ctx).Info().Uint("run_id", runID).Msg("Processing approved payroll (scheduled)")

	// Fetch run with system access (businessID = 0)
	run, err := s.payrollRepo.FindByID(ctx, runID, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch payroll run: %w", err)
	}
	
	if run.Status != domain.StatusApproved {
		log.Ctx(ctx).Warn().Str("status", string(run.Status)).Msg("Skipping payroll processing, not in approved state")
		return nil
	}

	// Use the same instant processing logic (reuse ProcessPayrollRunInstantly)
	_, err = s.ProcessPayrollRunInstantly(ctx, runID, run.BusinessID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to process approved payroll")
		return fmt.Errorf("failed to process approved payroll: %w", err)
	}

	log.Ctx(ctx).Info().Msg("Payroll processing completed successfully")
	return nil
}

// ListByBusinessID retrieves all payroll runs for a business
func (s *payrollService) ListByBusinessID(ctx context.Context, businessID uint) ([]*domain.PayrollRun, error) {
	return s.payrollRepo.FindByBusinessID(ctx, businessID)
}

// GetByID retrieves a specific payroll run by ID, ensuring it belongs to the specified business
func (s *payrollService) GetByID(ctx context.Context, runID, businessID uint) (*domain.PayrollRun, error) {
	return s.payrollRepo.FindByID(ctx, runID, businessID)
}

// ProcessPayrollRunInstantly processes a payroll run immediately, bypassing the scheduler.
// This allows businesses to pay employees instantly without waiting for scheduled processing.
// It executes bulk transfers and verifies them in the database.
func (s *payrollService) ProcessPayrollRunInstantly(ctx context.Context, runID, businessID uint) (*domain.PayrollRun, error) {
	slog.Info("Processing payroll run instantly",
		"run_id", runID,
		"business_id", businessID,
	)

	// Fetch the payroll run
	run, err := s.payrollRepo.FindByID(ctx, runID, businessID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payroll run: %w", err)
	}

	// Validate status
	if run.Status != domain.StatusApproved && run.Status != domain.StatusDraft {
		return nil, fmt.Errorf("%w: can only process approved or draft payroll runs", domain.ErrValidationFailed)
	}

	// Update status to processing
	run.Status = domain.StatusProcessing
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to update payroll run status: %w", err)
	}

	// Load employees for entries that don't have employee data
	for i := range run.Entries {
		if run.Entries[i].Employee == nil {
			emp, err := s.employeeRepo.FindByID(ctx, run.Entries[i].EmployeeID, businessID)
			if err != nil {
				slog.Warn("Failed to load employee for entry", "entry_id", run.Entries[i].ID, "employee_id", run.Entries[i].EmployeeID, "error", err)
				continue
			}
			run.Entries[i].Employee = emp
		}
	}

	// Convert payroll entries to bulk transfer requests
	transfers := make([]domain.SingleTransferRequest, 0, len(run.Entries))
	for _, entry := range run.Entries {
		if entry.Employee == nil {
			slog.Warn("Skipping entry with nil employee", "entry_id", entry.ID)
			continue
		}

		// Convert net pay from smallest currency unit (kobo) to main unit (NGN)
		amountStr := fmt.Sprintf("%d", entry.NetPay)

		// Map bank name to bank code (simplified - should use a proper mapping)
		bankCode := s.mapBankNameToCode(entry.Employee.BankName)

		// Ensure account number is exactly 10 characters (Korapay requirement)
		accountNumber := entry.Employee.BankAccountNumber
		if len(accountNumber) > 10 {
			accountNumber = accountNumber[len(accountNumber)-10:] // Take last 10 digits
		} else if len(accountNumber) < 10 {
			accountNumber = fmt.Sprintf("%010s", accountNumber) // Pad with zeros
		}

		transfers = append(transfers, domain.SingleTransferRequest{
			Reference:     fmt.Sprintf("PAYROLL-%d-EMP-%d", run.ID, entry.EmployeeID),
			Amount:        amountStr,
			BankCode:      bankCode,
			AccountNumber: accountNumber,
			AccountName:   entry.Employee.FullName,
			Narration:     fmt.Sprintf("Salary payment for %s", run.Period.Format("January 2006")),
			Currency:      "NGN",
		})
	}

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
	if bulkResp.SuccessfulTransfers == len(transfers) {
		// All transfers succeeded via API
		run.Status = domain.StatusCompleted
		run.PaymentReference = bulkReq.BatchReference
		now := time.Now()
		run.ProcessedAt = &now
		slog.Info("Payroll run processed successfully",
			"run_id", runID,
			"total_transfers", len(transfers),
			"successful", bulkResp.SuccessfulTransfers,
			"verified_in_db", verificationResult.VerifiedCount,
		)
	} else if verificationResult.AllVerified {
		// Transfers created in DB but some may have failed at provider
		// If all transfers are verified in DB, mark as completed (provider will update status via webhook)
		run.Status = domain.StatusCompleted
		run.PaymentReference = bulkReq.BatchReference
		now := time.Now()
		run.ProcessedAt = &now
		slog.Info("Payroll run completed (all transfers verified in database)",
			"run_id", runID,
			"total_transfers", len(transfers),
			"api_successful", bulkResp.SuccessfulTransfers,
			"verified_in_db", verificationResult.VerifiedCount,
		)
	} else {
		// Some transfers failed or weren't created
		run.Status = domain.StatusFailed
		slog.Warn("Payroll run failed",
			"run_id", runID,
			"successful", bulkResp.SuccessfulTransfers,
			"total", len(transfers),
			"verified", verificationResult.VerifiedCount,
		)
	}

	if err := s.payrollRepo.Update(ctx, run); err != nil {
		slog.Error("Failed to update payroll run status", "error", err)
		return nil, fmt.Errorf("failed to update payroll run: %w", err)
	}

	return run, nil
}

// TransferVerificationResult holds the result of transfer verification
type TransferVerificationResult struct {
	AllVerified   bool
	VerifiedCount int
	TotalCount     int
	Transfers      []*domain.Transfer
}

// verifyTransfersInDatabase verifies that all transfers were created in the database for a specific payroll run
func (s *payrollService) verifyTransfersInDatabase(ctx context.Context, businessID uint, payrollRunID uint, expectedCount int) TransferVerificationResult {
	// Get all transfers for this business created recently (within last 2 minutes)
	transfers, total, err := s.transferRepo.FindByBusinessID(ctx, businessID, 1, 200)
	if err != nil {
		slog.Error("Failed to fetch transfers for verification", "error", err)
		return TransferVerificationResult{
			AllVerified:   false,
			VerifiedCount: 0,
			TotalCount:     expectedCount,
		}
	}

	// Filter transfers that match our payroll pattern (PAYROLL-{runID}-EMP-{empID})
	verifiedCount := 0
	var matchingTransfers []*domain.Transfer
	recentTime := time.Now().Add(-2 * time.Minute)
	expectedPattern := fmt.Sprintf("PAYROLL-%d-EMP-", payrollRunID)
	
	for _, transfer := range transfers {
		// Check if transfer was created recently and matches this payroll run
		if transfer.CreatedAt.After(recentTime) && transfer.Reference != "" {
			// Check if reference matches our payroll run pattern
			if len(transfer.Reference) >= len(expectedPattern) && transfer.Reference[:len(expectedPattern)] == expectedPattern {
				verifiedCount++
				matchingTransfers = append(matchingTransfers, transfer)
			}
		}
	}

	allVerified := verifiedCount >= expectedCount

	slog.Info("Transfer verification completed",
		"payroll_run_id", payrollRunID,
		"expected", expectedCount,
		"verified", verifiedCount,
		"total_fetched", total,
		"all_verified", allVerified,
	)

	return TransferVerificationResult{
		AllVerified:   allVerified,
		VerifiedCount: verifiedCount,
		TotalCount:     expectedCount,
		Transfers:      matchingTransfers,
	}
}

// processPayrollRunInstantly is an internal helper called by SubmitForApproval and ApprovePayrollRun
func (s *payrollService) processPayrollRunInstantly(ctx context.Context, run *domain.PayrollRun, businessID uint) (*domain.PayrollRun, error) {
	// Reload run with entries to ensure we have all data
	fullRun, err := s.payrollRepo.FindByID(ctx, run.ID, businessID)
	if err != nil {
		return nil, err
	}

	// Process using the public method
	return s.ProcessPayrollRunInstantly(ctx, fullRun.ID, businessID)
}

// GetPayrollRunForDisbursement fetches a payroll run for disbursement (implements domain.PayrollService)
func (s *payrollService) GetPayrollRunForDisbursement(ctx context.Context, runID uint) (*domain.PayrollRun, error) {
	return s.payrollRepo.FindByID(ctx, runID, 0) // System access
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
	return s.payrollRepo.Update(ctx, run)
}

// mapBankNameToCode maps bank names to bank codes (simplified mapping)
// In production, this should use a proper bank code lookup service
func (s *payrollService) mapBankNameToCode(bankName string) string {
	// Common Nigerian bank codes
	bankMap := map[string]string{
		"Access Bank":        "044",
		"Access":             "044",
		"United Bank for Africa": "033",
		"UBA":                "033",
		"Guaranty Trust Bank": "058",
		"GTB":                "058",
		"GTBank":             "058",
		"First Bank":         "011",
		"Zenith Bank":        "057",
		"Zenith":             "057",
		"Fidelity Bank":      "070",
		"Union Bank":         "032",
		"Stanbic IBTC":       "221",
		"Ecobank":            "050",
	}

	// Try exact match first
	if code, ok := bankMap[bankName]; ok {
		return code
	}

	// Try case-insensitive match
	for name, code := range bankMap {
		if len(bankName) >= len(name) && bankName[:len(name)] == name {
			return code
		}
	}

	// Default to Access Bank code if not found
	slog.Warn("Bank code not found, using default", "bank_name", bankName)
	return "044"
}
