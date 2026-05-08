package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sentoz/yc-sheduler/internal/calendar"
	"github.com/sentoz/yc-sheduler/internal/config"
)

const dateOnlyLayout = "2006-01-02"

// ScheduleProvider supplies current schedules for the calendar UI.
type ScheduleProvider interface {
	Schedules() []config.Schedule
	Timezone() string
	ValidationInterval() string
	ValidationResources() bool
	ResourceStatuses(ctx context.Context, schedules []config.Schedule) map[string]ResourceStatus
}

// ResourceStatus describes the current live state of a resource.
type ResourceStatus struct {
	State string `json:"state"`
	Error string `json:"error,omitempty"`

	IsTransitional bool `json:"is_transitional,omitempty"`
}

type calendarResponse struct {
	Timezone            string           `json:"timezone"`
	ValidationInterval  string           `json:"validation_interval"`
	From                string           `json:"from"`
	To                  string           `json:"to"`
	Title               string           `json:"title"`
	Events              []calendar.Event `json:"events"`
	ValidationResources bool             `json:"validation_resources"`
}

func registerCalendarAPI(mux *http.ServeMux, provider ScheduleProvider) {
	mux.HandleFunc("/api/calendar", func(w http.ResponseWriter, r *http.Request) {
		handleCalendar(w, r, provider)
	})
}

func handleCalendar(w http.ResponseWriter, r *http.Request, provider ScheduleProvider) {
	location, err := loadLocation(provider.Timezone())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	from, to, err := parseCalendarRange(r, location)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	schedules := provider.Schedules()
	events, err := calendar.EventsInRange(schedules, provider.Timezone(), from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	mergeResourceStatuses(events, provider.ResourceStatuses(r.Context(), schedules))

	response := calendarResponse{
		Timezone:            location.String(),
		ValidationInterval:  provider.ValidationInterval(),
		ValidationResources: provider.ValidationResources(),
		From:                from.Format(dateOnlyLayout),
		To:                  to.Format(dateOnlyLayout),
		Title:               calendar.FormatMonthTitle(from.In(location)),
		Events:              events,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func mergeResourceStatuses(events []calendar.Event, statuses map[string]ResourceStatus) {
	for i := range events {
		status, exists := statuses[events[i].ResourceKey]
		if !exists {
			continue
		}
		events[i].State = status.State
		events[i].StatusError = status.Error
		events[i].Transitional = status.IsTransitional
	}
}

func parseCalendarRange(r *http.Request, location *time.Location) (time.Time, time.Time, error) {
	fromRaw := r.URL.Query().Get("from")
	toRaw := r.URL.Query().Get("to")

	if fromRaw == "" && toRaw == "" {
		now := time.Now().In(location)
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
		to := from.AddDate(0, 1, -1)
		return from, to, nil
	}

	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("both from and to must be provided")
	}

	from, err := time.ParseInLocation(dateOnlyLayout, fromRaw, location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from date %q, expected YYYY-MM-DD", fromRaw)
	}

	to, err := time.ParseInLocation(dateOnlyLayout, toRaw, location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to date %q, expected YYYY-MM-DD", toRaw)
	}

	if to.Before(from) {
		return time.Time{}, time.Time{}, fmt.Errorf("to date must be on or after from date")
	}

	return from, to, nil
}

func loadLocation(timezone string) (*time.Location, error) {
	if timezone == "" {
		return time.Local, nil
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", timezone, err)
	}

	return location, nil
}

// ResourceKey returns a stable key for resource status lookup in the UI layer.
func ResourceKey(resource config.Resource) string {
	return resource.Type + ":" + resource.FolderID + ":" + resource.ID
}
