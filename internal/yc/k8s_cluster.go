package yc

import (
	"context"

	k8spb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// StartCluster starts the specified Kubernetes cluster.
func (c *Client) StartCluster(ctx context.Context, folderID, clusterID string) error {
	// Use protoreflect.FullName as SDK v2 requires this format for endpoint resolution
	endpoint := protoreflect.FullName("yandex.cloud.k8s.v1.ClusterService.Start")
	return executeOperation(ctx, c, endpoint, "start cluster", clusterID, func(ctx context.Context, conn grpc.ClientConnInterface) (string, error) {
		client := k8spb.NewClusterServiceClient(conn)
		op, err := client.Start(ctx, &k8spb.StartClusterRequest{
			ClusterId: clusterID,
		})
		if err != nil {
			return "", err
		}
		return op.GetId(), nil
	})
}

// StopCluster stops the specified Kubernetes cluster.
func (c *Client) StopCluster(ctx context.Context, folderID, clusterID string) error {
	// Use protoreflect.FullName as SDK v2 requires this format for endpoint resolution
	endpoint := protoreflect.FullName("yandex.cloud.k8s.v1.ClusterService.Stop")
	return executeOperation(ctx, c, endpoint, "stop cluster", clusterID, func(ctx context.Context, conn grpc.ClientConnInterface) (string, error) {
		client := k8spb.NewClusterServiceClient(conn)
		op, err := client.Stop(ctx, &k8spb.StopClusterRequest{
			ClusterId: clusterID,
		})
		if err != nil {
			return "", err
		}
		return op.GetId(), nil
	})
}

// GetCluster retrieves the current state of a Kubernetes cluster.
func (c *Client) GetCluster(ctx context.Context, folderID, clusterID string) (*k8spb.Cluster, error) {
	// Use protoreflect.FullName to specify the endpoint, as SDK v2 may require this format
	// Reference: https://github.com/yandex-cloud/go-sdk/blob/v2/services/k8s/v1/cluster.go
	endpoint := protoreflect.FullName("yandex.cloud.k8s.v1.ClusterService.Get")
	return getResource(ctx, c, endpoint, "get cluster", clusterID, func(ctx context.Context, conn grpc.ClientConnInterface) (*k8spb.Cluster, error) {
		client := k8spb.NewClusterServiceClient(conn)
		return client.Get(ctx, &k8spb.GetClusterRequest{
			ClusterId: clusterID,
		})
	})
}
