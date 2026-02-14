package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// MockGHServer creates a test HTTP server that responds to GitHub API requests.
// Returns the server URL. The server is automatically closed when the test finishes.
func MockGHServer(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return server.URL
}

// GHRepoResponse creates an http.HandlerFunc that returns a GitHub repos API response.
func GHRepoResponse(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "100")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}
}

// GHPushableRepoJSON returns a JSON body for a repo with push permissions.
func GHPushableRepoJSON(owner, repo string) string {
	return fmt.Sprintf(`{
	"full_name": "%s/%s",
	"permissions": {
		"admin": false,
		"push": true,
		"pull": true
	}
}`, owner, repo)
}

// GHReadOnlyRepoJSON returns a JSON body for a repo with read-only permissions.
func GHReadOnlyRepoJSON(owner, repo string) string {
	return fmt.Sprintf(`{
	"full_name": "%s/%s",
	"permissions": {
		"admin": false,
		"push": false,
		"pull": true
	}
}`, owner, repo)
}

// GHNotFoundJSON returns a JSON body for a 404 response.
func GHNotFoundJSON() string {
	return `{"message": "Not Found", "documentation_url": "https://docs.github.com/rest"}`
}

// GHRateLimitResponse creates a handler that returns rate limit exceeded.
func GHRateLimitResponse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1700000000")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message": "API rate limit exceeded"}`)
	}
}
