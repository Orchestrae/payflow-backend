// internal/domain/business.go
package domain

import "time"

type Business struct {
	ID        uint
	AdminID   uint
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time

	// Relational fields (optional but useful)
	Admin *User
}
