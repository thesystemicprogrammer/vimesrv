package job

import (
	"time"

	"github.com/robfig/cron/v3"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type RobfigCronParser struct {
	parser cron.Parser
}

func NewRobfigCronParser() *RobfigCronParser {
	return &RobfigCronParser{
		parser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
	}
}

func (parser *RobfigCronParser) Parse(spec string) (ports.CronSchedule, error) {
	s, err := parser.parser.Parse(spec)
	if err != nil {
		return nil, err
	}
	return robfigSchedule{s: s}, nil
}

type robfigSchedule struct {
	s cron.Schedule
}

func (r robfigSchedule) Next(t time.Time) time.Time {
	return r.s.Next(t)
}
