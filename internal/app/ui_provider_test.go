package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sentoz/yc-sheduler/internal/config"
)

func TestUIProviderCachesResourceStatus(t *testing.T) {
	var calls atomic.Int32
	checker := fakeStateChecker{
		getState: func(context.Context, config.Resource) (string, bool, error) {
			calls.Add(1)
			return "running", false, nil
		},
	}

	store := NewScheduleStore("Europe/Moscow", nil)
	provider := NewUIProvider(store, checker, "10m", true)

	current := time.Date(2026, time.April, 29, 12, 0, 0, 0, time.UTC)
	provider.now = func() time.Time { return current }

	schedules := []config.Schedule{
		{
			Name: "a",
			Resource: config.Resource{
				Type:     "vm",
				ID:       "id",
				FolderID: "folder",
			},
		},
	}

	provider.ResourceStatuses(t.Context(), schedules)
	provider.ResourceStatuses(t.Context(), schedules)

	if got := calls.Load(); got != 1 {
		t.Fatalf("GetState call count = %d, want 1", got)
	}

	current = current.Add(statusCacheTTL + time.Second)
	provider.ResourceStatuses(t.Context(), schedules)

	if got := calls.Load(); got != 2 {
		t.Fatalf("GetState call count after ttl = %d, want 2", got)
	}
}

func TestUIProviderSanitizesResourceStatusError(t *testing.T) {
	checker := fakeStateChecker{
		getState: func(context.Context, config.Resource) (string, bool, error) {
			return "", false, errors.New("secret token detail")
		},
	}

	store := NewScheduleStore("Europe/Moscow", nil)
	provider := NewUIProvider(store, checker, "10m", true)

	schedules := []config.Schedule{
		{
			Name: "a",
			Resource: config.Resource{
				Type:     "vm",
				ID:       "id",
				FolderID: "folder",
			},
		},
	}

	statuses := provider.ResourceStatuses(t.Context(), schedules)
	status := statuses["vm:folder:id"]

	if status.State != "unknown" {
		t.Fatalf("State = %q, want unknown", status.State)
	}
	if status.Error != "failed to fetch state" {
		t.Fatalf("Error = %q, want sanitized error", status.Error)
	}
}

type fakeStateChecker struct {
	getState func(context.Context, config.Resource) (string, bool, error)
}

func (f fakeStateChecker) GetState(ctx context.Context, resource config.Resource) (string, bool, error) {
	return f.getState(ctx, resource)
}
