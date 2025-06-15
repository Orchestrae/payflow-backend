package postgres

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/repository"

	"gorm.io/gorm"
)

type deductionRuleRepository struct {
	db *gorm.DB
}

func NewDeductionRuleRepository(db *gorm.DB) repository.DeductionRuleRepository {
	return &deductionRuleRepository{db: db}
}

func (r *deductionRuleRepository) Create(ctx context.Context, rule *domain.DeductionRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *deductionRuleRepository) Update(ctx context.Context, rule *domain.DeductionRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *deductionRuleRepository) FindByID(ctx context.Context, id uint) (*domain.DeductionRule, error) {
	var rule domain.DeductionRule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *deductionRuleRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&domain.DeductionRule{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *deductionRuleRepository) FindAllByBusinessID(ctx context.Context, businessID uint) ([]domain.DeductionRule, error) {
	var rules []domain.DeductionRule
	err := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *deductionRuleRepository) WithTx(tx *gorm.DB) repository.DeductionRuleRepository {
	return &deductionRuleRepository{db: tx}
}
