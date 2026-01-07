package app

import (
	"context"
	"fmt"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
)

const scheduleNamePeriodicLibraryScan = "periodic_library_scan"
const scheduleNamePeriodicRebuildExport = "periodic_rebuild_export"
const scheduleNamePeriodicRecommendations = "periodic_recommendations"

// initStartupSchedules initializes scheduled jobs that should run on application startup
// This function is called during application initialization and will fail-fast if
// configuration is invalid (e.g., invalid cron spec)
func initStartupSchedules(ctx context.Context, cfg *config.Config, useCases *UseCases, adapters *Adapters) error {
	logger.Info().Msg("initializing startup schedules")

	// Initialize periodic library scan if enabled
	if cfg.Media.LibraryScan.Enabled {
		if err := initPeriodicLibraryScan(ctx, cfg, useCases, adapters); err != nil {
			return fmt.Errorf("failed to initialize periodic library scan: %w", err)
		}
	} else {
		logger.Info().Msg("periodic library scan is disabled in configuration")
	}

	// Initialize periodic rebuild export if enabled
	if cfg.Rebuild.PeriodicExport.Enabled {
		if err := initPeriodicRebuildExport(ctx, cfg, useCases, adapters); err != nil {
			return fmt.Errorf("failed to initialize periodic rebuild export: %w", err)
		}
	} else {
		logger.Info().Msg("periodic rebuild export is disabled in configuration")
	}

	// Initialize periodic recommendations if enabled
	if cfg.Recommendations.Enabled {
		if err := initPeriodicRecommendations(ctx, cfg, useCases, adapters); err != nil {
			return fmt.Errorf("failed to initialize periodic recommendations: %w", err)
		}
	} else {
		logger.Info().Msg("periodic recommendations is disabled in configuration")
	}

	logger.Info().Msg("startup schedules initialized successfully")
	return nil
}

// initPeriodicLibraryScan initializes the periodic library scan schedule
func initPeriodicLibraryScan(ctx context.Context, cfg *config.Config, useCases *UseCases, adapters *Adapters) error {
	cronSpec := cfg.Media.LibraryScan.CronSpec
	priority := cfg.Media.LibraryScan.Priority

	logger.Info().
		Str("schedule_name", scheduleNamePeriodicLibraryScan).
		Str("cron_spec", cronSpec).
		Int("priority", priority).
		Msg("initializing periodic library scan schedule")

	// Validate cron spec before upserting (fail-fast)
	if _, err := adapters.CronParser.Parse(cronSpec); err != nil {
		return fmt.Errorf("invalid cron_spec '%s': %w", cronSpec, err)
	}

	// Upsert the schedule with ForceNextRunNow=true to ensure it runs immediately
	input := job.UpsertScheduleInput{
		Name:            scheduleNamePeriodicLibraryScan,
		CronSpec:        cronSpec,
		JobType:         shared.JobTypeScanLibrary,
		Priority:        priority,
		Enabled:         true,
		ForceNextRunNow: true, // Always run immediately on startup/restart
		Payload:         nil,  // No payload needed for library scan
	}

	if _, err := useCases.UpsertScheduleUseCase.Execute(ctx, input); err != nil {
		return fmt.Errorf("failed to upsert schedule: %w", err)
	}

	logger.Info().
		Str("schedule_name", scheduleNamePeriodicLibraryScan).
		Msg("periodic library scan schedule initialized successfully")

	return nil
}

// initPeriodicRebuildExport initializes the periodic rebuild export schedule
func initPeriodicRebuildExport(ctx context.Context, cfg *config.Config, useCases *UseCases, adapters *Adapters) error {
	cronSpec := cfg.Rebuild.PeriodicExport.CronSpec
	priority := cfg.Rebuild.PeriodicExport.Priority
	runAtStartup := cfg.Rebuild.PeriodicExport.RunAtStartup

	logger.Info().
		Str("schedule_name", scheduleNamePeriodicRebuildExport).
		Str("cron_spec", cronSpec).
		Int("priority", priority).
		Bool("run_at_startup", runAtStartup).
		Msg("initializing periodic rebuild export schedule")

	// Validate cron spec before upserting (fail-fast)
	if _, err := adapters.CronParser.Parse(cronSpec); err != nil {
		return fmt.Errorf("invalid cron_spec '%s': %w", cronSpec, err)
	}

	// Upsert the schedule
	input := job.UpsertScheduleInput{
		Name:            scheduleNamePeriodicRebuildExport,
		CronSpec:        cronSpec,
		JobType:         shared.JobTypePrepareRebuild,
		Priority:        priority,
		Enabled:         true,
		ForceNextRunNow: runAtStartup,
		Payload:         nil,
	}

	if _, err := useCases.UpsertScheduleUseCase.Execute(ctx, input); err != nil {
		return fmt.Errorf("failed to upsert schedule: %w", err)
	}

	logger.Info().
		Str("schedule_name", scheduleNamePeriodicRebuildExport).
		Msg("periodic rebuild export schedule initialized successfully")

	return nil
}

// initPeriodicRecommendations initializes the periodic recommendation model build schedule
func initPeriodicRecommendations(ctx context.Context, cfg *config.Config, useCases *UseCases, adapters *Adapters) error {
	cronSpec := cfg.Recommendations.CronSpec
	priority := cfg.Recommendations.Priority
	runAtStartup := cfg.Recommendations.RunAtStartup

	logger.Info().
		Str("schedule_name", scheduleNamePeriodicRecommendations).
		Str("cron_spec", cronSpec).
		Int("priority", priority).
		Bool("run_at_startup", runAtStartup).
		Msg("initializing periodic recommendations schedule")

	// Validate cron spec before upserting (fail-fast)
	if _, err := adapters.CronParser.Parse(cronSpec); err != nil {
		return fmt.Errorf("invalid cron_spec '%s': %w", cronSpec, err)
	}

	// Upsert the schedule
	input := job.UpsertScheduleInput{
		Name:            scheduleNamePeriodicRecommendations,
		CronSpec:        cronSpec,
		JobType:         shared.JobTypeBuildRecommendations,
		Priority:        priority,
		Enabled:         true,
		ForceNextRunNow: runAtStartup,
		Payload:         nil,
	}

	if _, err := useCases.UpsertScheduleUseCase.Execute(ctx, input); err != nil {
		return fmt.Errorf("failed to upsert schedule: %w", err)
	}

	logger.Info().
		Str("schedule_name", scheduleNamePeriodicRecommendations).
		Msg("periodic recommendations schedule initialized successfully")

	return nil
}
