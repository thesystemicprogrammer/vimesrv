package job

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type JobManagerInput struct {
	Config                config.JobConfig
	ProcessNextJobUseCase usecase.ProcessNextJobUseCase
	SchedulerTickUseCase  usecase.SchedulerTickUseCase
	JobRepository         ports.JobRepository
	ScheduleRepository    ports.ScheduleRepository
	Handlers              ports.HandlerResolver
	BackoffStrategy       ports.BackoffStrategy
	Cron                  ports.CronParser
	Clock                 ports.Clock
}

type JobManager struct {
	Config                config.JobConfig
	ProcessNextJobUseCase usecase.ProcessNextJobUseCase
	SchedulerTickUseCase  usecase.SchedulerTickUseCase
	JobRepository         ports.JobRepository
	ScheduleRepository    ports.ScheduleRepository
	Handlers              ports.HandlerResolver
	BackoffStrategy       ports.BackoffStrategy
	Cron                  ports.CronParser
	Clock                 ports.Clock
	stopCh                chan struct{}
	started               int32
	wg                    sync.WaitGroup
}

func NewJobManager(jobManagerInput JobManagerInput) *JobManager {
	if jobManagerInput.Clock == nil {
		jobManagerInput.Clock = ports.RealClock{}
	}

	return &JobManager{
		Config:                jobManagerInput.Config,
		ProcessNextJobUseCase: jobManagerInput.ProcessNextJobUseCase,
		SchedulerTickUseCase:  jobManagerInput.SchedulerTickUseCase,
		JobRepository:         jobManagerInput.JobRepository,
		ScheduleRepository:    jobManagerInput.ScheduleRepository,
		Handlers:              jobManagerInput.Handlers,
		BackoffStrategy:       jobManagerInput.BackoffStrategy,
		Cron:                  jobManagerInput.Cron,
		Clock:                 jobManagerInput.Clock,
		stopCh:                make(chan struct{}),
	}
}

func (manager *JobManager) Start() error {
	if !atomic.CompareAndSwapInt32(&manager.started, 0, 1) {
		return fmt.Errorf("manager already started")
	}

	// Workers
	for i := 0; i < manager.Config.WorkerCount; i++ {
		id := manager.workerID(i)
		manager.wg.Add(1)
		go func(workerID string) {
			defer manager.wg.Done()
			manager.workerLoop(workerID)
		}(id)
	}
	// Scheduler
	manager.wg.Go(func() {
		manager.schedulerLoop()
	})

	logger.Info().Int("workerCount", manager.Config.WorkerCount).Msg("job manager started")
	return nil
}

func (manager *JobManager) Stop(ctx context.Context) error {
	select {
	case <-manager.stopCh:
	default:
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

func (manager *JobManager) workerLoop(workerID string) {
	duration := time.Duration(manager.Config.PollingIntervalInSeconds) * time.Second
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	ctx := context.Background()

	for {
		select {
		case <-manager.stopCh:
			return
		default:
		}
		found, err := manager.ProcessNextJobUseCase.Execute(ctx, workerID)
		if err != nil {
			logger.Error().Err(err).Str("workerID", workerID).Msg("Use Case Processing Next Job failed")
			select {
			case <-time.After(500 * time.Millisecond):
			case <-manager.stopCh:
				return
			}
			continue
		}
		if !found {
			select {
			case <-ticker.C:
			case <-manager.stopCh:
				return
			}
		}
	}
}

func (manager *JobManager) schedulerLoop() {
	duration := time.Duration(manager.Config.SchedulerIntervalInSeconds) * time.Second
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	ctx := context.Background()

	for {
		select {
		case <-manager.stopCh:
			return
		case <-ticker.C:
			_ = manager.SchedulerTickUseCase.Execute(ctx)
		}
	}
}

func (manager *JobManager) workerID(i int) string {
	host, _ := os.Hostname()
	return fmt.Sprintf("%s-%d-%d", host, os.Getpid(), i)
}
