package postgres

import (
	"context"

	"gorm.io/gorm"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// --- LeaveType Repository ---

type leaveTypeRepository struct{ db *gorm.DB }

func NewLeaveTypeRepository(db *gorm.DB) repository.LeaveTypeRepository {
	return &leaveTypeRepository{db: db}
}

func (r *leaveTypeRepository) Create(ctx context.Context, lt *domain.LeaveType) error {
	return r.db.WithContext(ctx).Create(lt).Error
}

func (r *leaveTypeRepository) FindByID(ctx context.Context, id uint) (*domain.LeaveType, error) {
	var lt domain.LeaveType
	if err := r.db.WithContext(ctx).First(&lt, id).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return &lt, nil
}

func (r *leaveTypeRepository) FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.LeaveType, error) {
	var types []*domain.LeaveType
	err := r.db.WithContext(ctx).Where("business_id = ?", businessID).Find(&types).Error
	return types, err
}

// --- LeaveRequest Repository ---

type leaveRequestRepository struct{ db *gorm.DB }

func NewLeaveRequestRepository(db *gorm.DB) repository.LeaveRequestRepository {
	return &leaveRequestRepository{db: db}
}

func (r *leaveRequestRepository) Create(ctx context.Context, req *domain.LeaveRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *leaveRequestRepository) FindByID(ctx context.Context, id uint) (*domain.LeaveRequest, error) {
	var req domain.LeaveRequest
	if err := r.db.WithContext(ctx).Preload("Employee").Preload("LeaveType").First(&req, id).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return &req, nil
}

func (r *leaveRequestRepository) FindByBusinessID(ctx context.Context, businessID uint, page, limit int) ([]*domain.LeaveRequest, int, error) {
	var requests []*domain.LeaveRequest
	var total int64

	r.db.WithContext(ctx).Model(&domain.LeaveRequest{}).Where("business_id = ?", businessID).Count(&total)

	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).Preload("Employee").Preload("LeaveType").
		Where("business_id = ?", businessID).
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&requests).Error

	return requests, int(total), err
}

func (r *leaveRequestRepository) Update(ctx context.Context, req *domain.LeaveRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

// --- LeaveBalance Repository ---

type leaveBalanceRepository struct{ db *gorm.DB }

func NewLeaveBalanceRepository(db *gorm.DB) repository.LeaveBalanceRepository {
	return &leaveBalanceRepository{db: db}
}

func (r *leaveBalanceRepository) Create(ctx context.Context, balance *domain.LeaveBalance) error {
	return r.db.WithContext(ctx).Create(balance).Error
}

func (r *leaveBalanceRepository) FindByEmployeeAndType(ctx context.Context, employeeID, leaveTypeID uint, year int) (*domain.LeaveBalance, error) {
	var balance domain.LeaveBalance
	if err := r.db.WithContext(ctx).Where("employee_id = ? AND leave_type_id = ? AND year = ?", employeeID, leaveTypeID, year).First(&balance).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return &balance, nil
}

func (r *leaveBalanceRepository) FindByEmployee(ctx context.Context, employeeID uint, year int) ([]*domain.LeaveBalance, error) {
	var balances []*domain.LeaveBalance
	err := r.db.WithContext(ctx).Where("employee_id = ? AND year = ?", employeeID, year).Find(&balances).Error
	return balances, err
}

func (r *leaveBalanceRepository) Update(ctx context.Context, balance *domain.LeaveBalance) error {
	return r.db.WithContext(ctx).Save(balance).Error
}
