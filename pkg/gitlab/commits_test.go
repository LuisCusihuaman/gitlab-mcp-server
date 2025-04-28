package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mark3labs/mcp-go/mcp"
	gl "gitlab.com/gitlab-org/api/client-go"
)

// Add tests for GetProjectCommits here
func TestGetProjectCommitsHandler(t *testing.T) {
	ctx := context.Background()
	mockClient, mockCommits, ctrl := setupMockClientForCommits(t)
	defer ctrl.Finish()

	// Mock getClient function for commit tests
	mockGetClientCommits := func(_ context.Context) (*gl.Client, error) {
		return mockClient, nil
	}

	// --- Define the Tool and Handler ---
	getProjectCommitsTool, getProjectCommitsHandler := GetProjectCommits(mockGetClientCommits)

	projectID := "group/project"
	ref := "main"
	path := "src/main.go"
	sinceStr := "2024-01-01T00:00:00Z"
	untilStr := "2024-01-31T23:59:59Z"
	sinceTime, _ := time.Parse(time.RFC3339, sinceStr)
	untilTime, _ := time.Parse(time.RFC3339, untilStr)

	// Helper to create mock Commits
	createCommit := func(id, shortID, title string) *gl.Commit {
		return &gl.Commit{
			ID:             id,
			ShortID:        shortID,
			Title:          title,
			AuthorName:     "Test User",
			AuthorEmail:    "test@example.com",
			CommitterName:  "Test User",
			CommitterEmail: "test@example.com",
			// Add stats if needed for specific tests
		}
	}

	createCommitWithStats := func(id, shortID, title string, additions, deletions, total int) *gl.Commit {
		c := createCommit(id, shortID, title)
		c.Stats = &gl.CommitStats{
			Additions: additions,
			Deletions: deletions,
			Total:     total,
		}
		return c
	}

	// --- Test Cases ---
	tests := []struct {
		name               string
		inputArgs          map[string]any
		mockSetup          func()
		expectedResult     []*gl.Commit // Expecting a slice of commits
		expectHandlerError bool
		expectResultError  bool
		errorContains      string
	}{
		{
			name: "Success - List Commits - Basic",
			inputArgs: map[string]any{
				"projectId": projectID,
			},
			mockSetup: func() {
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListCommitsOptions{})
				mockCommits.EXPECT().
					ListCommits(projectID, expectedOptsMatcher, gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListCommitsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Commit, *gl.Response, error) {
						assert.Nil(t, opts.RefName)
						assert.Nil(t, opts.Path)
						assert.Nil(t, opts.Since)
						assert.Nil(t, opts.Until)
						assert.Nil(t, opts.WithStats)
						assert.Equal(t, 1, opts.Page)
						assert.Equal(t, DefaultPerPage, opts.PerPage)
						return []*gl.Commit{
							createCommit("sha1", "sha1abc", "Initial commit"),
							createCommit("sha2", "sha2def", "Add feature X"),
						}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Commit{
				createCommit("sha1", "sha1abc", "Initial commit"),
				createCommit("sha2", "sha2def", "Add feature X"),
			},
		},
		{
			name: "Success - List Commits - With All Filters",
			inputArgs: map[string]any{
				"projectId": projectID,
				"ref":       ref,
				"path":      path,
				"since":     sinceStr,
				"until":     untilStr,
				"withStats": true,
				"page":      1,
				"per_page":  5,
			},
			mockSetup: func() {
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListCommitsOptions{})
				mockCommits.EXPECT().
					ListCommits(projectID, expectedOptsMatcher, gomock.Any()).
					DoAndReturn(func(_ interface{}, opts *gl.ListCommitsOptions, _ ...gl.RequestOptionFunc) ([]*gl.Commit, *gl.Response, error) {
						require.NotNil(t, opts.RefName)
						assert.Equal(t, ref, *opts.RefName)
						require.NotNil(t, opts.Path)
						assert.Equal(t, path, *opts.Path)
						require.NotNil(t, opts.Since)
						assert.True(t, sinceTime.Equal(*opts.Since))
						require.NotNil(t, opts.Until)
						assert.True(t, untilTime.Equal(*opts.Until))
						require.NotNil(t, opts.WithStats)
						assert.True(t, *opts.WithStats)
						assert.Equal(t, 1, opts.Page)
						assert.Equal(t, 5, opts.PerPage)
						return []*gl.Commit{createCommitWithStats("sha3", "sha3ghi", "Refactor module", 10, 5, 15)}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil
					})
			},
			expectedResult: []*gl.Commit{createCommitWithStats("sha3", "sha3ghi", "Refactor module", 10, 5, 15)},
		},
		{
			name: "Success - Empty List",
			inputArgs: map[string]any{
				"projectId": projectID,
				"ref":       "nonexistent-branch",
			},
			mockSetup: func() {
				expectedOptsMatcher := gomock.AssignableToTypeOf(&gl.ListCommitsOptions{})
				mockCommits.EXPECT().
					ListCommits(projectID, expectedOptsMatcher, gomock.Any()).
					Return([]*gl.Commit{}, &gl.Response{Response: &http.Response{StatusCode: 200}}, nil)
			},
			expectedResult: []*gl.Commit{}, // Expect empty slice
		},
		{
			name: "Error - Project Not Found (404)",
			inputArgs: map[string]any{
				"projectId": "nonexistent/project",
			},
			mockSetup: func() {
				mockCommits.EXPECT().
					ListCommits("nonexistent/project", gomock.Any(), gomock.Any()).
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
				mockCommits.EXPECT().
					ListCommits(projectID, gomock.Any(), gomock.Any()).
					Return(nil, &gl.Response{Response: &http.Response{StatusCode: 500}}, errors.New("gitlab: 500 Internal Server Error"))
			},
			expectedResult:     nil,
			expectHandlerError: true,
			errorContains:      fmt.Sprintf("failed to list commits for project %q", projectID),
		},
		{
			name:               "Error - Missing projectId",
			inputArgs:          map[string]any{"ref": ref},
			mockSetup:          func() {},
			expectedResult:     nil,
			expectResultError:  true,
			expectHandlerError: false,
			errorContains:      "Validation Error: missing required parameter: projectId",
		},
		{
			name:               "Error - Invalid since date format",
			inputArgs:          map[string]any{"projectId": projectID, "since": "not-a-date"},
			mockSetup:          func() {},
			expectedResult:     nil,
			expectResultError:  true,
			expectHandlerError: false,
			errorContains:      "Validation Error: parameter 'since' must be a valid ISO 8601 timestamp string",
		},
		{
			name:               "Error - Invalid withStats type",
			inputArgs:          map[string]any{"projectId": projectID, "withStats": "maybe"},
			mockSetup:          func() {},
			expectedResult:     nil,
			expectResultError:  true,
			expectHandlerError: false,
			errorContains:      "Validation Error: parameter 'withStats' must be a boolean (or boolean-like string), got type string",
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
					Name:      getProjectCommitsTool.Name,
					Arguments: tc.inputArgs,
				},
			}

			// Execute the handler
			result, err := getProjectCommitsHandler(ctx, req)

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
					var actualCommits []*gl.Commit
					err = json.Unmarshal([]byte(textContent.Text), &actualCommits)
					require.NoError(t, err, "Failed to unmarshal actual result JSON")

					// Compare lengths first
					require.Equal(t, len(tc.expectedResult), len(actualCommits), "Number of commits mismatch")

					// Compare content using JSONEq
					expectedJSON, _ := json.Marshal(tc.expectedResult)
					actualJSON, _ := json.Marshal(actualCommits)
					assert.JSONEq(t, string(expectedJSON), string(actualJSON), "Commit list content mismatch")
				}
			}
		})
	}
}
