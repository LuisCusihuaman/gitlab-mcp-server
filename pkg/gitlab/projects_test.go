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
	"go.uber.org/mock/gomock" // Import gomock

	"github.com/mark3labs/mcp-go/mcp"
	gl "gitlab.com/gitlab-org/api/client-go"                  // GitLab client library
	mock_gitlab "gitlab.com/gitlab-org/api/client-go/testing" // Correct import path for mocks
)

// Helper function to extract text content from the result, similar to github-mcp-server
func getTextResult(t *testing.T, result *mcp.CallToolResult) mcp.TextContent {
	t.Helper()
	require.NotNil(t, result, "getTextResult received nil result")
	require.NotEmpty(t, result.Content, "Expected result content, but got empty slice")
	content := result.Content[0] // Assume first content block is the relevant one

	textContent, ok := content.(mcp.TextContent)
	require.True(t, ok, "Expected result content to be of type TextContent, got %T", content)
	return textContent
}

// Helper to create a mock GetClientFn for testing handlers
func setupMockClient(t *testing.T) (*gl.Client, *mock_gitlab.MockProjectsServiceInterface, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockProjects := mock_gitlab.NewMockProjectsServiceInterface(ctrl) // Use correct mock type

	// Create a minimal client and attach the mock service
	// We don't need a real HTTP client for this type of mock
	client := &gl.Client{
		Projects: mockProjects,
		// Add other services here as needed for other tests
	}

	return client, mockProjects, ctrl
}

// Helper to create a mock GetClientFn for testing handlers
// Updated to return the specific mock type needed for file operations
func setupMockClientForFiles(t *testing.T) (*gl.Client, *mock_gitlab.MockRepositoryFilesServiceInterface, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockFiles := mock_gitlab.NewMockRepositoryFilesServiceInterface(ctrl) // Mock for files

	// Create a minimal client and attach the mock service
	client := &gl.Client{
		RepositoryFiles: mockFiles, // Attach the file service mock
		// Projects:        Needs mockProjects if testing combined scenarios
	}

	return client, mockFiles, ctrl
}

// Helper to create a mock GetClientFn for testing handlers
// Updated to return the specific mock type needed for repositories service
func setupMockClientForRepos(t *testing.T) (*gl.Client, *mock_gitlab.MockRepositoriesServiceInterface, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockRepos := mock_gitlab.NewMockRepositoriesServiceInterface(ctrl) // Mock for Repositories

	// Create a minimal client and attach the mock service
	client := &gl.Client{
		Repositories: mockRepos, // Attach the Repositories service mock
		// Add other service mocks here if testing combined scenarios
	}

	return client, mockRepos, ctrl
}

// Helper to create a mock GetClientFn for testing handlers for the Branches service
func setupMockClientForBranches(t *testing.T) (*gl.Client, *mock_gitlab.MockBranchesServiceInterface, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockBranches := mock_gitlab.NewMockBranchesServiceInterface(ctrl) // Mock for Branches

	// Create a minimal client and attach the mock service
	client := &gl.Client{
		Branches: mockBranches, // Attach the Branches service mock
	}

	return client, mockBranches, ctrl
}

