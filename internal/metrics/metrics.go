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
)

// Init registers all Prometheus metrics used by the application.
func Init() {
	prometheus.MustRegister(operationsTotal)
}

// IncOperation increments the operations counter for the given
// resource type, action and status ("success", "error", "dry_run").
func IncOperation(resourceType, action, status string) {
	operationsTotal.WithLabelValues(resourceType, action, status).Inc()
}
