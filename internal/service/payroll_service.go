// internal/service/payroll_service.go
package service

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"payflow/internal/repository"
	"payflow/internal/service/tax"
	"strings"
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
	// 1. Fetch business for statutory config + all employees.
	business, err := s.businessRepo.FindByID(ctx, businessID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch business")
		return nil, fmt.Errorf("failed to fetch business: %w", err)
	}

	allEmployees, err := s.employeeRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch employees")
		return nil, fmt.Errorf("failed to fetch employees: %w", err)
	}

	// Filter for active employees and batch-load their cadres
	var activeEmployees []*domain.Employee
	cadreIDSet := make(map[uint]struct{})
	for _, emp := range allEmployees {
		if emp.IsActive {
			activeEmployees = append(activeEmployees, emp)
			cadreIDSet[emp.CadreID] = struct{}{}
		}
	}

	// Batch-load all cadres in a single query (eliminates N+1)
	cadreIDs := make([]uint, 0, len(cadreIDSet))
	for id := range cadreIDSet {
		cadreIDs = append(cadreIDs, id)
	}
	cadres, err := s.cadreRepo.FindByIDs(ctx, cadreIDs, businessID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to batch-load cadres")
		return nil, fmt.Errorf("failed to load cadres: %w", err)
	}
	cadreMap := make(map[uint]*domain.Cadre, len(cadres))
	for _, c := range cadres {
		cadreMap[c.ID] = c
	}

	var employees []*domain.Employee
	for _, emp := range activeEmployees {
		cadre, ok := cadreMap[emp.CadreID]
		if !ok {
			log.Warn().Uint("employee_id", emp.ID).Uint("cadre_id", emp.CadreID).Msg("Failed to load cadre for employee")
			continue
		}
		emp.Cadre = cadre
		employees = append(employees, emp)
	}

	if len(employees) == 0 {
		return nil, fmt.Errorf("no active employees found for business to run payroll")
	}

	// 2. For each employee, calculate their pay.
	var totalGross, totalDeductions, totalNet, totalEmployerCosts int64
	runEntries := make([]domain.PayrollRunEntry, 0, len(employees))

	for _, emp := range employees {
		if emp.Cadre == nil {
			log.Warn().Uint("employee_id", emp.ID).Str("employee_name", emp.FullName).Uint("business_id", businessID).Msg("Employee without cadre")
			continue
		}

		var entryGross, entryDeductions, entryBonus int64
		var basicPay, housingPay, transportPay, otherPay int64
		details := make([]domain.PayrollRunEntryDetail, 0)

		// Calculate Gross Pay and classify earning components by type
		for _, ec := range emp.Cadre.EarningComponents {
			entryGross += ec.Amount
			switch ec.ComponentType {
			case domain.ComponentBasic:
				basicPay = ec.Amount
			case domain.ComponentHousing:
				housingPay = ec.Amount
			case domain.ComponentTransport:
				transportPay = ec.Amount
			default:
				otherPay += ec.Amount
			}
			details = append(details, domain.PayrollRunEntryDetail{
				Type:   domain.DetailTypeEarning,
				Name:   ec.Name,
				Amount: ec.Amount,
			})
		}

		// Fallback: if no component tagged as basic, use name matching
		if basicPay == 0 {
			for _, ec := range emp.Cadre.EarningComponents {
				if strings.EqualFold(ec.Name, "basic pay") || strings.EqualFold(ec.Name, "basic salary") || strings.EqualFold(ec.Name, "basic") {
					basicPay = ec.Amount
					break
				}
			}
		}

		// Calculate custom deductions (business-defined rules)
		for _, dr := range emp.Cadre.DeductionRules {
			var deductionAmount int64
			if dr.Type == domain.DeductionTypePercentage {
				baseAmount := entryGross
				if dr.CalculationBasis == domain.BasisBasicPay && basicPay > 0 {
					baseAmount = basicPay
				}
				deductionAmount = int64(float64(baseAmount) * (dr.Value / 100.0))
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

		// Compute statutory deductions based on business country (Nigeria/Ghana)
		taxResult := tax.CalculateForCountry(business.Currency, tax.Input{
			BasicPay:       basicPay,
			HousingPay:     housingPay,
			TransportPay:   transportPay,
			OtherPay:       otherPay,
			GrossPay:       entryGross,
			AnnualRentPaid: emp.AnnualRentPaid,
			PensionEnabled: business.PensionEnabled,
			NHFEnabled:     business.NHFEnabled,
			NSITFEnabled:   business.NSITFEnabled,
			PAYEEnabled:    business.PAYEEnabled,
		})

		// Add statutory employee deductions
		if taxResult.PAYE > 0 {
			entryDeductions += taxResult.PAYE
			details = append(details, domain.PayrollRunEntryDetail{
				Type:        domain.DetailTypeStatutoryDeduction,
				Name:        "PAYE Income Tax",
				Amount:      taxResult.PAYE,
				Description: fmt.Sprintf("Monthly PAYE (annual taxable: %d kobo, rent relief: %d kobo)", taxResult.AnnualTaxableIncome, taxResult.RentRelief),
			})
		}
		if taxResult.EmployeePension > 0 {
			entryDeductions += taxResult.EmployeePension
			details = append(details, domain.PayrollRunEntryDetail{
				Type:        domain.DetailTypeStatutoryDeduction,
				Name:        "Pension (Employee 8%)",
				Amount:      taxResult.EmployeePension,
				Description: fmt.Sprintf("RSA contribution (pension base: %d kobo)", taxResult.PensionBase),
			})
		}
		if taxResult.NHF > 0 {
			entryDeductions += taxResult.NHF
			details = append(details, domain.PayrollRunEntryDetail{
				Type: domain.DetailTypeStatutoryDeduction,
				Name: "NHF (2.5%)",
				Amount: taxResult.NHF,
			})
		}

		// Track employer costs (do NOT reduce net pay)
		if taxResult.EmployerPension > 0 {
			details = append(details, domain.PayrollRunEntryDetail{
				Type:        domain.DetailTypeEmployerCost,
				Name:        "Pension (Employer 10%)",
				Amount:      taxResult.EmployerPension,
				Description: fmt.Sprintf("Employer RSA contribution (pension base: %d kobo)", taxResult.PensionBase),
			})
		}
		if taxResult.NSITF > 0 {
			details = append(details, domain.PayrollRunEntryDetail{
				Type: domain.DetailTypeEmployerCost,
				Name: "NSITF (Employer 1%)",
				Amount: taxResult.NSITF,
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
			EmployeeID:         emp.ID,
			Employee:           emp,
			GrossPay:           entryGross,
			TotalDeductions:    entryDeductions,
			Bonuses:            entryBonus,
			NetPay:             entryNet,
			EmployerPension:    taxResult.EmployerPension,
			EmployerNSITF:      taxResult.NSITF,
			TotalEmployerCost:  taxResult.TotalEmployerCosts,
			TotalCostToCompany: entryGross + taxResult.TotalEmployerCosts,
			Details:            details,
		})

		totalGross += entryGross
		totalDeductions += entryDeductions
		totalNet += entryNet
		totalEmployerCosts += taxResult.TotalEmployerCosts
	}

	// 3. Assemble the final payroll run object.
	now := time.Now()
	// Normalize period to first day of the month
	normalizedPeriod := time.Date(period.Year(), period.Month(), 1, 0, 0, 0, 0, time.UTC)

	payrollRun := &domain.PayrollRun{
		BusinessID:         businessID,
		Period:             normalizedPeriod,
		Status:             domain.StatusDraft,
		TotalGrossPay:      totalGross,
		TotalDeductions:    totalDeductions,
		TotalNetPay:        totalNet,
		TotalEmployerCosts: totalEmployerCosts,
		TotalCostToCompany: totalGross + totalEmployerCosts,
		ScheduledFor:       now.AddDate(0, 0, 5), // Default to 5 days from now
		Entries:            runEntries,
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

	// Send approval confirmation to operators
	if s.notificationSvc != nil {
		go func() {
			operators, _ := s.userRepo.FindByBusinessID(context.Background(), approver.BusinessID)
			for _, op := range operators {
				if op.Role == domain.RoleOperator {
					s.notificationSvc.SendEmail(context.Background(), op.Email,
						"Payroll Approved: "+run.Period.Format("January 2006"),
						fmt.Sprintf("The payroll run for %s has been approved and is scheduled for processing.", run.Period.Format("January 2006")))
				}
			}
		}()
	}

	// If auto-process is enabled, process immediately (for testing)
	if business.PayrollAutoProcess {
		log.Info().Uint("run_id", runID).Uint("business_id", approver.BusinessID).Msg("Auto-processing payroll run (business config: auto-process enabled)")
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
		log.Info().Uint("run_id", runID).Uint("business_id", user.BusinessID).Msg("Auto-approving payroll run (business config: no approval required)")

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
	log.Info().Uint("run_id", runID).Uint("business_id", businessID).Msg("Processing payroll run instantly")

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

	// Batch-load employees for entries that don't have employee data (eliminates N+1)
	var missingEmpIDs []uint
	for i := range run.Entries {
		if run.Entries[i].Employee == nil {
			missingEmpIDs = append(missingEmpIDs, run.Entries[i].EmployeeID)
		}
	}
	if len(missingEmpIDs) > 0 {
		loadedEmployees, err := s.employeeRepo.FindByIDs(ctx, missingEmpIDs, businessID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to batch-load employees for payroll entries")
		} else {
			empMap := make(map[uint]*domain.Employee, len(loadedEmployees))
			for _, emp := range loadedEmployees {
				empMap[emp.ID] = emp
			}
			for i := range run.Entries {
				if run.Entries[i].Employee == nil {
					if emp, ok := empMap[run.Entries[i].EmployeeID]; ok {
						run.Entries[i].Employee = emp
					} else {
						log.Warn().Uint("entry_id", run.Entries[i].ID).Uint("employee_id", run.Entries[i].EmployeeID).Msg("Failed to load employee for entry")
					}
				}
			}
		}
	}

	// Convert payroll entries to bulk transfer requests
	transfers := make([]domain.SingleTransferRequest, 0, len(run.Entries))
	for _, entry := range run.Entries {
		if entry.Employee == nil {
			log.Warn().Uint("entry_id", entry.ID).Msg("Skipping entry with nil employee")
			continue
		}

		// Convert net pay from smallest currency unit (kobo) to main unit (NGN)
		amountStr := fmt.Sprintf("%d", entry.NetPay)

		// Map bank name to bank code (simplified - should use a proper mapping)
		bankCode, err := s.mapBankNameToCode(entry.Employee.BankName)
		if err != nil {
			log.Warn().Err(err).Uint("employee_id", entry.EmployeeID).Str("bank_name", entry.Employee.BankName).Msg("Skipping entry: could not map bank name to code")
			continue
		}

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
		log.Info().Uint("run_id", runID).Int("total_transfers", len(transfers)).Int("successful", bulkResp.SuccessfulTransfers).Int("verified_in_db", verificationResult.VerifiedCount).Msg("Payroll run processed successfully")
	} else if verificationResult.AllVerified {
		// Transfers created in DB but some may have failed at provider
		// If all transfers are verified in DB, mark as completed (provider will update status via webhook)
		run.Status = domain.StatusCompleted
		run.PaymentReference = bulkReq.BatchReference
		now := time.Now()
		run.ProcessedAt = &now
		log.Info().Uint("run_id", runID).Int("total_transfers", len(transfers)).Int("api_successful", bulkResp.SuccessfulTransfers).Int("verified_in_db", verificationResult.VerifiedCount).Msg("Payroll run completed (all transfers verified in database)")
	} else {
		// Some transfers failed or weren't created
		run.Status = domain.StatusFailed
		log.Warn().Uint("run_id", runID).Int("successful", bulkResp.SuccessfulTransfers).Int("total", len(transfers)).Int("verified", verificationResult.VerifiedCount).Msg("Payroll run failed")
	}

	if err := s.payrollRepo.Update(ctx, run); err != nil {
		log.Error().Err(err).Msg("Failed to update payroll run status")
		return nil, fmt.Errorf("failed to update payroll run: %w", err)
	}

	// Send payslip notifications to employees on successful processing
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
		log.Error().Err(err).Msg("Failed to fetch transfers for verification")
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

	log.Info().Uint("payroll_run_id", payrollRunID).Int("expected", expectedCount).Int("verified", verifiedCount).Int("total_fetched", total).Bool("all_verified", allVerified).Msg("Transfer verification completed")

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

// mapBankNameToCode maps bank names to bank codes (simplified mapping).
func formatKoboAsNGN(kobo int64) string {
	return fmt.Sprintf("%.2f", float64(kobo)/100.0)
}

// Returns an error if the bank name cannot be mapped.
// In production, this should use a proper bank code lookup service.
func (s *payrollService) mapBankNameToCode(bankName string) (string, error) {
	// Common Nigerian bank codes (keys are lowercase for case-insensitive matching)
	bankMap := map[string]string{
		"access bank":             "044",
		"access":                  "044",
		"united bank for africa":  "033",
		"uba":                     "033",
		"guaranty trust bank":     "058",
		"gtb":                     "058",
		"gtbank":                  "058",
		"first bank":              "011",
		"zenith bank":             "057",
		"zenith":                  "057",
		"fidelity bank":           "070",
		"union bank":              "032",
		"stanbic ibtc":            "221",
		"ecobank":                 "050",
	}

	// Case-insensitive lookup
	normalizedName := strings.ToLower(strings.TrimSpace(bankName))
	if code, ok := bankMap[normalizedName]; ok {
		return code, nil
	}

	// Try prefix match (case-insensitive)
	for name, code := range bankMap {
		if strings.HasPrefix(normalizedName, name) {
			return code, nil
		}
	}

	return "", fmt.Errorf("unsupported bank name: %q", bankName)
}
