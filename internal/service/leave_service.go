package service

import (
	"context"
	"fmt"
	"time"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// LeaveService manages leave types, requests, and balances.
type LeaveService interface {
	// Leave Types
	CreateLeaveType(ctx context.Context, lt *domain.LeaveType) (*domain.LeaveType, error)
	ListLeaveTypes(ctx context.Context, businessID uint) ([]*domain.LeaveType, error)

	// Leave Requests
	SubmitRequest(ctx context.Context, req *domain.LeaveRequest) (*domain.LeaveRequest, error)
	ListRequests(ctx context.Context, businessID uint, page, limit int) ([]*domain.LeaveRequest, int, error)
	ApproveRequest(ctx context.Context, requestID, approverID uint) error
	RejectRequest(ctx context.Context, requestID, approverID uint, reason string) error

	// Leave Balances
	GetBalance(ctx context.Context, employeeID uint, year int) ([]*domain.LeaveBalance, error)
}

type leaveService struct {
	leaveTypeRepo    repository.LeaveTypeRepository
	leaveRequestRepo repository.LeaveRequestRepository
	leaveBalanceRepo repository.LeaveBalanceRepository
}

// NewLeaveService creates a new leave service.
func NewLeaveService(
	typeRepo repository.LeaveTypeRepository,
	requestRepo repository.LeaveRequestRepository,
	balanceRepo repository.LeaveBalanceRepository,
) LeaveService {
	return &leaveService{
		leaveTypeRepo:    typeRepo,
		leaveRequestRepo: requestRepo,
		leaveBalanceRepo: balanceRepo,
	}
}

func (s *leaveService) CreateLeaveType(ctx context.Context, lt *domain.LeaveType) (*domain.LeaveType, error) {
	if err := s.leaveTypeRepo.Create(ctx, lt); err != nil {
		return nil, fmt.Errorf("failed to create leave type: %w", err)
	}
	return lt, nil
}

func (s *leaveService) ListLeaveTypes(ctx context.Context, businessID uint) ([]*domain.LeaveType, error) {
	return s.leaveTypeRepo.FindByBusinessID(ctx, businessID)
}

func (s *leaveService) SubmitRequest(ctx context.Context, req *domain.LeaveRequest) (*domain.LeaveRequest, error) {
	// Check balance
	year := req.StartDate.Year()
	balance, err := s.leaveBalanceRepo.FindByEmployeeAndType(ctx, req.EmployeeID, req.LeaveTypeID, year)
	if err != nil {
		// Create balance if doesn't exist
		leaveType, ltErr := s.leaveTypeRepo.FindByID(ctx, req.LeaveTypeID)
		if ltErr != nil {
			return nil, domain.ErrNotFound
		}
		balance = &domain.LeaveBalance{
			EmployeeID:  req.EmployeeID,
			LeaveTypeID: req.LeaveTypeID,
			Year:        year,
			Entitled:    leaveType.DefaultDays,
			Used:        0,
			Remaining:   leaveType.DefaultDays,
		}
		s.leaveBalanceRepo.Create(ctx, balance)
	}

	if balance.Remaining < req.Days {
		return nil, fmt.Errorf("%w: insufficient leave balance (remaining: %d, requested: %d)", domain.ErrValidationFailed, balance.Remaining, req.Days)
	}

	req.Status = "pending"
	if err := s.leaveRequestRepo.Create(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to submit leave request: %w", err)
	}
	return req, nil
}

func (s *leaveService) ListRequests(ctx context.Context, businessID uint, page, limit int) ([]*domain.LeaveRequest, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	return s.leaveRequestRepo.FindByBusinessID(ctx, businessID, page, limit)
}

func (s *leaveService) ApproveRequest(ctx context.Context, requestID, approverID uint) error {
	req, err := s.leaveRequestRepo.FindByID(ctx, requestID)
	if err != nil {
		return domain.ErrNotFound
	}
	if req.Status != "pending" {
		return fmt.Errorf("%w: can only approve pending requests", domain.ErrValidationFailed)
	}

	req.Status = "approved"
	req.ApprovedByID = &approverID
	if err := s.leaveRequestRepo.Update(ctx, req); err != nil {
		return err
	}

	// Update balance
	year := req.StartDate.Year()
	balance, err := s.leaveBalanceRepo.FindByEmployeeAndType(ctx, req.EmployeeID, req.LeaveTypeID, year)
	if err == nil {
		balance.Used += req.Days
		balance.Remaining = balance.Entitled - balance.Used
		s.leaveBalanceRepo.Update(ctx, balance)
	}

	return nil
}

func (s *leaveService) RejectRequest(ctx context.Context, requestID, approverID uint, reason string) error {
	req, err := s.leaveRequestRepo.FindByID(ctx, requestID)
	if err != nil {
		return domain.ErrNotFound
	}
	if req.Status != "pending" {
		return fmt.Errorf("%w: can only reject pending requests", domain.ErrValidationFailed)
	}

	req.Status = "rejected"
	req.ApprovedByID = &approverID
	req.Reason = reason
	return s.leaveRequestRepo.Update(ctx, req)
}

func (s *leaveService) GetBalance(ctx context.Context, employeeID uint, year int) ([]*domain.LeaveBalance, error) {
	if year == 0 {
		year = time.Now().Year()
	}
	return s.leaveBalanceRepo.FindByEmployee(ctx, employeeID, year)
}
