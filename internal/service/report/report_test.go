package report

import (
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"payflow/internal/domain"
)

func strPtr(s string) *string { return &s }

func testPayrollRun() *domain.PayrollRun {
	return &domain.PayrollRun{
		Period: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Entries: []domain.PayrollRunEntry{
			{
				EmployeeID: 1,
				GrossPay:   30000000, // 300,000 NGN
				TotalDeductions: 5986800,
				NetPay:     24013200,
				EmployerPension: 2800000,
				EmployerNSITF:   300000,
				Employee: &domain.Employee{
					FullName:          "John Doe",
					Email:             "john@example.com",
					BankName:          "GTBank",
					BankCode:          "058",
					BankAccountNumber: "0123456789",
					TIN:               strPtr("1234567890"),
					PensionRSAPIN:     strPtr("PEN001234"),
					NHFNumber:         strPtr("NHF001234"),
				},
				Details: []domain.PayrollRunEntryDetail{
					{Type: domain.DetailTypeEarning, Name: "Basic Salary", Amount: 20000000},
					{Type: domain.DetailTypeEarning, Name: "Housing Allowance", Amount: 5000000},
					{Type: domain.DetailTypeEarning, Name: "Transport Allowance", Amount: 3000000},
					{Type: domain.DetailTypeStatutoryDeduction, Name: "PAYE Income Tax", Amount: 3246800},
					{Type: domain.DetailTypeStatutoryDeduction, Name: "Pension (Employee 8%)", Amount: 2240000},
					{Type: domain.DetailTypeStatutoryDeduction, Name: "NHF (2.5%)", Amount: 500000},
					{Type: domain.DetailTypeEmployerCost, Name: "Pension (Employer 10%)", Amount: 2800000},
					{Type: domain.DetailTypeEmployerCost, Name: "NSITF (Employer 1%)", Amount: 300000},
				},
			},
		},
	}
}

func testBusiness() *domain.Business {
	return &domain.Business{
		Name: "Test Company Ltd",
	}
}

func parseCSV(t *testing.T, data []byte) [][]string {
	t.Helper()
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}
	return records
}

func TestGeneratePAYEReport(t *testing.T) {
	data, err := GeneratePAYEReport(testPayrollRun(), testBusiness())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := parseCSV(t, data)
	if len(records) != 2 { // header + 1 employee
		t.Fatalf("expected 2 rows, got %d", len(records))
	}

	// Check header
	if records[0][0] != "Employee Name" {
		t.Errorf("expected header 'Employee Name', got '%s'", records[0][0])
	}

	// Check data
	if records[1][0] != "John Doe" {
		t.Errorf("expected 'John Doe', got '%s'", records[1][0])
	}
	if records[1][1] != "1234567890" {
		t.Errorf("expected TIN '1234567890', got '%s'", records[1][1])
	}
	// PAYE amount: 3,246,800 kobo = 32,468.00 NGN
	if records[1][3] != "32468.00" {
		t.Errorf("expected PAYE '32468.00', got '%s'", records[1][3])
	}
	if records[1][4] != "2026-01" {
		t.Errorf("expected period '2026-01', got '%s'", records[1][4])
	}
}

func TestGeneratePensionSchedule(t *testing.T) {
	data, err := GeneratePensionSchedule(testPayrollRun(), testBusiness())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := parseCSV(t, data)
	if len(records) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(records))
	}

	// Employee pension: 2,240,000 kobo = 22,400.00
	if records[1][3] != "22400.00" {
		t.Errorf("expected employee pension '22400.00', got '%s'", records[1][3])
	}
	// Employer pension: 2,800,000 kobo = 28,000.00
	if records[1][4] != "28000.00" {
		t.Errorf("expected employer pension '28000.00', got '%s'", records[1][4])
	}
	// Total: 50,400.00
	if records[1][5] != "50400.00" {
		t.Errorf("expected total pension '50400.00', got '%s'", records[1][5])
	}
	// RSA PIN
	if records[1][1] != "PEN001234" {
		t.Errorf("expected RSA PIN 'PEN001234', got '%s'", records[1][1])
	}
}

func TestGenerateNHFSchedule(t *testing.T) {
	data, err := GenerateNHFSchedule(testPayrollRun(), testBusiness())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := parseCSV(t, data)
	if len(records) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(records))
	}

	// NHF: 500,000 kobo = 5,000.00
	if records[1][2] != "5000.00" {
		t.Errorf("expected NHF '5000.00', got '%s'", records[1][2])
	}
	if records[1][1] != "NHF001234" {
		t.Errorf("expected NHF number 'NHF001234', got '%s'", records[1][1])
	}
}

func TestGenerateBankSchedule(t *testing.T) {
	data, err := GenerateBankSchedule(testPayrollRun(), testBusiness())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := parseCSV(t, data)
	if len(records) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(records))
	}

	if records[1][1] != "GTBank" {
		t.Errorf("expected bank 'GTBank', got '%s'", records[1][1])
	}
	if records[1][3] != "0123456789" {
		t.Errorf("expected account '0123456789', got '%s'", records[1][3])
	}
	// Net pay: 24,013,200 kobo = 240,132.00
	if records[1][4] != "240132.00" {
		t.Errorf("expected net pay '240132.00', got '%s'", records[1][4])
	}
}

func TestEmptyPayroll(t *testing.T) {
	emptyRun := &domain.PayrollRun{
		Period:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Entries: []domain.PayrollRunEntry{},
	}
	business := testBusiness()

	// All should return headers only
	tests := []struct {
		name string
		gen  func(*domain.PayrollRun, *domain.Business) ([]byte, error)
	}{
		{"PAYE", GeneratePAYEReport},
		{"Pension", GeneratePensionSchedule},
		{"NHF", GenerateNHFSchedule},
		{"Bank", GenerateBankSchedule},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.gen(emptyRun, business)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			records := parseCSV(t, data)
			if len(records) != 1 { // headers only
				t.Errorf("expected 1 row (header), got %d", len(records))
			}
		})
	}
}

func TestGeneratePayrollSummary(t *testing.T) {
	data, err := GeneratePayrollSummary(testPayrollRun(), testBusiness())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := parseCSV(t, data)
	// header + 1 employee + totals = 3 rows
	if len(records) != 3 {
		t.Fatalf("expected 3 rows (header + data + totals), got %d", len(records))
	}

	if records[0][0] != "Employee Name" {
		t.Errorf("expected header 'Employee Name', got '%s'", records[0][0])
	}
	if records[1][0] != "John Doe" {
		t.Errorf("expected 'John Doe', got '%s'", records[1][0])
	}
	if records[2][0] != "TOTALS" {
		t.Errorf("expected 'TOTALS', got '%s'", records[2][0])
	}
	// Totals gross should match single employee gross
	if records[2][2] != records[1][2] {
		t.Errorf("totals gross (%s) should match data gross (%s)", records[2][2], records[1][2])
	}
}
