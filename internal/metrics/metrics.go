package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	operationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "yc_scheduler_operations_total",
			Help: "Total number of resource operations by type, action and status.",
		},
		[]string{"resource_type", "action", "status"},
	)

	validatorCorrectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "yc_scheduler_validator_corrections_total",
			Help: "Total number of corrective jobs created by validator to fix state mismatches.",
		},
		[]string{"resource_type", "action"},
	)

	schedulerSkipsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "yc_scheduler_scheduler_skips_total",
			Help: "Total number of scheduled job executions skipped (resource already in desired state or in transitional state).",
		},
		[]string{"resource_type", "action", "reason"},
	)
)

// Init registers all Prometheus metrics used by the application.
func Init() {
	prometheus.MustRegister(operationsTotal)
	prometheus.MustRegister(validatorCorrectionsTotal)
	prometheus.MustRegister(schedulerSkipsTotal)
}

// IncOperation increments the operations counter for the given
// resource type, action and status ("success", "error", "dry_run", "skipped").
func IncOperation(resourceType, action, status string) {
	operationsTotal.WithLabelValues(resourceType, action, status).Inc()
}

// IncValidatorCorrection increments the validator corrections counter for the given
// resource type and action.
func IncValidatorCorrection(resourceType, action string) {
	validatorCorrectionsTotal.WithLabelValues(resourceType, action).Inc()
}

// IncSchedulerSkip increments the scheduler skips counter for the given
// resource type, action and reason ("already_in_state", "transitional_state").
func IncSchedulerSkip(resourceType, action, reason string) {
	schedulerSkipsTotal.WithLabelValues(resourceType, action, reason).Inc()
}
