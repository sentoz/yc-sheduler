package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSchedulesFromDirAndMultiDoc(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "config.yaml")
	schedulesDir := filepath.Join(tmpDir, "schedules")

	mustWriteFile(t, configPath, []byte(strings.TrimSpace(`
timezone: Europe/Moscow
max_concurrent_jobs: 5
validation_interval: 10m
shutdown_timeout: 5m
metrics_enabled: false
metrics_port: 9090
schedules_dir: ./schedules
`)))
	mustMkdirAll(t, schedulesDir)

	schedulePath := filepath.Join(schedulesDir, "multi.yaml")
	mustWriteFile(t, schedulePath, []byte(strings.TrimSpace(`
apiVersion: scheduler.yc/v1alpha1
kind: Schedule
metadata:
  name: vm-start
spec:
  type: daily
  resource:
    type: vm
    id: fhm1234567890abcdef
    folder_id: b1g1234567890abcdef
  actions:
    start:
      enabled: true
      time: 09:00
---
apiVersion: scheduler.yc/v1alpha1
kind: Schedule
metadata:
  name: vm-stop
spec:
  type: cron
  resource:
    type: vm
    id: fhm1234567890abcdef
    folder_id: b1g1234567890abcdef
  actions:
    stop:
      enabled: true
      crontab: 0 18 * * *
`)))

	cfg, err := Load(context.Background(), configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SchedulesDir != schedulesDir {
		t.Fatalf("SchedulesDir = %q, want %q", cfg.SchedulesDir, schedulesDir)
	}

	if len(cfg.Schedules) != 2 {
		t.Fatalf("len(Schedules) = %d, want 2", len(cfg.Schedules))
	}

	if cfg.Schedules[0].Name != "vm-start" {
		t.Fatalf("Schedules[0].Name = %q, want vm-start", cfg.Schedules[0].Name)
	}
	if cfg.Schedules[1].Name != "vm-stop" {
		t.Fatalf("Schedules[1].Name = %q, want vm-stop", cfg.Schedules[1].Name)
	}
}

func TestLoadSchedulesDuplicateNames(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "config.yaml")
	schedulesDir := filepath.Join(tmpDir, "schedules")

	mustWriteFile(t, configPath, []byte(strings.TrimSpace(`
timezone: Europe/Moscow
max_concurrent_jobs: 5
validation_interval: 10m
shutdown_timeout: 5m
metrics_enabled: false
metrics_port: 9090
schedules_dir: ./schedules
`)))
	mustMkdirAll(t, schedulesDir)

	mustWriteFile(t, filepath.Join(schedulesDir, "a.yaml"), []byte(strings.TrimSpace(`
apiVersion: scheduler.yc/v1alpha1
kind: Schedule
metadata:
  name: same-name
spec:
  type: daily
  resource:
    type: vm
    id: fhm1234567890abcdef
    folder_id: b1g1234567890abcdef
  actions:
    start:
      enabled: true
      time: 09:00
`)))
	mustWriteFile(t, filepath.Join(schedulesDir, "b.yaml"), []byte(strings.TrimSpace(`
apiVersion: scheduler.yc/v1alpha1
kind: Schedule
metadata:
  name: same-name
spec:
  type: daily
  resource:
    type: vm
    id: fhm1234567890abcdef
    folder_id: b1g1234567890abcdef
  actions:
    stop:
      enabled: true
      time: 18:00
`)))

	_, err := Load(context.Background(), configPath)
	if err == nil {
		t.Fatal("Load() error = nil, want duplicate schedule name error")
	}
	if !strings.Contains(err.Error(), "duplicate schedule name") {
		t.Fatalf("Load() error = %v, want duplicate schedule name", err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}
