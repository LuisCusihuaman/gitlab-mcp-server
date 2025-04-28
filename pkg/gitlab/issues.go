package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

// ListIssues defines the MCP tool for listing issues with filtering and pagination.
func ListIssues(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"listIssues",
			mcp.WithDescription("Retrieves a list of issues in a GitLab project with pagination and filtering."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "List GitLab Issues",
				ReadOnlyHint: true,
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
				mcp.Required(),
			),
			// Optional filtering parameters
			mcp.WithString("state",
				mcp.Description("Return issues with the specified state (opened, closed, all)."),
				mcp.Enum("opened", "closed", "all"),
			),
			mcp.WithString("labels",
				mcp.Description("Comma-separated list of label names to filter by."),
			),
			mcp.WithString("milestone",
				mcp.Description("Milestone title to filter by."),
			),
			mcp.WithString("scope",
				mcp.Description("Return issues for the given scope (created_by_me, assigned_to_me, all)."),
				mcp.Enum("created_by_me", "assigned_to_me", "all"),
			),
			mcp.WithNumber("authorId",
				mcp.Description("Return issues created by the given user ID (integer)."),
			),
			mcp.WithNumber("assigneeId",
				mcp.Description("Return issues assigned to the given user ID (integer)."),
			),
			mcp.WithString("search",
				mcp.Description("Search issues against their title and description."),
			),
			mcp.WithString("orderBy",
				mcp.Description("Return issues ordered by this field (created_at, updated_at, priority)."),
				mcp.Enum("created_at", "updated_at", "priority"),
			),
			mcp.WithString("sort",
				mcp.Description("Return issues sorted in asc or desc order."),
				mcp.Enum("asc", "desc"),
			),
			mcp.WithString("createdAfter",
				mcp.Description("Return issues created on or after the given time (ISO 8601 format: YYYY-MM-DDTHH:MM:SSZ)."),
			),
			mcp.WithString("createdBefore",
				mcp.Description("Return issues created on or before the given time (ISO 8601 format: YYYY-MM-DDTHH:MM:SSZ)."),
			),
			mcp.WithString("updatedAfter",
				mcp.Description("Return issues updated on or after the given time (ISO 8601 format: YYYY-MM-DDTHH:MM:SSZ)."),
			),
			mcp.WithString("updatedBefore",
				mcp.Description("Return issues updated on or before the given time (ISO 8601 format: YYYY-MM-DDTHH:MM:SSZ)."),
			),
			// Add standard MCP pagination parameters
			WithPagination(),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Get client using context
			client, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Parse required parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Parse optional filtering parameters
			state, err := OptionalParam[string](&request, "state")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			labels, err := OptionalParam[string](&request, "labels")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			milestone, err := OptionalParam[string](&request, "milestone")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			scope, err := OptionalParam[string](&request, "scope")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// Parse numeric parameters as float64 and convert to int
			authorIDFloat, err := OptionalParam[float64](&request, "authorId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			assigneeIDFloat, err := OptionalParam[float64](&request, "assigneeId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			search, err := OptionalParam[string](&request, "search")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			orderBy, err := OptionalParam[string](&request, "orderBy")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			sort, err := OptionalParam[string](&request, "sort")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// Date parameters are passed as strings in ISO 8601 format
			createdAfter, err := OptionalTimeParam(&request, "createdAfter")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			createdBefore, err := OptionalTimeParam(&request, "createdBefore")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			updatedAfter, err := OptionalTimeParam(&request, "updatedAfter")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			updatedBefore, err := OptionalTimeParam(&request, "updatedBefore")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Parse pagination parameters
			page, perPage, err := OptionalPaginationParams(&request)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Construct GitLab API options
			opts := &gl.ListProjectIssuesOptions{
				ListOptions: gl.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
			}

			// Only set the optional parameters if they were provided
			if state != "" {
				opts.State = &state
			}

			if labels != "" {
				// Convert comma-separated string to a slice of strings
				labelSlice := strings.Split(labels, ",")
				// Convert to gl.LabelOptions
				labelOpts := gl.LabelOptions(labelSlice)
				opts.Labels = &labelOpts
			}

			if milestone != "" {
				opts.Milestone = &milestone
			}

			if scope != "" {
				opts.Scope = &scope
			}

			// Only set the author and assignee IDs if they were provided
			if authorIDFloat != 0 {
				authorID := int(authorIDFloat)
				opts.AuthorID = &authorID
			}

			if assigneeIDFloat != 0 {
				assigneeID := int(assigneeIDFloat)
				opts.AssigneeID = &assigneeID
			}

			if search != "" {
				opts.Search = &search
			}

			if orderBy != "" {
				opts.OrderBy = &orderBy
			}

			if sort != "" {
				opts.Sort = &sort
			}

			// Set date fields if provided - these are pointers to time.Time
			opts.CreatedAfter = createdAfter
			opts.CreatedBefore = createdBefore
			opts.UpdatedAfter = updatedAfter
			opts.UpdatedBefore = updatedBefore

			// --- Call GitLab API
			issues, resp, err := client.Issues.ListProjectIssues(projectID, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					return mcp.NewToolResultError(fmt.Sprintf("project %q not found or access denied (%d)", projectID, code)), nil
				}
				// For other errors, return internal error
				return nil, fmt.Errorf("failed to list issues from project %q: %w (status: %d)", projectID, err, code)
			}

			// --- Handle empty result gracefully
			if len(issues) == 0 {
				return mcp.NewToolResultText("[]"), nil // Return empty JSON array
			}

			// --- Format successful response
			data, err := json.Marshal(issues)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal issues list: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}
