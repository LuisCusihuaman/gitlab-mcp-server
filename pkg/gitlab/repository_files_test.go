package gitlab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mark3labs/mcp-go/mcp"
	gl "gitlab.com/gitlab-org/api/client-go"
)

func TestGetProjectFileHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockFiles, ctrl := setupMockClientForFiles(t)
	defer ctrl.Finish()

	// Mock getClient function for file tests
	mockGetClientFiles := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler ---
	getProjectFileTool, getProjectFileHandler := GetProjectFile(mockGetClientFiles)

	projectID := "group/project"
	filePath := "src/main.go"
	ref := "main"
	fileContentBase64 := base64.StdEncoding.EncodeToString([]byte("package main\n\nfunc main() {}\n"))
	fileContentDecoded := "package main\n\nfunc main() {}\n"

	// --- Test Cases ---
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     string // Expecting the decoded file content string
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Get File Content",
			inputArgs: map[string]any{
				"projectId": projectID,
				"filePath":  filePath,
				"ref":       ref,
			},
			mockSetup: func() {
				expectedOpts := &gl.GetFileOptions{Ref: &ref}
				mockFiles.EXPECT().
					GetFile(projectID, filePath, expectedOpts, gomock.Any()).
					Return(&gl.File{Content: fileContentBase64, FileName: filePath}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: fileContentDecoded,
		},
		{
			name: "Success - Get File Content - Default Ref",
			inputArgs: map[string]any{
				"projectId": projectID,
				"filePath":  filePath,
				// ref is omitted
			},
			mockSetup: func() {
				// Expect opts with Ref being nil (or pointer to empty string depending on OptionalParam)
				// Using AssignableToTypeOf is safer if nil vs empty string pointer matters
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.GetFileOptions{})
				mockFiles.EXPECT().
					GetFile(projectID, filePath, expectedOptsMatcher, gomock.Any()).
					// Check that Ref is nil or points to empty within DoAndReturn if needed
					Return(&gl.File{Content: fileContentBase64, FileName: filePath}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: fileContentDecoded,
		},
		{
			name: "Error - File Not Found (404)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"filePath":  "nonexistent.txt",
				"ref":       ref,
			},
			mockSetup: func() {
				expectedOpts := &gl.GetFileOptions{Ref: &ref}
				mockFiles.EXPECT().
					GetFile(projectID, "nonexistent.txt", expectedOpts, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 File Not Found"))
			},
			expectedResult:     "", // No content expected for error result
			expectResultError:  true,
			expectHandlerError: false,
			errorContains:      fmt.Sprintf("project %q or file %q not found", projectID, "nonexistent.txt"),
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"filePath":  filePath,
				"ref":       ref,
			},
			mockSetup: func() {
				expectedOpts := &gl.GetFileOptions{Ref: &ref}
				mockFiles.EXPECT().
					GetFile(projectID, filePath, expectedOpts, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:     "",
			expectHandlerError: true,
			errorContains:      fmt.Sprintf("failed to get file %q from project %q", filePath, projectID),
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"filePath": filePath, "ref": ref},
			mockSetup:          func() {},
			expectedResult:     "",
			expectResultError:  true,
			expectHandlerError: false,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name:               "Error - Missing filePath",
			inputArgs:          map[string]any{"projectId": projectID, "ref": ref},
			mockSetup:          func() {},
			expectedResult:     "",
			expectResultError:  true,
			expectHandlerError: false,
			errorContains:      "Validation Error: missing required parameter: filePath",
		},
	}

	// --- Run Tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			// Create the request
			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      getProjectFileTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Execute the handler
			result, err := getProjectFileHandler(ctx, req)

			// Assertions
			if tc.expectHandlerError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)

				if tc.expectResultError {
					assert.Contains(t, textContent.Text, tc.errorContains, "Error message mismatch")
				} else {
					assert.Equal(t, tc.expectedResult, textContent.Text, "Decoded file content mismatch")
				}
			}
		})
	}
}