func TestGetProjectHandler(t *testing.T) {
	ctx := context.Background()
	projectIDInt := 123
	projectIDStr := "123"          // Use string for ID in tests for consistency with API calls expecting 'any'
	projectPath := "group/project" // Example path

	mockClient, mockProjects, ctrl := setupMockClient(t)
	defer ctrl.Finish()

	// Create the mock getClient function
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler ---
	getProjectTool, getProjectHandler := GetProject(mockGetClient)

	// --- Test Cases ---
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     interface{} // Can be *gl.Project for success, or string for error message
		expectHandlerError bool        // Whether the handler itself should return an error
		expectResultError  bool        // Whether the returned mcp.CallToolResult should represent an error
		errorContains      string      // Substring to check in the actual error returned by handler
	}{
		{
			name: "Success - Get Project by ID",
			inputArgs: map[string]any{
				"projectId": projectIDStr,
			},
			mockSetup: func() {
				expectedProject := &gl.Project{
					ID:                projectIDInt,
					Name:              "Test Project",
					PathWithNamespace: projectPath,
				}
				mockProjects.EXPECT().
					GetProject(projectIDStr, gomock.Any(), gomock.Any()). // Expect string ID
					Return(expectedProject, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: &gl.Project{ // Store expected struct for success
				ID:                projectIDInt,
				Name:              "Test Project",
				PathWithNamespace: projectPath,
			},
			expectHandlerError: false,
			expectResultError:  false,
		},
		{
			name: "Success - Get Project by Path",
			inputArgs: map[string]any{
				"projectId": projectPath,
			},
			mockSetup: func() {
				expectedProject := &gl.Project{
					ID:                456,
					Name:              "Another Project",
					PathWithNamespace: projectPath,
				}
				mockProjects.EXPECT().
					GetProject(projectPath, gomock.Any(), gomock.Any()).
					Return(expectedProject, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: &gl.Project{
				ID:                456,
				Name:              "Another Project",
				PathWithNamespace: projectPath,
			},
			expectHandlerError: false,
			expectResultError:  false,
		},
		{
			name: "Error - Project Not Found (404)",
			inputArgs: map[string]any{
				"projectId": "nonexistent",
			},
			mockSetup: func() {
				mockProjects.EXPECT().
					GetProject("nonexistent", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Project Not Found"))
			},
			expectedResult:     "project \"nonexistent\" not found or access denied (404)",
			expectHandlerError: false, // Handler returns error within the result
			expectResultError:  true,  // The result itself represents an error
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"projectId": projectIDStr,
			},
			mockSetup: func() {
				mockProjects.EXPECT().
					GetProject(projectIDStr, gomock.Any(), gomock.Any()). // Expect string ID
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:     nil,  // No result content expected when handler errors
			expectHandlerError: true, // Handler returns an actual error
			expectResultError:  true, // Result is nil due to handler error
			errorContains:      "failed to get project \"123\": gitlab: 500 Internal Server Error",
		},
		{
			name:               "Error - Missing projectId parameter",
			inputArgs:          map[string]any{},
			mockSetup:          func() {},
			expectedResult:     "Validation Error: missing required parameter: projectId",
			expectHandlerError: false,
			expectResultError:  true,
		},
		{
			name: "Error - Invalid projectId type (e.g., non-string/int like)",
			// Note: The requiredParam helper now expects 'any'. The *handler* itself
			// might later fail if the GitLab client cannot handle the type,
			// but the parameter extraction itself might succeed if non-empty.
			// Let's adjust the test to check for empty string specifically,
			// as requiredParam likely checks for zero value.
			inputArgs: map[string]any{
				"projectId": "", // Test empty string specifically
			},
			mockSetup: func() {},
			// Exact error depends on requiredParam implementation details - Updated to match requiredParam
			expectedResult:     "Validation Error: required parameter 'projectId' cannot be empty or zero value",
			expectHandlerError: false,
			expectResultError:  true,
		},
	}

	// --- Run Tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()

			// Create the request using the correct structure
			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      getProjectTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Execute the handler
			result, err := getProjectHandler(ctx, req)

			// Assertions
			if tc.expectHandlerError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				textContent := getTextResult(t, result) // Use helper

				if tc.expectResultError {
					expectedErrString, ok := tc.expectedResult.(string)
					require.True(t, ok, "Expected error result should be a string")
					assert.Contains(t, textContent.Text, expectedErrString, "Error message mismatch")
					// TODO: Verify if result structure indicates error status (mcp-go might not have a dedicated flag)
				} else {
					// Check for successful result content by unmarshalling
					expectedProject, ok := tc.expectedResult.(*gl.Project)
					require.True(t, ok, "Expected success result should be a *gl.Project")

					var actualProject gl.Project
					err = json.Unmarshal([]byte(textContent.Text), &actualProject)
					require.NoError(t, err, "Failed to unmarshal success result JSON")

					// Compare relevant fields
					assert.Equal(t, expectedProject.ID, actualProject.ID, "Project ID mismatch")
					assert.Equal(t, expectedProject.Name, actualProject.Name, "Project Name mismatch")
					assert.Equal(t, expectedProject.PathWithNamespace, actualProject.PathWithNamespace, "Project PathWithNamespace mismatch")
					// Add more field assertions if necessary
				}
			}
		})
	}
}

