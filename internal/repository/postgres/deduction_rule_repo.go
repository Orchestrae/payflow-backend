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

func (r *deductionRuleRepository) FindByID(ctx context.Context, id uint, businessID uint) (*domain.DeductionRule, error) {
	var rule domain.DeductionRule
	err := r.db.WithContext(ctx).
		Where("id = ? AND business_id = ?", id, businessID).
		First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *deductionRuleRepository) Delete(ctx context.Context, id uint, businessID uint) error {
	result := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Delete(&domain.DeductionRule{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *deductionRuleRepository) FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.DeductionRule, error) {
	var rules []domain.DeductionRule
	err := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Find(&rules).Error
	if err != nil {
		return nil, err
	}

	domainRules := make([]*domain.DeductionRule, len(rules))
	for i, rule := range rules {
		r := rule // copy
		domainRules[i] = &r
	}
	return domainRules, nil
}

func (r *deductionRuleRepository) WithTx(tx repository.Transactioner) repository.DeductionRuleRepository {
	if txr, ok := tx.(*transactioner); ok {
		return &deductionRuleRepository{db: txr.db}
	}
	return r
}
