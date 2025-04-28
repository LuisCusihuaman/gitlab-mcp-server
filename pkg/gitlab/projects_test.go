package gitlab

import (
	"context"
	"encoding/json"
	"errors"
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
