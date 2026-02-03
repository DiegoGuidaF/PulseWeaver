package config

import (
	"fmt"
	"log"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Conf struct {
	Server      ConfServer
	DB          ConfDB
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	TZ          string `env:"TZ" envDefault:"UTC"`
}

type ConfServer struct {
	Port int `env:"SERVER_PORT" envDefault:"8080"`
}

type ConfDB struct {
	File  string `env:"DB_FILE" envDefault:"data.db"`
	Debug bool   `env:"DB_DEBUG" envDefault:"false"`
	Dsn   string
}

func Load() (*Conf, error) {
	var c Conf

	// Load .env file if present
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
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
