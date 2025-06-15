// internal/domain/employee.go
package domain

import "time"

type Employee struct {
	ID                uint
	BusinessID        uint
	CadreID           uint
	FullName          string
	Email             string
	BankName          string
	BankAccountNumber string
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time

	// Relational field
	Cadre *Cadre
}
