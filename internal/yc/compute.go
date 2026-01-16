package yc

import (
	"context"
	"fmt"
	"time"

	computepb "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	operationpb "github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// StartInstance starts a compute instance in the specified folder.
func (c *Client) StartInstance(ctx context.Context, folderID, instanceID string) error {
	// Use protoreflect.FullName as SDK v2 requires this format for endpoint resolution
	endpoint := protoreflect.FullName("yandex.cloud.compute.v1.InstanceService.Start")
	return executeOperation(ctx, c, endpoint, "start instance", instanceID, func(ctx context.Context, conn grpc.ClientConnInterface) (string, error) {
		client := computepb.NewInstanceServiceClient(conn)
		op, err := client.Start(ctx, &computepb.StartInstanceRequest{
			InstanceId: instanceID,
		})
		if err != nil {
			return "", err
		}
		return op.GetId(), nil
	})
}

// StopInstance stops a compute instance in the specified folder.
func (c *Client) StopInstance(ctx context.Context, folderID, instanceID string) error {
	// Use protoreflect.FullName as SDK v2 requires this format for endpoint resolution
	endpoint := protoreflect.FullName("yandex.cloud.compute.v1.InstanceService.Stop")
	return executeOperation(ctx, c, endpoint, "stop instance", instanceID, func(ctx context.Context, conn grpc.ClientConnInterface) (string, error) {
		client := computepb.NewInstanceServiceClient(conn)
		op, err := client.Stop(ctx, &computepb.StopInstanceRequest{
			InstanceId: instanceID,
		})
		if err != nil {
			return "", err
		}
		return op.GetId(), nil
	})
}

// GetInstance retrieves the current state of a compute instance.
func (c *Client) GetInstance(ctx context.Context, folderID, instanceID string) (*computepb.Instance, error) {
	// Use protoreflect.FullName as SDK v2 requires this format for endpoint resolution
	endpoint := protoreflect.FullName("yandex.cloud.compute.v1.InstanceService.Get")
	return getResource(ctx, c, endpoint, "get instance", instanceID, func(ctx context.Context, conn grpc.ClientConnInterface) (*computepb.Instance, error) {
		client := computepb.NewInstanceServiceClient(conn)
		return client.Get(ctx, &computepb.GetInstanceRequest{
			InstanceId: instanceID,
		})
	})
}

// waitOperation polls the Operation service until the operation with the
// given ID is completed or the context is canceled.
func waitOperation(ctx context.Context, sdk *ycsdk.SDK, operationID string) error {
	if operationID == "" {
		return fmt.Errorf("yc: %w: empty operation id", ErrOperationFailed)
	}

	// Use protoreflect.FullName as SDK v2 requires this format for endpoint resolution
	endpoint := protoreflect.FullName("yandex.cloud.operation.OperationService.Get")
	conn, err := sdk.GetConnection(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("yc: get connection for operation %s: %w", operationID, err)
	}

	client := operationpb.NewOperationServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			op, err := client.Get(ctx, &operationpb.GetOperationRequest{OperationId: operationID})
			if err != nil {
				return fmt.Errorf("yc: get operation %s: %w", operationID, err)
			}

			if !op.GetDone() {
				continue
			}

			if op.GetError() != nil {
				return fmt.Errorf("yc: %w: %s", ErrOperationFailed, op.GetError().GetMessage())
			}

			return nil
		}
	}
}
