package executor

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/woozymasta/yc-scheduler/internal/metrics"
	"github.com/woozymasta/yc-scheduler/internal/yc"
	pkgconfig "github.com/woozymasta/yc-scheduler/pkg/config"
)

// Make returns a job function that executes the given action for the schedule's resource.
// The returned function has no parameters to match gocron's expectations.
func Make(client *yc.Client, sch pkgconfig.Schedule, action string, dryRun bool) func() {
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
