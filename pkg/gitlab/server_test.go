package gitlab

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createMCPRequest is a helper function to create a CallToolRequest pointer for tests
func createMCPRequest(params map[string]interface{}) *mcp.CallToolRequest {
	return &mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: params,
		},
	}
}

func TestRequiredParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		paramName   string
		requestFunc func(r *mcp.CallToolRequest, p string) (any, error) // Use any for result type
		expectedVal any
		expectError bool
		errContains string
	}{
		{
			name:      "String: Parameter present and correct type",
			params:    map[string]interface{}{"testParam": "value1"},
			paramName: "testParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, error) {
				return requiredParam[string](r, p)
			},
			expectedVal: "value1",
			expectError: false,
		},
		{
			name:      "Int: Parameter present and correct type (float64)",
			params:    map[string]interface{}{"testParam": float64(123)},
			paramName: "testParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, error) {
				return requiredParam[float64](r, p) // Test with float64 first
			},
			expectedVal: float64(123),
			expectError: false,
		},
		{
			name:      "Parameter missing",
			params:    map[string]interface{}{},
			paramName: "testParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, error) {
				return requiredParam[string](r, p)
			},
			expectError: true,
			errContains: "missing required parameter",
		},
		{
			name:      "String: Parameter present but wrong type (int)",
			params:    map[string]interface{}{"testParam": 123},
			paramName: "testParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, error) {
				return requiredParam[string](r, p)
			},
			expectError: true,
			errContains: "not of expected type string, got int", // Updated error check
		},
		{
			name:      "String: Parameter present but empty string (zero value)",
			params:    map[string]interface{}{"testParam": ""},
			paramName: "testParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, error) {
				return requiredParam[string](r, p)
			},
			expectError: true,
			errContains: "cannot be empty or zero value", // Updated error check
		},
		{
			name:      "Int: Parameter present but zero value (float64)",
			params:    map[string]interface{}{"testParam": float64(0)},
			paramName: "testParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, error) {
				return requiredParam[float64](r, p)
			},
			expectError: true,
			errContains: "cannot be empty or zero value", // Updated error check
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := createMCPRequest(tc.params)

			val, err := tc.requestFunc(req, tc.paramName)

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

func TestOptionalParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		paramName   string
		requestFunc func(r *mcp.CallToolRequest, p string) (any, error) // Use any for result type
		expectedVal any                                                 // Expect zero value if absent/error
		expectError bool
		errContains string
	}{
		{
			name:      "String: Parameter present",
			params:    map[string]interface{}{"optParam": "value1"},
			paramName: "optParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, error) {
				return OptionalParam[string](r, p)
			},
			expectedVal: "value1",
			expectError: false,
		},
		{
			name:      "String: Parameter missing",
			params:    map[string]interface{}{},
			paramName: "optParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, error) {
				return OptionalParam[string](r, p)
			},
			expectedVal: "", // Zero value for string
			expectError: false,
		},
		{
			name:      "Int: Parameter missing",
			params:    map[string]interface{}{},
			paramName: "optParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, error) {
				return OptionalParam[int](r, p)
			},
			expectedVal: 0, // Zero value for int
			expectError: false,
		},
		{
			name:      "String: Parameter present but wrong type (int)",
			params:    map[string]interface{}{"optParam": 123},
			paramName: "optParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, error) {
				return OptionalParam[string](r, p)
			},
			expectedVal: "", // Zero value on type error
			expectError: true,
			errContains: "not of expected type string, got int",
		},
		{
			name:      "String: Parameter present but empty string",
			params:    map[string]interface{}{"optParam": ""},
			paramName: "optParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, error) {
				return OptionalParam[string](r, p)
			},
			expectedVal: "", // Empty string is a valid value for optional string
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := createMCPRequest(tc.params)
			val, err := tc.requestFunc(req, tc.paramName)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				// Assert zero value on error is implicit via tc.expectedVal
			} else {
				require.NoError(t, err)
			}
			// Always assert the expected value (which might be the zero value)
			assert.Equal(t, tc.expectedVal, val)
		})
	}
}

