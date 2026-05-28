package service

import (
	"strings"
	"testing"
)

func TestParseEmployeeCSV_Valid(t *testing.T) {
	csv := `full_name,email,cadre_id,bank_name,bank_code,bank_account_number,tin,pension_rsa_pin,nhf_number,annual_rent_paid
John Doe,john@acme.com,1,GTBank,058,0123456789,TIN001,PEN001,NHF001,240000000
Jane Smith,jane@acme.com,2,Access Bank,044,9876543210,,,, `

	employees, errors, err := ParseEmployeeCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errors) != 0 {
		t.Errorf("expected no errors, got %v", errors)
	}
	if len(employees) != 2 {
		t.Fatalf("expected 2 employees, got %d", len(employees))
	}

	// First employee - all fields
	if employees[0].FullName != "John Doe" {
		t.Errorf("expected 'John Doe', got '%s'", employees[0].FullName)
	}
	if employees[0].CadreID != 1 {
		t.Errorf("expected cadre_id 1, got %d", employees[0].CadreID)
	}
	if employees[0].TIN == nil || *employees[0].TIN != "TIN001" {
		t.Errorf("expected TIN 'TIN001'")
	}
	if employees[0].AnnualRentPaid != 240000000 {
		t.Errorf("expected annual_rent_paid 240000000, got %d", employees[0].AnnualRentPaid)
	}

	// Second employee - optional fields empty
	if employees[1].TIN != nil {
		t.Errorf("expected nil TIN for second employee")
	}
}

func TestParseEmployeeCSV_MissingHeaders(t *testing.T) {
	csv := `name,email_address
John,john@test.com`

	_, _, err := ParseEmployeeCSV(strings.NewReader(csv))
	if err == nil {
		t.Error("expected error for missing required headers")
	}
	if !strings.Contains(err.Error(), "missing required CSV header") {
		t.Errorf("expected 'missing required CSV header' error, got: %v", err)
	}
}

func TestParseEmployeeCSV_InvalidData(t *testing.T) {
	csv := `full_name,email,cadre_id,bank_name,bank_account_number
John Doe,john@acme.com,abc,GTBank,0123456789
,jane@acme.com,1,Access,9876543210
Valid User,valid@acme.com,2,Zenith,1111111111`

	employees, errors, err := ParseEmployeeCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Row 2: invalid cadre_id "abc"
	// Row 3: empty full_name
	if len(errors) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(errors), errors)
	}

	// Row 4: valid
	if len(employees) != 1 {
		t.Errorf("expected 1 valid employee, got %d", len(employees))
	}
	if employees[0].FullName != "Valid User" {
		t.Errorf("expected 'Valid User', got '%s'", employees[0].FullName)
	}
}

func TestParseEmployeeCSV_EmptyFile(t *testing.T) {
	csv := `full_name,email,cadre_id,bank_name,bank_account_number`

	employees, errors, err := ParseEmployeeCSV(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(employees) != 0 {
		t.Errorf("expected 0 employees, got %d", len(employees))
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}
}
