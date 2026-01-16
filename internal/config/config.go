package config

// Config represents the main application configuration.
type Config struct {
	// Schedules contains all scheduled tasks.
	Schedules []Schedule `yaml:"schedules" json:"schedules" jsonschema:"minItems=1"`

	// MetricsEnabled toggles Prometheus metrics HTTP server.
	MetricsEnabled bool `yaml:"metrics_enabled,omitempty" json:"metrics_enabled,omitempty" default:"false" jsonschema:"default=false"`

	// MetricsPort defines the port for the metrics HTTP server.
	MetricsPort int `yaml:"metrics_port,omitempty" json:"metrics_port,omitempty" default:"9090" jsonschema:"default=9090"`

	// ValidationInterval defines how often the state validator runs.
	ValidationInterval Duration `yaml:"validation_interval,omitempty" json:"validation_interval,omitempty" default:"10m" jsonschema:"example=10m"`

	// Timezone specifies the timezone for schedules (IANA timezone name).
	// If empty, system timezone is used.
	Timezone Timezone `yaml:"timezone,omitempty" json:"timezone,omitempty" jsonschema:"example=Europe/Moscow"`

	// MaxConcurrentJobs limits the number of concurrent job executions.
	MaxConcurrentJobs int `yaml:"max_concurrent_jobs,omitempty" json:"max_concurrent_jobs,omitempty" default:"5" jsonschema:"default=5,minimum=1"`

	// ShutdownTimeout defines the timeout for graceful shutdown.
	ShutdownTimeout Duration `yaml:"shutdown_timeout,omitempty" json:"shutdown_timeout,omitempty" default:"5m" jsonschema:"example=5m"`
}

// Schedule defines a scheduled task for managing cloud resources.
type Schedule struct {

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

	// Resource defines the target resource to manage.
	Resource Resource `yaml:"resource" json:"resource"`

	// Name is a unique identifier for the schedule.
	Name string `yaml:"name" json:"name" default:"" jsonschema:"minLength=1,example=vm-production-start"`

	// Type specifies the schedule type (cron, daily, weekly, monthly).
	Type string `yaml:"type" json:"type" default:"" jsonschema:"enum=cron,enum=daily,enum=weekly,enum=monthly,example=daily"`
}

// Resource defines a cloud resource to manage.
type Resource struct {
	// Type specifies the resource type (vm, k8s_cluster).
	Type string `yaml:"type" json:"type" default:"" jsonschema:"enum=vm,enum=k8s_cluster,example=vm"`

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
}

// ActionConfig defines configuration for a specific action.
type ActionConfig struct {
	// Enabled indicates whether this action is enabled.
	Enabled bool `yaml:"enabled" json:"enabled" jsonschema:"example=true"`

	// Time specifies the time to perform the action.
	// For daily, weekly, monthly schedules: HH:MM or HH:MM:SS format (e.g., "09:00").
	Time string `yaml:"time,omitempty" json:"time,omitempty"`

	// Crontab is a cron expression for cron-based schedules (e.g., "0 9 * * *" for daily at 9 AM).
	Crontab Crontab `yaml:"crontab,omitempty" json:"crontab,omitempty"`

	// Day specifies the day of the week (0=Sunday, 1=Monday, ..., 6=Saturday) for weekly schedules,
	// or the day of the month (1-31) for monthly schedules.
	Day int `yaml:"day,omitempty" json:"day,omitempty" jsonschema:"example=1"`
}

// CronJobConfig defines configuration for a cron-based schedule.
// Deprecated: Parameters are now read from ActionConfig.
type CronJobConfig struct {
	// Crontab is a cron expression (e.g., "0 9 * * *" for daily at 9 AM).
	Crontab Crontab `yaml:"crontab" json:"crontab" default:""`
}

// DailyJobConfig defines configuration for a daily schedule.
// Deprecated: Parameters are now read from ActionConfig.
type DailyJobConfig struct {
	// Time specifies the time of day (HH:MM or HH:MM:SS format).
	Time Time `yaml:"time" json:"time" default:""`
}

// WeeklyJobConfig defines configuration for a weekly schedule.
// Deprecated: Parameters are now read from ActionConfig.
type WeeklyJobConfig struct {
	// Time specifies the time of day (HH:MM or HH:MM:SS format).
	Time Time `yaml:"time" json:"time" default:""`

	// Day specifies the day of the week (0=Sunday, 1=Monday, ..., 6=Saturday).
	Day int `yaml:"day" json:"day" default:"0" jsonschema:"minimum=0,maximum=6,example=1"`
}

// MonthlyJobConfig defines configuration for a monthly schedule.
// Deprecated: Parameters are now read from ActionConfig.
type MonthlyJobConfig struct {
	// Time specifies the time of day (HH:MM or HH:MM:SS format).
	Time Time `yaml:"time" json:"time" default:""`

	// Day specifies the day of the month (1-31).
	Day int `yaml:"day" json:"day" default:"1" jsonschema:"minimum=1,maximum=31,example=1"`
}
