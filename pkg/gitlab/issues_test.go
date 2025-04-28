package gitlab

import (
	"context"       // Needed for assertions
	"encoding/json" // Needed for assertions
	"errors"        // Added for creating mock errors
	"fmt"
	"net/http"

	// "net/http/httptest"
	// "net/url" // No longer needed for http server
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go" // GitLab client library

	// Gomock mocks
	"go.uber.org/mock/gomock" // Added for gomock
)

// mockGetClientFn is NOT defined locally, assumed provided by other tests or helpers (like setupMockClient*)
// getTextResult is assumed defined elsewhere in package gitlab_test

func TestGetIssueHandler(t *testing.T) {
	ctx := context.Background()

	// --- Setup Mock Client and GetClientFn once ---
	mockClient, mockIssues, ctrl := setupMockClientForIssues(t)
	defer ctrl.Finish()

	// Create the mock getClient function once, capturing the mockClient
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler once ---
	getIssueTool, handler := GetIssue(mockGetClient)

	// --- Test Cases ---
	tests := []struct {
		name                string
		projectID           string      // Can be string or int representation for API
		issueIid            float64     // MCP number type maps to float64
		mockSetup           func()      // Gomock setup (no args, uses mockIssues from outer scope)
		expectedResult      interface{} // Expect *gl.Issue for success, or string for user error message
		expectResultError   bool        // True if the returned mcp.CallToolResult should represent an error
		expectInternalError bool        // True if the handler itself should return a non-nil internal error
		errorContains       string      // Substring for internal error check ONLY
	}{
		// --- Success Case ---
		{
			name:      "Success - Get Issue by ID",
			projectID: "group/project",
			issueIid:  1.0,
			mockSetup: func() {
				expectedIssue := &gl.Issue{
					ID:          123,
					IID:         1,
					ProjectID:   456,
					Title:       "Test Issue",
					Description: "This is a test issue.",
				}
				mockIssues.EXPECT().
					GetIssue("group/project", 1, gomock.Any(), gomock.Any()).
					Return(expectedIssue, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: &gl.Issue{ // Store expected struct for success
				ID:          123,
				IID:         1,
				ProjectID:   456,
				Title:       "Test Issue",
				Description: "This is a test issue.",
			},
			expectResultError:   false,
			expectInternalError: false,
		},
		// --- User Error Cases ---
		{
			name:      "Error - Issue Not Found (404)",
			projectID: "group/project",
			issueIid:  999.0,
			mockSetup: func() {
				mockIssues.EXPECT().
					GetIssue("group/project", 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Issue Not Found"))
			},
			expectedResult:      "issue 999 not found in project \"group/project\" or access denied (404)", // Expect error message string
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Missing projectId parameter",
			projectID:           "", // Will be omitted from args
			issueIid:            1.0,
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: missing required parameter: projectId", // Expect error message string
			expectResultError:   true,
			expectInternalError: false,
		},
		{
			name:                "Error - Invalid issueIid (not integer)",
			projectID:           "group/project",
			issueIid:            1.5, // Non-integer float
			mockSetup:           func() { /* No API call expected */ },
			expectedResult:      "Validation Error: issueIid 1.5 is not a valid integer", // Expect error message string
			expectResultError:   true,
			expectInternalError: false,
		},
		// --- Internal Error Cases ---
		{
			name:      "Error - GitLab API Error (500)",
			projectID: "group/project",
			issueIid:  2.0,
			mockSetup: func() {
				mockIssues.EXPECT().
					GetIssue("group/project", 2, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:      nil,                                                                                       // No result content expected when handler errors
			expectResultError:   true,                                                                                      // Result is nil due to handler error
			expectInternalError: true,                                                                                      // Handler returns an actual error
			errorContains:       "failed to get issue 2 from project \"group/project\": gitlab: 500 Internal Server Error", // Check internal error here
		},
	}

	// Test case for Client Initialization Error (outside the loop)
	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		// Define a GetClientFn that returns an error
		errorGetClientFn := func(ctx context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := GetIssue(errorGetClientFn)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name:      getIssueTool.Name,
				Arguments: map[string]any{"projectId": "any", "issueIid": 1.0},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to initialize GitLab client: mock init error")
		assert.Nil(t, result)
	})

	// --- Run Loop for other tests ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectations for this specific test case
			tc.mockSetup()

			// Prepare request arguments
			args := map[string]any{}
			if tc.projectID != "" { // Only add if not empty for missing param test
				args["projectId"] = tc.projectID
			}
			// issueIid is always added as it's required, invalid type tested separately
			args["issueIid"] = tc.issueIid

			// Prepare request using correct structure
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      getIssueTool.Name, // Use the tool name from the definition
					Arguments: args,
				},
			}

			// Execute handler (defined outside the loop)
			result, err := handler(ctx, request)

			// Assertions
			if tc.expectInternalError {
				require.Error(t, err) // Expect the handler itself to return an error
				if tc.errorContains != "" {
					assert.ErrorContains(t, err, tc.errorContains)
				}
				assert.Nil(t, result) // Result should be nil when handler errors
			} else {
				require.NoError(t, err)                 // Handler should not return an error
				require.NotNil(t, result)               // Result should not be nil
				textContent := getTextResult(t, result) // Use helper

				if tc.expectResultError {
					// User-facing errors (validation, not found) check textContent.Text
					expectedErrString, ok := tc.expectedResult.(string)
					require.True(t, ok, "Expected user error result to be a string")
					assert.Contains(t, textContent.Text, expectedErrString, "User error message mismatch")
				} else {
					// Successful result - compare JSON content
					expectedIssue, ok := tc.expectedResult.(*gl.Issue)
					require.True(t, ok, "Expected success result should be *gl.Issue")
					expectedJSON, err := json.Marshal(expectedIssue)
					require.NoError(t, err, "Failed to marshal expected success result")
					assert.JSONEq(t, string(expectedJSON), textContent.Text, "Result JSON mismatch")
				}
			}
		})
	}
}
