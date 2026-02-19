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
	LogLevel  string         `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat logging.Format `env:"LOG_FORMAT" envDefault:"text"` // "json" or "text" (tint)
	TZ        string         `env:"TZ" envDefault:"UTC"`
}

type ConfServer struct {
	AdminPassword string `env:"ADMIN_PASSWORD,required"`
	Port          int    `env:"SERVER_PORT" envDefault:"8080"`
	TrustedProxy  string `env:"TRUSTED_PROXY"`
}

type ConfDB struct {
	File  string `env:"DB_FILE" envDefault:"data.db"`
	Debug bool   `env:"DB_DEBUG" envDefault:"false"`
	Dsn   string
}

type ConfWhitelist struct {
	FilePath      string        `env:"WHITELIST_FILE_PATH" envDefault:"./whitelist.txt"`
	DebounceDelay time.Duration `env:"WHITELIST_DEBOUNCE_DELAY" envDefault:"5s"`
}

func Load() (*Conf, error) {
	var c Conf

	// Load .env file if present
	if err := godotenv.Load(); err != nil {
	}

	// Create config struct from env variables
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Validate after parsing
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", c.Server.Port)
	}

	return &c, nil
}
