package report

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"payflow/internal/domain"
)

// GenerateNHFSchedule generates an NHF contribution schedule CSV for FMBN upload.
func GenerateNHFSchedule(run *domain.PayrollRun, business *domain.Business) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	period := run.Period.Format("2006-01")

	// Header
	w.Write([]string{
		"Employee Name", "NHF Number", "NHF Contribution (NGN)", "Period",
		"Employer Name",
	})

	for _, entry := range run.Entries {
		if entry.Employee == nil {
			continue
		}

		nhfDetail := findDetail(entry.Details, domain.DetailTypeStatutoryDeduction, "NHF")
		nhfAmount := int64(0)
		if nhfDetail != nil {
			nhfAmount = nhfDetail.Amount
		}

		// Skip employees with no NHF contribution
		if nhfAmount == 0 {
			continue
		}

		w.Write([]string{
			entry.Employee.FullName,
			safeString(entry.Employee.NHFNumber),
			formatNGN(nhfAmount),
			period,
			business.Name,
		})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("failed to generate NHF schedule: %w", err)
	}

	return buf.Bytes(), nil
}
