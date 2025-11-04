package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew_ValidLevels tests creating loggers with valid log levels
func TestNew_ValidLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			log, err := New(level, "text")
			require.NoError(t, err)
			assert.NotNil(t, log)
		})
	}
}

// TestNew_InvalidLevel tests creating logger with invalid log level
func TestNew_InvalidLevel(t *testing.T) {
	log, err := New("invalid", "text")

	assert.Error(t, err)
	assert.Nil(t, log)
	assert.Contains(t, err.Error(), "invalid log level")
}

// TestNew_TextFormat tests creating logger with text format
func TestNew_TextFormat(t *testing.T) {
	log, err := New("info", "text")

	require.NoError(t, err)
	assert.NotNil(t, log)
}

// TestNew_JSONFormat tests creating logger with JSON format
func TestNew_JSONFormat(t *testing.T) {
	log, err := New("info", "json")

	require.NoError(t, err)
	assert.NotNil(t, log)
}

// TestNew_InvalidFormat tests creating logger with invalid format
func TestNew_InvalidFormat(t *testing.T) {
	log, err := New("info", "invalid")

	assert.Error(t, err)
	assert.Nil(t, log)
	assert.Contains(t, err.Error(), "invalid log format")
}

// TestNewWithWriter_TextFormat tests logger with custom writer in text format
func TestNewWithWriter_TextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("info", "text", buf)

	require.NoError(t, err)
	assert.NotNil(t, log)

	log.Info("test message")
	output := buf.String()

	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "level=info")
}

// TestNewWithWriter_JSONFormat tests logger with custom writer in JSON format
func TestNewWithWriter_JSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("info", "json", buf)

	require.NoError(t, err)
	assert.NotNil(t, log)

	log.Info("test message")
	output := buf.String()

	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "\"level\":\"info\"")
	assert.Contains(t, output, "\"msg\":\"test message\"")
}

// TestWithRequestID tests adding request ID context
func TestWithRequestID(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("info", "json", buf)
	require.NoError(t, err)

	entry := log.WithRequestID("req-12345")
	entry.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "\"request_id\":\"req-12345\"")
}

// TestWithError tests adding error context
func TestWithError(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("info", "json", buf)
	require.NoError(t, err)

	testErr := assert.AnError
	entry := log.WithError(testErr)
	entry.Error("test error")

	output := buf.String()
	assert.Contains(t, output, "\"error\":\"assert.AnError")
}

// TestWithHomeID tests adding home ID context
func TestWithHomeID(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("info", "json", buf)
	require.NoError(t, err)

	entry := log.WithHomeID(12345)
	entry.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "\"home_id\":12345")
}

// TestWithZoneID tests adding zone ID context
func TestWithZoneID(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("info", "json", buf)
	require.NoError(t, err)

	entry := log.WithZoneID(1)
	entry.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "\"zone_id\":1")
}

// TestWithZoneName tests adding zone name context
func TestWithZoneName(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("info", "json", buf)
	require.NoError(t, err)

	entry := log.WithZoneName("Living Room")
	entry.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "\"zone_name\":\"Living Room\"")
}

// TestLogLevels tests that log levels are respected
func TestLogLevels(t *testing.T) {
	tests := []struct {
		name         string
		level        string
		shouldLog    string
		shouldNotLog string
	}{
		{
			name:         "debug level logs everything",
			level:        "debug",
			shouldLog:    "debug",
			shouldNotLog: "",
		},
		{
			name:         "info level skips debug",
			level:        "info",
			shouldLog:    "info",
			shouldNotLog: "debug",
		},
		{
			name:         "warn level skips info and debug",
			level:        "warn",
			shouldLog:    "warn",
			shouldNotLog: "info",
		},
		{
			name:         "error level only logs errors",
			level:        "error",
			shouldLog:    "error",
			shouldNotLog: "warn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log, err := NewWithWriter(tt.level, "text", buf)
			require.NoError(t, err)

			log.Debug("debug message")
			log.Info("info message")
			log.Warn("warn message")
			log.Error("error message")

			output := buf.String()

			if tt.shouldLog != "" {
				assert.Contains(t, output, tt.shouldLog+" message")
			}
			if tt.shouldNotLog != "" {
				assert.NotContains(t, output, tt.shouldNotLog+" message")
			}
		})
	}
}

// TestJSONFormatValidation tests that JSON output is valid
func TestJSONFormatValidation(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("info", "json", buf)
	require.NoError(t, err)

	log.Info("test message")
	output := buf.String()

	// Check for valid JSON structure
	assert.Contains(t, output, "\"level\":")
	assert.Contains(t, output, "\"msg\":")
	assert.Contains(t, output, "\"time\":")
	// Logrus adds a trailing newline, so check it's only one line of JSON
	lines := bytes.Split(bytes.TrimSpace([]byte(output)), []byte("\n"))
	assert.Equal(t, 1, len(lines))
}

// TestTextFormatTimestamps tests that text format includes timestamps
func TestTextFormatTimestamps(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("info", "text", buf)
	require.NoError(t, err)

	log.Info("test message")
	output := buf.String()

	// Check for timestamp pattern (YYYY-MM-DD HH:MM:SS)
	assert.Regexp(t, `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`, output)
}

// TestChainingContext tests that multiple contexts can be chained
func TestChainingContext(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("info", "json", buf)
	require.NoError(t, err)

	entry := log.Logger.WithFields(map[string]interface{}{
		"request_id": "req-123",
		"home_id":    12345,
		"zone_id":    1,
	})
	entry.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "\"request_id\":\"req-123\"")
	assert.Contains(t, output, "\"home_id\":12345")
	assert.Contains(t, output, "\"zone_id\":1")
}

// TestLogMessagePreservation tests that messages are correctly logged
func TestLogMessagePreservation(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("info", "json", buf)
	require.NoError(t, err)

	messages := []string{
		"Exporter started successfully",
		"Failed to collect metrics: connection timeout",
		"Zone metrics collected for home 12345",
	}

	for _, msg := range messages {
		buf.Reset()
		log.Info(msg)
		output := buf.String()
		assert.Contains(t, output, msg)
	}
}

// TestErrorLogging tests error logging with context
func TestErrorLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	log, err := NewWithWriter("error", "json", buf)
	require.NoError(t, err)

	testErr := assert.AnError
	log.Logger.WithFields(map[string]interface{}{
		"error":   testErr.Error(),
		"home_id": 12345,
	}).Error("Failed to fetch metrics")

	output := buf.String()
	assert.Contains(t, output, "\"level\":\"error\"")
	assert.Contains(t, output, "Failed to fetch metrics")
	assert.Contains(t, output, "home_id")
}
