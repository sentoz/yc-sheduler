# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][],
and this project adheres to [Semantic Versioning][].

## [0.1.0][] - 2026-01-14

### Added

* Core scheduler functionality based on gocron/v2 for managing
  Yandex Cloud resources
* Support for virtual machines (VM) and Kubernetes clusters
  (k8s_cluster) resource types
* Multiple schedule types:
  * `cron` - cron expression-based scheduling
  * `daily` - daily execution at specified time
  * `weekly` - weekly execution on specified weekday
  * `monthly` - monthly execution on specified day of month
  * `duration` - periodic execution with fixed interval
  * `one-time` - single execution at specified time
* Flexible action configuration: schedule parameters defined per
  action (`actions.start` and `actions.stop`), allowing different
  schedules for start and stop operations
* Yandex Cloud SDK v2 integration for Compute and Kubernetes
  services
* Resource operations:
  * Start and stop virtual machines
  * Start and stop Kubernetes clusters
  * Automatic operation status tracking and completion waiting
* Configuration management:
  * YAML and JSON configuration file support
  * Environment variable expansion via `jamle`
  * JSON Schema validation for configuration files
  * Default values using `github.com/creasty/defaults`
  * Global timezone configuration for all schedules
* Authentication methods:
  * Service account key file authentication (recommended) with
    automatic IAM token rotation
  * OAuth/IAM token authentication (fallback, not recommended for
    long-running processes)
  * Environment variable support (`YC_SA_KEY_FILE`, `YC_TOKEN`)
  * Credential validation at application startup
* Command-line interface:
  * Configuration file path specification (`--config`, `-c`)
  * Service account key file path (`--sa-key`)
  * OAuth/IAM token (`--token`, `-t`)
  * Dry-run mode (`--dry-run`, `-n`) for safe operation testing
  * Logging configuration flags (`--log-level`, `--log-format`)
* Structured logging with `zerolog`:
  * Multiple log levels: trace, debug, info, warn, error
  * JSON and console output formats
  * Environment variable support (`LOG_LEVEL`, `LOG_FORMAT`)
* Resource state validator:
  * Periodic resource state validation with configurable interval
  * Automatic expected state calculation based on schedule and
    current time
  * Last execution time comparison for start and stop actions
  * Automatic corrective action creation for state mismatches
  * Transitional state detection and validation skipping
* Prometheus metrics integration:
  * HTTP server for metrics and health endpoints
  * Metrics endpoint (`/metrics`) with operation counters
  * Health check endpoints (`/health/live`, `/health/ready`)
  * Build information endpoint (`/`) with JSON response
  * Operation metrics with labels: resource_type, action, status
* Graceful shutdown:
  * SIGINT and SIGTERM signal handling
  * Configurable shutdown timeout
  * Proper resource cleanup and connection closing
* Concurrency control:
  * Configurable maximum concurrent job execution limit
  * Job queue management with wait mode
* Extended configuration options:
  * `metrics_enabled` - enable/disable Prometheus metrics
  * `metrics_port` - HTTP server port for metrics
  * `validation_interval` - resource state validation interval
  * `timezone` - global timezone for all schedules
  * `max_concurrent_jobs` - maximum concurrent job limit
  * `shutdown_timeout` - graceful shutdown timeout
* Centralized executor package for resource operation execution
* Signal handling package for graceful application shutdown
* Build-time metadata injection (version, commit, build time,
  repository URL)

## [0.0.0][] - 2026-01-13

### Added

* Base project struct

[0.1.0]: https://github.com/WoozyMasta/yc-scheduler/tree/v0.1.0
[0.0.0]: https://github.com/WoozyMasta/yc-scheduler/tree/v0.0.0

<!--links-->
[Keep a Changelog]: https://keepachangelog.com/en/1.1.0/
[Semantic Versioning]: https://semver.org/spec/v2.0.0.html
