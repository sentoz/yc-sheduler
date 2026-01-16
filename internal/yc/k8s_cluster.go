package yc

import (
	"context"
	"fmt"

	k8spb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// StartCluster starts the specified Kubernetes cluster.
func (c *Client) StartCluster(ctx context.Context, folderID, clusterID string) error {
	if err := c.ensureInitialized(); err != nil {
		return err
	}

	// Use protoreflect.FullName as SDK v2 requires this format for endpoint resolution
	endpoint := protoreflect.FullName("yandex.cloud.k8s.v1.ClusterService.Start")
	conn, err := c.getConnection(ctx, endpoint, "start cluster", clusterID)
	if err != nil {
		return err
	}

	client := k8spb.NewClusterServiceClient(conn)

	op, err := client.Start(ctx, &k8spb.StartClusterRequest{
		ClusterId: clusterID,
	})
	if err != nil {
		return fmt.Errorf("yc: start cluster %s: %w", clusterID, err)
	}

	return waitOperation(ctx, c.sdk, op.GetId())
}

// StopCluster stops the specified Kubernetes cluster.
func (c *Client) StopCluster(ctx context.Context, folderID, clusterID string) error {
	if err := c.ensureInitialized(); err != nil {
		return err
	}

	// Use protoreflect.FullName as SDK v2 requires this format for endpoint resolution
	endpoint := protoreflect.FullName("yandex.cloud.k8s.v1.ClusterService.Stop")
	conn, err := c.getConnection(ctx, endpoint, "stop cluster", clusterID)
	if err != nil {
		return err
	}

	client := k8spb.NewClusterServiceClient(conn)

	op, err := client.Stop(ctx, &k8spb.StopClusterRequest{
		ClusterId: clusterID,
	})
	if err != nil {
		return fmt.Errorf("yc: stop cluster %s: %w", clusterID, err)
	}

	return waitOperation(ctx, c.sdk, op.GetId())
}

// GetCluster retrieves the current state of a Kubernetes cluster.
func (c *Client) GetCluster(ctx context.Context, folderID, clusterID string) (*k8spb.Cluster, error) {
	if err := c.ensureInitialized(); err != nil {
		return nil, err
	}

	// Use protoreflect.FullName to specify the endpoint, as SDK v2 may require this format
	// Reference: https://github.com/yandex-cloud/go-sdk/blob/v2/services/k8s/v1/cluster.go
	endpoint := protoreflect.FullName("yandex.cloud.k8s.v1.ClusterService.Get")
	conn, err := c.getConnection(ctx, endpoint, "get cluster", clusterID)
	if err != nil {
		return nil, err
	}

	client := k8spb.NewClusterServiceClient(conn)

	cluster, err := client.Get(ctx, &k8spb.GetClusterRequest{
		ClusterId: clusterID,
	})
	if err != nil {
		return nil, fmt.Errorf("yc: get cluster %s: %w", clusterID, err)
	}

	return cluster, nil
}
