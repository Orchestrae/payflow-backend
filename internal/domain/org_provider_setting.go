package domain

// OrgProviderSetting stores per-business provider API key overrides.
// Values are AES-256-GCM encrypted at rest.
type OrgProviderSetting struct {
	Model
	BusinessID     uint   `gorm:"index;uniqueIndex:idx_org_provider_key" json:"business_id"`
	Provider       string `gorm:"size:50;uniqueIndex:idx_org_provider_key" json:"provider"`
	SettingKey     string `gorm:"size:100;uniqueIndex:idx_org_provider_key" json:"setting_key"`
	EncryptedValue string `gorm:"type:text" json:"-"`
	IsActive       bool   `gorm:"default:true" json:"is_active"`
}

// Common org-level provider setting keys.
const (
	OrgSettingPaystackSecretKey = "paystack_secret_key"
	OrgSettingPaystackPublicKey = "paystack_public_key"
	OrgSettingKorapayAPIKey     = "korapay_api_key"
	OrgSettingKorapayEncKey     = "korapay_encryption_key"
)
