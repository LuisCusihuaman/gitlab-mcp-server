package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go" // GitLab client library
)

// GetProjectCommits defines the MCP tool for listing commits in a project.
func GetProjectCommits(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getProjectCommits",
			mcp.WithDescription("Retrieves a list of repository commits in a project, optionally filtered by ref, path, dates, and stats."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "List Project Commits",
				ReadOnlyHint: true,
			}),
			mcp.WithString("projectId",
				mcp.Required(),
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			),
			mcp.WithString("ref",
				mcp.Description("The name of a repository branch, tag or commit SHA. Default: the repository's default branch."),
			),
			mcp.WithString("path",
				mcp.Description("The file path to retrieve commits for."),
			),
			mcp.WithString("since", // GitLab API expects time as string, use OptionalTimeParam helper
				mcp.Description("Only commits after or on this date are returned (YYYY-MM-DDTHH:MM:SSZ). Format: ISO 8601"),
			),
			mcp.WithString("until", // GitLab API expects time as string, use OptionalTimeParam helper
				mcp.Description("Only commits before or on this date are returned (YYYY-MM-DDTHH:MM:SSZ). Format: ISO 8601"),
			),
			mcp.WithBoolean("withStats",
				mcp.Description("Include commit stats (additions, deletions). Default is false."),
			),
			// Add standard MCP pagination parameters
			WithPagination(),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse parameters
			projectIDStr, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			refName, err := OptionalParam[string](&request, "ref")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			path, err := OptionalParam[string](&request, "path")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			since, err := OptionalTimeParam(&request, "since") // Use time helper
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			until, err := OptionalTimeParam(&request, "until") // Use time helper
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			withStats, err := OptionalBoolParam(&request, "withStats")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			page, perPage, err := OptionalPaginationParams(&request)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Construct GitLab API options
			opts := &gl.ListCommitsOptions{
				ListOptions: gl.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
				Since:     since,     // Assign *time.Time pointer directly
				Until:     until,     // Assign *time.Time pointer directly
				WithStats: withStats, // Correct field name is WithStats
			}
			if refName != "" {
				opts.RefName = &refName
			}
			if path != "" {
				opts.Path = &path
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitLab client: %w", err)
			}

			// --- Call GitLab API
			commits, resp, err := glClient.Commits.ListCommits(projectIDStr, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					msg := fmt.Sprintf("project %q not found or access denied (%d)", projectIDStr, code)
					return mcp.NewToolResultError(msg), nil
				}
				return nil, fmt.Errorf("failed to list commits for project %q: %w (status: %d)", projectIDStr, err, code)
			}

			// --- Marshal and return success
			// Handle empty list gracefully
			if len(commits) == 0 {
				return mcp.NewToolResultText("[]"), nil // Return empty JSON array
			}

			data, err := json.Marshal(commits)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal commit list data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}