func TestOptionalParamOK(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		paramName   string
		requestFunc func(r *mcp.CallToolRequest, p string) (value any, ok bool, err error) // Use any for result type
		expectedVal any
		expectedOK  bool
		expectedErr bool
		errContains string
	}{
		{
			name:      "String: Parameter present",
			params:    map[string]interface{}{"optParam": "value1"},
			paramName: "optParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, bool, error) {
				return OptionalParamOK[string](r, p)
			},
			expectedVal: "value1",
			expectedOK:  true,
			expectedErr: false,
		},
		{
			name:      "String: Parameter missing",
			params:    map[string]interface{}{},
			paramName: "optParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, bool, error) {
				return OptionalParamOK[string](r, p)
			},
			expectedVal: "", // Zero value
			expectedOK:  false,
			expectedErr: false,
		},
		{
			name:      "Int: Parameter missing",
			params:    map[string]interface{}{},
			paramName: "optParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, bool, error) {
				return OptionalParamOK[int](r, p)
			},
			expectedVal: 0, // Zero value
			expectedOK:  false,
			expectedErr: false,
		},
		{
			name:      "String: Parameter present but wrong type (int)",
			params:    map[string]interface{}{"optParam": 123},
			paramName: "optParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, bool, error) {
				return OptionalParamOK[string](r, p)
			},
			expectedVal: "",   // Zero value on type error
			expectedOK:  true, // OK is true because param *was* present
			expectedErr: true,
			errContains: "not of expected type string, got int",
		},
		{
			name:      "String: Parameter present and empty string",
			params:    map[string]interface{}{"optParam": ""},
			paramName: "optParam",
			requestFunc: func(r *mcp.CallToolRequest, p string) (any, bool, error) {
				return OptionalParamOK[string](r, p)
			},
			expectedVal: "",
			expectedOK:  true,
			expectedErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := createMCPRequest(tc.params)
			val, ok, err := tc.requestFunc(req, tc.paramName)

			if tc.expectedErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.expectedOK, ok, "Presence flag (ok)")
			assert.Equal(t, tc.expectedVal, val, "Returned value")
		})
	}
}

func TestOptionalIntParam(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		paramName   string
		expectedVal int
		expectError bool
		errContains string
	}{
		{
			name:        "Parameter present as float64 (JSON number)",
			params:      map[string]interface{}{"optInt": float64(42)},
			paramName:   "optInt",
			expectedVal: 42,
			expectError: false,
		},
		{
			name:        "Parameter present as integer string",
			params:      map[string]interface{}{"optInt": "123"},
			paramName:   "optInt",
			expectedVal: 123,
			expectError: false,
		},
		{
			name:        "Parameter present as int",
			params:      map[string]interface{}{"optInt": 55},
			paramName:   "optInt",
			expectedVal: 55,
			expectError: false,
		},
		{
			name:        "Parameter missing",
			params:      map[string]interface{}{},
			paramName:   "optInt",
			expectedVal: 0, // Zero value for int
			expectError: false,
		},
		{
			name:        "Parameter present but empty string",
			params:      map[string]interface{}{"optInt": ""},
			paramName:   "optInt",
			expectedVal: 0, // Treat empty string as 0/absent
			expectError: false,
		},
		{
			name:        "Parameter present but wrong type (boolean)",
			params:      map[string]interface{}{"optInt": true},
			paramName:   "optInt",
			expectedVal: 0,
			expectError: true,
			errContains: "must be convertible to an integer",
		},
		{
			name:        "Parameter present but non-integer float64",
			params:      map[string]interface{}{"optInt": float64(42.5)},
			paramName:   "optInt",
			expectedVal: 0,
			expectError: true,
			errContains: "must be a whole number",
		},
		{
			name:        "Parameter present but invalid integer string",
			params:      map[string]interface{}{"optInt": "abc"},
			paramName:   "optInt",
			expectedVal: 0,
			expectError: true,
			errContains: "must be a valid integer string",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := createMCPRequest(tc.params)
			// Call the updated OptionalIntParam (no default value)
			val, err := OptionalIntParam(req, tc.paramName)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.expectedVal, val)
		})
	}
}

