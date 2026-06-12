package config

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"reflect"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Conf struct {
	Server    ConfServer
	DB        ConfDB
	Rules     ConfRules
	Policy    ConfPolicy
	GeoIP     ConfGeoIP
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
	DataDir string `env:"DB_DIR" envDefault:"./data"`
	Dsn     string
}

type ConfPolicy struct {
	APISecret string `env:"POLICY_ENGINE_API_SECRET,required,notEmpty"`
}

// ConfRules holds configuration for background rule/scheduler behaviour.
type ConfRules struct {
	CheckInterval     time.Duration `env:"RULE_CHECK_INTERVAL" envDefault:"1m"`
	DataRetentionDays int           `env:"DATA_RETENTION_DAYS" envDefault:"30"`
	// AggregateRetentionDays bounds hourly_traffic_aggregates, which serve
	// dashboard windows wider than 24h. Must cover at least DataRetentionDays,
	// otherwise wide dashboard windows would show less history than the raw
	// access log still holds.
	AggregateRetentionDays int `env:"AGGREGATE_RETENTION_DAYS" envDefault:"365"`
}

// ConfGeoIP holds configuration for the GeoIP enrichment feature.
type ConfGeoIP struct {
	Enabled bool   `env:"GEOIP_ENABLED"   envDefault:"true"`
	DataDir string `env:"GEOIP_DATA_DIR"  envDefault:"./data/geoip"`
}

func Load() (*Conf, error) {
	c := new(Conf)

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
	if err := env.ParseWithOptions(c, envParsingOpts); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	// Ensure rule scheduler has a valid interval
	if c.Rules.CheckInterval <= 0 {
		return nil, fmt.Errorf("check interval must be bigger than 0: %d", c.Rules.CheckInterval)
	}

	// Aggregates must outlive raw data (0 = unbounded); a shorter horizon would
	// make wide dashboard windows show less history than the raw access log.
	if c.Rules.AggregateRetentionDays != 0 &&
		(c.Rules.DataRetentionDays == 0 || c.Rules.AggregateRetentionDays < c.Rules.DataRetentionDays) {
		return nil, fmt.Errorf("AGGREGATE_RETENTION_DAYS (%d) must be 0 (unbounded) or >= DATA_RETENTION_DAYS (%d)",
			c.Rules.AggregateRetentionDays, c.Rules.DataRetentionDays)
	}

	// Ensure api secret for Policy endpoint is defined and secure
	if len(c.Policy.APISecret) < 16 {
		return nil, fmt.Errorf("policy api secret is too short (got %d chars, minimum 16); generate one with: openssl rand -base64 16", len(c.Policy.APISecret))
	}

	if err := validateWritableDir(c.DB.DataDir); err != nil {
		return nil, fmt.Errorf("DB dir is not valid: %w", err)
	}

	if err := validateWritableDir(c.GeoIP.DataDir); err != nil {
		return nil, fmt.Errorf("GeoIPDB dir is not valid: %w", err)
	}

	return c, nil
}

func parseIPAddressFunc(rawIP string) (any, error) {
	if rawIP == "" {
		return netip.Addr{}, nil
	}

	addr, err := netip.ParseAddr(rawIP)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("invalid IP '%s': %w", rawIP, err)
	}

	// Unmap so an IPv4-mapped IPv6 trusted proxy still compares equal to the
	// canonical client address the policy engine evaluates.
	return addr.Unmap(), nil
}

func validateWritableDir(dir string) error {
	// Check if it is a directory (or can become one)
	info, err := os.Stat(dir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat directory %q: %w", dir, err)
		}
		// Directory doesn't exist yet, but we can create it later
	} else if !info.IsDir() {
		return fmt.Errorf("path %q exists but is not a directory", dir)
	}

	// Check writability by attempting to create a temp file
	f, err := os.CreateTemp(dir, "test-dir-write-check-*")
	if err != nil {
		return fmt.Errorf("dir %q is not writable: %w", dir, err)

	}
	defer func(f *os.File) {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}(f)

	return nil
}
