// Package scheduler provides the scheduler implementation.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog/log"

	"github.com/sentoz/yc-sheduler/internal/config"
	"github.com/sentoz/yc-sheduler/internal/executor"
	"github.com/sentoz/yc-sheduler/internal/metrics"
	"github.com/sentoz/yc-sheduler/internal/resource"
	"github.com/sentoz/yc-sheduler/internal/schedule"
)

// Interface defines the interface for scheduler operations.
type Interface interface {
	AddJob(def gocron.JobDefinition, name string, fn func(), timezone string) error
	Start(ctx context.Context) error
	Stop()
	AddOneTimeJob(name string, fn func()) error
	RegisterSchedules(stateChecker resource.StateChecker, operator resource.Operator, cfg *config.Config, dryRun bool, m *metrics.Metrics) error
}

// Scheduler wraps gocron.Scheduler and provides a higher-level API
// tailored for yc-scheduler configuration.
type Scheduler struct {
	s  gocron.Scheduler
	mu sync.Mutex
}

const managedScheduleTag = "managed_schedule"

// Ensure Scheduler implements Interface.
var _ Interface = (*Scheduler)(nil)

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

	log.Info().
		Str("timezone", location.String()).
		Int("max_concurrent_jobs", maxConcurrentJobs).
		Msg("Scheduler initialized")

	return &Scheduler{s: s}, nil
}

