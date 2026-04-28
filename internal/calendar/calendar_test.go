package calendar

import (
	"testing"
	"time"

	"github.com/sentoz/yc-sheduler/internal/config"
)

func TestEventsInRangeDaily(t *testing.T) {
	events, err := EventsInRange([]config.Schedule{
		makeSchedule("vm-work", "daily", "start", &config.ActionConfig{Enabled: true, Time: "09:00"}),
	}, "Europe/Moscow", mustDate(t, "2026-04-01"), mustDate(t, "2026-04-03"))
	if err != nil {
		t.Fatalf("EventsInRange() error = %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	if events[0].LocalDate != "2026-04-01" || events[0].LocalTime != "09:00:00" {
		t.Fatalf("first event = %+v, want 2026-04-01 09:00:00", events[0])
	}
}

func TestEventsInRangeWeekly(t *testing.T) {
	events, err := EventsInRange([]config.Schedule{
		makeSchedule("vm-weekly", "weekly", "stop", &config.ActionConfig{Enabled: true, Time: "18:30", Day: 1}),
	}, "Europe/Moscow", mustDate(t, "2026-04-01"), mustDate(t, "2026-04-15"))
	if err != nil {
		t.Fatalf("EventsInRange() error = %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].LocalDate != "2026-04-06" || events[1].LocalDate != "2026-04-13" {
		t.Fatalf("events dates = %+v, want Mondays 2026-04-06 and 2026-04-13", events)
	}
}

func TestEventsInRangeMonthly(t *testing.T) {
	events, err := EventsInRange([]config.Schedule{
		makeSchedule("vm-monthly", "monthly", "start", &config.ActionConfig{Enabled: true, Time: "07:15", Day: 15}),
	}, "Europe/Moscow", mustDate(t, "2026-04-01"), mustDate(t, "2026-05-31"))
	if err != nil {
		t.Fatalf("EventsInRange() error = %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].LocalDate != "2026-04-15" || events[1].LocalDate != "2026-05-15" {
		t.Fatalf("events = %+v, want 15th of each month", events)
	}
}

func TestEventsInRangeCron(t *testing.T) {
	events, err := EventsInRange([]config.Schedule{
		makeSchedule("vm-cron", "cron", "start", &config.ActionConfig{Enabled: true, Crontab: config.Crontab("0 8 * * 1-5")}),
	}, "Europe/Moscow", mustDate(t, "2026-04-06"), mustDate(t, "2026-04-08"))
	if err != nil {
		t.Fatalf("EventsInRange() error = %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	if events[0].LocalTime != "08:00:00" {
		t.Fatalf("first event time = %s, want 08:00:00", events[0].LocalTime)
	}
}

func TestEventsInRangeSkipsDisabledAction(t *testing.T) {
	events, err := EventsInRange([]config.Schedule{
		makeSchedule("vm-disabled", "daily", "start", &config.ActionConfig{Enabled: false, Time: "09:00"}),
	}, "Europe/Moscow", mustDate(t, "2026-04-01"), mustDate(t, "2026-04-03"))
	if err != nil {
		t.Fatalf("EventsInRange() error = %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("len(events) = %d, want 0", len(events))
	}
}

func TestEventsInRangeSortsByTime(t *testing.T) {
	events, err := EventsInRange([]config.Schedule{
		makeSchedule("b", "daily", "start", &config.ActionConfig{Enabled: true, Time: "18:00"}),
		makeSchedule("a", "daily", "start", &config.ActionConfig{Enabled: true, Time: "09:00"}),
	}, "Europe/Moscow", mustDate(t, "2026-04-01"), mustDate(t, "2026-04-01"))
	if err != nil {
		t.Fatalf("EventsInRange() error = %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].ScheduleName != "a" || events[1].ScheduleName != "b" {
		t.Fatalf("events order = %+v, want a then b", events)
	}
}

func makeSchedule(name, scheduleType, actionName string, action *config.ActionConfig) config.Schedule {
	schedule := config.Schedule{
		Name: name,
		Type: scheduleType,
		Resource: config.Resource{
			Type:     "vm",
			ID:       "resource-id",
			FolderID: "folder-id",
		},
	}

	switch actionName {
	case "start":
		schedule.Actions.Start = action
	case "stop":
		schedule.Actions.Stop = action
	}

	return schedule
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()

	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("time.Parse(%q) error = %v", value, err)
	}
	return date
}
