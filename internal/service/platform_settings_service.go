package service

import (
	"context"
	"fmt"

	"payflow/internal/domain"
	"payflow/internal/repository"
	"payflow/pkg/utils"
)

// PlatformSettingsService manages encrypted platform configuration.
type PlatformSettingsService interface {
	SetSetting(ctx context.Context, key, value, description, category string) error
	GetSetting(ctx context.Context, key string) (string, error)
	GetSettingsByCategory(ctx context.Context, category string) ([]SettingSummary, error)
	ListSettings(ctx context.Context) ([]SettingSummary, error)
	DeleteSetting(ctx context.Context, key string) error
}

// SettingSummary is a safe view of a setting (no decrypted values).
type SettingSummary struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Category    string `json:"category"`
	IsSet       bool   `json:"is_set"`
	MaskedValue string `json:"masked_value,omitempty"` // e.g., "sk_test_****xyz"
}

type platformSettingsService struct {
	repo          repository.PlatformSettingRepository
	encryptionKey string // 32-byte AES key
}

// NewPlatformSettingsService creates a new platform settings service.
func NewPlatformSettingsService(repo repository.PlatformSettingRepository, encryptionKey string) PlatformSettingsService {
	return &platformSettingsService{repo: repo, encryptionKey: encryptionKey}
}

func (s *platformSettingsService) SetSetting(ctx context.Context, key, value, description, category string) error {
	encrypted, err := utils.Encrypt(value, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt setting: %w", err)
	}

	setting := &domain.PlatformSetting{
		Key:            key,
		EncryptedValue: encrypted,
		Description:    description,
		Category:       category,
		IsSet:          true,
	}

	return s.repo.Upsert(ctx, setting)
}

func (s *platformSettingsService) GetSetting(ctx context.Context, key string) (string, error) {
	setting, err := s.repo.FindByKey(ctx, key)
	if err != nil {
		return "", err
	}
	if !setting.IsSet {
		return "", fmt.Errorf("setting %q is not configured", key)
	}

	decrypted, err := utils.Decrypt(setting.EncryptedValue, s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt setting: %w", err)
	}

	return decrypted, nil
}

func (s *platformSettingsService) GetSettingsByCategory(ctx context.Context, category string) ([]SettingSummary, error) {
	settings, err := s.repo.FindByCategory(ctx, category)
	if err != nil {
		return nil, err
	}
	return s.toSummaries(settings), nil
}

func (s *platformSettingsService) ListSettings(ctx context.Context) ([]SettingSummary, error) {
	settings, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	return s.toSummaries(settings), nil
}

func (s *platformSettingsService) DeleteSetting(ctx context.Context, key string) error {
	return s.repo.Delete(ctx, key)
}

func (s *platformSettingsService) toSummaries(settings []*domain.PlatformSetting) []SettingSummary {
	summaries := make([]SettingSummary, len(settings))
	for i, setting := range settings {
		summaries[i] = SettingSummary{
			Key:         setting.Key,
			Description: setting.Description,
			Category:    setting.Category,
			IsSet:       setting.IsSet,
		}
		// Show masked value if set
		if setting.IsSet {
			if decrypted, err := utils.Decrypt(setting.EncryptedValue, s.encryptionKey); err == nil && len(decrypted) > 6 {
				summaries[i].MaskedValue = decrypted[:4] + "****" + decrypted[len(decrypted)-3:]
			} else {
				summaries[i].MaskedValue = "****"
			}
		}
	}
	return summaries
}
