package scheduler

import "errors"

var (
	// ErrInvalidScheduleType is returned when a schedule has an unknown type.
	ErrInvalidScheduleType = errors.New("invalid schedule type")

	// ErrMissingJobConfig is returned when a schedule does not contain the
	// corresponding job configuration for its type.
	ErrMissingJobConfig = errors.New("missing job configuration for schedule")
)
