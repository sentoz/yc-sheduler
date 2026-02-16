package scheduler

import (
	"context"
	"testing"

	"github.com/sentoz/yc-sheduler/internal/config"
	"github.com/sentoz/yc-sheduler/internal/resource"
)

type testStateChecker struct{}

func (testStateChecker) GetState(context.Context, config.Resource) (string, bool, error) {
	return "stopped", false, nil
}

type testOperator struct{}

func (testOperator) Start(context.Context, config.Resource) error { return nil }
func (testOperator) Stop(context.Context, config.Resource) error  { return nil }

func TestReplaceSchedules_ReplacesManagedJobsOnly(t *testing.T) {
	t.Parallel()

	s, err := New("", 1)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	checker := testStateChecker{}
	op := testOperator{}

	cfg := &config.Config{
		Schedules: []config.Schedule{makeSchedule("old", "daily", true, false)},
	}

	if err := s.RegisterSchedules(checker, op, cfg, false, nil); err != nil {
		t.Fatalf("RegisterSchedules() error = %v", err)
	}

	if err := s.AddOneTimeJob("validator:keep", func() {}); err != nil {
		t.Fatalf("AddOneTimeJob() error = %v", err)
	}

	if got := len(s.s.Jobs()); got != 2 {
		t.Fatalf("jobs before replace = %d, want 2", got)
	}

	replacement := []config.Schedule{makeSchedule("new", "daily", false, true)}
	if err := s.ReplaceSchedules(checker, op, replacement, false, nil); err != nil {
		t.Fatalf("ReplaceSchedules() error = %v", err)
	}

	jobs := s.s.Jobs()
	if len(jobs) != 2 {
		t.Fatalf("jobs after replace = %d, want 2", len(jobs))
	}

	names := make(map[string]struct{}, len(jobs))
	for _, job := range jobs {
		names[job.Name()] = struct{}{}
	}

	if _, ok := names["old:start"]; ok {
		t.Fatal("old schedule job still present after replace")
	}
	if _, ok := names["new:stop"]; !ok {
		t.Fatal("new schedule job is missing after replace")
	}
	if _, ok := names["validator:keep"]; !ok {
		t.Fatal("one-time validator job should not be removed by replace")
	}
}

func makeSchedule(name, kind string, withStart, withStop bool) config.Schedule {
	sch := config.Schedule{
		Name: name,
		Type: kind,
		Resource: config.Resource{
			Type:     "vm",
			ID:       "id-1",
			FolderID: "folder-1",
		},
	}

	if withStart {
		sch.Actions.Start = &config.ActionConfig{Enabled: true, Time: "09:00"}
	}
	if withStop {
		sch.Actions.Stop = &config.ActionConfig{Enabled: true, Time: "18:00"}
	}

	return sch
}

var (
	_ resource.StateChecker = testStateChecker{}
	_ resource.Operator     = testOperator{}
)
