package validator

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	computepb "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	k8spb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"

	"github.com/woozymasta/yc-scheduler/internal/executor"
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
		// Both enabled: simplified logic - prefer running
		// In a more sophisticated implementation, we would check
		// the actual schedule times to determine the expected state.
		return "running", "start"
	}

	// No actions enabled
	return "", ""
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
