package job

import (
	"time"

	"github.com/robfig/cron/v3"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

type RobfigCronParser struct {
	parser cron.Parser
}

// NewRobfigCronParser creates a cron parser that supports second-level precision.
// This parser is used for all cron expressions in the system, including production use cases
// like periodic library scans that need sub-minute intervals.
//
// Format: "second minute hour day month weekday" (6 fields required)
// Examples:
//   - "*/30 * * * * *" = every 30 seconds
//   - "0 * * * * *" = every minute (at :00 seconds)
//   - "0 */5 * * * *" = every 5 minutes
//   - "0 0 3 * * *" = daily at 3:00 AM
func NewRobfigCronParser() *RobfigCronParser {
	return &RobfigCronParser{
		parser: cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
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
