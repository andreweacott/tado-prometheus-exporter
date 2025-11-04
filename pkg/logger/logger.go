// Package logger provides structured logging for the exporter.
//
// It wraps logrus to provide:
//   - Structured logging with JSON and text output
//   - Configurable log levels (debug, info, warn, error)
//   - Convenience methods for adding context fields
//   - Output routing to files, stdout, or custom writers
//
// Example usage:
//
//	log, err := logger.New("info", "json")
//	if err != nil {
//		fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
//	}
//	log.Info("Application started")
//	log.WithField("home_id", 12345).Warn("Failed to collect metrics", "error", err)
package logger

import (
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus.Logger with convenience methods
type Logger struct {
	*logrus.Logger
}

// New creates a new logger with specified level and format
func New(level, format string) (*Logger, error) {
	log := logrus.New()

	// Set log level
	parsedLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %s", level)
	}
	log.SetLevel(parsedLevel)

	// Set output to stderr (standard for structured logging)
	log.SetOutput(os.Stderr)

	// Set format based on configuration
	switch format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	case "text":
		log.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
		})
	default:
		return nil, fmt.Errorf("invalid log format: %s (must be 'json' or 'text')", format)
	}

	return &Logger{log}, nil
}

// NewWithWriter creates a new logger with custom output writer
func NewWithWriter(level, format string, out io.Writer) (*Logger, error) {
	log := logrus.New()

	// Set log level
	parsedLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %s", level)
	}
	log.SetLevel(parsedLevel)

	// Set output
	log.SetOutput(out)

	// Set format
	switch format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	case "text":
		log.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
		})
	default:
		return nil, fmt.Errorf("invalid log format: %s (must be 'json' or 'text')", format)
	}

	return &Logger{log}, nil
}

// WithRequestID returns a logger entry with request ID context
func (l *Logger) WithRequestID(requestID string) *logrus.Entry {
	return l.WithField("request_id", requestID)
}

// WithError returns a logger entry with error context
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.WithField("error", err.Error())
}

// WithHomeID returns a logger entry with home ID context
func (l *Logger) WithHomeID(homeID int64) *logrus.Entry {
	return l.WithField("home_id", homeID)
}

// WithZoneID returns a logger entry with zone ID context
func (l *Logger) WithZoneID(zoneID int64) *logrus.Entry {
	return l.WithField("zone_id", zoneID)
}

// WithZoneName returns a logger entry with zone name context
func (l *Logger) WithZoneName(zoneName string) *logrus.Entry {
	return l.WithField("zone_name", zoneName)
}

// Info logs an info level message
func (l *Logger) Info(msg string, fields ...interface{}) {
	if len(fields) > 0 {
		l.Logger.WithFields(toFields(fields)).Info(msg)
	} else {
		l.Logger.Info(msg)
	}
}

// Debug logs a debug level message
func (l *Logger) Debug(msg string, fields ...interface{}) {
	if len(fields) > 0 {
		l.Logger.WithFields(toFields(fields)).Debug(msg)
	} else {
		l.Logger.Debug(msg)
	}
}

// Warn logs a warning level message
func (l *Logger) Warn(msg string, fields ...interface{}) {
	if len(fields) > 0 {
		l.Logger.WithFields(toFields(fields)).Warn(msg)
	} else {
		l.Logger.Warn(msg)
	}
}

// Error logs an error level message
func (l *Logger) Error(msg string, fields ...interface{}) {
	if len(fields) > 0 {
		l.Logger.WithFields(toFields(fields)).Error(msg)
	} else {
		l.Logger.Error(msg)
	}
}

// toFields converts variadic key-value pairs to logrus.Fields
func toFields(args []interface{}) logrus.Fields {
	fields := logrus.Fields{}
	for i := 0; i < len(args)-1; i += 2 {
		key := fmt.Sprintf("%v", args[i])
		value := args[i+1]
		fields[key] = value
	}
	return fields
}
