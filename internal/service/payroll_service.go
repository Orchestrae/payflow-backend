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
	employeeRepo    repository.EmployeeRepository // Assumed to be created
	cadreRepo       repository.CadreRepository    // Assumed to be created
	txer            repository.Transactioner
	payoutSvc       PayoutService
	notificationSvc NotificationService
	scheduler       domain.Scheduler
	userRepo        repository.UserRepository
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
	txer repository.Transactioner,
	payoutSvc PayoutService,
	notificationSvc NotificationService,
	userRepo repository.UserRepository,
	scheduler domain.Scheduler,
) PayrollService {
	return &payrollService{
		payrollRepo:     payrollRepo,
		employeeRepo:    employeeRepo,
		cadreRepo:       cadreRepo,
		txer:            txer,
		payoutSvc:       payoutSvc,
		notificationSvc: notificationSvc,
		userRepo:        userRepo,
		scheduler:       scheduler,
	}
}

// CalculatePayrollRun performs an in-memory calculation of the payroll.
func (s *payrollService) CalculatePayrollRun(ctx context.Context, businessID uint, adjustments map[uint]int64) (*domain.PayrollRun, error) {
	// 1. Fetch all employees for the business.
	allEmployees, err := s.employeeRepo.FindByBusinessID(ctx, businessID)
	if err != nil {
		slog.Error("Failed to fetch employees", "error", err)
		return nil, fmt.Errorf("failed to fetch employees: %w", err)
	}

	// Filter for active employees
	var employees []*domain.Employee
	for _, emp := range allEmployees {
		if emp.IsActive {
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

		// Adjustments
		if adj, ok := adjustments[emp.ID]; ok {
			entryBonus = adj
			details = append(details, domain.PayrollRunEntryDetail{
				Type:   domain.DetailTypeBonus,
				Name:   "One-time Adjustment",
				Amount: adj,
			})
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
	payrollRun := &domain.PayrollRun{
		BusinessID:      businessID,
		Period:          time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		Status:          domain.StatusDraft,
		TotalGrossPay:   totalGross,
		TotalDeductions: totalDeductions,
		TotalNetPay:     totalNet,
		ScheduledFor:    now.AddDate(0, 0, 5),
		Entries:         runEntries,
	}

	return payrollRun, nil
}

// CreateAndStorePayrollRun ... (Keeping the wrapper)
func (s *payrollService) CreateAndStorePayrollRun(ctx context.Context, businessID uint, adjustments map[uint]int64) (*domain.PayrollRun, error) {
	payrollRun, err := s.CalculatePayrollRun(ctx, businessID, adjustments)
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

	run.Status = domain.StatusApproved
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, err
	}

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

	run.Status = domain.StatusPendingApproval
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to update payroll run status: %w", err)
	}

	return run, nil
}

// ProcessApprovedPayroll implements the method required by the scheduler's interface.
func (s *payrollService) ProcessApprovedPayroll(ctx context.Context, runID uint) error {
	log.Ctx(ctx).Info().Uint("run_id", runID).Msg("Processing approved payroll")

	// 1. Fetch run. Pass 0 as businessID to indicate system access (implementation must support this).
	run, err := s.payrollRepo.FindByID(ctx, runID, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch payroll run: %w", err)
	}
	if run.Status != domain.StatusApproved {
		log.Ctx(ctx).Warn().Str("status", string(run.Status)).Msg("Skipping payroll processing, not in approved state")
		return nil
	}

	// 2. Update status to 'Processing'
	run.Status = domain.StatusProcessing
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		return fmt.Errorf("failed to update payroll run status to processing: %w", err)
	}

	// 3. Call the external payment gateway service.
	ref, err := s.payoutSvc.DisburseBulkPayment(ctx, *run)
	if err != nil {
		run.Status = domain.StatusFailed
		if updateErr := s.payrollRepo.Update(ctx, run); updateErr != nil {
			log.Ctx(ctx).Error().Err(updateErr).Msg("Failed to update payroll run status to failed")
		}
		log.Ctx(ctx).Error().Err(err).Msg("Disbursement failed")
		return fmt.Errorf("disbursement failed: %w", err)
	}

	// 4. Payment succeeded, update status to 'Completed'.
	run.Status = domain.StatusCompleted
	run.PaymentReference = ref
	now := time.Now()
	run.ProcessedAt = &now
	if err := s.payrollRepo.Update(ctx, run); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("CRITICAL: Disbursement succeeded but failed to update run status")
		return fmt.Errorf("failed to update payroll run status to completed: %w", err)
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
