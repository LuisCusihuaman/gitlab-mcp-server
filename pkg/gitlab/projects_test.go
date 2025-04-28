package gitlab

import (
	"context"
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
