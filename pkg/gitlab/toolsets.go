package gitlab

import (
	// Import necessary packages, including your toolsets package
	"github.com/LuisCusihuaman/gitlab-mcp-server/pkg/toolsets" // Adjust path if needed
	// "github.com/LuisCusihuaman/gitlab-mcp-server/pkg/translations" // Removed for now
	// "github.com/mark3labs/mcp-go/mcp"
	// "github.com/mark3labs/mcp-go/server"
)

// DefaultTools defines the list of toolsets enabled by default.
var DefaultTools = []string{"all"}

// InitToolsets initializes the ToolsetGroup with GitLab-specific toolsets.
// It creates the toolset definitions but does not populate them with actual tools yet.
func InitToolsets(
	enabledToolsets []string,
	readOnly bool,
	_ GetClientFn, // Renamed getClient to _
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
	// --- Add tools to projectsTS (Task 7 & 12) ---
	// projectsTS.AddReadTools(...)
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
