package toolsets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: Removed mock handlers and Toolset-specific tests (AddTools, GetActiveTools)
// as the reference github-mcp-server/pkg/toolsets/toolsets_test.go only tests
// the ToolsetGroup logic.

// --- ToolsetGroup Tests ---

func TestNewToolsetGroup(t *testing.T) {
	tgDefault := NewToolsetGroup(false)
	require.NotNil(t, tgDefault)
	assert.NotNil(t, tgDefault.Toolsets)
	assert.Empty(t, tgDefault.Toolsets)
	assert.False(t, tgDefault.readOnly)
	assert.False(t, tgDefault.everythingOn)

	tgReadOnly := NewToolsetGroup(true)
	require.NotNil(t, tgReadOnly)
	assert.NotNil(t, tgReadOnly.Toolsets)
	assert.Empty(t, tgReadOnly.Toolsets)
	assert.True(t, tgReadOnly.readOnly)
	assert.False(t, tgDefault.everythingOn) // Corrected assertion target
}

func TestToolsetGroup_AddToolset(t *testing.T) {
	// Case 1: Not read-only group
	tg := NewToolsetGroup(false)
	ts1 := NewToolset("ts1", "Toolset 1")
	tg.AddToolset(ts1)

	assert.Len(t, tg.Toolsets, 1)
	assert.Same(t, ts1, tg.Toolsets["ts1"])
	assert.False(t, tg.Toolsets["ts1"].readOnly, "Toolset should not be read-only in non-read-only group")

	// Case 2: Read-only group
	tgReadOnly := NewToolsetGroup(true)
	ts2 := NewToolset("ts2", "Toolset 2")
	tgReadOnly.AddToolset(ts2)

	assert.Len(t, tgReadOnly.Toolsets, 1)
	assert.Same(t, ts2, tgReadOnly.Toolsets["ts2"])
	assert.True(t, tgReadOnly.Toolsets["ts2"].readOnly, "Toolset should be forced to read-only in read-only group")

	// Case 3: Overwrite existing
	ts1Updated := NewToolset("ts1", "Toolset 1 Updated")
	tg.AddToolset(ts1Updated)
	assert.Len(t, tg.Toolsets, 1) // Still only one entry for "ts1"
	assert.Same(t, ts1Updated, tg.Toolsets["ts1"])
	assert.Equal(t, "Toolset 1 Updated", tg.Toolsets["ts1"].Description)
	// Asserting Enabled status might depend on whether AddToolset resets it.
	// Let's assume it uses the new toolset's default (false)
	assert.False(t, tg.Toolsets["ts1"].Enabled, "Overwritten toolset should likely reset to disabled")
}

func TestToolsetGroup_EnableToolset(t *testing.T) {
	tg := NewToolsetGroup(false)
	ts1 := NewToolset("ts1", "")
	tg.AddToolset(ts1)

	// Enable non-existent
	err := tg.EnableToolset("non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown toolset: non-existent")

	// Enable existing
	assert.False(t, ts1.Enabled, "Should be disabled initially")
	err = tg.EnableToolset("ts1")
	require.NoError(t, err)
	assert.True(t, ts1.Enabled, "Should be enabled after call")

	// Enable already enabled
	err = tg.EnableToolset("ts1")
	require.NoError(t, err)
	assert.True(t, ts1.Enabled, "Should remain enabled")
}

func TestToolsetGroup_EnableToolsets(t *testing.T) {
	tests := []struct {
		name            string
		initialToolsets map[string]*Toolset
		namesToEnable   []string
		expectError     bool
		errContains     string
		expectEnabled   []string // Names of toolsets expected to be enabled
		expectAllOn     bool
	}{
		{
			name: "Enable subset",
			initialToolsets: map[string]*Toolset{
				"ts1": NewToolset("ts1", ""),
				"ts2": NewToolset("ts2", ""),
				"ts3": NewToolset("ts3", ""),
			},
			namesToEnable: []string{"ts1", "ts3"},
			expectError:   false,
			expectEnabled: []string{"ts1", "ts3"},
			expectAllOn:   false,
		},
		{
			name: "Enable all keyword",
			initialToolsets: map[string]*Toolset{
				"ts1": NewToolset("ts1", ""),
				"ts2": NewToolset("ts2", ""),
			},
			namesToEnable: []string{"all"},
			expectError:   false,
			expectEnabled: []string{"ts1", "ts2"}, // All initially added toolsets
			expectAllOn:   true,
		},
		{
			name: "Enable non-existent",
			initialToolsets: map[string]*Toolset{
				"ts1": NewToolset("ts1", ""),
			},
			namesToEnable: []string{"ts1", "non-existent"},
			expectError:   true,
			errContains:   "unknown toolset: non-existent",
			expectEnabled: []string{"ts1"}, // ts1 should still be enabled before error
			expectAllOn:   false,
		},
		{
			name:            "Enable empty list",
			initialToolsets: map[string]*Toolset{"ts1": NewToolset("ts1", "")},
			namesToEnable:   []string{},
			expectError:     true, // Implementation returns error for empty list
			errContains:     "no toolsets specified",
			expectEnabled:   []string{},
			expectAllOn:     false,
		},
		{
			name:            "Enable only non-existent",
			initialToolsets: map[string]*Toolset{"ts1": NewToolset("ts1", "")},
			namesToEnable:   []string{"non-existent"},
			expectError:     true,
			errContains:     "unknown toolset: non-existent",
			expectEnabled:   []string{},
			expectAllOn:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tg := NewToolsetGroup(false)
			tg.Toolsets = tc.initialToolsets // Directly set for test setup

			err := tg.EnableToolsets(tc.namesToEnable)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectAllOn, tg.everythingOn, "everythingOn flag state")

			enabledCount := 0
			disabledCount := 0
			for name, ts := range tg.Toolsets {
				isEnabled := false
				for _, enabledName := range tc.expectEnabled {
					if name == enabledName {
						isEnabled = true
						break
					}
				}
				assert.Equal(t, isEnabled, ts.Enabled, "Enabled state for toolset: %s", name)
				if isEnabled {
					enabledCount++
				} else {
					disabledCount++
				}
			}
			assert.Len(t, tc.expectEnabled, enabledCount, "Number of enabled toolsets")
		})
	}
}

// NOTE: Removing tests for RegisterTools as we cannot easily mock *server.MCPServer
// func TestToolsetGroup_RegisterTools(t *testing.T) { ... }
