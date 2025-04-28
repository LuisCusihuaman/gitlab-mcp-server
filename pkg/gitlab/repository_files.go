package gitlab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	gl "gitlab.com/gitlab-org/api/client-go" // GitLab client library
)

// GetProjectFile defines the MCP tool for retrieving the content of a file in a project.
func GetProjectFile(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"getProjectFile",
			mcp.WithDescription("Retrieves the content of a specific file within a GitLab project repository."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "Get Project File Content",
				ReadOnlyHint: true,
			}),
			mcp.WithString("projectId",
				mcp.Required(),
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			),
			mcp.WithString("filePath",
				mcp.Required(),
				mcp.Description("The path to the file within the repository."),
			),
			mcp.WithString("ref",
				mcp.Description("The name of branch, tag, or commit SHA (defaults to the repository's default branch)."),
			),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse parameters
			projectIDStr, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			filePath, err := requiredParam[string](&request, "filePath")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			ref, err := OptionalParam[string](&request, "ref")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil // Should not happen with string?
			}

			// --- Construct GitLab API options
			opts := &gl.GetFileOptions{}
			if ref != "" {
				opts.Ref = &ref
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitLab client: %w", err)
			}

			// --- Call GitLab API
			file, resp, err := glClient.RepositoryFiles.GetFile(projectIDStr, filePath, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					// Could be project not found or file not found
					msg := fmt.Sprintf("project %q or file %q not found, or access denied (ref: %q) (%d)", projectIDStr, filePath, ref, code)
					return mcp.NewToolResultError(msg), nil
				}
				return nil, fmt.Errorf("failed to get file %q from project %q (ref: %q): %w (status: %d)", filePath, projectIDStr, ref, err, code)
			}

			// --- Decode Base64 content
			decodedContent, err := base64.StdEncoding.DecodeString(file.Content)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 content for file %q: %w", filePath, err)
			}

			// --- Return success
			// Consider returning metadata as well? For now, just content.
			return mcp.NewToolResultText(string(decodedContent)), nil
		}
}

// ListProjectFiles defines the MCP tool for listing files in a project directory.
func ListProjectFiles(getClient GetClientFn) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool(
			"listProjectFiles",
			mcp.WithDescription("Retrieves a list of files and directories within a specific path in a GitLab project repository."),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				Title:        "List Project Files/Directories",
				ReadOnlyHint: true,
			}),
			mcp.WithString("projectId",
				mcp.Required(),
				mcp.Description("The ID (integer) or URL-encoded path (string) of the project."),
			),
			mcp.WithString("path",
				mcp.Description("The path inside the repository. Used to list files in a subdirectory. Defaults to the root directory."),
			),
			mcp.WithString("ref",
				mcp.Description("The name of branch, tag, or commit SHA (defaults to the repository's default branch). Should be URL-encoded if it contains slashes."),
			),
			mcp.WithBoolean("recursive",
				mcp.Description("Flag indicating whether to list files recursively."),
			),
			// Add standard MCP pagination parameters for potentially large listings
			WithPagination(),
		),
		// Handler function implementation
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// --- Parse parameters
			projectIDStr, err := requiredParam[string](&request, "projectId")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			path, err := OptionalParam[string](&request, "path")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			ref, err := OptionalParam[string](&request, "ref")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			recursive, err := OptionalBoolParam(&request, "recursive")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}
			page, perPage, err := OptionalPaginationParams(&request)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Validation Error: %v", err)), nil
			}

			// --- Construct GitLab API options
			opts := &gl.ListTreeOptions{
				ListOptions: gl.ListOptions{
					Page:    page,
					PerPage: perPage,
				},
				Recursive: recursive, // Assign directly, pointers handled by OptionalBoolParam
			}
			if path != "" {
				opts.Path = &path
			}
			if ref != "" {
				opts.Ref = &ref
			}

			// --- Obtain GitLab client
			glClient, err := getClient(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitLab client: %w", err)
			}

			// --- Call GitLab API
			tree, resp, err := glClient.Repositories.ListTree(projectIDStr, opts, gl.WithContext(ctx))

			// --- Handle API errors
			if err != nil {
				code := http.StatusInternalServerError
				if resp != nil {
					code = resp.StatusCode
				}
				if code == http.StatusNotFound {
					// Could be project not found or path not found
					msg := fmt.Sprintf("project %q or path %q not found, or access denied (ref: %q) (%d)", projectIDStr, path, ref, code)
					return mcp.NewToolResultError(msg), nil
				}
				return nil, fmt.Errorf("failed to list repository tree for project %q (path: %q, ref: %q): %w (status: %d)", projectIDStr, path, ref, err, code)
			}

			// --- Marshal and return success
			// Handle empty list gracefully
			if len(tree) == 0 {
				return mcp.NewToolResultText("[]"), nil // Return empty JSON array
			}

			data, err := json.Marshal(tree)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal repository tree data: %w", err)
			}
			return mcp.NewToolResultText(string(data)), nil
		}
}
