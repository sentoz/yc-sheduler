package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sentoz/yc-sheduler/internal/config"
	"github.com/sentoz/yc-sheduler/internal/executor"
	"github.com/sentoz/yc-sheduler/internal/metrics"
	"github.com/sentoz/yc-sheduler/internal/resource"
	"github.com/sentoz/yc-sheduler/internal/schedule"
	"github.com/sentoz/yc-sheduler/internal/scheduler"
)

// Interface defines the interface for validator operations.
type Interface interface {
	Start(ctx context.Context, interval time.Duration)
}

// Validator periodically inspects resources and logs their state.
// If a discrepancy is found, it creates a one-time job to fix it.
type Validator struct {
	stateChecker resource.StateChecker
	operator     resource.Operator
	cfg          *config.Config
	scheduler    scheduler.Interface
	metrics      *metrics.Metrics
	dryRun       bool
}

// Ensure Validator implements Interface.
var _ Interface = (*Validator)(nil)

// New creates a new Validator instance.
// If m is nil, metrics will not be recorded.
func New(stateChecker resource.StateChecker, operator resource.Operator, cfg *config.Config, sched scheduler.Interface, m *metrics.Metrics, dryRun bool) *Validator {
	v := &Validator{
		stateChecker: stateChecker,
		operator:     operator,
		cfg:          cfg,
		scheduler:    sched,
		metrics:      m,
		dryRun:       dryRun,
	}
	log.Info().
		Int("schedules", len(cfg.Schedules)).
		Msg("Validator initialized")
	return v
}

// Start runs validation in the background until the context is canceled.
func (v *Validator) Start(ctx context.Context, interval time.Duration) {
	if v == nil || v.stateChecker == nil || v.cfg == nil {
		return
	}

	go func() {
		log.Info().
			Dur("interval", interval).
			Msg("Validator loop started")

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Validator loop stopped")
				return
			case <-ticker.C:
				v.runOnce(ctx)
			}
		}
	}()
}

func (v *Validator) runOnce(ctx context.Context) {
	now := time.Now()

	for _, sch := range v.cfg.Schedules {
		log.Trace().
			Str("schedule", sch.Name).
			Str("resource_type", sch.Resource.Type).
			Str("resource_id", sch.Resource.ID).
			Time("now", now).
			Msg("Validator is about to check resource state")

		actualState, isTransitional, err := v.stateChecker.GetState(ctx, sch.Resource)
		if err != nil {
			log.Warn().Err(err).
				Str("schedule", sch.Name).
				Str("resource_type", sch.Resource.Type).
				Str("resource_id", sch.Resource.ID).
				Msg("Failed to get actual resource state")
			continue
		}

		// If resource is in transitional state, skip validation and wait for stable state
		if isTransitional {
			log.Debug().
				Str("schedule", sch.Name).
				Str("resource_type", sch.Resource.Type).
				Str("resource_id", sch.Resource.ID).
				Str("current_state", actualState).
				Msg("Resource is in transitional state, deferring validation until stable")
			continue
		}

		// Determine expected state based on schedule and current time
		expectedState, expectedAction := v.determineExpectedState(sch, now)
		if expectedAction == "" {
			log.Debug().
				Str("schedule", sch.Name).
				Str("resource_type", sch.Resource.Type).
				Str("resource_id", sch.Resource.ID).
				Str("actual_state", actualState).
				Msg("No corrective action needed")
			continue
		}

		if actualState != expectedState {
			log.Warn().
				Str("schedule", sch.Name).
				Str("resource_type", sch.Resource.Type).
				Str("resource_id", sch.Resource.ID).
				Str("expected_state", expectedState).
				Str("actual_state", actualState).
				Str("corrective_action", expectedAction).
				Msg("State mismatch detected, creating corrective job")

			jobName := sch.Name + ":validator:" + expectedAction
			if err := v.scheduler.AddOneTimeJob(jobName, executor.Make(v.stateChecker, v.operator, sch, expectedAction, v.dryRun, v.metrics)); err != nil {
				log.Error().Err(err).
					Str("schedule", sch.Name).
					Str("resource_type", sch.Resource.Type).
					Str("resource_id", sch.Resource.ID).
					Str("action", expectedAction).
					Msg("Failed to create corrective job")
			} else {
				if v.metrics != nil {
					v.metrics.IncValidatorCorrection(sch.Resource.Type, expectedAction)
				}
				log.Info().
					Str("schedule", sch.Name).
					Str("resource_type", sch.Resource.Type).
					Str("resource_id", sch.Resource.ID).
					Str("action", expectedAction).
					Msg("Corrective job created")
			}
		} else {
			log.Debug().
				Str("schedule", sch.Name).
				Str("resource_type", sch.Resource.Type).
				Str("resource_id", sch.Resource.ID).
				Str("state", actualState).
				Msg("Resource state matches expected state")
		}
	}
}

