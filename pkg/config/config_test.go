package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLoad_FromEnvironmentVariables tests loading configuration from environment variables
func TestLoad_FromEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("TADO_PORT", "9091")
	os.Setenv("TADO_TOKEN_PASSPHRASE", "test-passphrase")
	os.Setenv("TADO_HOME_ID", "12345")
	os.Setenv("TADO_SCRAPE_TIMEOUT", "20")
	os.Setenv("TADO_LOG_LEVEL", "debug")
	os.Setenv("TADO_TOKEN_PATH", "/tmp/token.json")
	defer func() {
		os.Unsetenv("TADO_PORT")
		os.Unsetenv("TADO_TOKEN_PASSPHRASE")
		os.Unsetenv("TADO_HOME_ID")
		os.Unsetenv("TADO_SCRAPE_TIMEOUT")
		os.Unsetenv("TADO_LOG_LEVEL")
		os.Unsetenv("TADO_TOKEN_PATH")
	}()

	// Call with empty args (no CLI flags)
	cfg := LoadWithArgs([]string{})

	assert.Equal(t, 9091, cfg.Port)
	assert.Equal(t, "test-passphrase", cfg.TokenPassphrase)
	assert.Equal(t, "12345", cfg.HomeID)
	assert.Equal(t, 20, cfg.ScrapeTimeout)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "/tmp/token.json", cfg.TokenPath)
}

// TestLoad_Defaults tests loading configuration with default values
func TestLoad_Defaults(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("TADO_PORT")
	os.Unsetenv("TADO_TOKEN_PASSPHRASE")
	os.Unsetenv("TADO_HOME_ID")
	os.Unsetenv("TADO_SCRAPE_TIMEOUT")
	os.Unsetenv("TADO_LOG_LEVEL")
	os.Unsetenv("TADO_TOKEN_PATH")

	cfg := LoadWithArgs([]string{})

	assert.Equal(t, 9100, cfg.Port) // default port
	assert.Equal(t, 10, cfg.ScrapeTimeout) // default timeout
	assert.Equal(t, "info", cfg.LogLevel) // default log level
	assert.Equal(t, "", cfg.HomeID) // optional
	assert.Equal(t, "", cfg.TokenPassphrase) // required (but empty by default)
}

// TestLoad_InvalidEnvironmentVariables tests handling of invalid environment variables
func TestLoad_InvalidEnvironmentVariables(t *testing.T) {
	os.Setenv("TADO_PORT", "invalid")
	os.Setenv("TADO_SCRAPE_TIMEOUT", "not-a-number")
	defer func() {
		os.Unsetenv("TADO_PORT")
		os.Unsetenv("TADO_SCRAPE_TIMEOUT")
	}()

	cfg := LoadWithArgs([]string{})

	// Should fall back to defaults when invalid
	assert.Equal(t, 9100, cfg.Port)
	assert.Equal(t, 10, cfg.ScrapeTimeout)
}

// TestValidate_MissingPassphrase tests validation fails without passphrase
func TestValidate_MissingPassphrase(t *testing.T) {
	cfg := &Config{
		TokenPath:       "/tmp/token.json",
		TokenPassphrase: "",
		Port:            9100,
		ScrapeTimeout:   10,
		LogLevel:        "info",
	}

	err := cfg.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token-passphrase is required")
}

// TestValidate_InvalidPort tests validation of port range
func TestValidate_InvalidPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		valid   bool
	}{
		{"valid port 1", 1, true},
		{"valid port 9100", 9100, true},
		{"valid port 65535", 65535, true},
		{"invalid port 0", 0, false},
		{"invalid port -1", -1, false},
		{"invalid port 65536", 65536, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				TokenPath:       "/tmp/token.json",
				TokenPassphrase: "test",
				Port:            tt.port,
				ScrapeTimeout:   10,
				LogLevel:        "info",
			}

			err := cfg.Validate()

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid port")
			}
		})
	}
}

// TestValidate_InvalidTimeout tests validation of timeout
func TestValidate_InvalidTimeout(t *testing.T) {
	tests := []struct {
		name     string
		timeout  int
		valid    bool
	}{
		{"valid timeout 1", 1, true},
		{"valid timeout 10", 10, true},
		{"invalid timeout 0", 0, false},
		{"invalid timeout -1", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				TokenPath:       "/tmp/token.json",
				TokenPassphrase: "test",
				Port:            9100,
				ScrapeTimeout:   tt.timeout,
				LogLevel:        "info",
			}

			err := cfg.Validate()

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "scrape-timeout")
			}
		})
	}
}

// TestValidate_InvalidLogLevel tests validation of log level
func TestValidate_InvalidLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		valid    bool
	}{
		{"valid debug", "debug", true},
		{"valid info", "info", true},
		{"valid warn", "warn", true},
		{"valid error", "error", true},
		{"invalid invalid", "invalid", false},
		{"invalid TRACE", "TRACE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				TokenPath:       "/tmp/token.json",
				TokenPassphrase: "test",
				Port:            9100,
				ScrapeTimeout:   10,
				LogLevel:        tt.logLevel,
			}

			err := cfg.Validate()

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "log-level")
			}
		})
	}
}

// TestValidate_ValidConfig tests validation of valid config
func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		TokenPath:       "/tmp/token.json",
		TokenPassphrase: "secure-passphrase",
		Port:            9100,
		ScrapeTimeout:   15,
		LogLevel:        "info",
		HomeID:          "12345",
	}

	err := cfg.Validate()

	assert.NoError(t, err)
}

// TestParseEnvInt tests integer parsing from environment values
func TestParseEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"valid value", "42", 100, 42},
		{"empty value uses default", "", 100, 100},
		{"invalid value uses default", "not-a-number", 100, 100},
		{"negative value", "-10", 100, -10},
		{"zero value", "0", 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEnvInt(tt.envValue, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestString tests the String method for debug output
func TestString(t *testing.T) {
	cfg := &Config{
		TokenPath:       "/tmp/token.json",
		TokenPassphrase: "secret",
		Port:            9100,
		ScrapeTimeout:   10,
		LogLevel:        "info",
		HomeID:          "12345",
	}

	str := cfg.String()

	assert.Contains(t, str, "Port: 9100")
	assert.Contains(t, str, "LogLevel: info")
	assert.Contains(t, str, "ScrapeTimeout: 10s")
	assert.NotContains(t, str, "secret") // Don't leak password
}
