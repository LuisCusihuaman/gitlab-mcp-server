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

// GetMergeRequest defines the MCP tool for retrieving details of a specific merge request.
func GetMergeRequest(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getMergeRequest",
			mcp.WithDescription("Retrieves details for a specific GitLab merge request."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "Get GitLab Merge Request",
				ReadOnlyHint: true,
			}),
			mcp.WithString("projectId",
				mcp.Required(),
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			),
			mcp.WithNumber("mergeRequestIid",
				mcp.Required(),
				mcp.Description("The IID (internal ID, integer) of the merge request within the project."),
			),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse required parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			mrIidFloat, err := requiredParam[float64](&request, "mergeRequestIid")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			mrIid := int(mrIidFloat) // Convert float64 to int for API call
			// Check if conversion lost precision
			if float64(mrIid) != mrIidFloat {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: mergeRequestIid %v is not a valid integer", mrIidFloat)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitLab client: %w", err)
			}

			// --- Call GitLab API
			mr, resp, err := glClient.MergeRequests.GetMergeRequest(projectID, mrIid, nil, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					msg := fmt.Sprintf("merge request %d not found in project %q or access denied (%d)", mrIid, projectID, code)
					return mcp.NewToolResultError(msg), nil
				}
				return nil, fmt.Errorf("failed to get merge request %d from project %q: %w (status: %d)", mrIid, projectID, err, code)
			}

			// --- Marshal and return success
			data, err := json.Marshal(mr)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal merge request data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}

// GetMergeRequestComments defines the MCP tool for retrieving comments/notes for a specific merge request.
func GetMergeRequestComments(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getMergeRequestComments",
			mcp.WithDescription("Retrieves comments or notes from a specific merge request in a GitLab project."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "Get Merge Request Comments",
				ReadOnlyHint: true,
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(),
			),
			mcp.WithNumber("mergeRequestIid",
				mcp.Description("The IID (internal ID, integer) of the merge request within the project."),
				mcp.Required(),
			),
			// Add standard MCP pagination parameters
			WithPagination(),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			mrIidFloat, err := requiredParam[float64](&request, "mergeRequestIid")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			mrIid := int(mrIidFloat) // Convert float64 to int for API call
			// Check if conversion lost precision
			if float64(mrIid) != mrIidFloat {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: mergeRequestIid %v is not a valid integer", mrIidFloat)), nil
			}

			// Get pagination parameters
			page, perPage, err := OptionalPaginationParams(&request)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Construct GitLab API options
			opts := &gl.ListMergeRequestNotesOptions{
				ListOptions: gl.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
			}

			// --- Call GitLab API
			notes, resp, err := glClient.Notes.ListMergeRequestNotes(projectID, mrIid, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					msg := fmt.Sprintf("merge request %d not found in project %q or access denied (%d)", mrIid, projectID, code)
					return mcp.NewToolResultError(msg), nil
				}
				return nil, fmt.Errorf("failed to get comments for merge request %d from project %q: %w (status: %d)", mrIid, projectID, err, code)
			}

			// --- Marshal and return success
			// Handle empty list gracefully
			if len(notes) == 0 {
				return mcp.NewToolResultText("[]"), nil // Return empty JSON array
			}

			data, err := json.Marshal(notes)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal merge request comments data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}