// determineExpectedState determines the expected state and corrective action
// based on the schedule configuration and current time.
// Returns (expectedState, correctiveAction).
// expectedState: "running" or "stopped"
// correctiveAction: "start", "stop", or "" if no action needed.
func (v *Validator) determineExpectedState(sch config.Schedule, now time.Time) (string, string) {
	hasStart := sch.Actions.Start != nil && sch.Actions.Start.Enabled
	hasStop := sch.Actions.Stop != nil && sch.Actions.Stop.Enabled

	if hasStart && !hasStop {
		// Only start is enabled, expect running
		return "running", "start"
	}

	if hasStop && !hasStart {
		// Only stop is enabled, expect stopped
		return "stopped", "stop"
	}

	if hasStart && hasStop {
		// Both enabled: determine which action should have occurred last
		// by comparing the last execution times of start and stop actions.
		location := time.Local
		if v.cfg.Timezone.String() != "" {
			loc, err := time.LoadLocation(v.cfg.Timezone.String())
			if err == nil {
				location = loc
			}
		}
		nowInTZ := now.In(location)

		lastStartTime, err := v.getLastExecutionTime(sch, sch.Actions.Start, nowInTZ, location)
		if err != nil {
			log.Debug().Err(err).
				Str("schedule", sch.Name).
				Msg("Failed to calculate last start time, defaulting to running")
			return "running", "start"
		}

		lastStopTime, err := v.getLastExecutionTime(sch, sch.Actions.Stop, nowInTZ, location)
		if err != nil {
			log.Debug().Err(err).
				Str("schedule", sch.Name).
				Msg("Failed to calculate last stop time, defaulting to stopped")
			return "stopped", "stop"
		}

		// If last start happened after last stop, resource should be running
		// If last stop happened after last start, resource should be stopped
		if lastStartTime.After(lastStopTime) {
			return "running", "start"
		}
		return "stopped", "stop"
	}

	// No actions enabled
	return "", ""
}

// getLastExecutionTime calculates the last execution time of an action before the given time.
// Returns the last execution time or an error if calculation fails.
func (v *Validator) getLastExecutionTime(sch config.Schedule, action *config.ActionConfig, now time.Time, location *time.Location) (time.Time, error) {
	switch sch.Type {
	case "daily":
		if action.Time == "" {
			return time.Time{}, fmt.Errorf("daily schedule missing time")
		}
		return schedule.GetLastDailyTime(action.Time, now, location)
	case "weekly":
		if action.Time == "" {
			return time.Time{}, fmt.Errorf("weekly schedule missing time")
		}
		if action.Day < 0 || action.Day > 6 {
			return time.Time{}, fmt.Errorf("weekly schedule invalid day: %d", action.Day)
		}
		return schedule.GetLastWeeklyTime(action.Time, action.Day, now, location)
	case "monthly":
		if action.Time == "" {
			return time.Time{}, fmt.Errorf("monthly schedule missing time")
		}
		if action.Day < 1 || action.Day > 31 {
			return time.Time{}, fmt.Errorf("monthly schedule invalid day: %d", action.Day)
		}
		return schedule.GetLastMonthlyTime(action.Time, action.Day, now, location)
	case "cron":
		if action.Crontab.String() == "" {
			return time.Time{}, fmt.Errorf("cron schedule missing crontab")
		}
		return schedule.GetLastCronTime(action.Crontab.String(), now)
	default:
		return time.Time{}, fmt.Errorf("unknown schedule type: %s", sch.Type)
	}
}
