package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/thesystemicprogrammer/vimesrv/internal/utils/config"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		cfg         config.LoggingConfig
		expectError bool
	}{
		{
			name: "console format, info level",
			cfg: config.LoggingConfig{
				Level:  "info",
				Format: "console",
				File:   "",
			},
			expectError: false,
		},
		{
			name: "json format, debug level",
			cfg: config.LoggingConfig{
				Level:  "debug",
				Format: "json",
				File:   "",
			},
			expectError: false,
		},
		{
			name: "invalid level",
			cfg: config.LoggingConfig{
				Level:  "invalid",
				Format: "console",
				File:   "",
			},
			expectError: true,
		},
		{
			name: "invalid format",
			cfg: config.LoggingConfig{
				Level:  "info",
				Format: "xml",
				File:   "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, err := New(tt.cfg)

			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if !tt.expectError && log == nil {
				t.Error("Expected logger, got nil")
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zerolog.Level
		wantErr  bool
	}{
		{"debug", "debug", zerolog.DebugLevel, false},
		{"info", "info", zerolog.InfoLevel, false},
		{"warn", "warn", zerolog.WarnLevel, false},
		{"error", "error", zerolog.ErrorLevel, false},
		{"invalid", "invalid", zerolog.InfoLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := parseLogLevel(tt.level)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if !tt.wantErr && level != tt.expected {
				t.Errorf("Expected level %v, got %v", tt.expected, level)
			}
		})
	}
}

func TestGlobalLogger(t *testing.T) {
	// Test default logger
	defaultLogger := GetGlobal()
	if defaultLogger == nil {
		t.Fatal("Expected default logger, got nil")
	}

	// Test setting custom logger
	cfg := config.LoggingConfig{
		Level:  "debug",
		Format: "json",
		File:   "",
	}

	customLogger, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	SetGlobal(customLogger)

	// Verify global logger was set
	retrieved := GetGlobal()
	if retrieved != customLogger {
		t.Error("Global logger was not set correctly")
	}
}

func TestLoggerOutput(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a logger that writes to the buffer
	logger := zerolog.New(&buf).Level(zerolog.InfoLevel).With().Timestamp().Logger()

	// Log a message
	logger.Info().Msg("test message")

	// Verify output contains the message
	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
}

func TestLoggerLevels(t *testing.T) {
	var buf bytes.Buffer

	// Create logger with info level
	logger := zerolog.New(&buf).Level(zerolog.InfoLevel)

	// Debug should not be logged
	logger.Debug().Msg("debug message")
	if strings.Contains(buf.String(), "debug message") {
		t.Error("Debug message should not be logged at info level")
	}

	buf.Reset()

	// Info should be logged
	logger.Info().Msg("info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Error("Info message should be logged at info level")
	}

	buf.Reset()

	// Warn should be logged
	logger.Warn().Msg("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Error("Warn message should be logged at info level")
	}

	buf.Reset()

	// Error should be logged
	logger.Error().Msg("error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Error("Error message should be logged at info level")
	}
}

func TestLoggerWithFile(t *testing.T) {
	// Create temporary log file
	tmpFile := "/tmp/vimesrv_test.log"
	defer os.Remove(tmpFile)

	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
		File:   tmpFile,
	}

	logger, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Log a message
	logger.Info().Msg("test file message")

	// Read the file
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify content
	if !strings.Contains(string(content), "test file message") {
		t.Errorf("Expected log file to contain 'test file message', got: %s", string(content))
	}
}

func TestHelperFunctions(t *testing.T) {
	// Create a buffer logger
	var buf bytes.Buffer
	logger := zerolog.New(&buf).Level(zerolog.DebugLevel)
	SetGlobal(&logger)

	// Test helper functions
	Info().Msg("info test")
	if !strings.Contains(buf.String(), "info test") {
		t.Error("Info() helper did not log message")
	}

	buf.Reset()
	Debug().Msg("debug test")
	if !strings.Contains(buf.String(), "debug test") {
		t.Error("Debug() helper did not log message")
	}

	buf.Reset()
	Warn().Msg("warn test")
	if !strings.Contains(buf.String(), "warn test") {
		t.Error("Warn() helper did not log message")
	}

	buf.Reset()
	Error().Msg("error test")
	if !strings.Contains(buf.String(), "error test") {
		t.Error("Error() helper did not log message")
	}
}
