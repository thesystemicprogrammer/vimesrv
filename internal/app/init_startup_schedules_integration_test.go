package app

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/repository"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	usecasejob "github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// setupTestDatabase creates an in-memory SQLite database with migrations for integration tests
func setupTestDatabase(t *testing.T) (*database.DB, *sql.DB) {
	t.Helper()

	cfg := database.Config{
		Path:            "file::memory:?cache=shared",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}

	db, err := database.New(cfg)
	require.NoError(t, err, "failed to create test database")

	// Run migrations
	migration := database.NewDatabaseMigration(db.DB)
	err = migration.Migrate()
	require.NoError(t, err, "failed to run migrations")

	t.Cleanup(func() {
		db.Close()
	})

	return db, db.DB
}

// setupTestDependencies creates all dependencies needed for integration testing
func setupTestDependencies(t *testing.T) (*Adapters, *UseCases, *sql.DB) {
	t.Helper()

	db, sqlDB := setupTestDatabase(t)

	// Create repositories
	jobRepo := repository.NewJobRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)

	// Create adapters
	cronParser := job.NewRobfigCronParser()
	backoffStrategy := job.NewExponentialBackoff(1, 60)
	handlerRegistry := job.NewHandlerRegistry()

	adapters := &Adapters{
		CronParser:         cronParser,
		BackoffStrategy:    backoffStrategy,
		HandlerRegistry:    handlerRegistry,
		JobRepository:      jobRepo,
		ScheduleRepository: scheduleRepo,
	}

	// Create use cases
	jobCfg := config.JobConfig{
		MaxAttempts: 3,
	}

	upsertScheduleUC := usecasejob.NewUpsertScheduleUseCase(jobCfg, scheduleRepo, cronParser, ports.RealClock{})
	enqueueJobUC := usecasejob.NewEnqueueJobUseCase(jobCfg, jobRepo, ports.RealClock{}, &ports.NoOpJobNotifier{})

	useCases := &UseCases{
		UpsertScheduleUseCase: upsertScheduleUC,
		EnqueueJobUseCase:     enqueueJobUC,
	}

	return adapters, useCases, sqlDB
}

// TestInitStartupSchedules_Integration_LibraryScanEnabled tests the full integration
// with real database, verifying schedule is created correctly
func TestInitStartupSchedules_Integration_LibraryScanEnabled(t *testing.T) {
	ctx := context.Background()
	adapters, useCases, sqlDB := setupTestDependencies(t)

	cfg := &config.Config{
		Media: config.MediaConfig{
			LibraryScan: config.LibraryScanConfig{
				Enabled:  true,
				CronSpec: "0 * * * * *",
				Priority: 5,
			},
		},
	}

	// Execute initialization
	err := initStartupSchedules(ctx, cfg, useCases, adapters)
	require.NoError(t, err)

	// Verify schedule was created in database
	var count int
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM schedules WHERE name = ?", scheduleNamePeriodicLibraryScan).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should have created exactly one schedule")

	// Verify schedule details
	var schedule struct {
		ID          int64
		Name        string
		CronSpec    string
		JobType     string
		Priority    int
		MaxAttempts int
		Enabled     bool
		NextRunAt   sql.NullTime
	}

	err = sqlDB.QueryRow(`
		SELECT id, name, cron_spec, job_type, priority, max_attempts, enabled, next_run_at
		FROM schedules
		WHERE name = ?
	`, scheduleNamePeriodicLibraryScan).Scan(
		&schedule.ID,
		&schedule.Name,
		&schedule.CronSpec,
		&schedule.JobType,
		&schedule.Priority,
		&schedule.MaxAttempts,
		&schedule.Enabled,
		&schedule.NextRunAt,
	)
	require.NoError(t, err)

	// Assert schedule fields
	assert.Equal(t, scheduleNamePeriodicLibraryScan, schedule.Name)
	assert.Equal(t, "0 * * * * *", schedule.CronSpec)
	assert.Equal(t, shared.JobTypeScanLibrary, schedule.JobType)
	assert.Equal(t, 5, schedule.Priority)
	assert.Equal(t, 3, schedule.MaxAttempts) // From JobConfig
	assert.True(t, schedule.Enabled)
	assert.True(t, schedule.NextRunAt.Valid, "next_run_at should be set")

	// Verify next_run_at is approximately NOW (within 5 seconds)
	now := time.Now()
	diff := now.Sub(schedule.NextRunAt.Time).Abs()
	assert.Less(t, diff, 5*time.Second, "next_run_at should be set to approximately NOW")
}

