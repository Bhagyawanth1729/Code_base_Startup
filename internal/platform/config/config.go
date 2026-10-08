package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv           string        `mapstructure:"APP_ENV"`
	HTTPPort         string        `mapstructure:"HTTP_PORT"`
	DatabaseURL      string        `mapstructure:"DATABASE_URL"`
	RedisURL         string        `mapstructure:"REDIS_URL"`
	JWTSecret        string        `mapstructure:"JWT_SECRET"`
	AccessTokenTTL   time.Duration `mapstructure:"ACCESS_TOKEN_TTL"`
	RefreshTokenTTL  time.Duration `mapstructure:"REFRESH_TOKEN_TTL"`
	LogLevel         string        `mapstructure:"LOG_LEVEL"`
	RateLimitEnabled bool          `mapstructure:"RATE_LIMIT_ENABLED"`
	RateLimitRPS     int           `mapstructure:"RATE_LIMIT_RPS"`
	CORSAllowedOrigins []string    `mapstructure:"CORS_ALLOWED_ORIGINS"`
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./deployments/docker")
	viper.AutomaticEnv()

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults() {
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("HTTP_PORT", "8080")
	viper.SetDefault("DATABASE_URL", "postgres://forgestock:forgestock@localhost:5432/forgestock?sslmode=disable")
	viper.SetDefault("REDIS_URL", "redis://localhost:6379/0")
	viper.SetDefault("JWT_SECRET", "change-me-in-production")
	viper.SetDefault("ACCESS_TOKEN_TTL", "15m")
	viper.SetDefault("REFRESH_TOKEN_TTL", "168h")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("RATE_LIMIT_ENABLED", true)
	viper.SetDefault("RATE_LIMIT_RPS", 100)
	viper.SetDefault("CORS_ALLOWED_ORIGINS", []string{"*"})
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}