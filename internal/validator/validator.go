package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	computepb "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	k8spb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"

	"github.com/woozymasta/yc-scheduler/internal/executor"
	"github.com/woozymasta/yc-scheduler/internal/metrics"
	"github.com/woozymasta/yc-scheduler/internal/scheduler"
	"github.com/woozymasta/yc-scheduler/internal/yc"
	pkgconfig "github.com/woozymasta/yc-scheduler/pkg/config"
)

// Validator periodically inspects resources and logs their state.
// If a discrepancy is found, it creates a one-time job to fix it.
type Validator struct {
	client    *yc.Client
	cfg       *pkgconfig.Config
	scheduler *scheduler.Scheduler
	dryRun    bool
}

// New creates a new Validator instance.
func New(client *yc.Client, cfg *pkgconfig.Config, sched *scheduler.Scheduler, dryRun bool) *Validator {
	v := &Validator{
		client:    client,
		cfg:       cfg,
		scheduler: sched,
		dryRun:    dryRun,
	}
	log.Info().
		Int("schedules", len(cfg.Schedules)).
		Msg("Validator initialized")
	return v
}

// Start runs validation in the background until the context is canceled.
func (v *Validator) Start(ctx context.Context, interval time.Duration) {
	if v == nil || v.client == nil || v.cfg == nil {
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
		// Skip validation for duration-based schedules as they don't have
		// a predictable expected state based on time
		if sch.Type == "duration" {
			log.Trace().
				Str("schedule", sch.Name).
				Str("resource_type", sch.Resource.Type).
				Str("resource_id", sch.Resource.ID).
				Msg("Skipping validation for duration-based schedule")
			continue
		}

		log.Trace().
			Str("schedule", sch.Name).
			Str("resource_type", sch.Resource.Type).
			Str("resource_id", sch.Resource.ID).
			Time("now", now).
			Msg("Validator is about to check resource state")

		actualState, isTransitional, err := v.getActualState(ctx, sch.Resource)
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

		// Determine expected state based on schedule and current actual state
		expectedState, expectedAction := v.determineExpectedState(sch, now, actualState)
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
			if err := v.scheduler.AddOneTimeJob(jobName, executor.Make(v.client, sch, expectedAction, v.dryRun)); err != nil {
				log.Error().Err(err).
					Str("schedule", sch.Name).
					Str("resource_type", sch.Resource.Type).
					Str("resource_id", sch.Resource.ID).
					Str("action", expectedAction).
					Msg("Failed to create corrective job")
			} else {
				metrics.IncValidatorCorrection(sch.Resource.Type, expectedAction)
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
// based on the schedule configuration, current time, and actual resource state.
// Returns (expectedState, correctiveAction).
// expectedState: "running" or "stopped"
// correctiveAction: "start", "stop", or "" if no action needed.
func (v *Validator) determineExpectedState(sch pkgconfig.Schedule, now time.Time, actualState string) (string, string) {
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
func (v *Validator) getLastExecutionTime(sch pkgconfig.Schedule, action *pkgconfig.ActionConfig, now time.Time, location *time.Location) (time.Time, error) {
	switch sch.Type {
	case "daily":
		if action.Time == "" {
			return time.Time{}, fmt.Errorf("daily schedule missing time")
		}
		return v.getLastDailyTime(action.Time, now, location)
	case "weekly":
		if action.Time == "" {
			return time.Time{}, fmt.Errorf("weekly schedule missing time")
		}
		if action.Day < 0 || action.Day > 6 {
			return time.Time{}, fmt.Errorf("weekly schedule invalid day: %d", action.Day)
		}
		return v.getLastWeeklyTime(action.Time, action.Day, now, location)
	case "monthly":
		if action.Time == "" {
			return time.Time{}, fmt.Errorf("monthly schedule missing time")
		}
		if action.Day < 1 || action.Day > 31 {
			return time.Time{}, fmt.Errorf("monthly schedule invalid day: %d", action.Day)
		}
		return v.getLastMonthlyTime(action.Time, action.Day, now, location)
	case "cron":
		if action.Crontab.String() == "" {
			return time.Time{}, fmt.Errorf("cron schedule missing crontab")
		}
		return v.getLastCronTime(action.Crontab.String(), now, location)
	default:
		return time.Time{}, fmt.Errorf("unknown schedule type: %s", sch.Type)
	}
}

// getLastDailyTime calculates the last daily execution time before now.
func (v *Validator) getLastDailyTime(timeStr string, now time.Time, location *time.Location) (time.Time, error) {
	parts := [3]int{}
	n, err := fmt.Sscanf(timeStr, "%d:%d:%d", &parts[0], &parts[1], &parts[2])
	if err != nil && n < 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hour := parts[0]
	minute := parts[1]
	second := 0
	if n == 3 {
		second = parts[2]
	}

	// Create time for today at the specified time
	today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, location)

	// If today's time hasn't passed yet, use yesterday
	if today.After(now) || today.Equal(now) {
		today = today.AddDate(0, 0, -1)
	}

	return today, nil
}

// getLastWeeklyTime calculates the last weekly execution time before now.
func (v *Validator) getLastWeeklyTime(timeStr string, dayOfWeek int, now time.Time, location *time.Location) (time.Time, error) {
	parts := [3]int{}
	n, err := fmt.Sscanf(timeStr, "%d:%d:%d", &parts[0], &parts[1], &parts[2])
	if err != nil && n < 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hour := parts[0]
	minute := parts[1]
	second := 0
	if n == 3 {
		second = parts[2]
	}

	// Convert day of week (0=Sunday, 1=Monday, ..., 6=Saturday) to time.Weekday
	var targetWeekday time.Weekday
	switch dayOfWeek {
	case 0:
		targetWeekday = time.Sunday
	case 1:
		targetWeekday = time.Monday
	case 2:
		targetWeekday = time.Tuesday
	case 3:
		targetWeekday = time.Wednesday
	case 4:
		targetWeekday = time.Thursday
	case 5:
		targetWeekday = time.Friday
	case 6:
		targetWeekday = time.Saturday
	default:
		return time.Time{}, fmt.Errorf("invalid day of week: %d", dayOfWeek)
	}

	// Find the last occurrence of the target weekday
	currentWeekday := now.Weekday()
	daysBack := int(currentWeekday - targetWeekday)
	if daysBack < 0 {
		daysBack += 7
	}

	// If today is the target day, check if the time has passed
	if daysBack == 0 {
		today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, location)
		if today.After(now) {
			daysBack = 7 // Use last week's occurrence
		}
	}

	targetDate := now.AddDate(0, 0, -daysBack)
	return time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), hour, minute, second, 0, location), nil
}

