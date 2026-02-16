package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sentoz/yc-sheduler/internal/config"
	"github.com/sentoz/yc-sheduler/internal/metrics"
	"github.com/sentoz/yc-sheduler/internal/resource"
)

var operationLocks = newInFlightLocks()

type inFlightLocks struct {
	locks map[string]struct{}
	mu    sync.Mutex
}

func newInFlightLocks() *inFlightLocks {
	return &inFlightLocks{
		locks: make(map[string]struct{}),
	}
}

func (l *inFlightLocks) tryLock(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.locks[key]; exists {
		return false
	}

	l.locks[key] = struct{}{}
	return true
}

func (l *inFlightLocks) unlock(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.locks, key)
}

// Make returns a job function that executes the given action for the schedule's resource.
// The returned function has no parameters to match gocron's expectations.
// If m is nil, metrics will not be recorded.
func Make(stateChecker resource.StateChecker, operator resource.Operator, sch config.Schedule, action string, dryRun bool, m *metrics.Metrics) func() {
	resource := sch.Resource

	return func() {
		// Use a background context with a reasonable timeout for YC operations.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		resourceType := resource.Type
		lockKey := resourceType + ":" + resource.ID + ":" + action

		if !operationLocks.tryLock(lockKey) {
			log.Info().
				Str("schedule", sch.Name).
				Str("resource_type", resourceType).
				Str("resource_id", resource.ID).
				Str("action", action).
				Msg("Operation for resource/action is already in progress, skipping")
			if m != nil {
				m.IncOperation(resourceType, action, "skipped")
				m.IncSchedulerSkip(resourceType, action, "in_flight")
			}
			return
		}
		defer operationLocks.unlock(lockKey)

		if dryRun {
			log.Info().
				Str("schedule", sch.Name).
				Str("resource_type", resourceType).
				Str("resource_id", resource.ID).
				Str("action", action).
				Msg("Dry-run: planned operation")
			if m != nil {
				m.IncOperation(resourceType, action, "dry_run")
			}
			return
		}

		// Validate action
		if action != "start" && action != "stop" {
			log.Error().
				Str("resource_type", resourceType).
				Str("resource_id", resource.ID).
				Str("action", action).
				Msg("Unsupported action for resource")
			if m != nil {
				m.IncOperation(resourceType, action, "error")
			}
			return
		}

		// Check current state before executing operation to avoid conflicts
		currentState, isTransitional, stateErr := stateChecker.GetState(ctx, resource)
		if stateErr != nil {
			log.Warn().Err(stateErr).
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
				if m != nil {
					m.IncOperation(resourceType, action, "skipped")
					m.IncSchedulerSkip(resourceType, action, "transitional_state")
				}
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
				if m != nil {
					m.IncOperation(resourceType, action, "skipped")
					m.IncSchedulerSkip(resourceType, action, "already_in_state")
				}
				return
			}
		}

		log.Debug().
			Str("schedule", sch.Name).
			Str("resource_type", resourceType).
			Str("resource_id", resource.ID).
			Str("action", action).
			Msg("Executing resource operation")

		var opErr error
		switch action {
		case "start":
			opErr = operator.Start(ctx, resource)
		case "stop":
			opErr = operator.Stop(ctx, resource)
		default:
			opErr = fmt.Errorf("unsupported action: %s", action)
		}

		if opErr != nil {
			log.Error().Err(opErr).
				Str("resource_type", resourceType).
				Str("resource_id", resource.ID).
				Str("action", action).
				Msg("Resource operation failed")
			if m != nil {
				m.IncOperation(resourceType, action, "error")
			}
			return
		}

		if m != nil {
			m.IncOperation(resourceType, action, "success")
		}
	}
}
