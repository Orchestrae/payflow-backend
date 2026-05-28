package report

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"payflow/internal/domain"
)

// GeneratePAYEReport generates a PAYE return CSV for TaxProMax upload.
func GeneratePAYEReport(run *domain.PayrollRun, business *domain.Business) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	period := run.Period.Format("2006-01")

	// Header
	w.Write([]string{
		"Employee Name", "TIN", "Gross Pay (NGN)", "PAYE Tax (NGN)", "Period",
		"Employer Name",
	})

	for _, entry := range run.Entries {
		if entry.Employee == nil {
			continue
		}

		// Find PAYE detail
		payeDetail := findDetail(entry.Details, domain.DetailTypeStatutoryDeduction, "PAYE")
		payeAmount := int64(0)
		if payeDetail != nil {
			payeAmount = payeDetail.Amount
		}

		w.Write([]string{
			entry.Employee.FullName,
			safeString(entry.Employee.TIN),
			formatNGN(entry.GrossPay),
			formatNGN(payeAmount),
			period,
			business.Name,
		})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("failed to generate PAYE report: %w", err)
	}

	return buf.Bytes(), nil
}
