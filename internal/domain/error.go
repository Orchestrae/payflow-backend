// internal/domain/error.go
package domain

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound indicates that a requested resource could not be found.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict indicates a duplicate resource was created (e.g., user with same email).
	ErrConflict = errors.New("resource conflict or duplicate")

	// ErrUnauthorized indicates a failed authentication attempt.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates that the authenticated user does not have permission.
	ErrForbidden = errors.New("forbidden: insufficient permissions")

	// ErrValidationFailed indicates that request data failed validation.
	ErrValidationFailed = errors.New("validation failed")

	// ErrPaymentGatewayFailed indicates a problem with the external payment provider.
	ErrPaymentGatewayFailed = errors.New("payment gateway operation failed")

	// ErrInternalServer indicates an unexpected error in the system.
	ErrInternalServer = errors.New("internal server error")
)

// TransferAmountError represents an error for invalid transfer amount
type TransferAmountError struct {
	Amount    string
	MinAmount int64
	MaxAmount int64
	Message   string
}

func (e *TransferAmountError) Error() string {
	return e.Message
}

// NewTransferAmountBelowMinError creates an error for amount below minimum
func NewTransferAmountBelowMinError(amount string, minAmount int64) *TransferAmountError {
	return &TransferAmountError{
		Amount:    amount,
		MinAmount: minAmount,
		Message:   fmt.Sprintf("transfer amount %s is below the minimum allowed amount of %d", amount, minAmount),
	}
}

// NewTransferAmountAboveMaxError creates an error for amount above maximum
func NewTransferAmountAboveMaxError(amount string, maxAmount int64) *TransferAmountError {
	return &TransferAmountError{
		Amount:    amount,
		MaxAmount: maxAmount,
		Message:   fmt.Sprintf("transfer amount %s exceeds the maximum allowed amount of %d", amount, maxAmount),
	}
}
