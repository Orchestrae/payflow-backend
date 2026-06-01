package postgres

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/repository"

	"gorm.io/gorm"
)

type orgProviderSettingRepository struct {
	db *gorm.DB
}

func NewOrgProviderSettingRepository(db *gorm.DB) repository.OrgProviderSettingRepository {
	return &orgProviderSettingRepository{db: db}
}

func (r *orgProviderSettingRepository) Upsert(ctx context.Context, setting *domain.OrgProviderSetting) error {
	return r.db.WithContext(ctx).
		Where("business_id = ? AND provider = ? AND setting_key = ?", setting.BusinessID, setting.Provider, setting.SettingKey).
		Assign(domain.OrgProviderSetting{
			EncryptedValue: setting.EncryptedValue,
			IsActive:       setting.IsActive,
		}).
		FirstOrCreate(setting).Error
}

func (r *orgProviderSettingRepository) FindByBusinessAndProvider(ctx context.Context, businessID uint, provider string) ([]*domain.OrgProviderSetting, error) {
	var settings []*domain.OrgProviderSetting
	err := r.db.WithContext(ctx).
		Where("business_id = ? AND provider = ? AND is_active = true", businessID, provider).
		Find(&settings).Error
	return settings, err
}

func (r *orgProviderSettingRepository) FindByBusinessID(ctx context.Context, businessID uint) ([]*domain.OrgProviderSetting, error) {
	var settings []*domain.OrgProviderSetting
	err := r.db.WithContext(ctx).
		Where("business_id = ?", businessID).
		Find(&settings).Error
	return settings, err
}

func (r *orgProviderSettingRepository) Delete(ctx context.Context, businessID uint, provider, key string) error {
	return r.db.WithContext(ctx).
		Where("business_id = ? AND provider = ? AND setting_key = ?", businessID, provider, key).
		Delete(&domain.OrgProviderSetting{}).Error
}
