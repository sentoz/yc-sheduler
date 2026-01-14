// Package yc provides Yandex Cloud client functionality.
package yc

import (
	"context"
	"fmt"
	"sync"

	k8spb "github.com/yandex-cloud/go-genproto/yandex/cloud/k8s/v1"
	ycsdk "github.com/yandex-cloud/go-sdk/v2"
	"github.com/yandex-cloud/go-sdk/v2/credentials"
	"github.com/yandex-cloud/go-sdk/v2/pkg/options"
)

// Client wraps Yandex Cloud SDK and provides a narrow interface for
// higher-level components such as the scheduler.
type Client struct {
	sdk *ycsdk.SDK

	mu              sync.RWMutex
	nodeGroupPolicy map[string]*k8spb.ScalePolicy
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
		sdk:             sdk,
		nodeGroupPolicy: make(map[string]*k8spb.ScalePolicy),
	}, nil
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
