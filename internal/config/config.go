package config

import (
	"fmt"
	"net/netip"
	"reflect"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/logging"
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Conf struct {
	Server    ConfServer
	DB        ConfDB
	Rules     ConfRules
	Authz     ConfAuthz
	LogLevel  string         `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat logging.Format `env:"LOG_FORMAT" envDefault:"text"` // "json" or "text" (tint)
	LogColor  bool           `env:"LOG_COLOR" envDefault:"true"`  // Enable colored output (only for text format)
}

type ConfServer struct {
	AdminPassword string     `env:"ADMIN_PASSWORD,required,notEmpty"`
	Port          int        `env:"SERVER_PORT" envDefault:"8080"`
	TrustedProxy  netip.Addr `env:"TRUSTED_PROXY"`
	TZ            string     `env:"TZ" envDefault:"UTC"`
}

type ConfDB struct {
	File  string `env:"DB_FILE" envDefault:"data.db"`
	Debug bool   `env:"DB_DEBUG" envDefault:"false"`
	Dsn   string
}

type ConfAuthz struct {
	APISecret string `env:"AUTHZ_API_SECRET,required,notEmpty"`
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

	envParsingOpts := env.Options{
		FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeOf(netip.Addr{}): parseIPAddressFunc,
		},
	}

	// Create config struct from env variables
	if err := env.ParseWithOptions(&c, envParsingOpts); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	// Ensure rule scheduler has a valid interval
	if c.Rules.CheckInterval <= 0 {
		return nil, fmt.Errorf("check interval must be bigger than 0: %d", c.Rules.CheckInterval)
	}

	// Ensure api secret for Authz endpoint is defined and secure
	if len(c.Authz.APISecret) < 16 {
		return nil, fmt.Errorf("authz api secret is too short (got %d chars, minimum 16); generate one with: openssl rand -base64 16", len(c.Authz.APISecret))
	}
	return &c, nil
}

func parseIPAddressFunc(v string) (any, error) {
	if v == "" {
		return netip.Addr{}, nil
	}

	addr, err := netip.ParseAddr(v)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("invalid IP '%s': %w", v, err)
	}

	return addr, nil
}
