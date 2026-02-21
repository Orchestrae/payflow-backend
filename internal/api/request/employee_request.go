// internal/api/request/employee_request.go
package request

type CreateEmployeeRequest struct {
	CadreID           uint   `json:"cadre_id" validate:"required"`
	FullName          string `json:"full_name" validate:"required"`
	Email             string `json:"email" validate:"required,email"`
	BankName          string `json:"bank_name" validate:"required"`
	BankCode          string `json:"bank_code"`
	BankAccountNumber string `json:"bank_account_number" validate:"required"`
}

type UpdateEmployeeRequest struct {
	CadreID           uint   `json:"cadre_id,omitempty"`
	FullName          string `json:"full_name,omitempty"`
	Email             string `json:"email,omitempty,email"`
	BankName          string `json:"bank_name,omitempty"`
	BankCode          string `json:"bank_code,omitempty"`
	BankAccountNumber string `json:"bank_account_number,omitempty"`
	IsActive          *bool  `json:"is_active,omitempty"` // Pointer to distinguish between false and not provided
}
