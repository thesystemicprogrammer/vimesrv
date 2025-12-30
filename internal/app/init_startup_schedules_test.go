package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// Mock for CronParser
type MockCronParser struct {
	mock.Mock
}

func (m *MockCronParser) Parse(spec string) (ports.CronSchedule, error) {
	args := m.Called(spec)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ports.CronSchedule), args.Error(1)
}

// Mock for CronSchedule (needed for successful parse)
type MockCronSchedule struct {
	mock.Mock
}

// TestInitStartupSchedules_LibraryScanDisabled tests that function completes when library scan is disabled
func TestInitStartupSchedules_LibraryScanDisabled(t *testing.T) {
	ctx := context.Background()

	cfg := &config.Config{
		Media: config.MediaConfig{
			LibraryScan: config.LibraryScanConfig{
				Enabled:  false,
				CronSpec: "0 * * * * *",
				Priority: 0,
			},
		},
	}

	adapters := &Adapters{}
	useCases := &UseCases{}

	// Should complete successfully without calling any services
	err := initStartupSchedules(ctx, cfg, useCases, adapters)
	require.NoError(t, err)
}

// TestInitPeriodicLibraryScan_InvalidCronSpec tests fail-fast behavior with invalid cron spec
func TestInitPeriodicLibraryScan_InvalidCronSpec(t *testing.T) {
	ctx := context.Background()
	mockParser := new(MockCronParser)

	cfg := &config.Config{
		Media: config.MediaConfig{
			LibraryScan: config.LibraryScanConfig{
				Enabled:  true,
				CronSpec: "invalid cron spec",
				Priority: 0,
			},
		},
	}

	adapters := &Adapters{
		CronParser: mockParser,
	}

	useCases := &UseCases{}

	// Expect cron validation to fail
	mockParser.On("Parse", "invalid cron spec").Return(nil, errors.New("invalid cron format")).Once()

	err := initPeriodicLibraryScan(ctx, cfg, useCases, adapters)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron_spec")
	assert.Contains(t, err.Error(), "invalid cron format")
	mockParser.AssertExpectations(t)
}

// TestCronSpecValidation tests various cron spec formats with mock parser
func TestCronSpecValidation(t *testing.T) {
	tests := []struct {
		name      string
		cronSpec  string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "invalid empty cron spec",
			cronSpec:  "",
			shouldErr: true,
			errMsg:    "empty spec string",
		},
		{
			name:      "invalid cron spec format",
			cronSpec:  "not a cron",
			shouldErr: true,
			errMsg:    "expected exactly 6 fields",
		},
		{
			name:      "invalid 5-field cron spec",
			cronSpec:  "0 * * * *",
			shouldErr: true,
			errMsg:    "expected exactly 6 fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockParser := new(MockCronParser)

			cfg := &config.Config{
				Media: config.MediaConfig{
					LibraryScan: config.LibraryScanConfig{
						Enabled:  true,
						CronSpec: tt.cronSpec,
						Priority: 0,
					},
				},
			}

			adapters := &Adapters{
				CronParser: mockParser,
			}

			useCases := &UseCases{}

			if tt.shouldErr {
				mockParser.On("Parse", tt.cronSpec).Return(nil, errors.New(tt.errMsg)).Once()
				err := initPeriodicLibraryScan(ctx, cfg, useCases, adapters)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid cron_spec")
			}

			mockParser.AssertExpectations(t)
		})
	}
}

// TestScheduleNameConstant tests that the schedule name constant is correct
func TestScheduleNameConstant(t *testing.T) {
	assert.Equal(t, "periodic_library_scan", scheduleNamePeriodicLibraryScan)
}

// TestInitStartupSchedules_Integration tests the overall flow
// This test verifies the error paths work correctly
func TestInitStartupSchedules_Integration(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		cronSpec  string
		parseErr  error
		expectErr bool
		errMsg    string
	}{
		{
			name:      "disabled scan - should succeed",
			enabled:   false,
			cronSpec:  "",
			expectErr: false,
		},
		{
			name:      "enabled with invalid cron - should fail",
			enabled:   true,
			cronSpec:  "invalid",
			parseErr:  errors.New("parse error"),
			expectErr: true,
			errMsg:    "invalid cron_spec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockParser := new(MockCronParser)

			cfg := &config.Config{
				Media: config.MediaConfig{
					LibraryScan: config.LibraryScanConfig{
						Enabled:  tt.enabled,
						CronSpec: tt.cronSpec,
						Priority: 0,
					},
				},
			}

			adapters := &Adapters{
				CronParser: mockParser,
			}

			useCases := &UseCases{}

			if tt.enabled && tt.parseErr != nil {
				mockParser.On("Parse", tt.cronSpec).Return(nil, tt.parseErr).Once()
			}

			err := initStartupSchedules(ctx, cfg, useCases, adapters)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}

			if tt.enabled {
				mockParser.AssertExpectations(t)
			}
		})
	}
}