// Add tests for ListProjectFiles here
func TestListProjectFilesHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockRepos, ctrl := setupMockClientForRepos(t)
	defer ctrl.Finish()

	// Mock getClient function for repo tests
	mockGetClientRepos := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler ---
	listProjectFilesTool, listProjectFilesHandler := ListProjectFiles(mockGetClientRepos)

	projectID := "group/project"
	path := "src/app"
	ref := "feature/new-ui"

	// Helper to create mock TreeNodes
	createTreeNode := func(id, name, nodeType, path string, mode string) *gl.TreeNode {
		return &gl.TreeNode{ID: id, Name: name, Type: nodeType, Path: path, Mode: mode}
	}

	// --- Test Cases ---
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     []*gl.TreeNode // Expecting a slice of tree nodes
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - List Files - Root Path, Default Ref",
			inputArgs: map[string]any{
				"projectId": projectID,
				// path and ref omitted
			},
			mockSetup: func() {
				// Expect opts with Path and Ref being nil (or pointer to empty string)
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListTreeOptions{})
				mockRepos.EXPECT().
					ListTree(projectID, expectedOptsMatcher, gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListTreeOptions, _ ...gl.RequestOptionFunc) ([]*gl.TreeNode, *gl.Response, error) {
						assert.Nil(t, opts.Path, "Path should be nil for root")
						assert.Nil(t, opts.Ref, "Ref should be nil for default")
						assert.Nil(t, opts.Recursive, "Recursive should be nil by default")
						assert.Equal(t, 1, opts.Page, "Default page")
						assert.Equal(t, DefaultPerPage, opts.PerPage, "Default perPage")
						return []*gl.TreeNode{
							createTreeNode("abc", "README.md", "blob", "README.md", "100644"),
							createTreeNode("def", "src", "tree", "src", "040000"),
						}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.TreeNode{
				createTreeNode("abc", "README.md", "blob", "README.md", "100644"),
				createTreeNode("def", "src", "tree", "src", "040000"),
			},
		},
		{
			name: "Success - List Files - Specific Path and Ref",
			inputArgs: map[string]any{
				"projectId": projectID,
				"path":      path,
				"ref":       ref,
			},
			mockSetup: func() {
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListTreeOptions{})
				mockRepos.EXPECT().
					ListTree(projectID, expectedOptsMatcher, gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListTreeOptions, _ ...gl.RequestOptionFunc) ([]*gl.TreeNode, *gl.Response, error) {
						require.NotNil(t, opts.Path)
						assert.Equal(t, path, *opts.Path)
						require.NotNil(t, opts.Ref)
						assert.Equal(t, ref, *opts.Ref)
						return []*gl.TreeNode{createTreeNode("123", "main.js", "blob", "src/app/main.js", "100644")}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.TreeNode{createTreeNode("123", "main.js", "blob", "src/app/main.js", "100644")},
		},
		{
			name: "Success - List Files - Recursive",
			inputArgs: map[string]any{
				"projectId": projectID,
				"recursive": true,
				"page":      1,
				"per_page":  10,
			},
			mockSetup: func() {
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListTreeOptions{})
				mockRepos.EXPECT().
					ListTree(projectID, expectedOptsMatcher, gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListTreeOptions, _ ...gl.RequestOptionFunc) ([]*gl.TreeNode, *gl.Response, error) {
						require.NotNil(t, opts.Recursive)
						assert.True(t, *opts.Recursive)
						assert.Equal(t, 1, opts.Page)
						assert.Equal(t, 10, opts.PerPage)
						return []*gl.TreeNode{
							createTreeNode("f1", "file1.txt", "blob", "file1.txt", "100644"),
							createTreeNode("f2", "file2.txt", "blob", "subdir/file2.txt", "100644"),
						}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.TreeNode{
				createTreeNode("f1", "file1.txt", "blob", "file1.txt", "100644"),
				createTreeNode("f2", "file2.txt", "blob", "subdir/file2.txt", "100644"),
			},
		},
		{
			name: "Success - Empty Directory",
			inputArgs: map[string]any{
				"projectId": projectID,
				"path":      "empty_dir",
			},
			mockSetup: func() {
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListTreeOptions{})
				mockRepos.EXPECT().
					ListTree(projectID, expectedOptsMatcher, gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListTreeOptions, _ ...gl.RequestOptionFunc) ([]*gl.TreeNode, *gl.Response, error) {
						require.NotNil(t, opts.Path)
						assert.Equal(t, "empty_dir", *opts.Path)
						return []*gl.TreeNode{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil // Return empty slice
					})
			},
			expectedResult: []*gl.TreeNode{}, // Expect empty slice
		},
		{
			name: "Error - Project or Path Not Found (404)",
			inputArgs: map[string]any{
				"projectId": projectID,
				"path":      "nonexistent/path",
			},
			mockSetup: func() {
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListTreeOptions{})
				mockRepos.EXPECT().
					ListTree(projectID, expectedOptsMatcher, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Not Found"))
			},
			expectedResult:     nil,
			expectResultError:  true,
			expectHandlerError: false,
			errorContains:      fmt.Sprintf("project %q or path %q not found", projectID, "nonexistent/path"),
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockRepos.EXPECT().
					ListTree(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:     nil,
			expectHandlerError: true,
			errorContains:      fmt.Sprintf("failed to list repository tree for project %q", projectID),
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"path": path},
			mockSetup:          func() {},
			expectedResult:     nil,
			expectResultError:  true,
			expectHandlerError: false,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
	}

	// --- Run Tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			// Create the request
			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      listProjectFilesTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Execute the handler
			result, err := listProjectFilesHandler(ctx, req)

			// Assertions
			if tc.expectHandlerError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result)

				if tc.expectResultError {
					assert.Contains(t, textContent.Text, tc.errorContains, "Error message mismatch")
				} else {
					// Unmarshal expected and actual results
					var actualTree []*gl.TreeNode
					err = json.Unmarshal([]byte(textContent.Text), &actualTree)
					require.NoError(t, err, "Failed to unmarshal actual result JSON")

					// Compare lengths first
					require.Equal(t, len(tc.expectedResult), len(actualTree), "Number of tree nodes mismatch")

					// Compare content using JSONEq for simplicity
					expectedJSON, _ := json.Marshal(tc.expectedResult)
					actualJSON, _ := json.Marshal(actualTree)
					assert.JSONEq(t, string(expectedJSON), string(actualJSON), "Tree node list content mismatch")
				}
			}
		})
	}
}