// AddJob registers a new job in the underlying scheduler with the given
// definition and name.
// The job function is a simple func() without parameters to avoid reflection
// mismatches with gocron's task parameter handling.
// The timezone parameter is ignored as gocron v2 doesn't support per-job timezones.
// All jobs use the scheduler's timezone (set during initialization).
func (s *Scheduler) AddJob(def gocron.JobDefinition, name string, fn func(), timezone string) error {
	if s == nil || s.s == nil {
		return fmt.Errorf("scheduler: not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.addJobUnlocked(def, name, fn)
}

// Start starts the scheduler and blocks until the context is canceled.
func (s *Scheduler) Start(ctx context.Context) error {
	if s == nil || s.s == nil {
		return fmt.Errorf("scheduler: not initialized")
	}

	s.s.Start()

	log.Info().Msg("Scheduler event loop started")

	<-ctx.Done()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.s.Shutdown(); err != nil {
		log.Warn().Err(err).Msg("Scheduler shutdown error")
	}

	log.Info().Msg("Scheduler shutdown completed")

	return nil
}

// Stop stops the scheduler gracefully without waiting for the context.
func (s *Scheduler) Stop() {
	if s == nil || s.s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.s.Shutdown(); err != nil {
		log.Warn().Err(err).Msg("Scheduler stop error")
	}
}

// AddOneTimeJob adds a one-time job that will execute immediately.
// The job function is a simple func() without parameters.
func (s *Scheduler) AddOneTimeJob(name string, fn func()) error {
	if s == nil || s.s == nil {
		return fmt.Errorf("scheduler: not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.s.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
		gocron.NewTask(fn),
		gocron.WithName(name),
	)
	if err != nil {
		return fmt.Errorf("scheduler: add one-time job %q: %w", name, err)
	}

	log.Info().
		Str("job_name", name).
		Msg("One-time job registered")

	return nil
}

// RegisterSchedules registers all schedules from the configuration.
// It iterates through all schedules and registers start/stop actions as jobs.
// If m is nil, metrics will not be recorded.
func (s *Scheduler) RegisterSchedules(stateChecker resource.StateChecker, operator resource.Operator, cfg *config.Config, dryRun bool, m *metrics.Metrics) error {
	if s == nil || s.s == nil {
		return fmt.Errorf("scheduler: not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sch := range cfg.Schedules {
		if err := registerScheduleUnlocked(s, stateChecker, operator, sch, dryRun, m); err != nil {
			return err
		}
	}
	return nil
}

// ReplaceSchedules replaces all regular scheduled jobs with a new set from
// manifests. In-flight jobs are not interrupted.
func (s *Scheduler) ReplaceSchedules(stateChecker resource.StateChecker, operator resource.Operator, schedules []config.Schedule, dryRun bool, m *metrics.Metrics) error {
	if s == nil || s.s == nil {
		return fmt.Errorf("scheduler: not initialized")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.s.RemoveByTags(managedScheduleTag)

	for _, sch := range schedules {
		if err := registerScheduleUnlocked(s, stateChecker, operator, sch, dryRun, m); err != nil {
			return err
		}
	}

	log.Info().
		Int("jobs", len(s.s.Jobs())).
		Msg("Scheduler jobs reloaded")

	return nil
}

func registerScheduleUnlocked(s *Scheduler, stateChecker resource.StateChecker, operator resource.Operator, sch config.Schedule, dryRun bool, m *metrics.Metrics) error {
	if sch.Actions.Start != nil && sch.Actions.Start.Enabled {
		def, err := ScheduleToJobDefinition(sch, sch.Actions.Start)
		if err != nil {
			return fmt.Errorf("register schedule %q start action: %w", sch.Name, err)
		}
		name := sch.Name + ":start"
		if err := s.addJobUnlocked(def, name, executor.Make(stateChecker, operator, sch, "start", dryRun, m)); err != nil {
			return err
		}
	}
	if sch.Actions.Stop != nil && sch.Actions.Stop.Enabled {
		def, err := ScheduleToJobDefinition(sch, sch.Actions.Stop)
		if err != nil {
			return fmt.Errorf("register schedule %q stop action: %w", sch.Name, err)
		}
		name := sch.Name + ":stop"
		if err := s.addJobUnlocked(def, name, executor.Make(stateChecker, operator, sch, "stop", dryRun, m)); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scheduler) addJobUnlocked(def gocron.JobDefinition, name string, fn func()) error {
	if s == nil || s.s == nil {
		return fmt.Errorf("scheduler: not initialized")
	}

	_, err := s.s.NewJob(def, gocron.NewTask(fn), gocron.WithName(name), gocron.WithTags(managedScheduleTag))
	if err != nil {
		return fmt.Errorf("scheduler: add job %q: %w", name, err)
	}

	log.Debug().
		Str("job_name", name).
		Msg("Scheduler job registered")

	return nil
}

// ScheduleToJobDefinition converts a configuration schedule and action into a
// gocron.JobDefinition. The action config contains the schedule-specific parameters.
func ScheduleToJobDefinition(sch config.Schedule, action *config.ActionConfig) (gocron.JobDefinition, error) {
	switch sch.Type {
	case "cron":
		if action.Crontab.String() == "" {
			return nil, fmt.Errorf("scheduler: cron schedule %q missing crontab in action", sch.Name)
		}
		return gocron.CronJob(action.Crontab.String(), false), nil
	case "daily":
		if action.Time == "" {
			return nil, fmt.Errorf("scheduler: daily schedule %q missing time in action", sch.Name)
		}
		at, err := schedule.ParseTime(config.Time(action.Time))
		if err != nil {
			return nil, fmt.Errorf("scheduler: daily schedule %q: %w", sch.Name, err)
		}
		return gocron.DailyJob(1, at), nil
	case "weekly":
		if action.Time == "" {
			return nil, fmt.Errorf("scheduler: weekly schedule %q missing time in action", sch.Name)
		}
		if action.Day < 0 || action.Day > 6 {
			return nil, fmt.Errorf("scheduler: weekly schedule %q missing or invalid day in action (got %d, expected 0-6)", sch.Name, action.Day)
		}
		at, err := schedule.ParseTime(config.Time(action.Time))
		if err != nil {
			return nil, fmt.Errorf("scheduler: weekly schedule %q: %w", sch.Name, err)
		}
		weekday, err := schedule.ParseWeekday(action.Day)
		if err != nil {
			return nil, fmt.Errorf("scheduler: weekly schedule %q: %w", sch.Name, err)
		}
		return gocron.WeeklyJob(1, weekday, at), nil
	case "monthly":
		if action.Time == "" {
			return nil, fmt.Errorf("scheduler: monthly schedule %q missing time in action", sch.Name)
		}
		if action.Day < 1 || action.Day > 31 {
			return nil, fmt.Errorf("scheduler: monthly schedule %q missing or invalid day in action (got %d, expected 1-31)", sch.Name, action.Day)
		}
		at, err := schedule.ParseTime(config.Time(action.Time))
		if err != nil {
			return nil, fmt.Errorf("scheduler: monthly schedule %q: %w", sch.Name, err)
		}
		day, err := schedule.ParseDayOfMonth(action.Day)
		if err != nil {
			return nil, fmt.Errorf("scheduler: monthly schedule %q: %w", sch.Name, err)
		}
		return gocron.MonthlyJob(1, gocron.NewDaysOfTheMonth(day), at), nil
	default:
		return nil, fmt.Errorf("scheduler: unknown schedule type %q", sch.Type)
	}
}
