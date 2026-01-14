package yc

import (
	"context"
	"fmt"

	k8spb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
)

// StartCluster starts the specified Kubernetes cluster.
func (c *Client) StartCluster(ctx context.Context, folderID, clusterID string) error {
	if c == nil || c.sdk == nil {
		return fmt.Errorf("yc: client is not initialized")
	}

	conn, err := c.sdk.GetConnection(ctx, k8spb.ClusterService_Start_FullMethodName)
	if err != nil {
		return fmt.Errorf("yc: get connection for start cluster %s: %w", clusterID, err)
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
	if c == nil || c.sdk == nil {
		return fmt.Errorf("yc: client is not initialized")
	}

	conn, err := c.sdk.GetConnection(ctx, k8spb.ClusterService_Stop_FullMethodName)
	if err != nil {
		return fmt.Errorf("yc: get connection for stop cluster %s: %w", clusterID, err)
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

// RestartCluster stops and then starts the specified Kubernetes cluster.
func (c *Client) RestartCluster(ctx context.Context, folderID, clusterID string) error {
	if err := c.StopCluster(ctx, folderID, clusterID); err != nil {
		return fmt.Errorf("yc: restart cluster %s: stop: %w", clusterID, err)
	}
	if err := c.StartCluster(ctx, folderID, clusterID); err != nil {
		return fmt.Errorf("yc: restart cluster %s: start: %w", clusterID, err)
	}
	return nil
}
