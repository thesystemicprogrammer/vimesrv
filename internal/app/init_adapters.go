package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/job"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/media"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/repository"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type Adapters struct {
	BackoffStrategy          ports.BackoffStrategy
	CronParser               ports.CronParser
	HandlerRegistry          *job.HandlerRegistry
	JobRepository            ports.JobRepository
	ScheduleRepository       ports.ScheduleRepository
	FileHasher               ports.FileHasher
	FFProbeService           ports.FFProbeService
	FileSystemService        ports.FileSystemService
	MediaRepository          ports.MediaRepository
	AudioStreamRepository    ports.AudioStreamRepository
	SubtitleStreamRepository ports.SubtitleStreamRepository
	TranscodeRepository      ports.TranscodeRepository
	Transcoder               ports.Transcoder
}

func initAdapters(config *config.Config, db *database.DB) *Adapters {
	return &Adapters{
		CronParser:               job.NewRobfigCronParser(),
		BackoffStrategy:          job.NewExponentialBackoff(config.Job.BackoffBaseSeconds, config.Job.BackoffMaxSeconds),
		HandlerRegistry:          job.NewHandlerRegistry(),
		JobRepository:            repository.NewJobRepository(db),
		ScheduleRepository:       repository.NewScheduleRepository(db),
		FileHasher:               media.NewBlake2bHasher(),
		FFProbeService:           media.NewFFProbeAdapter(config.Media.FFProbeTimeoutSeconds),
		FileSystemService:        media.NewOSFileSystem(),
		MediaRepository:          repository.NewMediaRepository(db),
		AudioStreamRepository:    repository.NewAudioStreamRepository(db),
		SubtitleStreamRepository: repository.NewSubtitleStreamRepository(db),
		TranscodeRepository:      repository.NewTranscodeRepository(db),
		Transcoder:               media.NewFFmpegTranscoder(config.Media.TranscodeTimeoutSeconds),
	}
}
