package resource

import (
	"context"

	"github.com/woozymasta/yc-scheduler/internal/config"
	"github.com/woozymasta/yc-scheduler/internal/yc"
)

// Operator provides an interface for performing operations on resources.
type Operator interface {
	// Start starts the resource.
	Start(ctx context.Context, resource config.Resource) error
	// Stop stops the resource.
	Stop(ctx context.Context, resource config.Resource) error
}

// YCOperator implements Operator using Yandex Cloud client.
type YCOperator struct {
	client *yc.Client
}

// NewYCOperator creates a new YCOperator.
func NewYCOperator(client *yc.Client) *YCOperator {
	return &YCOperator{client: client}
}

// Start starts the resource.
func (o *YCOperator) Start(ctx context.Context, resource config.Resource) error {
	switch resource.Type {
	case "vm":
		return o.client.StartInstance(ctx, resource.FolderID, resource.ID)
	case "k8s_cluster":
		return o.client.StartCluster(ctx, resource.FolderID, resource.ID)
	default:
		return ErrUnsupportedResourceType
	}
}

// Stop stops the resource.
func (o *YCOperator) Stop(ctx context.Context, resource config.Resource) error {
	switch resource.Type {
	case "vm":
		return o.client.StopInstance(ctx, resource.FolderID, resource.ID)
	case "k8s_cluster":
		return o.client.StopCluster(ctx, resource.FolderID, resource.ID)
	default:
		return ErrUnsupportedResourceType
	}
}
