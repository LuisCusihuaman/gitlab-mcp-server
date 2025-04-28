package gitlab

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock GitLab API server for testing
func setupMockServer() *httptest.Server {
	mux := http.NewServeMux()
	// Add handlers for specific endpoints if needed for more complex tests
	mux.HandleFunc("/api/v4/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Private-Token") == "valid-token" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"id": 1, "username": "testuser"}`)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, `{"message": "401 Unauthorized"}`)
		}
	})

	return httptest.NewServer(mux)
}

func TestNewClient(t *testing.T) {
	server := setupMockServer()
	defer server.Close()

	tests := []struct {
		name        string
		token       string
		host        string
		expectError bool
		errContains string
	}{
		{
			name:        "Valid token and default host",
			token:       "valid-token",
			host:        "", // Use default gitlab.com
			expectError: false,
		},
		{
			name:        "Valid token and custom host",
			token:       "valid-token",
			host:        server.URL, // Use mock server URL
			expectError: false,
		},
		{
			name:        "Empty token",
			token:       "",
			host:        "",
			expectError: true,
			errContains: "token cannot be empty",
		},
		{
			name:        "Invalid custom host (client creation error)",
			token:       "valid-token",
			host:        ":invalid-url", // Force a client creation error
			expectError: true,
			errContains: "failed to create GitLab client",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(tc.token, tc.host)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				// Optionally, make a simple API call to verify connection if using mock host
				if tc.host == server.URL {
					user, _, apiErr := client.Users.CurrentUser()
					require.NoError(t, apiErr)
					require.NotNil(t, user)
					assert.Equal(t, "testuser", user.Username)
				}
			}
		})
	}
}
