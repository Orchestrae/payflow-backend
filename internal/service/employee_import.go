package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"payflow/internal/domain"
)

// ParseEmployeeCSV parses a CSV file into employee structs.
// Expected headers: full_name,email,cadre_id,bank_name,bank_code,bank_account_number,tin,pension_rsa_pin,nhf_number,annual_rent_paid
func ParseEmployeeCSV(reader io.Reader) ([]*domain.Employee, []string, error) {
	r := csv.NewReader(reader)
	r.TrimLeadingSpace = true

	// Read header
	headers, err := r.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Map headers to column indexes
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// Validate required headers
	required := []string{"full_name", "email", "cadre_id", "bank_name", "bank_account_number"}
	for _, req := range required {
		if _, ok := headerMap[req]; !ok {
			return nil, nil, fmt.Errorf("missing required CSV header: %s", req)
		}
	}

	var employees []*domain.Employee
	var errors []string
	rowNum := 1 // 1-based (header is row 0)

	for {
		rowNum++
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("row %d: %v", rowNum, err))
			continue
		}

		emp, parseErr := parseEmployeeRow(record, headerMap, rowNum)
		if parseErr != "" {
			errors = append(errors, parseErr)
			continue
		}

		employees = append(employees, emp)
	}

	return employees, errors, nil
}

func parseEmployeeRow(record []string, headerMap map[string]int, rowNum int) (*domain.Employee, string) {
	getField := func(name string) string {
		if idx, ok := headerMap[name]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	fullName := getField("full_name")
	email := getField("email")
	cadreIDStr := getField("cadre_id")

	if fullName == "" || email == "" || cadreIDStr == "" {
		return nil, fmt.Sprintf("row %d: full_name, email, and cadre_id are required", rowNum)
	}

	cadreID, err := strconv.ParseUint(cadreIDStr, 10, 32)
	if err != nil {
		return nil, fmt.Sprintf("row %d: invalid cadre_id '%s'", rowNum, cadreIDStr)
	}

	emp := &domain.Employee{
		FullName:          fullName,
		Email:             email,
		CadreID:           uint(cadreID),
		BankName:          getField("bank_name"),
		BankCode:          getField("bank_code"),
		BankAccountNumber: getField("bank_account_number"),
	}

	// Optional statutory fields
	if tin := getField("tin"); tin != "" {
		emp.TIN = &tin
	}
	if pin := getField("pension_rsa_pin"); pin != "" {
		emp.PensionRSAPIN = &pin
	}
	if nhf := getField("nhf_number"); nhf != "" {
		emp.NHFNumber = &nhf
	}
	if rent := getField("annual_rent_paid"); rent != "" {
		if v, err := strconv.ParseInt(rent, 10, 64); err == nil {
			emp.AnnualRentPaid = v
		}
	}

	return emp, ""
}
