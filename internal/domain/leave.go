package domain

import "time"

// LeaveType defines a type of leave available in a business.
type LeaveType struct {
	Model
	BusinessID       uint   `gorm:"index" json:"business_id"`
	Name             string `gorm:"size:100" json:"name"` // Annual, Sick, Compassionate
	DefaultDays      int    `json:"default_days"`
	RequiresApproval bool   `gorm:"default:true" json:"requires_approval"`
}

// LeaveRequest tracks an employee's leave request.
type LeaveRequest struct {
	Model
	EmployeeID   uint      `gorm:"index" json:"employee_id"`
	BusinessID   uint      `gorm:"index" json:"business_id"`
	LeaveTypeID  uint      `json:"leave_type_id"`
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
	Days         int       `json:"days"`
	Reason       string    `gorm:"size:500" json:"reason"`
	Status       string    `gorm:"size:20;default:'pending'" json:"status"` // pending, approved, rejected, cancelled
	ApprovedByID *uint     `json:"approved_by_id,omitempty"`

	Employee  *Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	LeaveType *LeaveType `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
}

// LeaveBalance tracks an employee's leave entitlement and usage per year.
type LeaveBalance struct {
	Model
	EmployeeID  uint `gorm:"uniqueIndex:idx_emp_leave_year" json:"employee_id"`
	LeaveTypeID uint `gorm:"uniqueIndex:idx_emp_leave_year" json:"leave_type_id"`
	Year        int  `gorm:"uniqueIndex:idx_emp_leave_year" json:"year"`
	Entitled    int  `json:"entitled"`
	Used        int  `gorm:"default:0" json:"used"`
	Remaining   int  `json:"remaining"`
}
