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
			req := &mcp.CallToolRequest{
				Params: mcp.ToolInputParams{
					Arguments: tc.params,
				},
			}

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

// TODO: Add tests for OptionalParam, OptionalIntParam, OptionalPaginationParams
