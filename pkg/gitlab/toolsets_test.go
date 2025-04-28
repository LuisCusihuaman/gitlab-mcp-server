package gitlab

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Import the actual gitlab client library
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Reintroduce mockGetClientFn for testing InitToolsets with the GetClientFn signature
func mockGetClientFn(_ context.Context) (*gitlab.Client, error) {
	// Return a nil client, as InitToolsets and tool definitions don't use it yet
	return nil, nil
}

// Mock TranslationHelperFunc (placeholder) - Kept commented out as it's not used
// func mockTranslationHelper(key string, defaultVal string, args ...interface{}) string {
// 	return defaultVal // Simple passthrough for now
// }

func TestInitToolsets(t *testing.T) {
	// Define the expected toolset names based on the implementation
	expectedToolsetNames := []string{
		"projects",
		"issues",
		"merge_requests",
		"security",
		"users",
		"search",
	}

	tests := []struct {
		name            string
		enabledToolsets []string
		readOnly        bool
		expectError     bool
		errContains     string
		expectEnabled   []string // Which toolsets should end up enabled
		// Removed checks for unexported fields (everythingOn, groupReadOnly)
		// These should be tested within the toolsets package itself.
	}{
		{
			name:            "Enable specific toolsets, not read-only",
			enabledToolsets: []string{"projects", "issues"},
			readOnly:        false,
			expectError:     false,
			expectEnabled:   []string{"projects", "issues"},
		},
		{
			name:            "Enable all toolsets, not read-only",
			enabledToolsets: []string{"all"},
			readOnly:        false,
			expectError:     false,
			expectEnabled:   expectedToolsetNames, // All defined toolsets
		},
		{
			name:            "Enable specific toolsets, read-only group",
			enabledToolsets: []string{"users", "search"},
			readOnly:        true,
			expectError:     false,
			expectEnabled:   []string{"users", "search"},
		},
		{
			name:            "Enable all toolsets, read-only group",
			enabledToolsets: []string{"all"},
			readOnly:        true,
			expectError:     false,
			expectEnabled:   expectedToolsetNames,
		},
		{
			name:            "Enable non-existent toolset",
			enabledToolsets: []string{"projects", "invalid-toolset"},
			readOnly:        false,
			expectError:     true,
			errContains:     "unknown toolset: invalid-toolset",
			expectEnabled:   []string{"projects"}, // projects should still be enabled before error
		},
		{
			name:            "Enable empty list",
			enabledToolsets: []string{},
			readOnly:        false,
			expectError:     true,
			errContains:     "no toolsets specified",
			expectEnabled:   []string{}, // None should be enabled
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Call InitToolsets using the mock function again
			// Pass nil for the translation helper for now
			tg, err := InitToolsets(tc.enabledToolsets, tc.readOnly, mockGetClientFn /*, nil */)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				// We might get a partially configured tg even on error
				if tg == nil {
					return // Nothing more to check if tg is nil
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, tg, "ToolsetGroup should not be nil on success")
			}

			// Removed checks for tg.readOnly and tg.everythingOn as they are unexported

			// Verify all expected toolsets exist in the returned group
			assert.Len(t, tg.Toolsets, len(expectedToolsetNames), "Should contain all defined toolsets")
			for _, name := range expectedToolsetNames {
				assert.Contains(t, tg.Toolsets, name, "Expected toolset %s to be in the group", name)
			}

			// Verify enabled status
			enabledMap := make(map[string]bool)
			for _, name := range tc.expectEnabled {
				enabledMap[name] = true
			}

			for name, ts := range tg.Toolsets {
				expectedEnabled := enabledMap[name]
				assert.Equal(t, expectedEnabled, ts.Enabled, "Enabled status mismatch for toolset: %s", name)
				// Removed check for ts.readOnly as it's unexported and tested in toolsets package
			}
		})
	}
}
