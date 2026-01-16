// Package yc provides Yandex Cloud client functionality.
package yc

import (
	"context"
	"fmt"
	"strings"

	computepb "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	"github.com/yandex-cloud/go-sdk/v2/credentials"
	"github.com/yandex-cloud/go-sdk/v2/pkg/options"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Client wraps Yandex Cloud SDK and provides a narrow interface for
// higher-level components such as the scheduler.
type Client struct {
	sdk *ycsdk.SDK
}

// AuthConfig describes how to authenticate against Yandex Cloud APIs.
// ServiceAccountKeyFile is the preferred method; Token is kept for
// backward compatibility and uses short-lived IAM/OAuth tokens.
type AuthConfig struct {
	// ServiceAccountKeyFile is a path to a service account key JSON file.
	// When set, SDK will automatically mint and refresh IAM tokens.
	ServiceAccountKeyFile string

	// Token is a pre-created IAM/OAuth token. This method is discouraged
	// because tokens are short-lived and require external rotation.
	Token string
}

// NewClient creates a new Yandex Cloud SDK client using the provided
// authentication configuration.
func NewClient(ctx context.Context, auth AuthConfig) (*Client, error) {
	var creds credentials.Credentials

	switch {
	case auth.ServiceAccountKeyFile != "":
		var err error
		creds, err = credentials.ServiceAccountKeyFile(auth.ServiceAccountKeyFile)
		if err != nil {
			return nil, fmt.Errorf("yc: load service account key file: %w", err)
		}
	case auth.Token != "":
		creds = credentials.OAuthToken(auth.Token)
	default:
		return nil, fmt.Errorf("yc: %w", ErrMissingCredentials)
	}

	sdk, err := ycsdk.Build(ctx, options.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("yc: build SDK: %w", err)
	}

	return &Client{
		sdk: sdk,
	}, nil
}

// ValidateCredentials checks if the current credentials are valid by attempting
// to get a connection to Compute service, which requires authentication. This verifies
// that the token/SA key is valid and not expired.
func (c *Client) ValidateCredentials(ctx context.Context) error {
	if err := c.ensureInitialized(); err != nil {
		return err
	}

	// Try to get a connection to Compute service, which requires authentication.
	// GetConnection will attempt to obtain an IAM token using the provided credentials.
	// If credentials are invalid or expired, this will fail.
	// Use protoreflect.FullName as SDK v2 may require this format for endpoint resolution.
	endpoint := protoreflect.FullName("yandex.cloud.compute.v1.InstanceService.List")
	conn, err := c.sdk.GetConnection(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("yc: %w: %v", ErrInvalidCredentials, err)
	}

	// Make a lightweight API call to verify the connection actually works.
	// We request an empty list from a non-existent folder, but the error should be
	// about the folder not found or permission denied, not about authentication.
	client := computepb.NewInstanceServiceClient(conn)
	_, err = client.List(ctx, &computepb.ListInstancesRequest{
		FolderId: "validation-check-folder-id",
		PageSize: 1,
	})
	if err != nil {
		// If error is about authentication/authorization, credentials are invalid.
		// If error is about folder not found or permission denied, that's expected
		// and credentials are valid (we just don't have access to that folder).
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "authorization") ||
			strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "invalid token") ||
			strings.Contains(errStr, "expired") || strings.Contains(errStr, "token") {
			return fmt.Errorf("yc: %w: %v", ErrInvalidCredentials, err)
		}
		// Any other error (like "folder not found", "permission denied" for that folder)
		// means credentials are valid, we just don't have access to that specific resource.
	}

	return nil
}

// ensureInitialized checks if the client is properly initialized.
// It returns ErrClientNotInitialized if the client or SDK is nil.
func (c *Client) ensureInitialized() error {
	if c == nil || c.sdk == nil {
		return fmt.Errorf("yc: %w", ErrClientNotInitialized)
	}
	return nil
}

// getConnection retrieves a gRPC connection for the specified endpoint.
// It handles error formatting consistently across all client methods.
func (c *Client) getConnection(ctx context.Context, endpoint protoreflect.FullName, operation, resourceID string) (grpc.ClientConnInterface, error) {
	conn, err := c.sdk.GetConnection(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("yc: get connection for %s %s: %w", operation, resourceID, err)
	}
	return conn, nil
}

// getResource is a generic helper function that encapsulates the common logic
// for Get operations (GetInstance, GetCluster, etc.).
// It handles initialization check, connection retrieval, and error formatting.
func getResource[T any](
	ctx context.Context,
	c *Client,
	endpoint protoreflect.FullName,
	operation, resourceID string,
	getFunc func(context.Context, grpc.ClientConnInterface) (T, error),
) (T, error) {
	var zero T
	if err := c.ensureInitialized(); err != nil {
		return zero, err
	}

	conn, err := c.getConnection(ctx, endpoint, operation, resourceID)
	if err != nil {
		return zero, err
	}

	result, err := getFunc(ctx, conn)
	if err != nil {
		return zero, fmt.Errorf("yc: %s %s: %w", operation, resourceID, err)
	}

	return result, nil
}

// executeOperation is a helper function that encapsulates the common logic
// for Start/Stop operations (StartInstance, StopInstance, StartCluster, StopCluster, etc.).
// It handles initialization check, connection retrieval, operation execution,
// and waiting for operation completion.
func executeOperation(
	ctx context.Context,
	c *Client,
	endpoint protoreflect.FullName,
	operation, resourceID string,
	opFunc func(context.Context, grpc.ClientConnInterface) (string, error),
) error {
	if err := c.ensureInitialized(); err != nil {
		return err
	}

	conn, err := c.getConnection(ctx, endpoint, operation, resourceID)
	if err != nil {
		return err
	}

	operationID, err := opFunc(ctx, conn)
	if err != nil {
		return fmt.Errorf("yc: %s %s: %w", operation, resourceID, err)
	}

	return waitOperation(ctx, c.sdk, operationID)
}

// Shutdown gracefully shuts down the underlying SDK, releasing any
// held resources and terminating background goroutines.
func (c *Client) Shutdown(ctx context.Context) error {
	if c == nil || c.sdk == nil {
		return nil
	}
	if err := c.sdk.Shutdown(ctx); err != nil {
		return fmt.Errorf("yc: shutdown SDK: %w", err)
	}
	return nil
}
