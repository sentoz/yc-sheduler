# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][],
and this project adheres to [Semantic Versioning][].

## [0.1.0][] - 2026-01-14

### Added

* Scheduler core based on configuration and gocron (VM, k8s_cluster, k8s_node_group)
* Yandex Cloud clients for compute instances, Kubernetes clusters and node groups
* Dry-run mode for safe verification of planned operations
* Prometheus metrics and HTTP server with `/metrics` and health-check endpoints
* Resource state validator with periodic execution
* Extended configuration:
  * `metrics_enabled`, `metrics_port`
  * `validation_interval`
  * `timezone`
  * `max_concurrent_jobs`
  * `shutdown_timeout`
* Service account authentication (key from file and inline JSON),
  environment variable support

### Changed

* Authentication: service account key is now recommended instead of long-lived tokens
* Updated documentation (`README.md`) with usage examples, configuration and monitoring

## [0.0.0][] - 2026-01-13

### Added

* Base project struct

[0.1.0]: https://github.com/WoozyMasta/yc-scheduler/tree/v0.1.0
[0.0.0]: https://github.com/WoozyMasta/yc-scheduler/tree/v0.0.0

<!--links-->
[Keep a Changelog]: https://keepachangelog.com/en/1.1.0/
[Semantic Versioning]: https://semver.org/spec/v2.0.0.html
