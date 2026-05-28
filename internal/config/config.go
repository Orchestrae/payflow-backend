// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
// Values are read by viper from a config file or environment variables.
type Config struct {
	ServerPort            string        `mapstructure:"SERVER_PORT"`
	LogLevel              string        `mapstructure:"LOG_LEVEL"`
	LogPretty             bool          `mapstructure:"LOG_PRETTY"`
	DatabaseURL           string        `mapstructure:"DB_URL"`
	JWTSecret             string        `mapstructure:"JWT_SECRET"`
	JWTExpiration         int           `mapstructure:"JWT_EXPIRATION_HOURS"`
	JWTExpirationDuration time.Duration `mapstructure:"-"`
	KoraPayAPIKey         string        `mapstructure:"KORAPAY_API_KEY"`
	KoraPayPublicKey      string        `mapstructure:"KORAPAY_PUBLIC_KEY"`
	KoraPayBaseURL        string        `mapstructure:"KORAPAY_BASE_URL"`

	// VFD Bank Configuration (New)
	VFDConsumerKey    string `mapstructure:"VFD_CONSUMER_KEY"`
	VFDConsumerSecret string `mapstructure:"VFD_CONSUMER_SECRET"`
	VFDBaseURL        string `mapstructure:"VFD_BASE_URL"`

	// Transfer Provider Configuration
	TransferDefaultProvider       string `mapstructure:"TRANSFER_DEFAULT_PROVIDER"`
	TransferProviderFallbackOrder string `mapstructure:"TRANSFER_PROVIDER_FALLBACK_ORDER"`

	// Transfer Limits (in minor currency units, e.g., kobo for NGN)
	TransferMinAmount int64 `mapstructure:"TRANSFER_MIN_AMOUNT"`
	TransferMaxAmount int64 `mapstructure:"TRANSFER_MAX_AMOUNT"`

	// SMTP Configuration
	SMTPHost     string `mapstructure:"SMTP_HOST"`
	SMTPPort     string `mapstructure:"SMTP_PORT"`
	SMTPUser     string `mapstructure:"SMTP_USER"`
	SMTPPassword string `mapstructure:"SMTP_PASSWORD"`
	SMTPFrom     string `mapstructure:"SMTP_FROM"`
	AppURL       string `mapstructure:"APP_URL"`

	// Redis Configuration
	RedisURL string `mapstructure:"REDIS_URL"`

	// CORS Configuration
	CORSAllowedOrigins string `mapstructure:"CORS_ALLOWED_ORIGINS"`

	// Paystack Configuration
	PaystackSecretKey string `mapstructure:"PAYSTACK_SECRET_KEY"`
	PaystackBaseURL   string `mapstructure:"PAYSTACK_BASE_URL"`

	// Provider Toggle
	EnabledProviders string `mapstructure:"ENABLED_PROVIDERS"`

	// SMS (Termii)
	TermiiAPIKey   string `mapstructure:"TERMII_API_KEY"`
	TermiiSenderID string `mapstructure:"TERMII_SENDER_ID"`

	// VFD Webhook Verification
	VFDWebhookSecret string `mapstructure:"VFD_WEBHOOK_SECRET"`

	// Database Migration Configuration
	EnableAutoMigration bool `mapstructure:"ENABLE_AUTO_MIGRATION"` // Set to false in production - use traditional migrations only
}

// Load loads configuration from the environment.
func Load() (*Config, error) {
	// Set default values
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_PRETTY", false)
	viper.SetDefault("JWT_EXPIRATION_HOURS", 72)
	viper.SetDefault("KORAPAY_BASE_URL", "https://api.korapay.com")
	// Setting the default VFD base URL for the development environment
	viper.SetDefault("VFD_BASE_URL", "https://api-devapps.vfdbank.systems")
	// Default transfer provider configuration - Korapay is primary, VFD is fallback
	viper.SetDefault("TRANSFER_DEFAULT_PROVIDER", "korapay")
	viper.SetDefault("TRANSFER_PROVIDER_FALLBACK_ORDER", "vfd")

	// Transfer limits - Korapay requires minimum NGN 1000 (100000 kobo) and max NGN 10,000,000
	viper.SetDefault("TRANSFER_MIN_AMOUNT", 1000)    // NGN 1000 minimum
	viper.SetDefault("TRANSFER_MAX_AMOUNT", 10000000) // NGN 10,000,000 maximum
	// SMTP defaults (MailHog for local dev, Brevo/SendGrid for production)
	viper.SetDefault("SMTP_HOST", "localhost")
	viper.SetDefault("SMTP_PORT", "1025")
	viper.SetDefault("SMTP_FROM", "no-reply@payflow.com")
	viper.SetDefault("APP_URL", "http://localhost:3000")
	// Paystack defaults
	viper.SetDefault("PAYSTACK_BASE_URL", "https://api.paystack.co")
	// Provider toggle: comma-separated list of enabled providers (backward compatible)
	viper.SetDefault("ENABLED_PROVIDERS", "korapay,vfd")
	// CORS: production origin by default; override with comma-separated list in dev
	viper.SetDefault("CORS_ALLOWED_ORIGINS", "https://payflowio.vercel.app,https://payflowio.netlify.app")
	// Auto-migration: Disabled by default for safety. Enable only for local development.
	// In production, use traditional migrations only (golang-migrate).
	viper.SetDefault("ENABLE_AUTO_MIGRATION", false)
	// Tell viper to look for an .env file
	viper.AddConfigPath(".")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	// Automatically read environment variables
	viper.AutomaticEnv()
	// This allows `DB_URL` to be read as `DB.URL` if needed, etc.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read the config file (if it exists)
	// This is useful for local development but env vars will override
	_ = viper.ReadInConfig()

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Viper doesn't handle time.Duration from env well, so we do it manually.
	config.JWTExpirationDuration = time.Duration(config.JWTExpiration) * time.Hour

	// Ensure database URL is read from env - viper can miss it in some cases
	// Check common PaaS env vars in order: Railway, Heroku, Render, etc.
	for _, key := range []string{"DB_URL", "DATABASE_URL", "DATABASE_PRIVATE_URL", "DATABASE_PUBLIC_URL"} {
		if config.DatabaseURL == "" {
			config.DatabaseURL = os.Getenv(key)
		}
	}

	// Fallback: build DSN from individual Postgres vars (Railway, etc.)
	if config.DatabaseURL == "" {
		host := os.Getenv("PGHOST")
		port := os.Getenv("PGPORT")
		user := os.Getenv("PGUSER")
		pass := os.Getenv("PGPASSWORD")
		dbname := os.Getenv("PGDATABASE")
		sslmode := os.Getenv("PGSSLMODE")
		if sslmode == "" {
			sslmode = "require"
		}
		if host != "" && user != "" && dbname != "" {
			if port == "" {
				port = "5432"
			}
			config.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
				user, pass, host, port, dbname, sslmode)
		}
	}

	// Railway, Heroku, Render inject PORT at runtime - use it when set
	if port := os.Getenv("PORT"); port != "" {
		config.ServerPort = port
	}

	// Temporary: auto-enable migrations on Railway/PaaS (no manual migrate needed)
	if config.DatabaseURL != "" &&
		(strings.Contains(config.DatabaseURL, "railway") || strings.Contains(config.DatabaseURL, "rlwy")) {
		config.EnableAutoMigration = true
	}

	return &config, nil
}
