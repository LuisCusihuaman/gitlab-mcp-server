package gitlab

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		paramName   string
		expectedVal string
		expectError bool
		errContains string
	}{
		{
			name:        "Parameter present and correct type",
			params:      map[string]interface{}{"testParam": "value1"},
			paramName:   "testParam",
			expectedVal: "value1",
			expectError: false,
		},
		{
			name:        "Parameter missing",
			params:      map[string]interface{}{},
			paramName:   "testParam",
			expectError: true,
			errContains: "missing required parameter",
		},
		{
			name:        "Parameter present but wrong type",
			params:      map[string]interface{}{"testParam": 123},
			paramName:   "testParam",
			expectError: true,
			errContains: "must be a string",
		},
		{
			name:        "Parameter present but empty string",
			params:      map[string]interface{}{"testParam": ""},
			paramName:   "testParam",
			expectError: true,
			errContains: "cannot be empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := createMCPRequest(tc.params)

			val, err := requiredParam(req, tc.paramName)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedVal, val)
			}
		})
	}
}

// createMCPRequest is a helper function to create a CallToolRequest for tests
func createMCPRequest(params map[string]interface{}) *mcp.CallToolRequest {
	return &mcp.CallToolRequest{
		Params: struct {
			Name      string                 "json:\"name\""
			Arguments map[string]interface{} "json:\"arguments,omitempty\""
			Meta      *struct {
				ProgressToken mcp.ProgressToken "json:\"progressToken,omitempty\""
			} "json:\"_meta,omitempty\""
		}{
			Arguments: params,
		},
	}
}

func TestOptionalParam(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		paramName    string
		defaultValue string
		expectedVal  string
	}{
		{
			name:         "Parameter present",
			params:       map[string]interface{}{"optParam": "value1"},
			paramName:    "optParam",
			defaultValue: "default",
			expectedVal:  "value1",
		},
		{
			name:         "Parameter missing",
			params:       map[string]interface{}{},
			paramName:    "optParam",
			defaultValue: "default",
			expectedVal:  "default",
		},
		{
			name:         "Parameter present but wrong type",
			params:       map[string]interface{}{"optParam": 123},
			paramName:    "optParam",
			defaultValue: "default",
			expectedVal:  "default", // Should return default on type error
		},
		{
			name:         "Parameter present but empty string",
			params:       map[string]interface{}{"optParam": ""},
			paramName:    "optParam",
			defaultValue: "default",
			expectedVal:  "default", // Should return default for empty optional string
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := createMCPRequest(tc.params)
			val := OptionalParam(req, tc.paramName, tc.defaultValue)
			assert.Equal(t, tc.expectedVal, val)
		})
	}
}

func TestOptionalIntParam(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		paramName    string
		defaultValue int
		expectedVal  int
		expectError  bool
		errContains  string
	}{
		{
			name:         "Parameter present as float64 (JSON number)",
			params:       map[string]interface{}{"optInt": float64(42)},
			paramName:    "optInt",
			defaultValue: 10,
			expectedVal:  42,
			expectError:  false,
		},
		{
			name:         "Parameter present as integer string",
			params:       map[string]interface{}{"optInt": "123"},
			paramName:    "optInt",
			defaultValue: 10,
			expectedVal:  123,
			expectError:  false,
		},
		{
			name:         "Parameter present as int",
			params:       map[string]interface{}{"optInt": 55},
			paramName:    "optInt",
			defaultValue: 10,
			expectedVal:  55,
			expectError:  false,
		},
		{
			name:         "Parameter missing",
			params:       map[string]interface{}{},
			paramName:    "optInt",
			defaultValue: 10,
			expectedVal:  10,
			expectError:  false,
		},
		{
			name:         "Parameter present but empty string",
			params:       map[string]interface{}{"optInt": ""},
			paramName:    "optInt",
			defaultValue: 10,
			expectedVal:  10,
			expectError:  false,
		},
		{
			name:         "Parameter present but wrong type (boolean)",
			params:       map[string]interface{}{"optInt": true},
			paramName:    "optInt",
			defaultValue: 10,
			expectError:  true,
			errContains:  "must be an integer",
		},
		{
			name:         "Parameter present but non-integer float64",
			params:       map[string]interface{}{"optInt": float64(42.5)},
			paramName:    "optInt",
			defaultValue: 10,
			expectError:  true,
			errContains:  "must be a whole number",
		},
		{
			name:         "Parameter present but invalid integer string",
			params:       map[string]interface{}{"optInt": "abc"},
			paramName:    "optInt",
			defaultValue: 10,
			expectError:  true,
			errContains:  "must be a valid integer string",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := createMCPRequest(tc.params)
			val, err := OptionalIntParam(req, tc.paramName, tc.defaultValue)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedVal, val)
			}
		})
	}
}

func TestOptionalPaginationParams(t *testing.T) {
	tests := []struct {
		name            string
		params          map[string]interface{}
		expectedPage    int
		expectedPerPage int
		expectError     bool
		errContains     string
	}{
		{
			name:            "No params, use defaults",
			params:          map[string]interface{}{},
			expectedPage:    1,
			expectedPerPage: DefaultPerPage,
			expectError:     false,
		},
		{
			name:            "Page provided",
			params:          map[string]interface{}{"page": float64(5)},
			expectedPage:    5,
			expectedPerPage: DefaultPerPage,
			expectError:     false,
		},
		{
			name:            "PerPage provided",
			params:          map[string]interface{}{"per_page": float64(50)},
			expectedPage:    1,
			expectedPerPage: 50,
			expectError:     false,
		},
		{
			name:            "Page and PerPage provided",
			params:          map[string]interface{}{"page": "3", "per_page": "25"},
			expectedPage:    3,
			expectedPerPage: 25,
			expectError:     false,
		},
		{
			name:            "PerPage exceeding max",
			params:          map[string]interface{}{"per_page": float64(200)},
			expectedPage:    1,
			expectedPerPage: MaxPerPage,
			expectError:     false,
		},
		{
			name:            "PerPage less than 1",
			params:          map[string]interface{}{"per_page": float64(0)},
			expectedPage:    1,
			expectedPerPage: DefaultPerPage, // Should reset to default
			expectError:     false,
		},
		{
			name:            "Page less than 1",
			params:          map[string]interface{}{"page": float64(0)},
			expectedPage:    1, // Should reset to 1
			expectedPerPage: DefaultPerPage,
			expectError:     false,
		},
		{
			name:        "Invalid page type",
			params:      map[string]interface{}{"page": true},
			expectError: true,
			errContains: "invalid 'page' parameter",
		},
		{
			name:        "Invalid per_page type",
			params:      map[string]interface{}{"per_page": "invalid"},
			expectError: true,
			errContains: "invalid 'per_page' parameter",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := createMCPRequest(tc.params)
			page, perPage, err := OptionalPaginationParams(req)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedPage, page)
				assert.Equal(t, tc.expectedPerPage, perPage)
			}
		})
	}
}
