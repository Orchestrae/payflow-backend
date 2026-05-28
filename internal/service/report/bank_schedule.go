package report

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"payflow/internal/domain"
)

// GenerateBankSchedule generates a bank transfer schedule CSV for bulk payment processing.
func GenerateBankSchedule(run *domain.PayrollRun, business *domain.Business) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	period := run.Period.Format("2006-01")

	// Header
	w.Write([]string{
		"Employee Name", "Bank Name", "Bank Code", "Account Number",
		"Net Pay (NGN)", "Period", "Employer Name",
	})

	for _, entry := range run.Entries {
		if entry.Employee == nil {
			continue
		}

		w.Write([]string{
			entry.Employee.FullName,
			entry.Employee.BankName,
			entry.Employee.BankCode,
			entry.Employee.BankAccountNumber,
			formatNGN(entry.NetPay),
			period,
			business.Name,
		})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("failed to generate bank schedule: %w", err)
	}

	return buf.Bytes(), nil
}
