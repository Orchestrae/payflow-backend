package report

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"payflow/internal/domain"
)

// GeneratePayrollSummary generates a comprehensive payroll register CSV with all employees and totals.
func GeneratePayrollSummary(run *domain.PayrollRun, business *domain.Business) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	period := run.Period.Format("2006-01")

	// Header
	w.Write([]string{
		"Employee Name", "Email", "Gross Pay (NGN)",
		"PAYE (NGN)", "Pension Employee (NGN)", "Pension Employer (NGN)",
		"NHF (NGN)", "Custom Deductions (NGN)", "Total Deductions (NGN)",
		"Net Pay (NGN)", "NSITF Employer (NGN)", "Total Cost to Company (NGN)",
		"Period", "Employer Name",
	})

	// Accumulators for totals row
	var totalGross, totalPAYE, totalPensionEmp, totalPensionEr int64
	var totalNHF, totalCustom, totalDeductions, totalNet int64
	var totalNSITF, totalCTC int64

	for _, entry := range run.Entries {
		if entry.Employee == nil {
			continue
		}

		// Extract amounts from details
		paye := detailAmount(entry.Details, domain.DetailTypeStatutoryDeduction, "PAYE")
		pensionEmp := detailAmount(entry.Details, domain.DetailTypeStatutoryDeduction, "Pension (Employee")
		nhf := detailAmount(entry.Details, domain.DetailTypeStatutoryDeduction, "NHF")

		// Custom deductions = total deductions - statutory deductions
		statutoryTotal := paye + pensionEmp + nhf
		customDeductions := entry.TotalDeductions - statutoryTotal
		if customDeductions < 0 {
			customDeductions = 0
		}

		// Accumulate
		totalGross += entry.GrossPay
		totalPAYE += paye
		totalPensionEmp += pensionEmp
		totalPensionEr += entry.EmployerPension
		totalNHF += nhf
		totalCustom += customDeductions
		totalDeductions += entry.TotalDeductions
		totalNet += entry.NetPay
		totalNSITF += entry.EmployerNSITF
		totalCTC += entry.TotalCostToCompany

		w.Write([]string{
			entry.Employee.FullName,
			entry.Employee.Email,
			formatNGN(entry.GrossPay),
			formatNGN(paye),
			formatNGN(pensionEmp),
			formatNGN(entry.EmployerPension),
			formatNGN(nhf),
			formatNGN(customDeductions),
			formatNGN(entry.TotalDeductions),
			formatNGN(entry.NetPay),
			formatNGN(entry.EmployerNSITF),
			formatNGN(entry.TotalCostToCompany),
			period,
			business.Name,
		})
	}

	// Totals row
	w.Write([]string{
		"TOTALS", "",
		formatNGN(totalGross),
		formatNGN(totalPAYE),
		formatNGN(totalPensionEmp),
		formatNGN(totalPensionEr),
		formatNGN(totalNHF),
		formatNGN(totalCustom),
		formatNGN(totalDeductions),
		formatNGN(totalNet),
		formatNGN(totalNSITF),
		formatNGN(totalCTC),
		period,
		business.Name,
	})

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("failed to generate payroll summary: %w", err)
	}

	return buf.Bytes(), nil
}

// detailAmount extracts the amount from a detail matching type and name prefix.
func detailAmount(details []domain.PayrollRunEntryDetail, detailType domain.PayrollEntryDetailType, namePrefix string) int64 {
	d := findDetail(details, detailType, namePrefix)
	if d != nil {
		return d.Amount
	}
	return 0
}
