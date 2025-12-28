package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
)

var (
	globalLogger *zerolog.Logger
	mu           sync.RWMutex
	defaultLog   zerolog.Logger
)

func init() {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	defaultLog = zerolog.New(output).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Caller().
		Logger()
}

func New(cfg config.LoggingConfig) (*zerolog.Logger, error) {
	level, err := parseLogLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	var writer io.Writer
	if cfg.File != "" {
		file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		writer = file
	} else {
		writer = os.Stdout
	}

	var output io.Writer
	switch cfg.Format {
	case "console":
		output = zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339,
			NoColor:    cfg.File != "", // Disable colors when logging to file
		}
	case "json":
		output = writer
	default:
		return nil, fmt.Errorf("invalid log format: %s (must be 'console' or 'json')", cfg.Format)
	}

	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()

	return &logger, nil
}

func parseLogLevel(level string) (zerolog.Level, error) {
	switch level {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	default:
		return zerolog.InfoLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

// SetGlobal sets the global logger instance
// This is thread-safe and can be called from multiple goroutines
func SetGlobal(log *zerolog.Logger) {
	mu.Lock()
	defer mu.Unlock()
	globalLogger = log
}

// GetGlobal returns the global logger instance
// If no global logger has been set, it returns a default logger
// This is thread-safe and can be called from multiple goroutines
func GetGlobal() *zerolog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if globalLogger == nil {
		return &defaultLog
	}
	return globalLogger
}

func Info() *zerolog.Event {
	return GetGlobal().Info()
}

func Debug() *zerolog.Event {
	return GetGlobal().Debug()
}

func Warn() *zerolog.Event {
	return GetGlobal().Warn()
}

func Error() *zerolog.Event {
	return GetGlobal().Error()
}

func Fatal() *zerolog.Event {
	return GetGlobal().Fatal()
}

func With() zerolog.Context {
	return GetGlobal().With()
}