func TestOptionalIntParamWithDefault(t *testing.T) {
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
			name:         "Parameter present as float64",
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
			name:         "Parameter missing",
			params:       map[string]interface{}{},
			paramName:    "optInt",
			defaultValue: 10,
			expectedVal:  10, // Should use default
			expectError:  false,
		},
		{
			name:         "Parameter present but empty string",
			params:       map[string]interface{}{"optInt": ""},
			paramName:    "optInt",
			defaultValue: 10,
			expectedVal:  10, // Should use default (as OptionalIntParam returns 0 for "")
			expectError:  false,
		},
		{
			name:         "Parameter present as 0",
			params:       map[string]interface{}{"optInt": float64(0)},
			paramName:    "optInt",
			defaultValue: 10,
			expectedVal:  10, // Expect default value, aligning with github-mcp-server behavior
			expectError:  false,
		},
		{
			name:         "Parameter present but wrong type (boolean)",
			params:       map[string]interface{}{"optInt": true},
			paramName:    "optInt",
			defaultValue: 10,
			expectedVal:  0, // Returns 0 on error from OptionalIntParam
			expectError:  true,
			errContains:  "must be convertible to an integer",
		},
		{
			name:         "Parameter present but invalid integer string",
			params:       map[string]interface{}{"optInt": "abc"},
			paramName:    "optInt",
			defaultValue: 10,
			expectedVal:  0,
			expectError:  true,
			errContains:  "must be a valid integer string",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := createMCPRequest(tc.params)
			val, err := OptionalIntParamWithDefault(req, tc.paramName, tc.defaultValue)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.expectedVal, val)
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
			name:            "Page provided as float64",
			params:          map[string]interface{}{"page": float64(5)},
			expectedPage:    5,
			expectedPerPage: DefaultPerPage,
			expectError:     false,
		},
		{
			name:            "Page provided as int string",
			params:          map[string]interface{}{"page": "3"},
			expectedPage:    3,
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
			name:            "PerPage provided as string",
			params:          map[string]interface{}{"per_page": "25"},
			expectedPage:    1,
			expectedPerPage: 25,
			expectError:     false,
		},
		{
			name:            "Both provided",
			params:          map[string]interface{}{"page": float64(2), "per_page": "15"},
			expectedPage:    2,
			expectedPerPage: 15,
			expectError:     false,
		},
		{
			name:            "Page zero provided, defaults to 1",
			params:          map[string]interface{}{"page": float64(0)},
			expectedPage:    1,
			expectedPerPage: DefaultPerPage,
			expectError:     false,
		},
		{
			name:            "Page negative provided, defaults to 1",
			params:          map[string]interface{}{"page": float64(-5)},
			expectedPage:    1,
			expectedPerPage: DefaultPerPage,
			expectError:     false,
		},
		{
			name:            "PerPage zero provided, uses default",
			params:          map[string]interface{}{"per_page": float64(0)},
			expectedPage:    1,
			expectedPerPage: DefaultPerPage,
			expectError:     false,
		},
		{
			name:            "PerPage negative provided, uses default",
			params:          map[string]interface{}{"per_page": float64(-10)},
			expectedPage:    1,
			expectedPerPage: DefaultPerPage,
			expectError:     false,
		},
		{
			name:            "PerPage greater than max, clamped to max",
			params:          map[string]interface{}{"per_page": float64(MaxPerPage + 50)},
			expectedPage:    1,
			expectedPerPage: MaxPerPage,
			expectError:     false,
		},
		{
			name:            "Invalid page type",
			params:          map[string]interface{}{"page": true},
			expectedPage:    0,
			expectedPerPage: 0,
			expectError:     true,
			errContains:     "invalid 'page' parameter: parameter 'page' must be convertible to an integer",
		},
		{
			name:            "Invalid per_page type",
			params:          map[string]interface{}{"per_page": "invalid"},
			expectedPage:    0,
			expectedPerPage: 0,
			expectError:     true,
			errContains:     "invalid 'per_page' parameter: parameter 'per_page' must be a valid integer string",
		},
		{
			name:            "Invalid page (non-whole float)",
			params:          map[string]interface{}{"page": 1.5},
			expectedPage:    0,
			expectedPerPage: 0,
			expectError:     true,
			errContains:     "invalid 'page' parameter: parameter 'page' must be a whole number",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := createMCPRequest(tc.params)
			page, perPage, err := OptionalPaginationParams(req)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				// Also check that page/perPage are zero on error for consistency
				assert.Zero(t, page, "page should be zero on error")
				assert.Zero(t, perPage, "perPage should be zero on error")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedPage, page, "page value")
				assert.Equal(t, tc.expectedPerPage, perPage, "perPage value")
			}
		})
	}
}
