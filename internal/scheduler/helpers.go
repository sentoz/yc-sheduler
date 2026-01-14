package scheduler

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"

	pkgconfig "github.com/woozymasta/yc-scheduler/pkg/config"
)

func parseTime(t pkgconfig.Time) (gocron.AtTimes, error) {
	value := t.String()
	if value == "" {
		return nil, fmt.Errorf("empty time")
	}

	parts := [3]int{}
	n, err := fmt.Sscanf(value, "%d:%d:%d", &parts[0], &parts[1], &parts[2])
	if err != nil && n < 2 {
		return nil, fmt.Errorf("invalid time format %q", value)
	}

	hour := parts[0]
	minute := parts[1]
	second := 0
	if n == 3 {
		second = parts[2]
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 59 {
		return nil, fmt.Errorf("time out of range %q", value)
	}

	at := gocron.NewAtTime(uint(hour), uint(minute), uint(second))
	return gocron.NewAtTimes(at), nil
}

func parseWeekday(day int) (gocron.Weekdays, error) {
	switch day {
	case 0:
		return gocron.NewWeekdays(time.Sunday), nil
	case 1:
		return gocron.NewWeekdays(time.Monday), nil
	case 2:
		return gocron.NewWeekdays(time.Tuesday), nil
	case 3:
		return gocron.NewWeekdays(time.Wednesday), nil
	case 4:
		return gocron.NewWeekdays(time.Thursday), nil
	case 5:
		return gocron.NewWeekdays(time.Friday), nil
	case 6:
		return gocron.NewWeekdays(time.Saturday), nil
	default:
		return nil, fmt.Errorf("invalid weekday %d", day)
	}
}

func parseDayOfMonth(day int) (int, error) {
	if day < 1 || day > 31 {
		return 0, fmt.Errorf("invalid day of month %d", day)
	}
	return day, nil
}
