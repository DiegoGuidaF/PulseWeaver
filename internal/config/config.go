package config

import (
	"fmt"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Conf struct {
	Server    ConfServer
	DB        ConfDB
	Whitelist ConfWhitelist
	Rules     ConfRules
	Caddy     ConfCaddy
	LogLevel  string         `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat logging.Format `env:"LOG_FORMAT" envDefault:"text"` // "json" or "text" (tint)
	LogColor  bool           `env:"LOG_COLOR" envDefault:"false"` // Enable colored output for tint format
}

type ConfServer struct {
	AdminPassword string `env:"ADMIN_PASSWORD,required"`
	Port          int    `env:"SERVER_PORT" envDefault:"8080"`
	TrustedProxy  string `env:"TRUSTED_PROXY"`
	TZ            string `env:"TZ" envDefault:"UTC"`
}

type ConfDB struct {
	File  string `env:"DB_FILE" envDefault:"data.db"`
	Debug bool   `env:"DB_DEBUG" envDefault:"false"`
	Dsn   string
}

type ConfWhitelist struct {
	FilePath  string        `env:"WHITELIST_FILE_PATH" envDefault:"./whitelist.txt"`
	RateLimit time.Duration `env:"WHITELIST_RATE_LIMIT" envDefault:"5s"`
}

type ConfCaddy struct {
	Endpoint  string `env:"CADDY_RELOADER_ENDPOINT"`
	AuthToken string `env:"CADDY_RELOADER_AUTH_TOKEN"`
}

// ConfRules holds configuration for background rule/scheduler behaviour.
type ConfRules struct {
	CheckInterval time.Duration `env:"RULE_CHECK_INTERVAL" envDefault:"1m"`
}

func Load() (*Conf, error) {
	var c Conf

	// Load .env file if present (optional, ignore errors)
	//nolint:staticcheck // Empty branch is intentional - .env file is optional
	if err := godotenv.Load(); err != nil {
		_ = err // Explicitly ignore error
	}

	// Create config struct from env variables
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Validate after parsing
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	if c.Caddy.Endpoint != "" {
		if c.Caddy.AuthToken == "" {
			return nil, fmt.Errorf("caddy endpoint defined but auth token is missing")
		}
	}

	return &c, nil
}
