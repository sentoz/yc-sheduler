package yc

import (
	"context"
	"fmt"
	"time"

	computepb "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	operationpb "github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
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

// RestartInstance restarts a compute instance by issuing a stop followed
// by a start and waiting for both operations to complete.
func (c *Client) RestartInstance(ctx context.Context, folderID, instanceID string) error {
	if err := c.StopInstance(ctx, folderID, instanceID); err != nil {
		return fmt.Errorf("yc: restart instance %s: stop: %w", instanceID, err)
	}
	if err := c.StartInstance(ctx, folderID, instanceID); err != nil {
		return fmt.Errorf("yc: restart instance %s: start: %w", instanceID, err)
	}
	return nil
}

// waitOperation polls the Operation service until the operation with the
// given ID is completed or the context is canceled.
func waitOperation(ctx context.Context, sdk *ycsdk.SDK, operationID string) error {
	if operationID == "" {
		return fmt.Errorf("yc: %w: empty operation id", ErrOperationFailed)
	}

	conn, err := sdk.GetConnection(ctx, operationpb.OperationService_Get_FullMethodName)
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
