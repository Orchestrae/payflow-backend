// internal/domain/error.go
package domain

import "errors"

var (
	// ErrNotFound indicates that a requested resource could not be found.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict indicates a duplicate resource was created (e.g., user with same email).
	ErrConflict = errors.New("resource conflict or duplicate")

	// ErrUnauthorized indicates a failed authentication attempt.
	ErrUnauthorized = errors.New("unauthorized: invalid credentials")

	// ErrForbidden indicates that the authenticated user does not have permission.
	ErrForbidden = errors.New("forbidden: insufficient permissions")

	// ErrValidationFailed indicates that request data failed validation.
	ErrValidationFailed = errors.New("validation failed")

	// ErrPaymentGatewayFailed indicates a problem with the external payment provider.
	ErrPaymentGatewayFailed = errors.New("payment gateway operation failed")

	// ErrInternalServer indicates an unexpected error in the system.
	ErrInternalServer = errors.New("internal server error")
)
