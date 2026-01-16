package executor

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	computepb "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	k8spb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"

	"github.com/woozymasta/yc-scheduler/internal/config"
	"github.com/woozymasta/yc-scheduler/internal/metrics"
	"github.com/woozymasta/yc-scheduler/internal/yc"
)

// Make returns a job function that executes the given action for the schedule's resource.
// The returned function has no parameters to match gocron's expectations.
func Make(client *yc.Client, sch config.Schedule, action string, dryRun bool) func() {
	resource := sch.Resource

	return func() {
		// Use a background context with a reasonable timeout for YC operations.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		resourceType := resource.Type

		if dryRun {
			log.Info().
				Str("schedule", sch.Name).
				Str("resource_type", resourceType).
				Str("resource_id", resource.ID).
				Str("action", action).
				Msg("Dry-run: planned operation")
			metrics.IncOperation(resourceType, action, "dry_run")
			return
		}

		var (
			op          func(context.Context) error
			resourceMsg string
		)

		switch resourceType {
		case "vm":
			switch action {
			case "start":
				op = func(ctx context.Context) error {
					return client.StartInstance(ctx, resource.FolderID, resource.ID)
				}
			case "stop":
				op = func(ctx context.Context) error {
					return client.StopInstance(ctx, resource.FolderID, resource.ID)
				}
			}
			resourceMsg = "VM operation failed"
		case "k8s_cluster":
			switch action {
			case "start":
				op = func(ctx context.Context) error {
					return client.StartCluster(ctx, resource.FolderID, resource.ID)
				}
			case "stop":
				op = func(ctx context.Context) error {
					return client.StopCluster(ctx, resource.FolderID, resource.ID)
				}
			}
			resourceMsg = "Cluster operation failed"
		default:
			log.Error().
				Str("resource_type", resourceType).
				Str("schedule", sch.Name).
				Msg("Unsupported resource type")
			metrics.IncOperation(resourceType, action, "error")
			return
		}

		if op == nil {
			log.Error().
				Str("resource_type", resourceType).
				Str("resource_id", resource.ID).
				Str("action", action).
				Msg("Unsupported action for resource")
			metrics.IncOperation(resourceType, action, "error")
			return
		}

		// Check current state before executing operation to avoid conflicts
		currentState, isTransitional, err := getCurrentState(ctx, client, resource)
		if err != nil {
			log.Warn().Err(err).
				Str("schedule", sch.Name).
				Str("resource_type", resourceType).
				Str("resource_id", resource.ID).
				Str("action", action).
				Msg("Failed to get current resource state, proceeding with operation")
		} else {
			// Skip operation if resource is in transitional state
			if isTransitional {
				log.Info().
					Str("schedule", sch.Name).
					Str("resource_type", resourceType).
					Str("resource_id", resource.ID).
					Str("action", action).
					Str("current_state", currentState).
					Msg("Resource is in transitional state, skipping operation")
				metrics.IncOperation(resourceType, action, "skipped")
				metrics.IncSchedulerSkip(resourceType, action, "transitional_state")
				return
			}

			// Skip operation if resource is already in desired state
			if (action == "start" && currentState == "running") ||
				(action == "stop" && currentState == "stopped") {
				log.Info().
					Str("schedule", sch.Name).
					Str("resource_type", resourceType).
					Str("resource_id", resource.ID).
					Str("action", action).
					Str("current_state", currentState).
					Msg("Resource is already in desired state, skipping operation")
				metrics.IncOperation(resourceType, action, "skipped")
				metrics.IncSchedulerSkip(resourceType, action, "already_in_state")
				return
			}
		}

		log.Debug().
			Str("schedule", sch.Name).
			Str("resource_type", resourceType).
			Str("resource_id", resource.ID).
			Str("action", action).
			Msg("Executing resource operation")

		if err := op(ctx); err != nil {
			log.Error().Err(err).
				Str("resource_type", resourceType).
				Str("resource_id", resource.ID).
				Str("action", action).
				Msg(resourceMsg)
			metrics.IncOperation(resourceType, action, "error")
			return
		}

		metrics.IncOperation(resourceType, action, "success")
	}
}

// getCurrentState retrieves the current state of the resource.
// Returns (state, isTransitional, error).
// state: "running", "stopped", or a transitional state name
// isTransitional: true if resource is in a transitional state
func getCurrentState(ctx context.Context, client *yc.Client, resource config.Resource) (string, bool, error) {
	switch resource.Type {
	case "vm":
		instance, err := client.GetInstance(ctx, resource.FolderID, resource.ID)
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
			// Resource is in transitional state
			return status.String(), true, nil
		}
	case "k8s_cluster":
		cluster, err := client.GetCluster(ctx, resource.FolderID, resource.ID)
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
			// Resource is in transitional state
			return status.String(), true, nil
		}
	default:
		return "", false, nil
	}
}
