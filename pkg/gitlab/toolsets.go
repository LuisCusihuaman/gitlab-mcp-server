package gitlab

import (
	"context" // Added for GetClientFn
	// Import necessary packages, including your toolsets package
	"github.com/LuisCusihuaman/gitlab-mcp-server/pkg/toolsets" // Adjust path if needed
	gl "gitlab.com/gitlab-org/api/client-go"                   // Import the GitLab client library
	// "github.com/LuisCusihuaman/gitlab-mcp-server/pkg/translations" // Removed for now
	// "github.com/mark3labs/mcp-go/mcp"
	// "github.com/mark3labs/mcp-go/server"
)

// GetClientFn defines the function signature for retrieving an initialized GitLab client.
// This allows decoupling toolset initialization from direct client creation.
type GetClientFn func(context.Context) (*gl.Client, error)

// DefaultTools defines the list of toolsets enabled by default.
var DefaultTools = []string{"all"}

// InitToolsets initializes the ToolsetGroup with GitLab-specific toolsets.
// It accepts a function to retrieve the GitLab client.
func InitToolsets(
	enabledToolsets []string,
	readOnly bool,
	getClient GetClientFn, // Restore parameter name
	// t translations.TranslationHelperFunc, // Removed for now
) (*toolsets.ToolsetGroup, error) {

	// 1. Create the ToolsetGroup
	tg := toolsets.NewToolsetGroup(readOnly)

	// 2. Define Toolsets (as per PDR section 5.3)
	projectsTS := toolsets.NewToolset("projects", "Tools for interacting with GitLab projects, repositories, branches, commits, tags.")
	issuesTS := toolsets.NewToolset("issues", "Tools for CRUD operations on GitLab issues, comments, labels.")
	mergeRequestsTS := toolsets.NewToolset("merge_requests", "Tools for CRUD operations on GitLab merge requests, comments, approvals, diffs.")
	securityTS := toolsets.NewToolset("security", "Tools for accessing GitLab security scan results (SAST, DAST, etc.).")
	usersTS := toolsets.NewToolset("users", "Tools for looking up GitLab user information.")
	searchTS := toolsets.NewToolset("search", "Tools for utilizing GitLab's scoped search capabilities.")

	// 3. Add Tools to Toolsets (Actual tool implementation TBD in separate tasks)
	//    Tool definition functions will need to accept GetClientFn or call it.
	//    Example (placeholder):
	//    getProjectTool := toolsets.NewServerTool(GetProject(getClient, t))

	// --- Add tools to projectsTS (Task 7 & 12) ---
	projectsTS.AddReadTools(
		toolsets.NewServerTool(GetProject(getClient /*, t */)),
		toolsets.NewServerTool(ListProjects(getClient /*, t */)),
		toolsets.NewServerTool(GetProjectFile(getClient /*, t */)),
	)
	// projectsTS.AddWriteTools(...)

	// --- Add tools to issuesTS (Task 8 & 13) ---
	// issuesTS.AddReadTools(...)
	// issuesTS.AddWriteTools(...)

	// --- Add tools to mergeRequestsTS (Task 9 & 14) ---
	// mergeRequestsTS.AddReadTools(...)
	// mergeRequestsTS.AddWriteTools(...)

	// --- Add tools to securityTS (Part of future tasks?) ---
	// securityTS.AddReadTools(...) // Likely read-only

	// --- Add tools to usersTS (Task 10) ---
	// usersTS.AddReadTools(...) // Likely read-only

	// --- Add tools to searchTS (Task 10/11) ---
	// searchTS.AddReadTools(...) // Likely read-only

	// 4. Add defined Toolsets to the Group
	tg.AddToolset(projectsTS)
	tg.AddToolset(issuesTS)
	tg.AddToolset(mergeRequestsTS)
	tg.AddToolset(securityTS)
	tg.AddToolset(usersTS)
	tg.AddToolset(searchTS)

	// 5. Enable Toolsets based on configuration
	err := tg.EnableToolsets(enabledToolsets)
	if err != nil {
		// Consider logging the error here in a real implementation
		return nil, err // Return error if enabling failed (e.g., unknown toolset name)
	}

	// 6. Return the configured group
	return tg, nil
}
