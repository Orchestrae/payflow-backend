package report

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"payflow/internal/domain"
)

// GeneratePensionSchedule generates a pension remittance schedule CSV for PFA upload.
func GeneratePensionSchedule(run *domain.PayrollRun, business *domain.Business) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	period := run.Period.Format("2006-01")

	// Header
	w.Write([]string{
		"Employee Name", "RSA PIN", "Employer Name",
		"Employee Contribution (NGN)", "Employer Contribution (NGN)",
		"Total Contribution (NGN)", "Period",
	})

	for _, entry := range run.Entries {
		if entry.Employee == nil {
			continue
		}

		// Find employee pension detail
		pensionDetail := findDetail(entry.Details, domain.DetailTypeStatutoryDeduction, "Pension (Employee")
		employeePension := int64(0)
		if pensionDetail != nil {
			employeePension = pensionDetail.Amount
		}

		totalPension := employeePension + entry.EmployerPension

		w.Write([]string{
			entry.Employee.FullName,
			safeString(entry.Employee.PensionRSAPIN),
			business.Name,
			formatNGN(employeePension),
			formatNGN(entry.EmployerPension),
			formatNGN(totalPension),
			period,
		})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("failed to generate pension schedule: %w", err)
	}

	return buf.Bytes(), nil
}
