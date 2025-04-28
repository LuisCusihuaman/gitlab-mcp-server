package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// ListMergeRequests defines the MCP tool for listing merge requests with pagination and filtering.
func ListMergeRequests(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"listMergeRequests",
			mcp.WithDescription("Lists merge requests for a GitLab project with filtering and pagination options."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "List GitLab Merge Requests",
				ReadOnlyHint: true,
			}),
			// Required parameters
			mcp.WithString("projectId",
				mcp.Required(),
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			),
			// Optional filtering parameters
			mcp.WithString("state",
				mcp.Description("Return merge requests with the specified state ('opened', 'closed', 'locked', 'merged', or 'all'). Default: 'all'."),
				mcp.Enum("opened", "closed", "locked", "merged", "all"),
			),
			mcp.WithString("scope",
				mcp.Description("Return merge requests for the specified scope ('created_by_me', 'assigned_to_me', or 'all'). Default: 'all'."),
				mcp.Enum("created_by_me", "assigned_to_me", "all"),
			),
			mcp.WithString("author_id",
				mcp.Description("Return merge requests created by the specified user ID."),
			),
			mcp.WithString("assignee_id",
				mcp.Description("Return merge requests assigned to the specified user ID."),
			),
			mcp.WithString("labels",
				mcp.Description("Return merge requests matching the comma-separated list of labels."),
			),
			mcp.WithString("milestone",
				mcp.Description("Return merge requests for the specified milestone title."),
			),
			mcp.WithString("search",
				mcp.Description("Return merge requests matching the search query in their title or description."),
			),
			mcp.WithString("created_after",
				mcp.Description("Return merge requests created on or after the given datetime (ISO 8601 format)."),
			),
			mcp.WithString("created_before",
				mcp.Description("Return merge requests created on or before the given datetime (ISO 8601 format)."),
			),
			mcp.WithString("updated_after",
				mcp.Description("Return merge requests updated on or after the given datetime (ISO 8601 format)."),
			),
			mcp.WithString("updated_before",
				mcp.Description("Return merge requests updated on or before the given datetime (ISO 8601 format)."),
			),
			mcp.WithString("sort",
				mcp.Description("Return merge requests sorted in the specified order ('asc' or 'desc'). Default: 'desc'."),
				mcp.Enum("asc", "desc"),
			),
			mcp.WithString("order_by",
				mcp.Description("Return merge requests ordered by the specified field ('created_at', 'updated_at', or 'title'). Default: 'created_at'."),
				mcp.Enum("created_at", "updated_at", "title"),
			),
			// Add standard MCP pagination parameters
			WithPagination(),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse required parameters
			projectID, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize GitLab client: %w", err)
			}

			// --- Parse optional filtering parameters
			opts := &gl.ListProjectMergeRequestsOptions{}

			// Get pagination parameters
			page, perPage, err := OptionalPaginationParams(&request)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			opts.Page = page
			opts.PerPage = perPage

			// String parameters
			if state, err := OptionalParam[string](&request, "state"); err == nil && state != "" {
				opts.State = &state
			}

			if scope, err := OptionalParam[string](&request, "scope"); err == nil && scope != "" {
				opts.Scope = &scope
			}

			if authorID, err := OptionalParam[string](&request, "author_id"); err == nil && authorID != "" {
				// Convert string to int
				id, err := strconv.Atoi(authorID)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: author_id must be a valid integer: %v", err)), nil
				}
				opts.AuthorID = &id
			}

			if assigneeID, err := OptionalParam[string](&request, "assignee_id"); err == nil && assigneeID != "" {
				// Convert string to int and wrap with AssigneeID
				id, err := strconv.Atoi(assigneeID)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: assignee_id must be a valid integer: %v", err)), nil
				}
				opts.AssigneeID = gl.AssigneeID(id)
			}

			if labels, err := OptionalParam[string](&request, "labels"); err == nil && labels != "" {
				// Convert to LabelOptions ([]string)
				labelsList := strings.Split(labels, ",")
				labelOpts := gl.LabelOptions(labelsList)
				opts.Labels = &labelOpts
			}

			if milestone, err := OptionalParam[string](&request, "milestone"); err == nil && milestone != "" {
				opts.Milestone = &milestone
			}

			if search, err := OptionalParam[string](&request, "search"); err == nil && search != "" {
				opts.Search = &search
			}

			// Handle time parameters - parse ISO 8601 strings to time.Time
			if createdAfter, err := OptionalParam[string](&request, "created_after"); err == nil && createdAfter != "" {
				t, err := time.Parse(time.RFC3339, createdAfter)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: created_after must be a valid ISO 8601 datetime: %v", err)), nil
				}
				opts.CreatedAfter = &t
			}

			if createdBefore, err := OptionalParam[string](&request, "created_before"); err == nil && createdBefore != "" {
				t, err := time.Parse(time.RFC3339, createdBefore)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: created_before must be a valid ISO 8601 datetime: %v", err)), nil
				}
				opts.CreatedBefore = &t
			}

			if updatedAfter, err := OptionalParam[string](&request, "updated_after"); err == nil && updatedAfter != "" {
				t, err := time.Parse(time.RFC3339, updatedAfter)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: updated_after must be a valid ISO 8601 datetime: %v", err)), nil
				}
				opts.UpdatedAfter = &t
			}

			if updatedBefore, err := OptionalParam[string](&request, "updated_before"); err == nil && updatedBefore != "" {
				t, err := time.Parse(time.RFC3339, updatedBefore)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Validation Error: updated_before must be a valid ISO 8601 datetime: %v", err)), nil
				}
				opts.UpdatedBefore = &t
			}

			if sort, err := OptionalParam[string](&request, "sort"); err == nil && sort != "" {
				opts.Sort = &sort
			}

			if orderBy, err := OptionalParam[string](&request, "order_by"); err == nil && orderBy != "" {
				opts.OrderBy = &orderBy
			}

			// --- Call GitLab API
			mrs, resp, err := glClient.MergeRequests.ListProjectMergeRequests(projectID, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					msg := fmt.Sprintf("project %q not found or access denied (%d)", projectID, code)
					return mcp.NewToolResultError(msg), nil
				}
				return nil, fmt.Errorf("failed to list merge requests for project %q: %w (status: %d)", projectID, err, code)
			}

			// --- Marshal and return success
			// Handle empty list gracefully
			if len(mrs) == 0 {
				return mcp.NewToolResultText("[]"), nil // Return empty JSON array
			}

			data, err := json.Marshal(mrs)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal merge requests data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}