// getLastMonthlyTime calculates the last monthly execution time before now.
func (v *Validator) getLastMonthlyTime(timeStr string, dayOfMonth int, now time.Time, location *time.Location) (time.Time, error) {
	parts := [3]int{}
	n, err := fmt.Sscanf(timeStr, "%d:%d:%d", &parts[0], &parts[1], &parts[2])
	if err != nil && n < 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hour := parts[0]
	minute := parts[1]
	second := 0
	if n == 3 {
		second = parts[2]
	}

	// Try this month first
	thisMonth := time.Date(now.Year(), now.Month(), dayOfMonth, hour, minute, second, 0, location)
	// If day doesn't exist in this month (e.g., Feb 31), go to last month
	if thisMonth.Month() != now.Month() {
		// Use last day of previous month
		lastMonth := now.AddDate(0, -1, 0)
		lastDay := time.Date(lastMonth.Year(), lastMonth.Month()+1, 0, hour, minute, second, 0, location)
		if dayOfMonth > lastDay.Day() {
			thisMonth = lastDay
		} else {
			thisMonth = time.Date(lastMonth.Year(), lastMonth.Month(), dayOfMonth, hour, minute, second, 0, location)
		}
	}

	// If this month's time hasn't passed yet, use last month
	if thisMonth.After(now) {
		lastMonth := now.AddDate(0, -1, 0)
		lastDay := time.Date(lastMonth.Year(), lastMonth.Month()+1, 0, hour, minute, second, 0, location)
		if dayOfMonth > lastDay.Day() {
			thisMonth = lastDay
		} else {
			thisMonth = time.Date(lastMonth.Year(), lastMonth.Month(), dayOfMonth, hour, minute, second, 0, location)
		}
	}

	return thisMonth, nil
}

// getLastCronTime calculates the last cron execution time before now.
func (v *Validator) getLastCronTime(crontab string, now time.Time, location *time.Location) (time.Time, error) {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(crontab)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Start from a point in the past (1 year ago) and iterate forward
	// to find the last execution time before now
	startTime := now.AddDate(-1, 0, 0)
	lastTime := schedule.Next(startTime)
	var prevTime time.Time

	// Iterate forward until we pass now
	maxIterations := 10000 // Safety limit for very frequent cron expressions
	for i := 0; i < maxIterations; i++ {
		if lastTime.After(now) || lastTime.Equal(now) {
			// We've passed now, so prevTime is the last execution before now
			if prevTime.IsZero() {
				return time.Time{}, fmt.Errorf("no cron execution found before now")
			}
			return prevTime, nil
		}
		prevTime = lastTime
		lastTime = schedule.Next(lastTime)
	}

	// If we exhausted iterations, return the last time we found
	if !prevTime.IsZero() {
		return prevTime, nil
	}

	return time.Time{}, fmt.Errorf("failed to find last cron execution time")
}

// getActualState retrieves the current state of the resource.
// Returns (state, isTransitional, error).
// state: "running", "stopped", or a transitional state name (e.g., "provisioning", "stopping")
// isTransitional: true if resource is in a transitional state (PROVISIONING, STOPPING, STARTING, etc.)
func (v *Validator) getActualState(ctx context.Context, resource pkgconfig.Resource) (string, bool, error) {
	switch resource.Type {
	case "vm":
		instance, err := v.client.GetInstance(ctx, resource.FolderID, resource.ID)
		if err != nil {
			return "", false, err
		}
		status := instance.GetStatus()
		switch status {
		case computepb.Instance_RUNNING:
			return "running", false, nil
		case computepb.Instance_STOPPED:
			return "stopped", false, nil
		default:
			// Resource is in transitional state (PROVISIONING, etc.)
			return status.String(), true, nil
		}
	case "k8s_cluster":
		cluster, err := v.client.GetCluster(ctx, resource.FolderID, resource.ID)
		if err != nil {
			return "", false, err
		}
		status := cluster.GetStatus()
		switch status {
		case k8spb.Cluster_RUNNING:
			return "running", false, nil
		case k8spb.Cluster_STOPPED:
			return "stopped", false, nil
		default:
			// Resource is in transitional state (PROVISIONING, RECONCILING, STOPPING, STARTING, DELETING, etc.)
			return status.String(), true, nil
		}
	default:
		return "", false, nil
	}
}
