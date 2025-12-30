package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/library"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/repository"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type Adapters struct {
	BackoffStrategy       ports.BackoffStrategy
	CronParser            ports.CronParser
	HandlerRegistry       *job.HandlerRegistry
	ScanLibraryRepository ports.ScanLibraryRepository
	JobRepository         ports.JobRepository
	ScheduleRepository    ports.ScheduleRepository
}

func initAdapters(config *config.Config, db *database.DB) *Adapters {
	return &Adapters{
		CronParser:            job.NewRobfigCronParser(),
		BackoffStrategy:       job.NewExponentialBackoff(config.Job.BackoffBaseSeconds, config.Job.BackoffMaxSeconds),
		HandlerRegistry:       job.NewHandlerRegistry(),
		ScanLibraryRepository: library.NewScanLibraryScanRepository(),
		JobRepository:         repository.NewJobRepository(db),
		ScheduleRepository:    repository.NewScheduleRepository(db),
	}
}
