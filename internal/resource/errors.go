package resource

import "errors"

var (
	// ErrUnsupportedResourceType is returned when an operation is attempted
	// on an unsupported resource type.
	ErrUnsupportedResourceType = errors.New("unsupported resource type")
)
