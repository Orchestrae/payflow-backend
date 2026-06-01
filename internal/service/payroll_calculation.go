// internal/service/payroll_calculation.go
package service

import (
	"context"
	"fmt"
	"payflow/internal/domain"
	"payflow/internal/service/tax"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

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

	var skippedEmployees []string
	for _, emp := range employees {
		if emp.Cadre == nil {
			skippedEmployees = append(skippedEmployees, fmt.Sprintf("%s (no cadre)", emp.FullName))
			log.Warn().Uint("employee_id", emp.ID).Str("employee_name", emp.FullName).Msg("Employee without cadre — skipped")
			continue
		}

		if len(emp.Cadre.EarningComponents) == 0 {
			skippedEmployees = append(skippedEmployees, fmt.Sprintf("%s (cadre '%s' has no earning components)", emp.FullName, emp.Cadre.Name))
			log.Warn().Uint("employee_id", emp.ID).Str("cadre", emp.Cadre.Name).Msg("Cadre has zero earning components — skipped")
			continue
		}

		entry, taxResult := s.calculateEmployeeEntry(ctx, emp, business, adjustments[emp.ID])

		runEntries = append(runEntries, entry)
		totalGross += entry.GrossPay
		totalDeductions += entry.TotalDeductions
		totalNet += entry.NetPay
		totalEmployerCosts += taxResult.TotalEmployerCosts
	}

	// 3. Assemble the final payroll run object.
	now := time.Now()
	normalizedPeriod := time.Date(period.Year(), period.Month(), 1, 0, 0, 0, 0, time.UTC)

	if len(runEntries) == 0 {
		msg := "no employees could be processed for payroll"
		if len(skippedEmployees) > 0 {
			msg = fmt.Sprintf("all employees were skipped: %v", skippedEmployees)
		}
		return nil, fmt.Errorf("%w: %s", domain.ErrValidationFailed, msg)
	}

	if len(skippedEmployees) > 0 {
		log.Warn().Strs("skipped", skippedEmployees).Msgf("%d employees skipped during payroll calculation", len(skippedEmployees))
	}

	payrollRun := &domain.PayrollRun{
		BusinessID:         businessID,
		Period:             normalizedPeriod,
		Status:             domain.StatusDraft,
		TotalGrossPay:      totalGross,
		TotalDeductions:    totalDeductions,
		TotalNetPay:        totalNet,
		TotalEmployerCosts: totalEmployerCosts,
		TotalCostToCompany: totalGross + totalEmployerCosts,
		ScheduledFor:       now.AddDate(0, 0, 5),
		Entries:            runEntries,
	}

	return payrollRun, nil
}

// calculateEmployeeEntry computes a single employee's payroll entry.
func (s *payrollService) calculateEmployeeEntry(ctx context.Context, emp *domain.Employee, business *domain.Business, employeeAdjustments []EmployeeAdjustment) (domain.PayrollRunEntry, tax.Result) {
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
			Type:   domain.DetailTypeStatutoryDeduction,
			Name:   "NHF (2.5%)",
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
			Type:   domain.DetailTypeEmployerCost,
			Name:   "NSITF (Employer 1%)",
			Amount: taxResult.NSITF,
		})
	}

	// Deduct active loan repayments
	if s.loanRepo != nil {
		activeLoans, err := s.loanRepo.FindActiveByEmployeeID(ctx, emp.ID)
		if err == nil {
			for _, loan := range activeLoans {
				if loan.RemainingBalance <= 0 {
					continue
				}
				deductionAmt := loan.MonthlyDeduction
				if deductionAmt > loan.RemainingBalance {
					deductionAmt = loan.RemainingBalance
				}
				entryDeductions += deductionAmt
				details = append(details, domain.PayrollRunEntryDetail{
					Type:        domain.DetailTypeDeduction,
					Name:        fmt.Sprintf("Loan Repayment: %s", loan.Description),
					Amount:      deductionAmt,
					Description: fmt.Sprintf("Loan #%d — remaining: %d kobo", loan.ID, loan.RemainingBalance-deductionAmt),
				})
			}
		} else {
			log.Warn().Err(err).Uint("employee_id", emp.ID).Msg("Failed to load loans for payroll — skipping loan deductions")
		}
	}

	// Process detailed adjustments for this employee
	for _, adj := range employeeAdjustments {
		componentType := adj.ComponentType
		if componentType == "" {
			if adj.Amount >= 0 {
				componentType = "earnings"
			} else {
				componentType = "deduction"
			}
		}

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
			entryDeductions += -adj.Amount
		}

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

	entryNet := entryGross + entryBonus - entryDeductions

	entry := domain.PayrollRunEntry{
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
	}

	return entry, taxResult
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
