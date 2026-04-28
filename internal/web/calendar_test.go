package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sentoz/yc-sheduler/internal/config"
)

func TestCalendarAPI(t *testing.T) {
	mux := newMux(false, testProvider{
		timezone: "Europe/Moscow",
		schedules: []config.Schedule{
			{
				Name: "vm-daily",
				Type: "daily",
				Resource: config.Resource{
					Type:     "vm",
					ID:       "resource-id",
					FolderID: "folder-id",
				},
				Actions: config.Actions{
					Start: &config.ActionConfig{
						Enabled: true,
						Time:    "09:00",
					},
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/calendar?from=2026-04-01&to=2026-04-02", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "\"schedule_name\":\"vm-daily\"") {
		t.Fatalf("body = %s, want schedule_name", body)
	}
	if !strings.Contains(body, "\"from\":\"2026-04-01\"") {
		t.Fatalf("body = %s, want from", body)
	}
	if !strings.Contains(body, "\"state\":\"running\"") {
		t.Fatalf("body = %s, want state", body)
	}
}

func TestCalendarAPIRejectsInvalidRange(t *testing.T) {
	mux := newMux(false, testProvider{timezone: "Europe/Moscow"})

	req := httptest.NewRequest(http.MethodGet, "/api/calendar?from=2026-04-02&to=2026-04-01", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUIIndexServed(t *testing.T) {
	mux := newMux(false, testProvider{timezone: "Europe/Moscow"})

	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if !strings.Contains(string(body), "<!DOCTYPE html>") {
		t.Fatalf("body = %s, want HTML doctype", string(body))
	}
}

func TestUIDisabledWithoutProvider(t *testing.T) {
	mux := newMux(false, nil)

	req := httptest.NewRequest(http.MethodGet, "/ui/", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "\"version\"") {
		t.Fatalf("body = %s, want build info response", rec.Body.String())
	}
}

type testProvider struct {
	timezone  string
	schedules []config.Schedule
}

func (p testProvider) Schedules() []config.Schedule {
	return append([]config.Schedule(nil), p.schedules...)
}

func (p testProvider) Timezone() string {
	return p.timezone
}

func (p testProvider) ResourceStatuses(_ context.Context, schedules []config.Schedule) map[string]ResourceStatus {
	statuses := make(map[string]ResourceStatus, len(schedules))
	for _, schedule := range schedules {
		statuses[ResourceKey(schedule.Resource)] = ResourceStatus{State: "running"}
	}
	return statuses
}
