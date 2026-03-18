package httpserver

import (
	"time"
)

// ServerConfig holds configuration for the HTTP server lifecycle.
type ServerConfig struct {
	Port              int
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	ShutdownTimeout   time.Duration
}

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16, // 64KB
		ShutdownTimeout:   5 * time.Second,
	}
}

func DefaultServerConfigFromConf(port int) ServerConfig {
	cfg := DefaultServerConfig()
	cfg.Port = port
	return cfg
}
