// internal/config/config.go
package config

import (
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

	return &config, nil
}
