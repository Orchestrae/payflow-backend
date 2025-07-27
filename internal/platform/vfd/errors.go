// internal/platform/vfd/errors.go
package vfd

import "errors"

var (
	// ErrCompanyExists is returned when VFD indicates a company with the same RC Number or Name already exists.
	ErrCompanyExists = errors.New("vfd: company with same RC number or name already exists")

	// ErrAccountCreationFailed is a generic error for when VFD fails the account creation for an unspecified reason.
	ErrAccountCreationFailed = errors.New("vfd: account creation failed")

	// ErrNotAuthorized is returned when the API credentials lack the permission to create clients.
	ErrNotAuthorized = errors.New("vfd: not authorized to create clients")

	// ErrAuthenticationFailed is returned when generating an access token fails.
	ErrAuthenticationFailed = errors.New("vfd: authentication failed")

	// ErrInvalidResponse is returned when the response from VFD is malformed or unexpected.
	ErrInvalidResponse = errors.New("vfd: invalid response from api")
)
