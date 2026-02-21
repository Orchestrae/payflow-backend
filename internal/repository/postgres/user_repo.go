// internal/repository/postgres/user_repo.go
package postgres

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/repository"

	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new GORM implementation of the UserRepository.
func NewUserRepository(db *gorm.DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) WithTx(tx repository.Transactioner) repository.UserRepository {
	if txr, ok := tx.(*transactioner); ok {
		return &userRepository{db: txr.db}
	}
	return r
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	dbUser := UserFromDomain(user)
	if err := r.db.WithContext(ctx).Create(dbUser).Error; err != nil {
		return DBErrToDomainErr(err)
	}
	*user = *dbUser.ToDomain()
	return nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	dbUser := UserFromDomain(user)
	result := r.db.WithContext(ctx).Model(&dbUser).Updates(dbUser)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (r *userRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	var dbUser User
	err := r.db.WithContext(ctx).First(&dbUser, id).Error
	if err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return dbUser.ToDomain(), nil
}

func (r *userRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var dbUser User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&dbUser).Error
	if err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return dbUser.ToDomain(), nil
}

func (r *userRepository) FindApproversByBusinessID(ctx context.Context, businessID uint) ([]domain.User, error) {
	var dbUsers []User
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND role = ?", businessID, domain.RoleApprover).
		Find(&dbUsers).Error
	if err != nil {
		return nil, err
	}
	domainUsers := make([]domain.User, len(dbUsers))
	for i, u := range dbUsers {
		domainUsers[i] = *u.ToDomain()
	}
	return domainUsers, nil
}

func (r *userRepository) FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.User, error) {
	var dbUsers []User
	err := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Find(&dbUsers).Error
	if err != nil {
		return nil, err
	}
	domainUsers := make([]*domain.User, len(dbUsers))
	for i, u := range dbUsers {
		domainUsers[i] = u.ToDomain()
	}
	return domainUsers, nil
}

func (r *userRepository) FindBusinessAdmin(ctx context.Context, businessID uint) (*domain.User, error) {
	var dbUser User
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND role = ?", businessID, domain.RoleAdmin).
		First(&dbUser).Error
	if err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return dbUser.ToDomain(), nil
}

func (r *userRepository) FindOperatorsByBusinessID(ctx context.Context, businessID uint) ([]domain.User, error) {
	var dbUsers []User
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND role = ?", businessID, domain.RoleOperator).
		Find(&dbUsers).Error

	if err != nil {
		return nil, err
	}

	domainUsers := make([]domain.User, len(dbUsers))
	for i, u := range dbUsers {
		domainUsers[i] = *u.ToDomain()
	}
	return domainUsers, nil
}
