// Package config loads runtime configuration from environment variables.
// 12-Factor: all config via env. Locally a .env file is loaded first (godotenv).
package config

import (
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all runtime configuration. Populated from environment variables.
type Config struct {
	Env             string        `env:"ENV" envDefault:"development"`
	Port            string        `env:"PORT" envDefault:"8080"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	DatabaseURL     string        `env:"DATABASE_URL,required"`
	JWTSecret       string        `env:"JWT_SECRET,required"`
	AccessTokenTTL  time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"720h"`
	BcryptCost      int           `env:"BCRYPT_COST" envDefault:"12"`
	CORSOrigins     []string      `env:"CORS_ORIGINS" envSeparator:","`

	// Seed-Admin (nur von cmd/seed genutzt; im API-Prozess leer/ungenutzt).
	SeedAdminEmail    string `env:"SEED_ADMIN_EMAIL"`
	SeedAdminPassword string `env:"SEED_ADMIN_PASSWORD"`
	SeedAdminName     string `env:"SEED_ADMIN_NAME"`
}

// IsProd reports whether the service runs in production mode.
func (c Config) IsProd() bool { return c.Env == "production" }

// Load reads .env (if present, local only) and parses the environment into Config.
func Load() (Config, error) {
	// godotenv.Load returns an error if .env is absent — intentionally ignored:
	// in production, env vars come from the platform, not a file.
	_ = godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
