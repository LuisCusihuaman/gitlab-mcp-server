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

// GetProjectBranches defines the MCP tool for listing branches in a project.
func GetProjectBranches(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getProjectBranches",
			mcp.WithDescription("Retrieves a list of repository branches from a project, sorted by name alphabetically."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "List Project Branches",
				ReadOnlyHint: true,
			}),
			mcp.WithString("projectId",
				mcp.Required(),
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			),
			mcp.WithString("search",
				mcp.Description("Return list of branches matching the search criteria."),
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
			search, err := OptionalParam[string](&request, "search")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			page, perPage, err := OptionalPaginationParams(&request)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Construct GitLab API options
			opts := &gl.ListBranchesOptions{
				ListOptions: gl.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
			}
			if search != "" {
				opts.Search = &search
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitLab client: %w", err)
			}

			// --- Call GitLab API
			branches, resp, err := glClient.Branches.ListBranches(projectIDStr, opts, gl.WithContext(ctx))

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
				return nil, fmt.Errorf("failed to list branches for project %q: %w (status: %d)", projectIDStr, err, code)
			}

			// --- Marshal and return success
			// Handle empty list gracefully
			if len(branches) == 0 {
				return mcp.NewToolResultText("[]"), nil // Return empty JSON array
			}

			data, err := json.Marshal(branches)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal branch list data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}
