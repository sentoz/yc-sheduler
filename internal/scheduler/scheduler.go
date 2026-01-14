// Package scheduler provides the scheduler implementation.
package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"

	pkgconfig "github.com/woozymasta/yc-scheduler/pkg/config"
)

// Scheduler wraps gocron.Scheduler and provides a higher-level API
// tailored for yc-scheduler configuration.
type Scheduler struct {
	s gocron.Scheduler
}

// New creates a new Scheduler configured with the provided timezone and
// concurrency limit. If timezone is empty, the local system timezone is
// used.
func New(timezone string, maxConcurrentJobs int) (*Scheduler, error) {
	location := time.Local
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			return nil, fmt.Errorf("scheduler: load location %q: %w", timezone, err)
		}
		location = loc
	}

	opts := []gocron.SchedulerOption{
		gocron.WithLocation(location),
	}
	if maxConcurrentJobs > 0 {
		opts = append(opts, gocron.WithLimitConcurrentJobs(uint(maxConcurrentJobs), gocron.LimitModeWait))
	}

	s, err := gocron.NewScheduler(opts...)
	if err != nil {
		return nil, fmt.Errorf("scheduler: new: %w", err)
	}

	return &Scheduler{s: s}, nil
}

// AddJob registers a new job in the underlying scheduler with the given
// definition and name.
func (s *Scheduler) AddJob(def gocron.JobDefinition, name string, fn func(context.Context)) error {
	if s == nil || s.s == nil {
		return fmt.Errorf("scheduler: not initialized")
	}

	_, err := s.s.NewJob(def, gocron.NewTask(fn), gocron.WithName(name))
	if err != nil {
		return fmt.Errorf("scheduler: add job %q: %w", name, err)
	}

	return nil
}

// Start starts the scheduler and blocks until the context is canceled.
func (s *Scheduler) Start(ctx context.Context) error {
	if s == nil || s.s == nil {
		return fmt.Errorf("scheduler: not initialized")
	}

	s.s.Start()

	<-ctx.Done()
	s.s.Shutdown()

	return nil
}

// Stop stops the scheduler gracefully without waiting for the context.
func (s *Scheduler) Stop() {
	if s == nil || s.s == nil {
		return
	}
	s.s.Shutdown()
}

// ScheduleToJobDefinition converts a configuration schedule into a
// gocron.JobDefinition.
func ScheduleToJobDefinition(sch pkgconfig.Schedule) (gocron.JobDefinition, error) {
	switch sch.Type {
	case "cron":
		if sch.CronJob == nil {
			return nil, fmt.Errorf("scheduler: cron schedule %q missing cron_job", sch.Name)
		}
		return gocron.CronJob(sch.CronJob.Crontab.String(), false), nil
	case "daily":
		if sch.DailyJob == nil {
			return nil, fmt.Errorf("scheduler: daily schedule %q missing daily_job", sch.Name)
		}
		at, err := parseTime(sch.DailyJob.Time)
		if err != nil {
			return nil, fmt.Errorf("scheduler: daily schedule %q: %w", sch.Name, err)
		}
		return gocron.DailyJob(1, at), nil
	case "weekly":
		if sch.WeeklyJob == nil {
			return nil, fmt.Errorf("scheduler: weekly schedule %q missing weekly_job", sch.Name)
		}
		at, err := parseTime(sch.WeeklyJob.Time)
		if err != nil {
			return nil, fmt.Errorf("scheduler: weekly schedule %q: %w", sch.Name, err)
		}
		weekday, err := parseWeekday(sch.WeeklyJob.Day)
		if err != nil {
			return nil, fmt.Errorf("scheduler: weekly schedule %q: %w", sch.Name, err)
		}
		return gocron.WeeklyJob(1, weekday, at), nil
	case "monthly":
		if sch.MonthlyJob == nil {
			return nil, fmt.Errorf("scheduler: monthly schedule %q missing monthly_job", sch.Name)
		}
		at, err := parseTime(sch.MonthlyJob.Time)
		if err != nil {
			return nil, fmt.Errorf("scheduler: monthly schedule %q: %w", sch.Name, err)
		}
		day, err := parseDayOfMonth(sch.MonthlyJob.Day)
		if err != nil {
			return nil, fmt.Errorf("scheduler: monthly schedule %q: %w", sch.Name, err)
		}
		return gocron.MonthlyJob(1, gocron.NewDaysOfTheMonth(day), at), nil
	case "duration":
		if sch.DurationJob == nil {
			return nil, fmt.Errorf("scheduler: duration schedule %q missing duration_job", sch.Name)
		}
		return gocron.DurationJob(sch.DurationJob.Duration.Std()), nil
	case "one-time":
		if sch.OneTimeJob == nil {
			return nil, fmt.Errorf("scheduler: one-time schedule %q missing one_time_job", sch.Name)
		}
		t, err := sch.OneTimeJob.Time.Time()
		if err != nil {
			return nil, fmt.Errorf("scheduler: one-time schedule %q: %w", sch.Name, err)
		}
		return gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(t)), nil
	default:
		return nil, fmt.Errorf("scheduler: unknown schedule type %q", sch.Type)
	}
}
