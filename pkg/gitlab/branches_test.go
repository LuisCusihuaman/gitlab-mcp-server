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
	"go.uber.org/mock/gomock"

	"github.com/mark3labs/mcp-go/mcp"
	gl "gitlab.com/gitlab-org/api/client-go"
)

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
					DoAndReturn(func(_ interface{}, opts *gl.ListBranchesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Branch, *gl.Response, error) {
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
					DoAndReturn(func(_ interface{}, opts *gl.ListBranchesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Branch, *gl.Response, error) {
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
					DoAndReturn(func(_ interface{}, opts *gl.ListBranchesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Branch, *gl.Response, error) {
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
