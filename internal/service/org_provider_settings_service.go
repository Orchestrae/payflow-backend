package service

import (
	"context"
	"fmt"

	"payflow/internal/domain"
	"payflow/internal/repository"
	"payflow/pkg/utils"
)

// OrgProviderSettingsService manages per-business provider API key overrides.
type OrgProviderSettingsService interface {
	SetKey(ctx context.Context, businessID uint, provider, key, value string) error
	GetKey(ctx context.Context, businessID uint, provider, key string) (string, error)
	GetProviderKeys(ctx context.Context, businessID uint, provider string) (map[string]string, error)
	ListSettings(ctx context.Context, businessID uint) ([]OrgProviderSettingSummary, error)
	DeleteKey(ctx context.Context, businessID uint, provider, key string) error
}

// OrgProviderSettingSummary is a safe view (no decrypted values).
type OrgProviderSettingSummary struct {
	Provider    string `json:"provider"`
	SettingKey  string `json:"setting_key"`
	IsActive    bool   `json:"is_active"`
	MaskedValue string `json:"masked_value,omitempty"`
}

type orgProviderSettingsService struct {
	repo          repository.OrgProviderSettingRepository
	encryptionKey string
}

func NewOrgProviderSettingsService(repo repository.OrgProviderSettingRepository, encryptionKey string) OrgProviderSettingsService {
	return &orgProviderSettingsService{repo: repo, encryptionKey: encryptionKey}
}

func (s *orgProviderSettingsService) SetKey(ctx context.Context, businessID uint, provider, key, value string) error {
	encrypted, err := utils.Encrypt(value, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt key: %w", err)
	}

	setting := &domain.OrgProviderSetting{
		BusinessID:     businessID,
		Provider:       provider,
		SettingKey:     key,
		EncryptedValue: encrypted,
		IsActive:       true,
	}

	return s.repo.Upsert(ctx, setting)
}

func (s *orgProviderSettingsService) GetKey(ctx context.Context, businessID uint, provider, key string) (string, error) {
	settings, err := s.repo.FindByBusinessAndProvider(ctx, businessID, provider)
	if err != nil {
		return "", err
	}

	for _, setting := range settings {
		if setting.SettingKey == key {
			return utils.Decrypt(setting.EncryptedValue, s.encryptionKey)
		}
	}

	return "", fmt.Errorf("key %q not found for provider %q", key, provider)
}

func (s *orgProviderSettingsService) GetProviderKeys(ctx context.Context, businessID uint, provider string) (map[string]string, error) {
	settings, err := s.repo.FindByBusinessAndProvider(ctx, businessID, provider)
	if err != nil {
		return nil, err
	}

	if len(settings) == 0 {
		return nil, nil
	}

	keys := make(map[string]string, len(settings))
	for _, setting := range settings {
		decrypted, err := utils.Decrypt(setting.EncryptedValue, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt key %q: %w", setting.SettingKey, err)
		}
		keys[setting.SettingKey] = decrypted
	}

	return keys, nil
}

func (s *orgProviderSettingsService) ListSettings(ctx context.Context, businessID uint) ([]OrgProviderSettingSummary, error) {
	settings, err := s.repo.FindByBusinessID(ctx, businessID)
	if err != nil {
		return nil, err
	}

	summaries := make([]OrgProviderSettingSummary, len(settings))
	for i, setting := range settings {
		summaries[i] = OrgProviderSettingSummary{
			Provider:   setting.Provider,
			SettingKey: setting.SettingKey,
			IsActive:   setting.IsActive,
		}
		if decrypted, err := utils.Decrypt(setting.EncryptedValue, s.encryptionKey); err == nil && len(decrypted) > 6 {
			summaries[i].MaskedValue = decrypted[:4] + "****" + decrypted[len(decrypted)-3:]
		} else {
			summaries[i].MaskedValue = "****"
		}
	}

	return summaries, nil
}

func (s *orgProviderSettingsService) DeleteKey(ctx context.Context, businessID uint, provider, key string) error {
	return s.repo.Delete(ctx, businessID, provider, key)
}
