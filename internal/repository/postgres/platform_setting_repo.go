package postgres

import (
	"context"
	"payflow/internal/domain"
	"payflow/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type platformSettingRepository struct {
	db *gorm.DB
}

func NewPlatformSettingRepository(db *gorm.DB) repository.PlatformSettingRepository {
	return &platformSettingRepository{db: db}
}

func (r *platformSettingRepository) Upsert(ctx context.Context, setting *domain.PlatformSetting) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"encrypted_value", "description", "category", "is_set", "updated_at"}),
		}).
		Create(setting).Error
}

func (r *platformSettingRepository) FindByKey(ctx context.Context, key string) (*domain.PlatformSetting, error) {
	var setting domain.PlatformSetting
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&setting).Error; err != nil {
		return nil, DBErrToDomainErr(err)
	}
	return &setting, nil
}

func (r *platformSettingRepository) FindByCategory(ctx context.Context, category string) ([]*domain.PlatformSetting, error) {
	var settings []*domain.PlatformSetting
	if err := r.db.WithContext(ctx).Where("category = ?", category).Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *platformSettingRepository) FindAll(ctx context.Context) ([]*domain.PlatformSetting, error) {
	var settings []*domain.PlatformSetting
	if err := r.db.WithContext(ctx).Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *platformSettingRepository) Delete(ctx context.Context, key string) error {
	return r.db.WithContext(ctx).Where("key = ?", key).Delete(&domain.PlatformSetting{}).Error
}
