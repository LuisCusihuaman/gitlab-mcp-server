package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go"
	// "github.com/your-org/gitlab-mcp-server/pkg/translations" // Uncomment if translations are ready
)

// func getIssueTool(getClient GetClientFn, t translations.TranslationHelperFunc) (mcp.Tool, server.ToolHandlerFunc) {
func GetIssue(getClient GetClientFn) (mcp.Tool, server.ToolHandlerFunc) { // Simplified for now
	return mcp.NewTool(
			"getIssue",

			mcp.WithDescription("Retrieves details for a specific GitLab issue."), // Plain text for now
			// Use WithString, WithNumber for parameters
			mcp.WithString("projectId",
				// t("mcp_gitlab_getIssue.projectId.description", "The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(), // Correct usage
			),
			mcp.WithNumber("issueIid", // Use WithNumber for integer types expected by API
				// t("mcp_gitlab_getIssue.issueIid.description", "The IID (internal ID, integer) of the issue within the project."),
				mcp.Description("The IID (internal ID, integer) of the issue within the project."),
				mcp.Required(), // Correct usage
			),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "Get GitLab Issue", // Add title
				ReadOnlyHint: true,
			}),
		),

		// Handler signature matches projects.go: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Get client using context
			client, err := getClient(ctx)
			if err != nil {
				// Return internal error using fmt.Errorf
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// Use type parameter and pass pointer to request for param helpers
			projectID, err := requiredParam[string](&req, "projectId")
			if err != nil {
				// Return user-facing error directly
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// Use WithNumber in tool definition, expect float64 here, then convert
			issueIidFloat, err := requiredParam[float64](&req, "issueIid")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			issueIid := int(issueIidFloat) // Convert float64 to int for API call
			// Check if conversion lost precision (optional but good practice)
			if float64(issueIid) != issueIidFloat {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: issueIid %v is not a valid integer", issueIidFloat)), nil
			}

			// Call GitLab API using alias 'gl' and passing context
			issue, resp, err := client.Issues.GetIssue(projectID, issueIid, nil, gl.WithContext(ctx))

			// Handle Errors (pattern from projects.go)
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					msg := fmt.Sprintf("issue %d not found in project %q or access denied (%d)", issueIid, projectID, code)
					// Return user-facing error directly
					return mcp.NewToolResultError(msg), nil
				}
				// Return internal error using fmt.Errorf
				return nil, fmt.Errorf("failed to get issue %d from project %q: %w (status code: %d)", issueIid, projectID, err, code)
			}

			// Format Success Response (pattern from projects.go)
			jsonData, err := json.Marshal(issue)
			if err != nil {
				// Return internal error using fmt.Errorf
				return nil, fmt.Errorf("failed to marshal issue data: %w", err)
			}
			// Use NewToolResultText
			return mcp.NewToolResultText(string(jsonData)), nil
		}
}

// Add other issue tool functions here later (e.g., ListIssues)
