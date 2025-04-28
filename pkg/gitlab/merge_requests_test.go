package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
	"go.uber.org/mock/gomock"
)

// TestGetMergeRequestHandler tests the GetMergeRequest tool
func TestGetMergeRequestHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockMRs, ctrl := setupMockClientForMergeRequests(t)
	defer ctrl.Finish()

	// Mock getClient function for merge request tests
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// Define the Tool and Handler
	getMergeRequestTool, getMergeRequestHandler := GetMergeRequest(mockGetClient)

	// Test data
	projectID := "group/project"
	mrIID := 1.0 // MCP number type maps to float64
	timeNow := time.Now()

	// Create a sample merge request for testing
	sampleMR := &gl.MergeRequest{
		BasicMergeRequest: gl.BasicMergeRequest{
			ID:           123,
			IID:          int(mrIID),
			ProjectID:    456,
			Title:        "Implement feature X",
			Description:  "This adds feature X which does Y",
			State:        "opened",
			CreatedAt:    &timeNow,
			WebURL:       fmt.Sprintf("https://gitlab.com/%s/merge_requests/%d", projectID, int(mrIID)),
			SourceBranch: "feature-x",
			TargetBranch: "main",
		},
	}

	// Test cases
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     interface{}
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - Get MR by ID",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIID,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					GetMergeRequest(projectID, int(mrIID), nil, gomock.Any()).
					Return(sampleMR, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: sampleMR,
		},
		{
			name: "Error - MR Not Found (404)",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": 999.0,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					GetMergeRequest(projectID, 999, nil, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Not Found"))
			},
			expectResultError: true,
			errorContains:     "merge request 999 not found in project",
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIID,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					GetMergeRequest(projectID, int(mrIID), nil, gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to get merge request",
		},
		{
			name: "Error - Missing projectId parameter",
			inputArgs: map[string]any{
				"mergeRequestIid": mrIID,
			},
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Missing mergeRequestIid parameter",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: mergeRequestIid",
		},
		{
			name: "Error - Invalid mergeRequestIid (not integer)",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": 1.5, // Non-integer value
			},
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: mergeRequestIid 1.5 is not a valid integer",
		},
	}

	// Run test cases
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
					Name:      getMergeRequestTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Call the handler
			result, err := getMergeRequestHandler(ctx, req)

			// Verify results
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
					// For successful responses, verify the returned data
					var actualMR gl.MergeRequest
					err = json.Unmarshal([]byte(textContent.Text), &actualMR)
					require.NoError(t, err, "Failed to unmarshal actual result JSON")

					// Marshal both expected and actual for JSONEq comparison
					expectedJSON, _ := json.Marshal(tc.expectedResult)
					actualJSON, _ := json.Marshal(actualMR)
					assert.JSONEq(t, string(expectedJSON), string(actualJSON), "Merge request content mismatch")
				}
			}
		})
	}

	// Test for client initialization error
	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := GetMergeRequest(errorGetClientFn)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: getMergeRequestTool.Name,
				Arguments: map[string]any{
					"projectId":       projectID,
					"mergeRequestIid": mrIID,
				},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get GitLab client")
		assert.Nil(t, result)
	})
}

