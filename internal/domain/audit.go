package domain

// AuditLog tracks who did what, when, and to which resource.
type AuditLog struct {
	Model
	UserID       uint   `gorm:"index" json:"user_id"`
	BusinessID   uint   `gorm:"index" json:"business_id"`
	Action       string `gorm:"size:50" json:"action"`                 // created, updated, deleted, approved, rejected, processed, invited
	ResourceType string `gorm:"size:50;index" json:"resource_type"`    // employee, cadre, payroll_run, transfer, business, user
	ResourceID   uint   `json:"resource_id"`
	Description  string `gorm:"size:500" json:"description"`
	IPAddress    string `gorm:"size:45" json:"ip_address,omitempty"`
}
