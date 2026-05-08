package calendar

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/sentoz/yc-sheduler/internal/config"
)

// Event represents a single scheduled action occurrence in the calendar.
type Event struct {
	ScheduleName        string `json:"schedule_name"`
	ScheduleDisplayName string `json:"schedule_display_name"`
	ResourceType        string `json:"resource_type"`
	ResourceID          string `json:"resource_id"`
	FolderID            string `json:"folder_id,omitempty"`
	ResourceKey         string `json:"resource_key"`
	Action              string `json:"action"`
	Time                string `json:"time"`
	LocalDate           string `json:"local_date"`
	LocalTime           string `json:"local_time"`
	State               string `json:"state,omitempty"`
	StatusError         string `json:"status_error,omitempty"`
	Transitional        bool   `json:"transitional,omitempty"`
}

// EventsInRange expands schedules into concrete calendar events in the inclusive
// [from, to] date range interpreted in the provided timezone.
func EventsInRange(schedules []config.Schedule, timezone string, from, to time.Time) ([]Event, error) {
	location := time.Local
	if timezone != "" {
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			return nil, fmt.Errorf("calendar: load timezone %q: %w", timezone, err)
		}
		location = loc
	}

	startDate := dateOnly(from.In(location))
	endDate := dateOnly(to.In(location))
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("calendar: invalid range: %s before %s", endDate.Format(dateOnlyLayout), startDate.Format(dateOnlyLayout))
	}

	rangeStart := startDate
	rangeEndExclusive := endDate.AddDate(0, 0, 1)

	events := make([]Event, 0)
	for _, schedule := range schedules {
		if schedule.Actions.Start != nil && schedule.Actions.Start.Enabled {
			actionEvents, err := expandAction(schedule, "start", schedule.Actions.Start, rangeStart, rangeEndExclusive, location)
			if err != nil {
				return nil, err
			}
			events = append(events, actionEvents...)
		}
		if schedule.Actions.Stop != nil && schedule.Actions.Stop.Enabled {
			actionEvents, err := expandAction(schedule, "stop", schedule.Actions.Stop, rangeStart, rangeEndExclusive, location)
			if err != nil {
				return nil, err
			}
			events = append(events, actionEvents...)
		}
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].Time != events[j].Time {
			return events[i].Time < events[j].Time
		}
		if events[i].ScheduleName != events[j].ScheduleName {
			return events[i].ScheduleName < events[j].ScheduleName
		}
		return events[i].Action < events[j].Action
	})

	return events, nil
}

func expandAction(
	schedule config.Schedule,
	actionName string,
	action *config.ActionConfig,
	rangeStart time.Time,
	rangeEndExclusive time.Time,
	location *time.Location,
) ([]Event, error) {
	switch schedule.Type {
	case "daily":
		hour, minute, second, err := parseClock(action.Time)
		if err != nil {
			return nil, fmt.Errorf("calendar: daily schedule %q: %w", schedule.Name, err)
		}
		events := make([]Event, 0)
		for day := rangeStart; day.Before(rangeEndExclusive); day = day.AddDate(0, 0, 1) {
			at := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, second, 0, location)
			events = append(events, newEvent(schedule, actionName, at))
		}
		return events, nil
	case "weekly":
		hour, minute, second, err := parseClock(action.Time)
		if err != nil {
			return nil, fmt.Errorf("calendar: weekly schedule %q: %w", schedule.Name, err)
		}
		if action.Day < 0 || action.Day > 6 {
			return nil, fmt.Errorf("calendar: weekly schedule %q: invalid day %d", schedule.Name, action.Day)
		}
		events := make([]Event, 0)
		for day := rangeStart; day.Before(rangeEndExclusive); day = day.AddDate(0, 0, 1) {
			if int(day.Weekday()) != action.Day {
				continue
			}
			at := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, second, 0, location)
			events = append(events, newEvent(schedule, actionName, at))
		}
		return events, nil
	case "monthly":
		hour, minute, second, err := parseClock(action.Time)
		if err != nil {
			return nil, fmt.Errorf("calendar: monthly schedule %q: %w", schedule.Name, err)
		}
		if action.Day < 1 || action.Day > 31 {
			return nil, fmt.Errorf("calendar: monthly schedule %q: invalid day %d", schedule.Name, action.Day)
		}
		events := make([]Event, 0)
		for day := rangeStart; day.Before(rangeEndExclusive); day = day.AddDate(0, 0, 1) {
			if day.Day() != action.Day {
				continue
			}
			at := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, second, 0, location)
			events = append(events, newEvent(schedule, actionName, at))
		}
		return events, nil
	case "cron":
		if action.Crontab.String() == "" {
			return nil, fmt.Errorf("calendar: cron schedule %q missing crontab", schedule.Name)
		}
		cronSchedule, err := parseCronSchedule(action.Crontab.String())
		if err != nil {
			return nil, fmt.Errorf("calendar: cron schedule %q: %w", schedule.Name, err)
		}
		events := make([]Event, 0)
		cursor := rangeStart.Add(-time.Second)
		for {
			next := cronSchedule.Next(cursor.In(location))
			if !next.Before(rangeEndExclusive) {
				break
			}
			if !next.Before(rangeStart) {
				events = append(events, newEvent(schedule, actionName, next.In(location)))
			}
			cursor = next
		}
		return events, nil
	default:
		return nil, fmt.Errorf("calendar: unknown schedule type %q for %q", schedule.Type, schedule.Name)
	}
}

func newEvent(schedule config.Schedule, actionName string, at time.Time) Event {
	return Event{
		ScheduleName:        schedule.Name,
		ScheduleDisplayName: scheduleDisplayName(schedule),
		ResourceType:        schedule.Resource.Type,
		ResourceID:          schedule.Resource.ID,
		FolderID:            schedule.Resource.FolderID,
		ResourceKey:         resourceKey(schedule.Resource),
		Action:              actionName,
		Time:                at.Format(time.RFC3339),
		LocalDate:           at.Format(dateOnlyLayout),
		LocalTime:           at.Format("15:04:05"),
	}
}

func parseClock(value string) (hour, minute, second int, err error) {
	for _, layout := range []string{"15:04:05", "15:04"} {
		parsed, parseErr := time.Parse(layout, value)
		if parseErr == nil {
			return parsed.Hour(), parsed.Minute(), parsed.Second(), nil
		}
	}

	return 0, 0, 0, fmt.Errorf("invalid time %q", value)
}

func parseCronSchedule(spec string) (cron.Schedule, error) {
	parserWithSeconds := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parserWithSeconds.Parse(spec)
	if err == nil {
		return schedule, nil
	}

	parserStandard := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, standardErr := parserStandard.Parse(spec)
	if standardErr != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", standardErr)
	}

	return schedule, nil
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

const dateOnlyLayout = "2006-01-02"

// FormatMonthTitle returns a human-friendly month caption for the UI.
func FormatMonthTitle(t time.Time) string {
	return strings.TrimSpace(t.Format("January 2006"))
}

func resourceKey(resource config.Resource) string {
	return resource.Type + ":" + resource.FolderID + ":" + resource.ID
}

func scheduleDisplayName(schedule config.Schedule) string {
	if schedule.DisplayName != "" {
		return schedule.DisplayName
	}
	return schedule.Name
}
