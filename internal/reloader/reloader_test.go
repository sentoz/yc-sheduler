package reloader

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestReloader_TriggersOnlyOnScheduleFileChanges(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	schedulePath := filepath.Join(dir, "a.yaml")
	if err := os.WriteFile(schedulePath, []byte("name: a\n"), 0o600); err != nil {
		t.Fatalf("write schedule: %v", err)
	}

	var reloadCalls atomic.Int32
	r, err := New(dir, 20*time.Millisecond, func(context.Context) error {
		reloadCalls.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.Start(ctx)
	}()

	time.Sleep(60 * time.Millisecond)

	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("ignore\n"), 0o600); err != nil {
		t.Fatalf("write non-schedule file: %v", err)
	}
	time.Sleep(80 * time.Millisecond)
	if got := reloadCalls.Load(); got != 0 {
		t.Fatalf("reload calls after non-yaml change = %d, want 0", got)
	}

	if err := os.WriteFile(schedulePath, []byte("name: b\n"), 0o600); err != nil {
		t.Fatalf("update schedule: %v", err)
	}

	deadline := time.Now().Add(700 * time.Millisecond)
	for reloadCalls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}

	if got := reloadCalls.Load(); got != 1 {
		t.Fatalf("reload calls after yaml change = %d, want 1", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("reloader did not stop after cancel")
	}
}
