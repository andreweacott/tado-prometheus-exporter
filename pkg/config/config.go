// Package config handles application configuration.
//
// It provides:
//   - Flag parsing with CLI arguments
//   - Environment variable support (with CLI override)
//   - Configuration validation
//   - Precedence: CLI flags > environment variables > defaults
//
// Supported environment variables:
//   - TADO_TOKEN_PATH: Path to token storage file
//   - TADO_TOKEN_PASSPHRASE: Passphrase for token encryption
//   - TADO_PORT: HTTP server port
//   - TADO_HOME_ID: Filter to specific Tado home
//   - TADO_SCRAPE_TIMEOUT: Timeout for API requests (seconds)
//   - TADO_LOG_LEVEL: Logging level (debug, info, warn, error)
//
// Example usage:
//
//	cfg := config.Load()
//	if err := cfg.Validate(); err != nil {
//		log.Fatal(err)
//	}
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

// Load parses environment variables and command-line flags and returns a Config
// Precedence: CLI flags > environment variables > defaults
func Load() *Config {
	return LoadWithArgs(os.Args[1:])
}

// LoadWithArgs loads configuration with explicit arguments (useful for testing)
func LoadWithArgs(args []string) *Config {
	cfg := &Config{}

	// Read environment variables
	envTokenPath := os.Getenv("TADO_TOKEN_PATH")
	envTokenPassphrase := os.Getenv("TADO_TOKEN_PASSPHRASE")
	envPort := os.Getenv("TADO_PORT")
	envHomeID := os.Getenv("TADO_HOME_ID")
	envScrapeTimeout := os.Getenv("TADO_SCRAPE_TIMEOUT")
	envLogLevel := os.Getenv("TADO_LOG_LEVEL")

	// Determine defaults
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "/root"
	}
	defaultTokenPath := filepath.Join(homeDir, ".tado-exporter", "token.json")

	// Use env var if set, otherwise use default
	if envTokenPath != "" {
		defaultTokenPath = envTokenPath
	}
	if envTokenPassphrase == "" {
		envTokenPassphrase = ""
	}
	if envPort == "" {
		envPort = "9100"
	}
	if envScrapeTimeout == "" {
		envScrapeTimeout = "10"
	}
	if envLogLevel == "" {
		envLogLevel = "info"
	}

	// Create a new FlagSet for this invocation (allows multiple calls in tests)
	fs := flag.NewFlagSet("config", flag.ContinueOnError)

	// Parse command-line flags (these override env vars)
	fs.StringVar(&cfg.TokenPath, "token-path", defaultTokenPath, "Path to store the encrypted token (env: TADO_TOKEN_PATH)")
	fs.StringVar(&cfg.TokenPassphrase, "token-passphrase", envTokenPassphrase, "Passphrase to encrypt/decrypt the token (env: TADO_TOKEN_PASSPHRASE, required)")

	// Server configuration
	fs.IntVar(&cfg.Port, "port", parseEnvInt(envPort, 9100), "HTTP server listen port (env: TADO_PORT)")
	fs.StringVar(&cfg.HomeID, "home-id", envHomeID, "Tado Home ID (env: TADO_HOME_ID, optional)")
	fs.IntVar(&cfg.ScrapeTimeout, "scrape-timeout", parseEnvInt(envScrapeTimeout, 10), "Maximum time in seconds to wait for API response (env: TADO_SCRAPE_TIMEOUT)")
	fs.StringVar(&cfg.LogLevel, "log-level", envLogLevel, "Logging verbosity: debug, info, warn, error (env: TADO_LOG_LEVEL)")

	// Parse args - in production this will be os.Args, in tests can be empty or custom
	// FlagSet is configured with ContinueOnError, so parse errors are handled gracefully
	_ = fs.Parse(args)

	return cfg
}

// parseEnvInt parses an environment variable as an integer, returning default if invalid
func parseEnvInt(envValue string, defaultValue int) int {
	if envValue == "" {
		return defaultValue
	}
	var result int
	_, err := fmt.Sscanf(envValue, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
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
