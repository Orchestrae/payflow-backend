package domain

// Notification represents an in-app notification for a user.
type Notification struct {
	Model
	UserID     uint   `gorm:"index" json:"user_id"`
	BusinessID uint   `gorm:"index" json:"business_id"`
	Title      string `gorm:"size:255" json:"title"`
	Message    string `gorm:"size:1000" json:"message"`
	Type       string `gorm:"size:50" json:"type"` // payroll, transfer, system, invite
	IsRead     bool   `gorm:"default:false" json:"is_read"`
	LinkURL    string `gorm:"size:500" json:"link_url,omitempty"`
}