// TestGetMergeRequestCommentsHandler tests the GetMergeRequestComments tool
func TestGetMergeRequestCommentsHandler(t *testing.T) {
	ctx := context.Background()

	// --- Setup Mock Client and GetClientFn once ---
	mockClient, mockNotes, ctrl := setupMockClientForNotes(t)
	defer ctrl.Finish()

	// Create the mock getClient function once, capturing the mockClient
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler once ---
	getMRCommentsTool, handler := GetMergeRequestComments(mockGetClient)

	// Define common test data
	projectID := "group/project"
	mrIid := 1.0 // MCP number type maps to float64

	// Create time values to use in tests
	timeNow := time.Now()
	time24HoursAgo := timeNow.Add(-24 * time.Hour)
	time12HoursAgo := timeNow.Add(-12 * time.Hour)
	time6HoursAgo := timeNow.Add(-6 * time.Hour)

	// --- Test Cases ---
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     interface{} // Either []*gl.Note for success or string for error message
		expectHandlerError bool        // Whether the handler itself should return an error
		expectResultError  bool        // Whether the returned mcp.CallToolResult should represent an error
		errorContains      string      // Substring to check in the error message
	}{
		// --- Success Cases ---
		{
			name: "Success - Get Merge Request Comments",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
			},
			mockSetup: func() {
				// Create expected notes for the response with correct fields
				expectedNotes := []*gl.Note{
					{
						ID:   123,
						Body: "This looks good to me",
						Author: gl.NoteAuthor{
							Name: "Test User",
						},
						CreatedAt: &time24HoursAgo,
					},
					{
						ID:   124,
						Body: "I have a suggestion for this line",
						Author: gl.NoteAuthor{
							Name: "Another User",
						},
						CreatedAt: &time12HoursAgo,
					},
				}

				mockNotes.EXPECT().
					ListMergeRequestNotes(projectID, int(mrIid), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.ListMergeRequestNotesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Note, *gl.Response, error) {
						// Verify pagination settings
						assert.Equal(t, 1, opts.Page)
						assert.Equal(t, DefaultPerPage, opts.PerPage)

						return expectedNotes, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Note{
				{
					ID:   123,
					Body: "This looks good to me",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					CreatedAt: &time24HoursAgo,
				},
				{
					ID:   124,
					Body: "I have a suggestion for this line",
					Author: gl.NoteAuthor{
						Name: "Another User",
					},
					CreatedAt: &time12HoursAgo,
				},
			},
		},
		{
			name: "Success - Merge Request Comments With Pagination",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
				"page":            2,
				"per_page":        5,
			},
			mockSetup: func() {
				// Create expected notes for the response with correct fields
				expectedNotes := []*gl.Note{
					{
						ID:   125,
						Body: "Paginated comment",
						Author: gl.NoteAuthor{
							Name: "Test User",
						},
						CreatedAt: &time6HoursAgo,
					},
				}

				mockNotes.EXPECT().
					ListMergeRequestNotes(projectID, int(mrIid), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ any, _ int, opts *gl.ListMergeRequestNotesOptions, _ ...gl.RequestOptionFunc) ([]*gl.Note, *gl.Response, error) {
						// Verify pagination settings
						assert.Equal(t, 2, opts.Page)
						assert.Equal(t, 5, opts.PerPage)

						return expectedNotes, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Note{
				{
					ID:   125,
					Body: "Paginated comment",
					Author: gl.NoteAuthor{
						Name: "Test User",
					},
					CreatedAt: &time6HoursAgo,
				},
			},
		},
		{
			name: "Success - Empty Comments",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": 2.0, // Different MR with no comments
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					ListMergeRequestNotes(projectID, 2, gomock.Any(), gomock.Any()).
					Return([]*gl.Note{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Note{}, // Empty array
		},
		// --- Error Cases ---
		{
			name: "Error - Merge Request Not Found (404)",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": 999.0,
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					ListMergeRequestNotes(projectID, 999, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Merge Request Not Found"))
			},
			expectResultError: true,
			errorContains:     "merge request 999 not found in project",
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": mrIid,
			},
			mockSetup: func() {
				mockNotes.EXPECT().
					ListMergeRequestNotes(projectID, int(mrIid), gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to get comments for merge request",
		},
		{
			name: "Error - Missing projectId parameter",
			inputArgs: map[string]any{
				"mergeRequestIid": mrIid,
			}, // Missing projectId
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Missing mergeRequestIid parameter",
			inputArgs: map[string]any{
				"projectId": projectID,
			}, // Missing mergeRequestIid
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: mergeRequestIid",
		},
		{
			name: "Error - Invalid mergeRequestIid (not integer)",
			inputArgs: map[string]any{
				"projectId":       projectID,
				"mergeRequestIid": 1.5,
			}, // Non-integer float
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: mergeRequestIid 1.5 is not a valid integer",
		},
	}

	// --- Run Test Cases ---
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock expectations for this specific test case
			tc.mockSetup()

			// Prepare request using correct structure
			request := mcp.CallToolRequest{
				Params: struct {
					Name      string                 `json:"name"`
					Arguments map[string]interface{} `json:"arguments,omitempty"`
					Meta      *struct {
						ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
					} `json:"_meta,omitempty"`
				}{
					Name:      getMRCommentsTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Execute handler
			result, err := handler(ctx, request)

			// Validate results following the pattern
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
					// Handle special case for empty array
					if notes, ok := tc.expectedResult.([]*gl.Note); ok && len(notes) == 0 {
						assert.Equal(t, "[]", textContent.Text, "Empty array mismatch")
					} else {
						// Unmarshal expected and actual results
						var actualNotes []*gl.Note
						err = json.Unmarshal([]byte(textContent.Text), &actualNotes)
						require.NoError(t, err, "Failed to unmarshal actual result JSON")

						// Compare lengths first
						expectedNotes, _ := tc.expectedResult.([]*gl.Note)
						require.Equal(t, len(expectedNotes), len(actualNotes), "Number of notes mismatch")

						// Compare content using JSONEq
						expectedJSON, _ := json.Marshal(tc.expectedResult)
						actualJSON, _ := json.Marshal(actualNotes)
						assert.JSONEq(t, string(expectedJSON), string(actualJSON), "Notes content mismatch")
					}
				}
			}
		})
	}

	// Test case for Client Initialization Error (outside the loop)
	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		// Define a GetClientFn that returns an error
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := GetMergeRequestComments(errorGetClientFn)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: getMRCommentsTool.Name,
				Arguments: map[string]any{
					"projectId":       projectID,
					"mergeRequestIid": mrIid,
				},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize GitLab client")
		assert.Nil(t, result)
	})
}

