# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][],
and this project adheres to [Semantic Versioning][].

## [0.1.0][] - 2026-01-14

### Added

* Scheduler core based on configuration and gocron/v2 (VM,
  k8s_cluster)
* Yandex Cloud clients for compute instances and Kubernetes
  clusters
* Support for multiple schedule types: cron, daily, weekly,
  monthly, duration, one-time
* Schedule parameters defined per action (`actions.start` and
  `actions.stop`), allowing different schedules for `start` and
  `stop` actions
* Dry-run mode for safe verification of planned operations
* Prometheus metrics and HTTP server with `/metrics` and
  health-check endpoints
* Resource state validator with periodic execution and automatic
  corrective actions
* Extended configuration:
  * `metrics_enabled`, `metrics_port`
  * `validation_interval`
  * `timezone` (global timezone for all schedules)
  * `max_concurrent_jobs`
  * `shutdown_timeout`
* Service account authentication (key from file) with automatic
  IAM token rotation
* OAuth/IAM token authentication (fallback, not recommended for
  long-running processes)
* Environment variable support for authentication
  (`YC_SA_KEY_FILE`, `YC_TOKEN`)
* Configurable logging with `--log-level` and `--log-format`
  flags
* Environment variable support for logging (`LOG_LEVEL`,
  `LOG_FORMAT`)
* Structured logging using `zerolog` with trace, debug, info,
  warn, error levels
* JSON Schema validation for configuration files
* Graceful shutdown with configurable timeout
* Credential validation at startup
* Centralized executor package for resource operations
* Signal handling package for graceful shutdown

### Changed

* Schedule parameters are defined in `actions.start` and
  `actions.stop` instead of separate `*_job` sections. Each
  action can have its own schedule parameters.
* Removed `restart` action support. Only `start` and `stop`
  actions are available.
* Removed per-job timezone configuration. All schedules use
  global timezone from `Config.Timezone` (or system timezone if
  not specified).
* Removed support for Kubernetes node groups (`k8s_node_group`).
  Only virtual machines (`vm`) and entire Kubernetes clusters
  (`k8s_cluster`) are supported.
* Configuration structure: schedule parameters (`time`, `day`,
  `crontab`, `duration`) are now defined within each action
  configuration, allowing different schedules for `start` and
  `stop`.
* Authentication: service account key is now recommended instead
  of long-lived tokens
* Default values are set using `github.com/creasty/defaults`
  package with struct tags
* Updated documentation (`README.md`) with usage examples,
  configuration and monitoring

## [0.0.0][] - 2026-01-13

### Added

* Base project struct

[0.1.0]: https://github.com/WoozyMasta/yc-scheduler/tree/v0.1.0
[0.0.0]: https://github.com/WoozyMasta/yc-scheduler/tree/v0.0.0

<!--links-->
[Keep a Changelog]: https://keepachangelog.com/en/1.1.0/
[Semantic Versioning]: https://semver.org/spec/v2.0.0.html
