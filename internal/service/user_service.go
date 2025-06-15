package service

import (
	"context"
	"payflow/internal/domain"
)

type UserService interface {
	GetUserByID(ctx context.Context, userID uint) (*domain.User, error)
	ListUsersByBusinessID(ctx context.Context, businessID uint) ([]domain.User, error)
	UpdateUserRole(ctx context.Context, userID, businessID uint, role domain.UserRole) error
	InviteUser(ctx context.Context, email string, businessID uint, role domain.UserRole) error
}
