package resource

import (
	"context"

	computepb "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	k8spb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"

	"github.com/sentoz/yc-sheduler/internal/config"
	"github.com/sentoz/yc-sheduler/internal/yc"
)

// StateChecker provides an interface for checking resource state.
type StateChecker interface {
	// GetState retrieves the current state of the resource.
	// Returns (state, isTransitional, error).
	// state: "running", "stopped", or a transitional state name
	// isTransitional: true if resource is in a transitional state
	GetState(ctx context.Context, resource config.Resource) (string, bool, error)
}

// YCStateChecker implements StateChecker using Yandex Cloud client.
type YCStateChecker struct {
	client *yc.Client
}

// NewYCStateChecker creates a new YCStateChecker.
func NewYCStateChecker(client *yc.Client) *YCStateChecker {
	return &YCStateChecker{client: client}
}

// GetState retrieves the current state of the resource.
func (c *YCStateChecker) GetState(ctx context.Context, resource config.Resource) (string, bool, error) {
	switch resource.Type {
	case "vm":
		return c.getVMState(ctx, resource)
	case "k8s_cluster":
		return c.getClusterState(ctx, resource)
	default:
		return "", false, nil
	}
}

func (c *YCStateChecker) getVMState(ctx context.Context, resource config.Resource) (string, bool, error) {
	instance, err := c.client.GetInstance(ctx, resource.FolderID, resource.ID)
	if err != nil {
		return "", false, err
	}
	status := instance.GetStatus()
	switch status {
	case computepb.Instance_RUNNING:
		return "running", false, nil
	case computepb.Instance_STOPPED:
		return "stopped", false, nil
	default:
		// Resource is in transitional state
		return status.String(), true, nil
	}
}

func (c *YCStateChecker) getClusterState(ctx context.Context, resource config.Resource) (string, bool, error) {
	cluster, err := c.client.GetCluster(ctx, resource.FolderID, resource.ID)
	if err != nil {
		return "", false, err
	}
	status := cluster.GetStatus()
	switch status {
	case k8spb.Cluster_RUNNING:
		return "running", false, nil
	case k8spb.Cluster_STOPPED:
		return "stopped", false, nil
	default:
		// Resource is in transitional state
		return status.String(), true, nil
	}
}
