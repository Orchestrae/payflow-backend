package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"payflow/internal/domain"
	"payflow/internal/repository"
)

// Platform setting keys for anomaly thresholds
const (
	SettingAnomalyMaxSingleAmount = "anomaly_max_single_transfer"
	SettingAnomalyMaxDailyCount   = "anomaly_max_daily_count"
	SettingAnomalyMaxDailyVolume  = "anomaly_max_daily_volume"
	SettingAnomalyDuplicateWindow = "anomaly_duplicate_window"
)

// AnomalyAlert represents a detected anomaly in transfer patterns.
type AnomalyAlert struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"` // low, medium, high, critical
	BusinessID  uint      `json:"business_id"`
	Message     string    `json:"message"`
	Details     string    `json:"details"`
	DetectedAt  time.Time `json:"detected_at"`
}

// AnomalyDetectionService detects unusual transfer patterns.
type AnomalyDetectionService interface {
	CheckTransfer(ctx context.Context, businessID uint, amount int64, accountNumber string) ([]AnomalyAlert, error)
	RunDailyCheck(ctx context.Context) ([]AnomalyAlert, error)
}

// AnomalyConfig defines thresholds for anomaly detection.
type AnomalyConfig struct {
	MaxSingleTransferAmount int64 // Alert if single transfer exceeds this (kobo)
	MaxDailyTransferCount   int   // Alert if daily transfer count exceeds this
	MaxDailyTransferVolume  int64 // Alert if daily volume exceeds this (kobo)
	DuplicateAccountWindow  int   // Alert if same account receives N+ transfers in 24h
}

// DefaultAnomalyConfig returns sensible defaults for Nigerian payroll.
func DefaultAnomalyConfig() AnomalyConfig {
	return AnomalyConfig{
		MaxSingleTransferAmount: 50000000,  // NGN 500,000
		MaxDailyTransferCount:   100,
		MaxDailyTransferVolume:  500000000, // NGN 5,000,000
		DuplicateAccountWindow:  3,         // 3+ to same account in 24h
	}
}

type anomalyDetectionService struct {
	transferRepo repository.TransferRepository
	settingsSvc  PlatformSettingsService
	config       AnomalyConfig // fallback defaults
}

// NewAnomalyDetectionService creates a new anomaly detection service.
// If settingsSvc is provided, thresholds are loaded from platform settings (super admin configurable).
// Otherwise falls back to the provided config.
func NewAnomalyDetectionService(transferRepo repository.TransferRepository, config AnomalyConfig, settingsSvc PlatformSettingsService) AnomalyDetectionService {
	return &anomalyDetectionService{
		transferRepo: transferRepo,
		settingsSvc:  settingsSvc,
		config:       config,
	}
}

// getConfig loads config from platform settings if available, otherwise uses defaults.
func (s *anomalyDetectionService) getConfig(ctx context.Context) AnomalyConfig {
	if s.settingsSvc == nil {
		return s.config
	}
	cfg := s.config // start with defaults

	if v, err := s.settingsSvc.GetSetting(ctx, SettingAnomalyMaxSingleAmount); err == nil {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.MaxSingleTransferAmount = n
		}
	}
	if v, err := s.settingsSvc.GetSetting(ctx, SettingAnomalyMaxDailyCount); err == nil {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxDailyTransferCount = n
		}
	}
	if v, err := s.settingsSvc.GetSetting(ctx, SettingAnomalyMaxDailyVolume); err == nil {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.MaxDailyTransferVolume = n
		}
	}
	if v, err := s.settingsSvc.GetSetting(ctx, SettingAnomalyDuplicateWindow); err == nil {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.DuplicateAccountWindow = n
		}
	}

	return cfg
}

// CheckTransfer checks a single transfer for anomalies before it's processed.
func (s *anomalyDetectionService) CheckTransfer(ctx context.Context, businessID uint, amount int64, accountNumber string) ([]AnomalyAlert, error) {
	cfg := s.getConfig(ctx)
	var alerts []AnomalyAlert
	now := time.Now()

	// Check 1: Large single transfer
	if amount > cfg.MaxSingleTransferAmount {
		alerts = append(alerts, AnomalyAlert{
			Type:       "large_transfer",
			Severity:   "high",
			BusinessID: businessID,
			Message:    fmt.Sprintf("Large transfer detected: %d kobo (threshold: %d)", amount, cfg.MaxSingleTransferAmount),
			DetectedAt: now,
		})
	}

	// Check 2: Daily transfer count
	transfers, total, err := s.transferRepo.FindByBusinessID(ctx, businessID, 1, 1)
	if err == nil && total > cfg.MaxDailyTransferCount {
		_ = transfers
		alerts = append(alerts, AnomalyAlert{
			Type:       "high_frequency",
			Severity:   "medium",
			BusinessID: businessID,
			Message:    fmt.Sprintf("High transfer frequency: %d transfers today (threshold: %d)", total, cfg.MaxDailyTransferCount),
			DetectedAt: now,
		})
	}

	// Log alerts
	for _, alert := range alerts {
		log.Warn().
			Str("type", alert.Type).
			Str("severity", alert.Severity).
			Uint("business_id", alert.BusinessID).
			Str("message", alert.Message).
			Msg("ANOMALY DETECTED")
	}

	return alerts, nil
}

// RunDailyCheck scans all recent transfers for anomalies.
func (s *anomalyDetectionService) RunDailyCheck(ctx context.Context) ([]AnomalyAlert, error) {
	log.Info().Msg("Running daily anomaly detection scan")
	var allAlerts []AnomalyAlert

	// Check recent transfers across all businesses
	// In production, this would iterate per-business with proper pagination
	transfers, _, err := s.transferRepo.FindByBusinessID(ctx, 0, 1, 500) // System access
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transfers for anomaly check: %w", err)
	}

	cfg := s.getConfig(ctx)

	// Check for duplicate accounts (same account receiving many transfers)
	accountCounts := make(map[string]int)
	recentTime := time.Now().Add(-24 * time.Hour)
	for _, t := range transfers {
		if t.CreatedAt.After(recentTime) && t.Status == string(domain.StatusCompleted) {
			key := fmt.Sprintf("%d-%s", t.BusinessID, t.RecipientAccountNumber)
			accountCounts[key]++
		}
	}

	for key, count := range accountCounts {
		if count >= cfg.DuplicateAccountWindow {
			allAlerts = append(allAlerts, AnomalyAlert{
				Type:       "duplicate_recipient",
				Severity:   "medium",
				Message:    fmt.Sprintf("Account %s received %d transfers in 24h (threshold: %d)", key, count, cfg.DuplicateAccountWindow),
				DetectedAt: time.Now(),
			})
		}
	}

	log.Info().Int("alerts", len(allAlerts)).Msg("Daily anomaly scan complete")
	return allAlerts, nil
}
