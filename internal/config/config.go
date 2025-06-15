// internal/config/config.go
package config

import (
	"github.com/spf13/viper"
	"strings"
	"time"
)

// Config holds all configuration for the application.
// Values are read by viper from a config file or environment variables.
type Config struct {
	ServerPort       string        `mapstructure:"SERVER_PORT"`
	LogLevel         string        `mapstructure:"LOG_LEVEL"`
	LogPretty        bool          `mapstructure:"LOG_PRETTY"`
	DatabaseURL      string        `mapstructure:"DB_URL"`
	JWTSecret        string        `mapstructure:"JWT_SECRET"`
	JWTExpiration    time.Duration `mapstructure:"JWT_EXPIRATION_HOURS"`
	KoraPayAPIKey    string        `mapstructure:"KORAPAY_API_KEY"`
	KoraPayPublicKey string        `mapstructure:"KORAPAY_PUBLIC_KEY"`
	KoraPayBaseURL   string        `mapstructure:"KORAPAY_BASE_URL"`
}

// Load loads configuration from the environment.
func Load() (*Config, error) {
	// Set default values
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_PRETTY", false)
	viper.SetDefault("JWT_EXPIRATION_HOURS", 72)
	viper.SetDefault("KORAPAY_BASE_URL", "https://api.korapay.com/v1")

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
	config.JWTExpiration = time.Duration(viper.GetInt("JWT_EXPIRATION_HOURS")) * time.Hour

	return &config, nil
}