// TestListMergeRequestsHandler tests the ListMergeRequests tool
func TestListMergeRequestsHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockMRs, ctrl := setupMockClientForMergeRequests(t)
	defer ctrl.Finish()

	// Mock getClient function for merge request tests
	mockGetClient := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// Define the Tool and Handler
	listMergeRequestsTool, listMergeRequestsHandler := ListMergeRequests(mockGetClient)

	// Test data
	projectID := "group/project"
	timeNow := time.Now()
	time24HoursAgo := timeNow.Add(-24 * time.Hour)

	// Sample merge requests for testing
	sampleMRs := []*gl.MergeRequest{
		{
			BasicMergeRequest: gl.BasicMergeRequest{
				ID:           123,
				IID:          1,
				ProjectID:    456,
				Title:        "Implement feature X",
				Description:  "This adds feature X which does Y",
				State:        "opened",
				CreatedAt:    &time24HoursAgo,
				SourceBranch: "feature-x",
				TargetBranch: "main",
			},
		},
		{
			BasicMergeRequest: gl.BasicMergeRequest{
				ID:           124,
				IID:          2,
				ProjectID:    456,
				Title:        "Fix bug Z",
				Description:  "This fixes critical bug Z",
				State:        "opened",
				CreatedAt:    &timeNow,
				SourceBranch: "fix-bug-z",
				TargetBranch: "main",
			},
		},
	}

	// Test cases
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     interface{}
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - List Project MRs",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						// Verify default pagination
						assert.Equal(t, projectID, proj)
						assert.Equal(t, 1, opts.Page)
						assert.Equal(t, DefaultPerPage, opts.PerPage)

						return convertToBasicMRs(sampleMRs), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: sampleMRs,
		},
		{
			name: "Success - List Project MRs with Filtering",
			inputArgs: map[string]any{
				"projectId": projectID,
				"state":     "opened",
				"labels":    "bug,critical",
				"sort":      "desc",
				"order_by":  "created_at",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						// Verify filter params
						assert.Equal(t, projectID, proj)
						assert.Equal(t, "opened", *opts.State)
						assert.ElementsMatch(t, gl.LabelOptions{"bug", "critical"}, *opts.Labels)
						assert.Equal(t, "desc", *opts.Sort)
						assert.Equal(t, "created_at", *opts.OrderBy)

						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[1]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[1]},
		},
		{
			name: "Success - List Project MRs with Pagination",
			inputArgs: map[string]any{
				"projectId": projectID,
				"page":      2,
				"per_page":  5,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(proj any, opts *gl.ListProjectMergeRequestsOptions, _ ...gl.RequestOptionFunc) ([]*gl.BasicMergeRequest, *gl.Response, error) {
						// Verify pagination
						assert.Equal(t, projectID, proj)
						assert.Equal(t, 2, opts.Page)
						assert.Equal(t, 5, opts.PerPage)

						return convertToBasicMRs([]*gl.MergeRequest{sampleMRs[1]}), &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.MergeRequest{sampleMRs[1]},
		},
		{
			name: "Success - Empty MRs List",
			inputArgs: map[string]any{
				"projectId": projectID,
				"state":     "merged", // Filter that returns no results
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]*gl.BasicMergeRequest{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.MergeRequest{},
		},
		{
			name: "Error - Project Not Found (404)",
			inputArgs: map[string]any{
				"projectId": "nonexistent/project",
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 404}}, errors.New("gitlab: 404 Project Not Found"))
			},
			expectResultError: true,
			errorContains:     "not found or access denied",
		},
		{
			name: "Error - GitLab API Error (500)",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				mockMRs.EXPECT().
					ListProjectMergeRequests(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectHandlerError: true,
			errorContains:      "failed to list merge requests",
		},
		{
			name:              "Error - Missing projectId parameter",
			inputArgs:         map[string]any{},
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "Validation Error: missing required parameter: projectId",
		},
		{
			name: "Error - Invalid date format",
			inputArgs: map[string]any{
				"projectId":     projectID,
				"created_after": "not-a-date",
			},
			mockSetup:         func() { /* No API call expected */ },
			expectResultError: true,
			errorContains:     "must be a valid ISO 8601 datetime",
		},
	}

	// Run test cases
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
					Name:      listMergeRequestsTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Call the handler
			result, err := listMergeRequestsHandler(ctx, req)

			// Verify results
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
					// For empty lists, check for empty JSON array
					mrList, ok := tc.expectedResult.([]*gl.MergeRequest)
					if ok && len(mrList) == 0 {
						assert.Equal(t, "[]", textContent.Text, "Empty array mismatch")
					} else {
						// For successful responses with data, verify JSON equality
						var actualMRs []*gl.MergeRequest
						err = json.Unmarshal([]byte(textContent.Text), &actualMRs)
						require.NoError(t, err, "Failed to unmarshal actual result JSON")

						// Marshal both expected and actual for JSONEq comparison
						expectedJSON, _ := json.Marshal(tc.expectedResult)
						actualJSON, _ := json.Marshal(actualMRs)
						assert.JSONEq(t, string(expectedJSON), string(actualJSON), "Merge request list content mismatch")
					}
				}
			}
		})
	}

	// Test for client initialization error
	t.Run("Error - Client Initialization Error", func(t *testing.T) {
		errorGetClientFn := func(_ context.Context) (*gl.Client, error) {
			return nil, fmt.Errorf("mock init error")
		}
		_, handler := ListMergeRequests(errorGetClientFn)

		request := mcp.CallToolRequest{
			Params: struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments,omitempty"`
				Meta      *struct {
					ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
				} `json:"_meta,omitempty"`
			}{
				Name: listMergeRequestsTool.Name,
				Arguments: map[string]any{
					"projectId": projectID,
				},
			},
		}

		result, err := handler(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize GitLab client")
		assert.Nil(t, result)
	})
}

func convertToBasicMRs(mrs []*gl.MergeRequest) []*gl.BasicMergeRequest {
	var basicMRs []*gl.BasicMergeRequest
	for _, mr := range mrs {
		basicMRs = append(basicMRs, &gl.BasicMergeRequest{
			ID:           mr.ID,
			IID:          mr.IID,
			ProjectID:    mr.ProjectID,
			Title:        mr.Title,
			Description:  mr.Description,
			State:        mr.State,
			CreatedAt:    mr.CreatedAt,
			UpdatedAt:    mr.UpdatedAt,
			SourceBranch: mr.SourceBranch,
			TargetBranch: mr.TargetBranch,
		})
	}
	return basicMRs
}
