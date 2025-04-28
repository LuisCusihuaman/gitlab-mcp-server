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

// ListProjects defines the MCP tool for listing GitLab projects.
func ListProjects(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"listProjects",
			mcp.WithDescription("Retrieves a list of projects based on specified criteria."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "List Projects",
				ReadOnlyHint: true,
			}),
			// GitLab API ListProjectsOptions parameters
			mcp.WithString("search", mcp.Description("Return list of projects matching the search criteria.")),
			mcp.WithBoolean("membership", mcp.Description("Limit by projects that the current user is a member of.")),
			mcp.WithBoolean("owned", mcp.Description("Limit by projects owned by the current user.")),
			mcp.WithBoolean("starred", mcp.Description("Limit by projects starred by the current user.")),
			mcp.WithString("visibility",
				mcp.Description("Limit by visibility level."),
				mcp.Enum("public", "internal", "private"),
			),
			mcp.WithString("orderBy",
				mcp.Description("Return projects ordered by field."),
				mcp.Enum("id", "name", "path", "created_at", "updated_at", "last_activity_at"), // Add more if needed
			),
			mcp.WithString("sort",
				mcp.Description("Return projects sorted in asc or desc order."),
				mcp.Enum("asc", "desc"),
			),
			// Add standard MCP pagination parameters
			WithPagination(),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse parameters using helpers
			searchVal, err := OptionalParam[string](&request, "search")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			membershipVal, err := OptionalBoolParam(&request, "membership")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			ownedVal, err := OptionalBoolParam(&request, "owned")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			starredVal, err := OptionalBoolParam(&request, "starred")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			visibilityValStr, err := OptionalParam[string](&request, "visibility")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			orderByVal, err := OptionalParam[string](&request, "orderBy")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			sortVal, err := OptionalParam[string](&request, "sort")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			page, perPage, err := OptionalPaginationParams(&request)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Construct GitLab API options
			opts := &gl.ListProjectsOptions{
				ListOptions: gl.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
				// Assign pointers only if the value was actually provided
				Membership: membershipVal,
				Owned:      ownedVal,
				Starred:    starredVal,
			}

			if searchVal != "" {
				opts.Search = &searchVal
			}
			if visibilityValStr != "" {
				vis := gl.VisibilityValue(visibilityValStr)
				opts.Visibility = &vis
			}
			if orderByVal != "" {
				opts.OrderBy = &orderByVal
			}
			if sortVal != "" {
				opts.Sort = &sortVal
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitLab client: %w", err)
			}

			// --- Call GitLab API
			projects, resp, err := glClient.Projects.ListProjects(opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				// Don't return specific 404-like errors here, as an empty list is a valid result
				return nil, fmt.Errorf("failed to list projects: %w (status: %d)", err, code)
			}

			// --- Marshal and return success
			// Handle empty list gracefully
			if len(projects) == 0 {
				return mcp.NewToolResultText("[]"), nil // Return empty JSON array
			}

			data, err := json.Marshal(projects)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal project list data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}