// Add tests for ListProjects here later (Subtask 7.1)

func TestListProjectsHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockProjects, ctrl := setupMockClient(t)
	defer ctrl.Finish()

	// Create the mock getClient function
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler ---
	listProjectsTool, listProjectsHandler := ListProjects(mockGetClient)

	// --- Helper for Creating Expected Projects ---
	createMockProject := func(id int, name string) *gl.Project {
		return &gl.Project{ID: id, Name: name, PathWithNamespace: fmt.Sprintf("group/%s", name)}
	}

	// --- Test Cases ---
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()        // Mock setup no longer needs expectedOpts
		expectedResult     []*gl.Project // Expect a slice of projects for success
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name:      "Success - List Projects - No Filters",
			inputArgs: map[string]any{}, // No filters, default pagination
			mockSetup: func() {
				mockProjects.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).                                                                        // Use gomock.Any() for options
					DoAndReturn(func(opts *gl.ListProjectsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Project, *gl.Response, error) { // Use DoAndReturn, ignore rofs
						// Assertions on the actual options passed
						assert.Equal(t, 1, opts.Page, "Default page should be 1")
						assert.Equal(t, DefaultPerPage, opts.PerPage, "Default perPage should be used")
						assert.Nil(t, opts.Search)
						assert.Nil(t, opts.Owned)
						// Return the expected result for this specific call
						return []*gl.Project{createMockProject(1, "proj1"), createMockProject(2, "proj2")}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Project{createMockProject(1, "proj1"), createMockProject(2, "proj2")},
		},
		{
			name: "Success - List Projects - With Pagination",
			inputArgs: map[string]any{
				"page":     2,
				"per_page": 5, // Use per_page
			},
			mockSetup: func() {
				mockProjects.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gl.ListProjectsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Project, *gl.Response, error) {
						assert.Equal(t, 2, opts.Page)
						assert.Equal(t, 5, opts.PerPage)
						return []*gl.Project{createMockProject(6, "proj6")}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Project{createMockProject(6, "proj6")},
		},
		{
			name:      "Success - List Projects - With Search",
			inputArgs: map[string]any{"search": "foo"},
			mockSetup: func() {
				mockProjects.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gl.ListProjectsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Project, *gl.Response, error) {
						require.NotNil(t, opts.Search)
						assert.Equal(t, "foo", *opts.Search)
						assert.Equal(t, 1, opts.Page)
						assert.Equal(t, DefaultPerPage, opts.PerPage)
						return []*gl.Project{createMockProject(10, "foobar")}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Project{createMockProject(10, "foobar")},
		},
		{
			name:      "Success - List Projects - Empty Result",
			inputArgs: map[string]any{"search": "nonexistent"},
			mockSetup: func() {
				mockProjects.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gl.ListProjectsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Project, *gl.Response, error) {
						require.NotNil(t, opts.Search)
						assert.Equal(t, "nonexistent", *opts.Search)
						return []*gl.Project{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Project{}, // Expect empty slice
		},
		{
			name:      "Success - List Projects - With Boolean Filter (Owned)",
			inputArgs: map[string]any{"owned": true},
			mockSetup: func() {
				mockProjects.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gl.ListProjectsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Project, *gl.Response, error) {
						require.NotNil(t, opts.Owned)
						assert.True(t, *opts.Owned)
						assert.Nil(t, opts.Search)
						return []*gl.Project{createMockProject(5, "my-proj")}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Project{createMockProject(5, "my-proj")},
		},
		{
			name:      "Success - List Projects - With Visibility and Sort",
			inputArgs: map[string]any{"visibility": "private", "sort": "desc", "orderBy": "name"},
			mockSetup: func() {
				mockProjects.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					DoAndReturn(func(opts *gl.ListProjectsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Project, *gl.Response, error) {
						require.NotNil(t, opts.Visibility)
						assert.Equal(t, gl.VisibilityValue("private"), *opts.Visibility)
						require.NotNil(t, opts.Sort)
						assert.Equal(t, "desc", *opts.Sort)
						require.NotNil(t, opts.OrderBy)
						assert.Equal(t, "name", *opts.OrderBy)
						return []*gl.Project{createMockProject(9, "zebra"), createMockProject(8, "aardvark")}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Project{createMockProject(9, "zebra"), createMockProject(8, "aardvark")},
		},
		{
			name:      "Error - GitLab API Error (500)",
			inputArgs: map[string]any{},
			mockSetup: func() { // No need to pass/use expectedOpts here
				mockProjects.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()). // Match any options
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:     nil,
			expectHandlerError: true,
			errorContains:      "failed to list projects: gitlab: 500 Internal Server Error",
		},
		{
			name:               "Error - Invalid Page Type",
			inputArgs:          map[string]any{"page": "not-a-number"},
			mockSetup:          func() {}, // No mock call expected, validation fails first
			expectedResult:     nil,       // Expect error in result, not handler error
			expectResultError:  true,
			expectHandlerError: false,
			errorContains:      "Validation Error: invalid 'page' parameter: parameter 'page' must be a valid integer string", // Updated expected error to be more precise
		},
	}

	// --- Run Tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// --- Setup Mock ---
			// No longer need to create expectedOpts here
			tc.mockSetup()

			// --- Create Request ---
			req := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      listProjectsTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// --- Execute Handler ---
			result, err := listProjectsHandler(ctx, req)

			// --- Assertions ---
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
					var actualProjects []*gl.Project
					err = json.Unmarshal([]byte(textContent.Text), &actualProjects)
					require.NoError(t, err, "Failed to unmarshal actual result JSON")

					// Compare lengths first
					require.Equal(t, len(tc.expectedResult), len(actualProjects), "Number of projects mismatch")

					// Compare content (simple comparison, might need deep equal for complex structs)
					expectedJSON, _ := json.Marshal(tc.expectedResult)
					actualJSON, _ := json.Marshal(actualProjects)
					assert.JSONEq(t, string(expectedJSON), string(actualJSON), "Project list content mismatch")
				}
			}
		})
	}
}

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
					DoAndReturn(func(pid interface{}, opts *gl.ListTreeOptions, _ ...gl.RequestOptionFunc) ([]*gl.TreeNode, *gl.Response, error) {
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
					DoAndReturn(func(pid interface{}, opts *gl.ListTreeOptions, _ ...gl.RequestOptionFunc) ([]*gl.TreeNode, *gl.Response, error) {
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
					DoAndReturn(func(pid interface{}, opts *gl.ListTreeOptions, _ ...gl.RequestOptionFunc) ([]*gl.TreeNode, *gl.Response, error) {
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
					DoAndReturn(func(pid interface{}, opts *gl.ListTreeOptions, _ ...gl.RequestOptionFunc) ([]*gl.TreeNode, *gl.Response, error) {
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

// Add tests for GetProjectBranches here
func TestGetProjectBranchesHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockBranches, ctrl := setupMockClientForBranches(t)
	defer ctrl.Finish()

	// Mock getClient function for branch tests
	mockGetClientBranches := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler ---
	getProjectBranchesTool, getProjectBranchesHandler := GetProjectBranches(mockGetClientBranches)

	projectID := "group/project"
	searchQuery := "feat"

	// Helper to create mock Branches
	createBranch := func(name string, merged bool, protected bool) *gl.Branch {
		return &gl.Branch{
			Name:      name,
			Merged:    merged,
			Protected: protected,
			Commit: &gl.Commit{
				ID: fmt.Sprintf("commit-sha-for-%s", name),
			},
			// Add other relevant fields if needed
		}
	}

	// --- Test Cases ---
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     []*gl.Branch // Expecting a slice of branches
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - List Branches - No Filters",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListBranchesOptions{})
				mockBranches.EXPECT().
					ListBranches(projectID, expectedOptsMatcher, gomock.Any()).
					DoAndReturn(func(pid interface{}, opts *gl.ListBranchesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Branch, *gl.Response, error) {
						assert.Nil(t, opts.Search, "Search should be nil by default")
						assert.Equal(t, 1, opts.Page, "Default page")
						assert.Equal(t, DefaultPerPage, opts.PerPage, "Default perPage")
						return []*gl.Branch{
							createBranch("main", false, true),
							createBranch("develop", false, false),
						}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Branch{
				createBranch("main", false, true),
				createBranch("develop", false, false),
			},
		},
		{
			name: "Success - List Branches - With Search",
			inputArgs: map[string]any{
				"projectId": projectID,
				"search":    searchQuery,
			},
			mockSetup: func() {
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListBranchesOptions{})
				mockBranches.EXPECT().
					ListBranches(projectID, expectedOptsMatcher, gomock.Any()).
					DoAndReturn(func(pid interface{}, opts *gl.ListBranchesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Branch, *gl.Response, error) {
						require.NotNil(t, opts.Search)
						assert.Equal(t, searchQuery, *opts.Search)
						return []*gl.Branch{createBranch("feat/new-login", false, false)}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Branch{createBranch("feat/new-login", false, false)},
		},
		{
			name: "Success - List Branches - With Pagination",
			inputArgs: map[string]any{
				"projectId": projectID,
				"page":      2,
				"per_page":  1,
			},
			mockSetup: func() {
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListBranchesOptions{})
				mockBranches.EXPECT().
					ListBranches(projectID, expectedOptsMatcher, gomock.Any()).
					DoAndReturn(func(pid interface{}, opts *gl.ListBranchesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Branch, *gl.Response, error) {
						assert.Equal(t, 2, opts.Page)
						assert.Equal(t, 1, opts.PerPage)
						return []*gl.Branch{createBranch("release/v1.1", false, true)}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Branch{createBranch("release/v1.1", false, true)},
		},
		{
			name: "Success - Empty List",
			inputArgs: map[string]any{
				"projectId": projectID,
				"search":    "no-match-branch",
			},
			mockSetup: func() {
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListBranchesOptions{})
				mockBranches.EXPECT().
					ListBranches(projectID, expectedOptsMatcher, gomock.Any()).
					Return([]*gl.Branch{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Branch{}, // Expect empty slice
		},
		{
			name: "Error - Project Not Found (404)",
			inputArgs: map[string]any{
				"projectId": "nonexistent/project",
			},
			mockSetup: func() {
				mockBranches.EXPECT().
					ListBranches("nonexistent/project", gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Not Found"))
			},
			expectedResult:     nil,
			expectResultError:  true,
			expectHandlerError: false,
			errorContains:      fmt.Sprintf("project %q not found or access denied", "nonexistent/project"),
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockBranches.EXPECT().
					ListBranches(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:     nil,
			expectHandlerError: true,
			errorContains:      fmt.Sprintf("failed to list branches for project %q", projectID),
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"search": searchQuery},
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
					Name:      getProjectBranchesTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Execute the handler
			result, err := getProjectBranchesHandler(ctx, req)

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
					var actualBranches []*gl.Branch
					err = json.Unmarshal([]byte(textContent.Text), &actualBranches)
					require.NoError(t, err, "Failed to unmarshal actual result JSON")

					// Compare lengths first
					require.Equal(t, len(tc.expectedResult), len(actualBranches), "Number of branches mismatch")

					// Compare content using JSONEq for simplicity (adjust if complex fields need specific checks)
					expectedJSON, _ := json.Marshal(tc.expectedResult)
					actualJSON, _ := json.Marshal(actualBranches)
					assert.JSONEq(t, string(expectedJSON), string(actualJSON), "Branch list content mismatch")
				}
			}
		})
	}
}