// TestInitStartupSchedules_Integration_RunTwice tests idempotency
// Running initialization twice should update the schedule (upsert behavior)
func TestInitStartupSchedules_Integration_RunTwice(t *testing.T) {
	ctx := context.Background()
	adapters, useCases, sqlDB := setupTestDependencies(t)

	cfg := &config.Config{
		Media: config.MediaConfig{
			LibraryScan: config.LibraryScanConfig{
				Enabled:  true,
				CronSpec: "0 * * * * *",
				Priority: 5,
			},
		},
	}

	// First initialization
	err := initStartupSchedules(ctx, cfg, useCases, adapters)
	require.NoError(t, err)

	// Get the next_run_at from first initialization
	var firstNextRunAt sql.NullTime
	err = sqlDB.QueryRow("SELECT next_run_at FROM schedules WHERE name = ?", scheduleNamePeriodicLibraryScan).Scan(&firstNextRunAt)
	require.NoError(t, err)
	require.True(t, firstNextRunAt.Valid)

	// Wait a bit to ensure time difference
	time.Sleep(100 * time.Millisecond)

	// Second initialization (simulates restart)
	err = initStartupSchedules(ctx, cfg, useCases, adapters)
	require.NoError(t, err)

	// Verify still only one schedule
	var count int
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM schedules WHERE name = ?", scheduleNamePeriodicLibraryScan).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should still have exactly one schedule")

	// Get the next_run_at from second initialization
	var secondNextRunAt sql.NullTime
	err = sqlDB.QueryRow("SELECT next_run_at FROM schedules WHERE name = ?", scheduleNamePeriodicLibraryScan).Scan(&secondNextRunAt)
	require.NoError(t, err)
	require.True(t, secondNextRunAt.Valid)

	// Verify next_run_at was updated (ForceNextRunNow behavior)
	assert.True(t, secondNextRunAt.Time.After(firstNextRunAt.Time),
		"next_run_at should be updated on second initialization due to ForceNextRunNow=true")
}

// TestInitStartupSchedules_Integration_DisabledDoesNotCreateSchedule tests that
// when library scan is disabled, no schedule is created
func TestInitStartupSchedules_Integration_DisabledDoesNotCreateSchedule(t *testing.T) {
	ctx := context.Background()
	adapters, useCases, sqlDB := setupTestDependencies(t)

	cfg := &config.Config{
		Media: config.MediaConfig{
			LibraryScan: config.LibraryScanConfig{
				Enabled:  false,
				CronSpec: "0 * * * * *",
				Priority: 0,
			},
		},
	}

	// Execute initialization
	err := initStartupSchedules(ctx, cfg, useCases, adapters)
	require.NoError(t, err)

	// Verify no schedule was created
	var count int
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM schedules WHERE name = ?", scheduleNamePeriodicLibraryScan).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should not have created any schedule when disabled")
}

// TestInitStartupSchedules_Integration_DifferentCronSpecs tests various cron specifications
func TestInitStartupSchedules_Integration_DifferentCronSpecs(t *testing.T) {
	tests := []struct {
		name     string
		cronSpec string
	}{
		{"every minute", "0 * * * * *"},
		{"every hour", "0 0 * * * *"},
		{"specific time", "0 30 14 * * *"}, // 2:30 PM daily
		{"every 5 minutes", "0 */5 * * * *"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			adapters, useCases, sqlDB := setupTestDependencies(t)

			cfg := &config.Config{
				Media: config.MediaConfig{
					LibraryScan: config.LibraryScanConfig{
						Enabled:  true,
						CronSpec: tt.cronSpec,
						Priority: 0,
					},
				},
			}

			// Execute initialization
			err := initStartupSchedules(ctx, cfg, useCases, adapters)
			require.NoError(t, err)

			// Verify schedule was created with correct cron spec
			var cronSpec string
			err = sqlDB.QueryRow("SELECT cron_spec FROM schedules WHERE name = ?", scheduleNamePeriodicLibraryScan).Scan(&cronSpec)
			require.NoError(t, err)
			assert.Equal(t, tt.cronSpec, cronSpec)
		})
	}
}

// TestInitStartupSchedules_Integration_InvalidCronSpecFailsFast tests fail-fast behavior
func TestInitStartupSchedules_Integration_InvalidCronSpecFailsFast(t *testing.T) {
	ctx := context.Background()
	adapters, useCases, sqlDB := setupTestDependencies(t)

	cfg := &config.Config{
		Media: config.MediaConfig{
			LibraryScan: config.LibraryScanConfig{
				Enabled:  true,
				CronSpec: "invalid cron spec",
				Priority: 0,
			},
		},
	}

	// Execute initialization - should fail
	err := initStartupSchedules(ctx, cfg, useCases, adapters)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron_spec")

	// Verify no schedule was created
	var count int
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM schedules WHERE name = ?", scheduleNamePeriodicLibraryScan).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "should not have created any schedule when cron spec is invalid")
}
