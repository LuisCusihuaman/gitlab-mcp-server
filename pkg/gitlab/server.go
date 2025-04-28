package gitlab

import (
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DefaultPerPage defines the default number of items per page for pagination.
const DefaultPerPage = 30

// MaxPerPage defines the maximum number of items per page allowed by GitLab.
const MaxPerPage = 100

// NewServer creates a new MCP server instance with default options suitable for GitLab.
func NewServer(appName, appVersion string) *server.MCPServer {
	// Configure default server options here if needed
	opts := []server.ServerOption{
		// Add server options similar to github-mcp-server if needed
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true), // Assuming these exist and are desired
		server.WithLogging(),                        // Assuming this exists
	}
	return server.NewMCPServer(appName, appVersion, opts...)
}

// Helper function to get a required string parameter from the request.
// Checks for presence, type, and non-empty value.
func requiredParam(req *mcp.CallToolRequest, name string) (string, error) {
	param, ok := req.Params.Arguments[name] // Use req.Params.Arguments
	if !ok {
		return "", fmt.Errorf("missing required parameter: %s", name)
	}
	strVal, ok := param.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string, got %T", name, param)
	}
	if strVal == "" {
		return "", fmt.Errorf("parameter %s cannot be empty", name)
	}
	return strVal, nil
}

// Helper function to get an optional string parameter from the request.
// Returns defaultValue if not present, empty, or wrong type.
func OptionalParam(req *mcp.CallToolRequest, name, defaultValue string) string {
	param, ok := req.Params.Arguments[name] // Use req.Params.Arguments
	if !ok {
		return defaultValue
	}
	strVal, ok := param.(string)
	if !ok {
		// Log warning or just return default?
		return defaultValue
	}
	// Consider empty string as not provided for optional params
	if strVal == "" {
		return defaultValue
	}
	return strVal
}

// Helper function to get an optional integer parameter from the request.
// Handles number (float64 from JSON) and string types.
// Returns defaultValue if not present, empty string, or error during conversion.
func OptionalIntParam(req *mcp.CallToolRequest, name string, defaultValue int) (int, error) {
	param, ok := req.Params.Arguments[name] // Use req.Params.Arguments
	if !ok {
		return defaultValue, nil
	}

	switch v := param.(type) {
	case float64: // JSON numbers are often float64
		if v != float64(int(v)) { // Check if it's a whole number
			return 0, fmt.Errorf("parameter %s must be a whole number, got %f", name, v)
		}
		// Ensure the integer conversion doesn't cause overflow/underflow issues if necessary
		// For typical page/perPage numbers, direct conversion is usually fine.
		return int(v), nil
	case int:
		return v, nil
	case string:
		if v == "" {
			return defaultValue, nil // Treat empty string as not provided
		}
		intVal, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("parameter %s must be a valid integer string, got '%s': %w", name, v, err)
		}
		return intVal, nil
	default:
		// Return an error indicating the type was unexpected
		return 0, fmt.Errorf("parameter %s must be an integer or integer string, got %T", name, param)
	}
}

// WithPagination returns a ToolOption to add standard 'page' and 'per_page' parameters.
// Parameters are optional by default.
func WithPagination() mcp.ToolOption {
	return func(tool *mcp.Tool) {
		// Apply options for 'page' parameter
		mcp.WithNumber("page",
			mcp.Description("Page number of the results to retrieve."), // (min 1)
			// mcp.Min(1), // Add Min if available and desired - Check mcp library for availability
		)(tool)

		// Apply options for 'per_page' parameter
		mcp.WithNumber("per_page",
			mcp.Description(fmt.Sprintf("Number of results to return per page (default: %d, max: %d).", DefaultPerPage, MaxPerPage)), // (min 1, max 100)
			// mcp.Min(1), // Add Min/Max if available
			// mcp.Max(MaxPerPage),
		)(tool)
	}
}

// OptionalPaginationParams extracts page and per_page parameters from the request.
// It applies default and max values.
func OptionalPaginationParams(req *mcp.CallToolRequest) (page, perPage int, err error) {
	page, err = OptionalIntParam(req, "page", 1) // GitLab pages are 1-based
	if err != nil {
		return 0, 0, fmt.Errorf("invalid 'page' parameter: %w", err)
	}
	if page < 1 {
		// Although GitLab API might handle page=0, standard practice is 1-based.
		// Log warning or adjust? For now, force to 1.
		page = 1
	}

	perPage, err = OptionalIntParam(req, "per_page", DefaultPerPage)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid 'per_page' parameter: %w", err)
	}

	if perPage < 1 {
		perPage = DefaultPerPage
	} else if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return page, perPage, nil
}

// Note: newSuccessResult and newErrorResult helpers removed temporarily
// due to unresolved type errors. Will need to be added back later.
