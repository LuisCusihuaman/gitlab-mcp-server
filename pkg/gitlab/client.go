package gitlab

import (
	"context"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GetClientFn is a function type that returns an initialized GitLab client.
// Tool handlers will use this to get access to the client.
type GetClientFn func(ctx context.Context) (*gitlab.Client, error)

// NewClient initializes a new GitLab client based on the provided token and host.
func NewClient(token, host string) (*gitlab.Client, error) {
	if token == "" {
		return nil, fmt.Errorf("GitLab access token cannot be empty")
	}

	var opts []gitlab.ClientOptionFunc
	if host != "" {
		opts = append(opts, gitlab.WithBaseURL(host))
	}

	client, err := gitlab.NewClient(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Consider setting a custom User-Agent here if desired
	// client.UserAgent = "gitlab-mcp-server/..."

	return client, nil
}
