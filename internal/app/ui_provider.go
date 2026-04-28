package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sentoz/yc-sheduler/internal/config"
	"github.com/sentoz/yc-sheduler/internal/resource"
	"github.com/sentoz/yc-sheduler/internal/web"
)

const statusCacheTTL = 30 * time.Second

// UIProvider supplies schedules and live resource states to the calendar UI.
type UIProvider struct {
	store        *ScheduleStore
	stateChecker resource.StateChecker
	now          func() time.Time
	cache        map[string]cachedResourceStatus

	mu sync.Mutex
}

// NewUIProvider creates a calendar UI provider.
func NewUIProvider(store *ScheduleStore, stateChecker resource.StateChecker) *UIProvider {
	return &UIProvider{
		store:        store,
		stateChecker: stateChecker,
		now:          time.Now,
		cache:        make(map[string]cachedResourceStatus),
	}
}

// Schedules returns current schedules.
func (p *UIProvider) Schedules() []config.Schedule {
	if p == nil || p.store == nil {
		return nil
	}
	return p.store.Schedules()
}

// Timezone returns the configured application timezone.
func (p *UIProvider) Timezone() string {
	if p == nil || p.store == nil {
		return ""
	}
	return p.store.Timezone()
}

// ResourceStatuses returns the current state for unique resources referenced by schedules.
func (p *UIProvider) ResourceStatuses(ctx context.Context, schedules []config.Schedule) map[string]web.ResourceStatus {
	statuses := make(map[string]web.ResourceStatus)
	if p == nil || p.stateChecker == nil {
		return statuses
	}

	seen := make(map[string]struct{}, len(schedules))
	for _, schedule := range schedules {
		key := web.ResourceKey(schedule.Resource)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		statuses[key] = p.getResourceStatus(ctx, key, schedule.Resource)
	}

	return statuses
}

func (p *UIProvider) getResourceStatus(ctx context.Context, key string, resource config.Resource) web.ResourceStatus {
	now := p.now()

	p.mu.Lock()
	if cached, exists := p.cache[key]; exists && now.Before(cached.expiresAt) {
		p.mu.Unlock()
		return cached.status
	}
	p.mu.Unlock()

	state, isTransitional, err := p.stateChecker.GetState(ctx, resource)
	status := web.ResourceStatus{}
	if err != nil {
		status.State = "unknown"
		status.Error = fmt.Sprintf("failed to fetch state: %v", err)
	} else {
		status.State = state
		status.IsTransitional = isTransitional
	}

	p.mu.Lock()
	p.cache[key] = cachedResourceStatus{
		status:    status,
		expiresAt: now.Add(statusCacheTTL),
	}
	p.mu.Unlock()

	return status
}

type cachedResourceStatus struct {
	expiresAt time.Time
	status    web.ResourceStatus
}
