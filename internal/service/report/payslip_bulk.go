package report

import (
	"archive/zip"
	"bytes"
	"fmt"

	"payflow/internal/domain"
)

// GenerateAllPayslips generates a ZIP file containing one PDF payslip per employee.
func GenerateAllPayslips(run *domain.PayrollRun, business *domain.Business) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for i := range run.Entries {
		entry := &run.Entries[i]
		if entry.Employee == nil {
			continue
		}

		pdfBytes, err := GeneratePayslip(entry, business, run.Period)
		if err != nil {
			return nil, fmt.Errorf("failed to generate payslip for employee %d: %w", entry.EmployeeID, err)
		}

		filename := fmt.Sprintf("payslip_%s_%s.pdf",
			sanitizeFilename(entry.Employee.FullName),
			run.Period.Format("2006-01"),
		)

		fw, err := zw.Create(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to create zip entry: %w", err)
		}
		if _, err := fw.Write(pdfBytes); err != nil {
			return nil, fmt.Errorf("failed to write pdf to zip: %w", err)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip: %w", err)
	}

	return buf.Bytes(), nil
}

// sanitizeFilename replaces spaces and special chars for safe filenames.
func sanitizeFilename(name string) string {
	var result []byte
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, byte(c))
		} else if c == ' ' {
			result = append(result, '_')
		}
	}
	return string(result)
}
