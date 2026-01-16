package metrics

import "github.com/prometheus/client_golang/prometheus"

// Metrics holds all Prometheus metrics for the application.
type Metrics struct {
	operationsTotal           *prometheus.CounterVec
	validatorCorrectionsTotal *prometheus.CounterVec
	schedulerSkipsTotal       *prometheus.CounterVec
}

// New creates and registers a new Metrics instance.
func New() *Metrics {
	m := &Metrics{
		operationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "yc_scheduler_operations_total",
				Help: "Total number of resource operations by type, action and status.",
			},
			[]string{"resource_type", "action", "status"},
		),
		validatorCorrectionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "yc_scheduler_validator_corrections_total",
				Help: "Total number of corrective jobs created by validator to fix state mismatches.",
			},
			[]string{"resource_type", "action"},
		),
		schedulerSkipsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "yc_scheduler_scheduler_skips_total",
				Help: "Total number of scheduled job executions skipped (resource already in desired state or in transitional state).",
			},
			[]string{"resource_type", "action", "reason"},
		),
	}

	prometheus.MustRegister(m.operationsTotal)
	prometheus.MustRegister(m.validatorCorrectionsTotal)
	prometheus.MustRegister(m.schedulerSkipsTotal)

	return m
}

// IncOperation increments the operations counter for the given
// resource type, action and status ("success", "error", "dry_run", "skipped").
func (m *Metrics) IncOperation(resourceType, action, status string) {
	m.operationsTotal.WithLabelValues(resourceType, action, status).Inc()
}

// IncValidatorCorrection increments the validator corrections counter for the given
// resource type and action.
func (m *Metrics) IncValidatorCorrection(resourceType, action string) {
	m.validatorCorrectionsTotal.WithLabelValues(resourceType, action).Inc()
}

// IncSchedulerSkip increments the scheduler skips counter for the given
// resource type, action and reason ("already_in_state", "transitional_state").
func (m *Metrics) IncSchedulerSkip(resourceType, action, reason string) {
	m.schedulerSkipsTotal.WithLabelValues(resourceType, action, reason).Inc()
}
