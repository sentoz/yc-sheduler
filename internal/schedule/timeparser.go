package schedule

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/robfig/cron/v3"

	"github.com/sentoz/yc-sheduler/internal/config"
)

// ParseTime parses a time string (HH:MM or HH:MM:SS) and returns gocron.AtTimes.
func ParseTime(t config.Time) (gocron.AtTimes, error) {
	value := t.String()
	if value == "" {
		return nil, fmt.Errorf("empty time")
	}

	hour, minute, second, err := parseTimeString(value)
	if err != nil {
		return nil, err
	}

	// Validate values are in valid range before conversion to uint
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 59 {
		return nil, fmt.Errorf("time out of range: %02d:%02d:%02d", hour, minute, second)
	}

	at := gocron.NewAtTime(uint(hour), uint(minute), uint(second))
	return gocron.NewAtTimes(at), nil
}

// ParseWeekday converts a day of week integer (0=Sunday, 1=Monday, ..., 6=Saturday)
// to gocron.Weekdays.
func ParseWeekday(day int) (gocron.Weekdays, error) {
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

// ParseDayOfMonth validates and returns a day of month (1-31).
func ParseDayOfMonth(day int) (int, error) {
	if day < 1 || day > 31 {
		return 0, fmt.Errorf("invalid day of month %d", day)
	}
	return day, nil
}

// GetLastDailyTime calculates the last daily execution time before now.
func GetLastDailyTime(timeStr string, now time.Time, location *time.Location) (time.Time, error) {
	hour, minute, second, err := parseTimeString(timeStr)
	if err != nil {
		return time.Time{}, err
	}

	// Create time for today at the specified time
	today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, location)

	// If today's time hasn't passed yet, use yesterday
	if today.After(now) || today.Equal(now) {
		today = today.AddDate(0, 0, -1)
	}

	return today, nil
}

// GetLastWeeklyTime calculates the last weekly execution time before now.
func GetLastWeeklyTime(timeStr string, dayOfWeek int, now time.Time, location *time.Location) (time.Time, error) {
	hour, minute, second, err := parseTimeString(timeStr)
	if err != nil {
		return time.Time{}, err
	}

	// Convert day of week (0=Sunday, 1=Monday, ..., 6=Saturday) to time.Weekday
	var targetWeekday time.Weekday
	switch dayOfWeek {
	case 0:
		targetWeekday = time.Sunday
	case 1:
		targetWeekday = time.Monday
	case 2:
		targetWeekday = time.Tuesday
	case 3:
		targetWeekday = time.Wednesday
	case 4:
		targetWeekday = time.Thursday
	case 5:
		targetWeekday = time.Friday
	case 6:
		targetWeekday = time.Saturday
	default:
		return time.Time{}, fmt.Errorf("invalid day of week: %d", dayOfWeek)
	}

	// Find the last occurrence of the target weekday
	currentWeekday := now.Weekday()
	daysBack := int(currentWeekday - targetWeekday)
	if daysBack < 0 {
		daysBack += 7
	}

	// If today is the target day, check if the time has passed
	if daysBack == 0 {
		today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, location)
		if today.After(now) {
			daysBack = 7 // Use last week's occurrence
		}
	}

	targetDate := now.AddDate(0, 0, -daysBack)
	return time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), hour, minute, second, 0, location), nil
}

// GetLastMonthlyTime calculates the last monthly execution time before now.
func GetLastMonthlyTime(timeStr string, dayOfMonth int, now time.Time, location *time.Location) (time.Time, error) {
	hour, minute, second, err := parseTimeString(timeStr)
	if err != nil {
		return time.Time{}, err
	}

	// Try this month first
	thisMonth := time.Date(now.Year(), now.Month(), dayOfMonth, hour, minute, second, 0, location)
	// If day doesn't exist in this month (e.g., Feb 31), go to last month
	if thisMonth.Month() != now.Month() {
		// Use last day of previous month
		lastMonth := now.AddDate(0, -1, 0)
		lastDay := time.Date(lastMonth.Year(), lastMonth.Month()+1, 0, hour, minute, second, 0, location)
		if dayOfMonth > lastDay.Day() {
			thisMonth = lastDay
		} else {
			thisMonth = time.Date(lastMonth.Year(), lastMonth.Month(), dayOfMonth, hour, minute, second, 0, location)
		}
	}

	// If this month's time hasn't passed yet, use last month
	if thisMonth.After(now) {
		lastMonth := now.AddDate(0, -1, 0)
		lastDay := time.Date(lastMonth.Year(), lastMonth.Month()+1, 0, hour, minute, second, 0, location)
		if dayOfMonth > lastDay.Day() {
			thisMonth = lastDay
		} else {
			thisMonth = time.Date(lastMonth.Year(), lastMonth.Month(), dayOfMonth, hour, minute, second, 0, location)
		}
	}

	return thisMonth, nil
}

// GetLastCronTime calculates the last cron execution time before now.
func GetLastCronTime(crontab string, now time.Time) (time.Time, error) {
	// Try parsing with seconds first (6 fields), then fall back to standard format (5 fields)
	var schedule cron.Schedule
	var err error

	// First try with seconds (6 fields)
	parserWithSeconds := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err = parserWithSeconds.Parse(crontab)
	if err != nil {
		// If that fails, try standard format (5 fields)
		parserStandard := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		schedule, err = parserStandard.Parse(crontab)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid cron expression: %w", err)
		}
	}

	// Start from a point in the past (1 year ago) and iterate forward
	// to find the last execution time before now
	startTime := now.AddDate(-1, 0, 0)
	lastTime := schedule.Next(startTime)
	var prevTime time.Time

	// Iterate forward until we pass now
	maxIterations := 10000 // Safety limit for very frequent cron expressions
	for i := 0; i < maxIterations; i++ {
		if lastTime.After(now) || lastTime.Equal(now) {
			// We've passed now, so prevTime is the last execution before now
			if prevTime.IsZero() {
				return time.Time{}, fmt.Errorf("no cron execution found before now")
			}
			return prevTime, nil
		}
		prevTime = lastTime
		lastTime = schedule.Next(lastTime)
	}

	// If we exhausted iterations, return the last time we found
	if !prevTime.IsZero() {
		return prevTime, nil
	}

	return time.Time{}, fmt.Errorf("failed to find last cron execution time")
}

// parseTimeString parses a time string (HH:MM or HH:MM:SS) and returns hour, minute, second.
func parseTimeString(timeStr string) (hour, minute, second int, err error) {
	parts := [3]int{}
	n, err := fmt.Sscanf(timeStr, "%d:%d:%d", &parts[0], &parts[1], &parts[2])
	if err != nil && n < 2 {
		return 0, 0, 0, fmt.Errorf("invalid time format %q", timeStr)
	}

	hour = parts[0]
	minute = parts[1]
	second = 0
	if n >= 3 && len(parts) > 2 {
		second = parts[2]
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 59 {
		return 0, 0, 0, fmt.Errorf("time out of range %q", timeStr)
	}

	return hour, minute, second, nil
}
