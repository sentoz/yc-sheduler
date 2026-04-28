package app

import (
	"sync"

	"github.com/sentoz/yc-sheduler/internal/config"
)

// ScheduleStore provides concurrent read access to current schedules for the UI.
type ScheduleStore struct {
	timezone  string
	schedules []config.Schedule

	mu sync.RWMutex
}

// NewScheduleStore creates a store initialized with the current schedules.
func NewScheduleStore(timezone string, schedules []config.Schedule) *ScheduleStore {
	store := &ScheduleStore{
		timezone: timezone,
	}
	store.Update(schedules)
	return store
}

// Schedules returns a copy of the current schedules.
func (s *ScheduleStore) Schedules() []config.Schedule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return append([]config.Schedule(nil), s.schedules...)
}

// Timezone returns the application timezone used for schedule calculations.
func (s *ScheduleStore) Timezone() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.timezone
}

// Update replaces the current schedules with a copy of the provided slice.
func (s *ScheduleStore) Update(schedules []config.Schedule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.schedules = append([]config.Schedule(nil), schedules...)
}
