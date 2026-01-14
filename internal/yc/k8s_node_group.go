package yc

import (
	"context"
	"fmt"

	k8spb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
	"google.golang.org/protobuf/proto"
)

// StartNodeGroup scales the node group to the desired size using UpdateNodeGroup.
// The desiredSize parameter must be greater than zero.
func (c *Client) StartNodeGroup(ctx context.Context, folderID, nodeGroupID string, desiredSize int64) error {
	if c == nil || c.sdk == nil {
		return fmt.Errorf("yc: client is not initialized")
	}
	if desiredSize <= 0 {
		return fmt.Errorf("yc: invalid desired size %d for node group %s", desiredSize, nodeGroupID)
	}

	conn, err := c.sdk.GetConnection(ctx, k8spb.NodeGroupService_Update_FullMethodName)
	if err != nil {
		return fmt.Errorf("yc: get connection for start node group %s: %w", nodeGroupID, err)
	}

	client := k8spb.NewNodeGroupServiceClient(conn)

	// Restore previously saved scale policy if available, otherwise build
	// a new fixed scale policy using desiredSize.
	c.mu.RLock()
	policy := c.nodeGroupPolicy[nodeGroupID]
	c.mu.RUnlock()

	var scalePolicy *k8spb.ScalePolicy
	if policy != nil {
		cloned, ok := proto.Clone(policy).(*k8spb.ScalePolicy)
		if !ok {
			return fmt.Errorf("yc: start node group %s: failed to clone scale policy", nodeGroupID)
		}
		// If explicit size requested and policy has FixedScale, override size.
		if desiredSize > 0 {
			if fixed := cloned.GetFixedScale(); fixed != nil {
				fixed.Size = desiredSize
			} else {
				cloned.ScaleType = &k8spb.ScalePolicy_FixedScale_{
					FixedScale: &k8spb.ScalePolicy_FixedScale{
						Size: desiredSize,
					},
				}
			}
		}
		scalePolicy = cloned
	} else {
		scalePolicy = &k8spb.ScalePolicy{
			ScaleType: &k8spb.ScalePolicy_FixedScale_{
				FixedScale: &k8spb.ScalePolicy_FixedScale{
					Size: desiredSize,
				},
			},
		}
	}

	op, err := client.Update(ctx, &k8spb.UpdateNodeGroupRequest{
		NodeGroupId: nodeGroupID,
		ScalePolicy: scalePolicy,
		UpdateMask:  nil,
	})
	if err != nil {
		return fmt.Errorf("yc: update node group %s: %w", nodeGroupID, err)
	}

	return waitOperation(ctx, c.sdk, op.GetId())
}

// StopNodeGroup scales the node group down to zero nodes using UpdateNodeGroup.
func (c *Client) StopNodeGroup(ctx context.Context, folderID, nodeGroupID string) error {
	if c == nil || c.sdk == nil {
		return fmt.Errorf("yc: client is not initialized")
	}

	conn, err := c.sdk.GetConnection(ctx, k8spb.NodeGroupService_Update_FullMethodName)
	if err != nil {
		return fmt.Errorf("yc: get connection for stop node group %s: %w", nodeGroupID, err)
	}

	client := k8spb.NewNodeGroupServiceClient(conn)

	// First get current node group to preserve its scale policy fields if needed.
	ng, err := client.Get(ctx, &k8spb.GetNodeGroupRequest{
		NodeGroupId: nodeGroupID,
	})
	if err != nil {
		return fmt.Errorf("yc: get node group %s: %w", nodeGroupID, err)
	}

	// Save current scale policy for future StartNodeGroup.
	if ng.ScalePolicy != nil {
		c.mu.Lock()
		c.nodeGroupPolicy[nodeGroupID] = ng.ScalePolicy
		c.mu.Unlock()
	}

	op, err := client.Update(ctx, &k8spb.UpdateNodeGroupRequest{
		NodeGroupId: nodeGroupID,
		ScalePolicy: &k8spb.ScalePolicy{
			ScaleType: &k8spb.ScalePolicy_FixedScale_{
				FixedScale: &k8spb.ScalePolicy_FixedScale{
					Size: 0,
				},
			},
		},
		UpdateMask: nil,
	})
	if err != nil {
		return fmt.Errorf("yc: update node group %s to size 0: %w", nodeGroupID, err)
	}

	// Optionally, we could remember ng.ScalePolicy to restore it later in StartNodeGroup.
	_ = ng

	return waitOperation(ctx, c.sdk, op.GetId())
}

// RestartNodeGroup stops and then starts the specified node group.
func (c *Client) RestartNodeGroup(ctx context.Context, folderID, nodeGroupID string, desiredSize int64) error {
	if err := c.StopNodeGroup(ctx, folderID, nodeGroupID); err != nil {
		return fmt.Errorf("yc: restart node group %s: stop: %w", nodeGroupID, err)
	}
	if err := c.StartNodeGroup(ctx, folderID, nodeGroupID, desiredSize); err != nil {
		return fmt.Errorf("yc: restart node group %s: start: %w", nodeGroupID, err)
	}
	return nil
}
