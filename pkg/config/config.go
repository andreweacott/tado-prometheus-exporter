package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	// Token storage
	TokenPath       string
	TokenPassphrase string

	// Server configuration
	Port int

	// Tado API configuration
	HomeID string

	// Collection configuration
	ScrapeTimeout int

	// Logging
	LogLevel string
}

// Load parses command-line flags and returns a Config
func Load() *Config {
	cfg := &Config{}

	// Token storage configuration
	tokenPath := os.Getenv("HOME")
	if tokenPath == "" {
		tokenPath = "/root"
	}
	defaultTokenPath := filepath.Join(tokenPath, ".tado-exporter", "token.json")
	flag.StringVar(&cfg.TokenPath, "token-path", defaultTokenPath, "Path to store the encrypted token")
	flag.StringVar(&cfg.TokenPassphrase, "token-passphrase", "", "Passphrase to encrypt/decrypt the token (required)")

	// Server configuration
	flag.IntVar(&cfg.Port, "port", 9100, "HTTP server listen port")
	flag.StringVar(&cfg.HomeID, "home-id", "", "Tado Home ID (optional, auto-detect if not provided)")
	flag.IntVar(&cfg.ScrapeTimeout, "scrape-timeout", 10, "Maximum time in seconds to wait for API response")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Logging verbosity (debug, info, warn, error)")

	flag.Parse()

	return cfg
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.TokenPassphrase == "" {
		return fmt.Errorf("token-passphrase is required (use -token-passphrase flag or TADO_TOKEN_PASSPHRASE env var)")
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be between 1 and 65535)", c.Port)
	}

	if c.ScrapeTimeout < 1 {
		return fmt.Errorf("invalid scrape-timeout: %d (must be at least 1 second)", c.ScrapeTimeout)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log-level: %s (must be one of: debug, info, warn, error)", c.LogLevel)
	}

	return nil
}

// String returns a string representation of the config (without sensitive data)
func (c *Config) String() string {
	return fmt.Sprintf("Config{Port: %d, TokenPath: %s, HomeID: %s, ScrapeTimeout: %ds, LogLevel: %s}",
		c.Port, c.TokenPath, c.HomeID, c.ScrapeTimeout, c.LogLevel)
}
