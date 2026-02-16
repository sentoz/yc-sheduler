# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][],
and this project adheres to [Semantic Versioning][].

## _Untagged_

### Added

* Added Kubernetes-like schedule manifests (`apiVersion`, `kind`, `metadata`,
  `spec`) loaded from a directory configured by `schedules_dir`.
* Added support for multiple YAML documents in one schedule file using `---`.
* Added a dedicated schedule JSON schema at `static/schemas/schedule.json`.
* Added schedule examples in `examples/schedules/` and deployment examples in
  `deploy/schedules/`.

### Changed

* Main config now stores global settings plus `schedules_dir`; schedules are
  loaded from manifest files instead of inline `schedules:` list.
* Kubernetes deployment now mounts an additional ConfigMap with schedule files
  into `/schedules`.

## [0.1.3][] - 2026-01-22

### Fixed

* Fixed cron expression parsing in validator to support both 5-field (standard)
  and 6-field (with seconds) formats. Previously, the validator only accepted
  6-field cron expressions, causing errors for standard 5-field expressions like
  `0 8 * * 1-5`.

## [0.1.2][] - 2026-01-22

### Fixed

* Fixed timezone loading error in Docker containers by including IANA timezone
  database in the image.

## [0.1.1][] - 2026-01-22

### Added

* Kubernetes deployment manifests under `deploy/` and documentation for
  Kubernetes deployment in README.

### Fixed

* Fixed Docker container permission denied error by adding execute permissions
  to the binary in the Dockerfile using `--chmod=755`.

## [0.1.0][] - 2026-01-13

### Added

* Support for multiple schedule types: cron, daily, weekly, monthly
* Resource management for virtual machines (VM) and Kubernetes clusters
  * Start and stop operations for VMs
  * Start and stop operations for Kubernetes clusters
* State validator that periodically checks resource states and automatically
  corrects discrepancies
* Prometheus metrics and HTTP server with health check endpoints
* Graceful shutdown with configurable timeout
* JSON Schema validation for configuration files
* CLI interface with comprehensive flag support:
  * `--config` / `-c` - configuration file path (required)
  * `--sa-key` - service account key file path
  * `--token` / `-t` - OAuth/IAM token (discouraged)
  * `--dry-run` / `-n` - dry run mode
  * `--version` - print version information
  * `--log-level` - logging level (trace, debug, info, warn, error)
  * `--log-format` - logging format (json, console)
* Structured logging via zerolog with configurable levels and formats
* Environment variable support through jamle for configuration values
* Credentials validation at application startup
* Per-action schedule configuration allowing different schedules for start and
  stop actions
* Automatic skipping of operations for resources in transitional states
* Build information endpoint with version, commit, and build time

## [0.0.0][] - 2026-01-13

### Added

* Base project struct

[0.1.3]: https://github.com/sentoz/yc-sheduler/tree/v0.1.3
[0.1.2]: https://github.com/sentoz/yc-sheduler/tree/v0.1.2
[0.1.1]: https://github.com/sentoz/yc-sheduler/tree/v0.1.1
[0.1.0]: https://github.com/sentoz/yc-sheduler/tree/v0.1.0
[0.0.0]: https://github.com/sentoz/yc-sheduler/tree/v0.0.0

<!--links-->
[Keep a Changelog]: https://keepachangelog.com/en/1.1.0/
[Semantic Versioning]: https://semver.org/spec/v2.0.0.html
