package yc

import "errors"

var (
	// ErrMissingCredentials is returned when no authentication credentials
	// are provided (neither service account key nor token).
	ErrMissingCredentials = errors.New("missing Yandex Cloud credentials")

	// ErrInstanceNotFound is returned when a requested compute instance
	// cannot be found in the specified folder.
	ErrInstanceNotFound = errors.New("instance not found")

	// ErrClusterNotFound is returned when a requested Kubernetes cluster
	// cannot be found in the specified folder.
	ErrClusterNotFound = errors.New("cluster not found")

	// ErrNodeGroupNotFound is returned when a requested Kubernetes node
	// group cannot be found in the specified folder.
	ErrNodeGroupNotFound = errors.New("node group not found")

	// ErrOperationFailed is returned when a long-running Yandex Cloud
	// operation finishes in a failed state.
	ErrOperationFailed = errors.New("operation failed")
)
