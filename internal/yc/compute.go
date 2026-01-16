package yc

import (
	"context"
	"fmt"
	"time"

	computepb "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	operationpb "github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// StartInstance starts a compute instance in the specified folder.
func (c *Client) StartInstance(ctx context.Context, folderID, instanceID string) error {
	if c == nil || c.sdk == nil {
		return fmt.Errorf("yc: client is not initialized")
	}

	conn, err := c.sdk.GetConnection(ctx, computepb.InstanceService_Start_FullMethodName)
	if err != nil {
		return fmt.Errorf("yc: get connection for start instance %s: %w", instanceID, err)
	}

	client := computepb.NewInstanceServiceClient(conn)

	op, err := client.Start(ctx, &computepb.StartInstanceRequest{
		InstanceId: instanceID,
	})
	if err != nil {
		return fmt.Errorf("yc: start instance %s: %w", instanceID, err)
	}

	return waitOperation(ctx, c.sdk, op.GetId())
}

// StopInstance stops a compute instance in the specified folder.
func (c *Client) StopInstance(ctx context.Context, folderID, instanceID string) error {
	if c == nil || c.sdk == nil {
		return fmt.Errorf("yc: client is not initialized")
	}

	conn, err := c.sdk.GetConnection(ctx, computepb.InstanceService_Stop_FullMethodName)
	if err != nil {
		return fmt.Errorf("yc: get connection for stop instance %s: %w", instanceID, err)
	}

	client := computepb.NewInstanceServiceClient(conn)

	op, err := client.Stop(ctx, &computepb.StopInstanceRequest{
		InstanceId: instanceID,
	})
	if err != nil {
		return fmt.Errorf("yc: stop instance %s: %w", instanceID, err)
	}

	return waitOperation(ctx, c.sdk, op.GetId())
}

// GetInstance retrieves the current state of a compute instance.
func (c *Client) GetInstance(ctx context.Context, folderID, instanceID string) (*computepb.Instance, error) {
	if c == nil || c.sdk == nil {
		return nil, fmt.Errorf("yc: client is not initialized")
	}

	// Use protoreflect.FullName as SDK v2 requires this format for endpoint resolution
	endpoint := protoreflect.FullName("yandex.cloud.compute.v1.InstanceService.Get")
	conn, err := c.sdk.GetConnection(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("yc: get connection for get instance %s: %w", instanceID, err)
	}

	client := computepb.NewInstanceServiceClient(conn)

	instance, err := client.Get(ctx, &computepb.GetInstanceRequest{
		InstanceId: instanceID,
	})
	if err != nil {
		return nil, fmt.Errorf("yc: get instance %s: %w", instanceID, err)
	}

	return instance, nil
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
