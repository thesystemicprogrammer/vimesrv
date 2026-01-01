package job

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	usecasejob "github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type JobManagerInput struct {
	Config                  config.JobConfig
	ProcessNextJobUseCase   *usecasejob.ProcessNextJobUseCase
	SchedulerTickUseCase    *usecasejob.SchedulerTickUseCase
	RecoverStuckJobsUseCase *usecasejob.RecoverStuckJobsUseCase
	JobRepository           ports.JobRepository
	ScheduleRepository      ports.ScheduleRepository
	Handlers                ports.HandlerResolver
	BackoffStrategy         ports.BackoffStrategy
	Cron                    ports.CronParser
	Clock                   ports.Clock
}

type JobManager struct {
	JobManagerInput
	stopCh     chan struct{}
	cancelFunc context.CancelFunc
	started    int32
	stopped    int32
	wg         sync.WaitGroup
}

func NewJobManager(input JobManagerInput) *JobManager {
	if input.Clock == nil {
		input.Clock = ports.RealClock{}
	}

	return &JobManager{
		JobManagerInput: input,
		stopCh:          make(chan struct{}),
	}
}

func (manager *JobManager) Start() error {
	if !atomic.CompareAndSwapInt32(&manager.started, 0, 1) {
		return fmt.Errorf("manager already started")
	}

	// Create a cancellable context for all workers
	ctx, cancel := context.WithCancel(context.Background())
	manager.cancelFunc = cancel

	// Workers
	for i := 0; i < manager.Config.WorkerCount; i++ {
		id := manager.workerID(i)
		manager.wg.Add(1)
		go func(workerID string) {
			defer manager.wg.Done()
			manager.workerLoop(ctx, workerID)
		}(id)
	}
	// Scheduler
	manager.wg.Add(1)
	go func() {
		defer manager.wg.Done()
		manager.schedulerLoop(ctx)
	}()
	// Stuck job recovery
	manager.wg.Add(1)
	go func() {
		defer manager.wg.Done()
		manager.stuckJobRecoveryLoop(ctx)
	}()

	logger.Info().Int("workerCount", manager.Config.WorkerCount).Msg("job manager started")
	return nil
}

func (manager *JobManager) Stop(ctx context.Context) error {
	if atomic.CompareAndSwapInt32(&manager.stopped, 0, 1) {
		// Cancel the worker context to interrupt ongoing jobs
		if manager.cancelFunc != nil {
			manager.cancelFunc()
		}
		close(manager.stopCh)
	}
	done := make(chan struct{})
	go func() { manager.wg.Wait(); close(done) }()
	select {
	case <-done:
		logger.Info().Msg("job manager stopped")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (manager *JobManager) workerLoop(ctx context.Context, workerID string) {
	duration := time.Duration(manager.Config.PollingIntervalInSeconds) * time.Second
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-manager.stopCh:
			return
		case <-ctx.Done():
			logger.Info().Str("workerID", workerID).Msg("worker context cancelled, shutting down")
			return
		default:
		}
		found, err := manager.ProcessNextJobUseCase.Execute(ctx, workerID)
		if err != nil {
			logger.Error().Err(err).Str("workerID", workerID).Msg("Use Case Processing Next Job failed")
			select {
			case <-time.After(shared.ErrorBackoffDuration):
			case <-manager.stopCh:
				return
			case <-ctx.Done():
				return
			}
			continue
		}
		if !found {
			select {
			case <-ticker.C:
			case <-manager.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}
}

func (manager *JobManager) schedulerLoop(ctx context.Context) {
	duration := time.Duration(manager.Config.SchedulerIntervalInSeconds) * time.Second
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-manager.stopCh:
			return
		case <-ctx.Done():
			logger.Info().Msg("scheduler context cancelled, shutting down")
			return
		case <-ticker.C:
			_ = manager.SchedulerTickUseCase.Execute(ctx)
		}
	}
}

func (manager *JobManager) stuckJobRecoveryLoop(ctx context.Context) {
	duration := time.Duration(manager.Config.StuckJobCheckIntervalMinutes) * time.Minute
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-manager.stopCh:
			return
		case <-ctx.Done():
			logger.Info().Msg("stuck job recovery context cancelled, shutting down")
			return
		case <-ticker.C:
			_ = manager.RecoverStuckJobsUseCase.Execute(ctx)
		}
	}
}

func (manager *JobManager) workerID(i int) string {
	host, err := os.Hostname()
	if err != nil {
		logger.Warn().Err(err).Msg("failed to get hostname, using fallback")
		host = "unknown-host"
	}
	return fmt.Sprintf("%s-%d-%d", host, os.Getpid(), i)
}
