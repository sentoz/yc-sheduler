package config

// Config represents the main application configuration.
type Config struct {
	// Schedules contains all scheduled tasks.
	Schedules []Schedule `yaml:"schedules" json:"schedules" jsonschema:"minItems=1"`
}

// Schedule defines a scheduled task for managing cloud resources.
type Schedule struct {
	// Name is a unique identifier for the schedule.
	Name string `yaml:"name" json:"name" default:"" jsonschema:"minLength=1,example=vm-production-start"`

	// Type specifies the schedule type (cron, daily, weekly, monthly, duration, one-time).
	Type string `yaml:"type" json:"type" default:"" jsonschema:"enum=cron,enum=daily,enum=weekly,enum=monthly,enum=duration,enum=one-time,example=daily"`

	// Resource defines the target resource to manage.
	Resource Resource `yaml:"resource" json:"resource"`

	// Actions defines what actions to perform at scheduled times.
	Actions Actions `yaml:"actions" json:"actions"`

	// CronJob configuration (used when Type is "cron").
	CronJob *CronJobConfig `yaml:"cron_job,omitempty" json:"cron_job,omitempty"`

	// DailyJob configuration (used when Type is "daily").
	DailyJob *DailyJobConfig `yaml:"daily_job,omitempty" json:"daily_job,omitempty"`

	// WeeklyJob configuration (used when Type is "weekly").
	WeeklyJob *WeeklyJobConfig `yaml:"weekly_job,omitempty" json:"weekly_job,omitempty"`

	// MonthlyJob configuration (used when Type is "monthly").
	MonthlyJob *MonthlyJobConfig `yaml:"monthly_job,omitempty" json:"monthly_job,omitempty"`

	// DurationJob configuration (used when Type is "duration").
	DurationJob *DurationJobConfig `yaml:"duration_job,omitempty" json:"duration_job,omitempty"`

	// OneTimeJob configuration (used when Type is "one-time").
	OneTimeJob *OneTimeJobConfig `yaml:"one_time_job,omitempty" json:"one_time_job,omitempty"`
}

// Resource defines a cloud resource to manage.
type Resource struct {
	// Type specifies the resource type (vm, k8s_cluster, k8s_node_group).
	Type string `yaml:"type" json:"type" default:"" jsonschema:"enum=vm,enum=k8s_cluster,enum=k8s_node_group,example=vm"`

	// ID is the resource identifier in Yandex Cloud.
	ID string `yaml:"id" json:"id" default:"" jsonschema:"minLength=1,example=fhm1234567890abcdef"`

	// FolderID is the Yandex Cloud folder ID containing the resource.
	FolderID string `yaml:"folder_id" json:"folder_id" default:"" jsonschema:"minLength=1,example=b1g1234567890abcdef"`
}

// Actions defines what actions to perform on the resource.
type Actions struct {
	// Start defines when to start the resource.
	Start *ActionConfig `yaml:"start,omitempty" json:"start,omitempty"`

	// Stop defines when to stop the resource.
	Stop *ActionConfig `yaml:"stop,omitempty" json:"stop,omitempty"`

	// Restart defines when to restart the resource.
	Restart *ActionConfig `yaml:"restart,omitempty" json:"restart,omitempty"`
}

// ActionConfig defines configuration for a specific action.
type ActionConfig struct {
	// Enabled indicates whether this action is enabled.
	Enabled bool `yaml:"enabled" json:"enabled" jsonschema:"example=true"`

	// Time specifies the time to perform the action (for time-based schedules).
	Time Time `yaml:"time,omitempty" json:"time,omitempty"`
}

// CronJobConfig defines configuration for a cron-based schedule.
type CronJobConfig struct {
	// Crontab is a cron expression (e.g., "0 9 * * *" for daily at 9 AM).
	Crontab Crontab `yaml:"crontab" json:"crontab" default:""`

	// Timezone specifies the timezone for the cron schedule.
	Timezone Timezone `yaml:"timezone,omitempty" json:"timezone,omitempty" default:"UTC"`
}

// DailyJobConfig defines configuration for a daily schedule.
type DailyJobConfig struct {
	// Time specifies the time of day (HH:MM or HH:MM:SS format).
	Time Time `yaml:"time" json:"time" default:""`

	// Timezone specifies the timezone for the schedule.
	Timezone Timezone `yaml:"timezone,omitempty" json:"timezone,omitempty" default:"UTC"`
}

// WeeklyJobConfig defines configuration for a weekly schedule.
type WeeklyJobConfig struct {
	// Day specifies the day of the week (0=Sunday, 1=Monday, ..., 6=Saturday).
	Day int `yaml:"day" json:"day" default:"0" jsonschema:"minimum=0,maximum=6,example=1"`

	// Time specifies the time of day (HH:MM or HH:MM:SS format).
	Time Time `yaml:"time" json:"time" default:""`

	// Timezone specifies the timezone for the schedule.
	Timezone Timezone `yaml:"timezone,omitempty" json:"timezone,omitempty" default:"UTC"`
}

// MonthlyJobConfig defines configuration for a monthly schedule.
type MonthlyJobConfig struct {
	// Day specifies the day of the month (1-31).
	Day int `yaml:"day" json:"day" default:"1" jsonschema:"minimum=1,maximum=31,example=1"`

	// Time specifies the time of day (HH:MM or HH:MM:SS format).
	Time Time `yaml:"time" json:"time" default:""`

	// Timezone specifies the timezone for the schedule.
	Timezone Timezone `yaml:"timezone,omitempty" json:"timezone,omitempty" default:"UTC"`
}

// DurationJobConfig defines configuration for a duration-based schedule.
type DurationJobConfig struct {
	// Duration specifies the interval duration (e.g., "5s", "1h", "30m").
	Duration Duration `yaml:"duration" json:"duration" jsonschema:"example=1h"`

	// StartTime specifies when to start the schedule (optional).
	StartTime RFC3339Time `yaml:"start_time,omitempty" json:"start_time,omitempty"`
}

// OneTimeJobConfig defines configuration for a one-time schedule.
type OneTimeJobConfig struct {
	// Time specifies when to execute the job (RFC3339 format).
	Time RFC3339Time `yaml:"time" json:"time" default:""`

	// Timezone specifies the timezone for the schedule.
	Timezone Timezone `yaml:"timezone,omitempty" json:"timezone,omitempty" default:"UTC"`
}
