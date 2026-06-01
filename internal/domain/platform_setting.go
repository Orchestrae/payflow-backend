package domain

// PlatformSetting stores encrypted configuration values (API keys, SMTP credentials, etc.).
// Values are AES-256-GCM encrypted at rest.
type PlatformSetting struct {
	Model
	Key            string `gorm:"uniqueIndex;size:100" json:"key"`
	EncryptedValue string `gorm:"type:text" json:"-"`           // Never exposed in API
	Description    string `gorm:"size:500" json:"description"`
	Category       string `gorm:"size:50;index" json:"category"` // e.g., "paystack", "smtp", "korapay"
	IsSet          bool   `gorm:"default:false" json:"is_set"`   // Whether a value has been configured
}

// Common platform setting keys.
const (
	SettingPaystackSecretKey  = "paystack_secret_key"
	SettingPaystackPublicKey  = "paystack_public_key"
	SettingSMTPHost           = "smtp_host"
	SettingSMTPPort           = "smtp_port"
	SettingSMTPUser           = "smtp_user"
	SettingSMTPPassword       = "smtp_password"
	SettingSMTPFrom           = "smtp_from"
	SettingKorapayAPIKey      = "korapay_api_key"
	SettingKorapayEncKey      = "korapay_encryption_key"
)
