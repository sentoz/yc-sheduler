package executor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sentoz/yc-sheduler/internal/config"
)

type lockTestStateChecker struct{}

func (lockTestStateChecker) GetState(context.Context, config.Resource) (string, bool, error) {
	return "stopped", false, nil
}

type lockTestOperator struct {
	mu         sync.Mutex
	startCalls int
}

func (o *lockTestOperator) Start(context.Context, config.Resource) error {
	time.Sleep(120 * time.Millisecond)
	o.mu.Lock()
	o.startCalls++
	o.mu.Unlock()
	return nil
}

func (o *lockTestOperator) Stop(context.Context, config.Resource) error {
	return nil
}

func (o *lockTestOperator) calls() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.startCalls
}

func TestMake_SkipsWhenSameResourceActionAlreadyInFlight(t *testing.T) {
	t.Parallel()

	operationLocks = newInFlightLocks()

	sch := config.Schedule{
		Name: "vm-start",
		Type: "daily",
		Resource: config.Resource{
			Type:     "vm",
			ID:       "vm-1",
			FolderID: "folder-1",
		},
		Actions: config.Actions{
			Start: &config.ActionConfig{Enabled: true, Time: "09:00"},
		},
	}

	checker := lockTestStateChecker{}
	op := &lockTestOperator{}

	job := Make(checker, op, sch, "start", false, nil)

	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		job()
	}()

	time.Sleep(20 * time.Millisecond)
	job()

	select {
	case <-firstDone:
	case <-time.After(1 * time.Second):
		t.Fatal("first job run did not finish in time")
	}

	if got := op.calls(); got != 1 {
		t.Fatalf("operator start calls = %d, want 1", got)
	}
}
