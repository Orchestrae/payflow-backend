package report

import (
	"fmt"

	"payflow/internal/domain"
)

// formatNGN converts kobo (int64) to NGN string (e.g., 50000000 → "500,000.00").
func formatNGN(kobo int64) string {
	naira := float64(kobo) / 100.0
	return fmt.Sprintf("%.2f", naira)
}

// findDetail finds a payroll entry detail by type and name prefix.
func findDetail(details []domain.PayrollRunEntryDetail, detailType domain.PayrollEntryDetailType, namePrefix string) *domain.PayrollRunEntryDetail {
	for i := range details {
		if details[i].Type == detailType && len(details[i].Name) >= len(namePrefix) && details[i].Name[:len(namePrefix)] == namePrefix {
			return &details[i]
		}
	}
	return nil
}

// findAllDetails finds all details matching a type.
func findAllDetails(details []domain.PayrollRunEntryDetail, detailType domain.PayrollEntryDetailType) []domain.PayrollRunEntryDetail {
	var result []domain.PayrollRunEntryDetail
	for _, d := range details {
		if d.Type == detailType {
			result = append(result, d)
		}
	}
	return result
}

// safeString returns the string value or empty if pointer is nil.
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
