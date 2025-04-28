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

// GetProject defines the MCP tool for retrieving a single GitLab project.
// Uses named return values to match the expected signature pattern.
// GetProject defines the MCP tool for retrieving a single GitLab project.
func GetProject(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getProject",
			mcp.WithDescription("Retrieves details for a specific GitLab project."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "Get Project Details",
				ReadOnlyHint: true,
			}),
			mcp.WithString("projectId",
				mcp.Required(),
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			),
		),

		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse parameters
			projectIDStr, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitLab client: %w", err)
			}

			// --- Call GitLab API
			// Pass the string value directly; the go-gitlab library handles it as interface{}
			project, resp, err := glClient.Projects.GetProject(projectIDStr, nil, gl.WithContext(ctx))

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
				return nil, fmt.Errorf("failed to get project %q: %w (status: %d)", projectIDStr, err, code)
			}

			// --- Marshal and return success
			data, err := json.Marshal(project)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal project data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		} // End handler func assignment
}
