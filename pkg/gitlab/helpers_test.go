package gitlab

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	gl "gitlab.com/gitlab-org/api/client-go"
	mock_gitlab "gitlab.com/gitlab-org/api/client-go/testing"
	"go.uber.org/mock/gomock"
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

// Helper to create a mock GetClientFn for testing handlers for the Commits service
func setupMockClientForCommits(t *testing.T) (*gl.Client, *mock_gitlab.MockCommitsServiceInterface, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockCommits := mock_gitlab.NewMockCommitsServiceInterface(ctrl) // Mock for Commits

	// Create a minimal client and attach the mock service
	client := &gl.Client{
		Commits: mockCommits, // Attach the Commits service mock
	}

	return client, mockCommits, ctrl
}

// Helper to create a mock GetClientFn for testing handlers for the Issues service
func setupMockClientForIssues(t *testing.T) (*gl.Client, *mock_gitlab.MockIssuesServiceInterface, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockIssues := mock_gitlab.NewMockIssuesServiceInterface(ctrl) // Mock for Issues

	// Create a minimal client and attach the mock service
	client := &gl.Client{
		Issues: mockIssues, // Attach the Issues service mock
	}

	return client, mockIssues, ctrl
}

// Helper to create a mock GetClientFn for testing handlers for the MergeRequests service
func setupMockClientForMergeRequests(t *testing.T) (*gl.Client, *mock_gitlab.MockMergeRequestsServiceInterface, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockMRs := mock_gitlab.NewMockMergeRequestsServiceInterface(ctrl) // Mock for MergeRequests

	// Create a minimal client and attach the mock service
	client := &gl.Client{
		MergeRequests: mockMRs, // Attach the MergeRequests service mock
	}

	return client, mockMRs, ctrl
}

// Helper to create a mock GetClientFn for testing handlers for the Notes service
func setupMockClientForNotes(t *testing.T) (*gl.Client, *mock_gitlab.MockNotesServiceInterface, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockNotes := mock_gitlab.NewMockNotesServiceInterface(ctrl) // Mock for Notes

	// Create a minimal client and attach the mock service
	client := &gl.Client{
		Notes: mockNotes, // Attach the Notes service mock
	}

	return client, mockNotes, ctrl
}
