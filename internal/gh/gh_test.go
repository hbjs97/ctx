package gh_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hbjs97/ctx/internal/gh"
	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeRepo_PushAccess(t *testing.T) {
	t.Parallel()

	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", `{"permissions":{"push":true,"admin":false}}`, nil)

	a := gh.NewAdapter(fake)
	result, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

	require.NoError(t, err)
	assert.True(t, result.HasAccess)
	assert.True(t, result.CanPush)

	// Verify GH_CONFIG_DIR was passed via RunWithEnv.
	require.Len(t, fake.EnvCalls, 1)
	assert.Equal(t, "/tmp/gh-work", fake.EnvCalls[0]["GH_CONFIG_DIR"])
}

func TestProbeRepo_ReadOnly(t *testing.T) {
	t.Parallel()

	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", `{"permissions":{"push":false}}`, nil)

	a := gh.NewAdapter(fake)
	result, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

	require.NoError(t, err)
	assert.True(t, result.HasAccess)
	assert.False(t, result.CanPush)

	require.Len(t, fake.EnvCalls, 1)
	assert.Equal(t, "/tmp/gh-work", fake.EnvCalls[0]["GH_CONFIG_DIR"])
}

func TestProbeRepo_NotFound(t *testing.T) {
	t.Parallel()

	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", "", fmt.Errorf("HTTP 404: Not Found"))

	a := gh.NewAdapter(fake)
	result, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

	require.NoError(t, err)
	assert.False(t, result.HasAccess)

	require.Len(t, fake.EnvCalls, 1)
	assert.Equal(t, "/tmp/gh-work", fake.EnvCalls[0]["GH_CONFIG_DIR"])
}

func TestProbeRepo_Unauthorized(t *testing.T) {
	t.Parallel()

	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", "", fmt.Errorf("HTTP 401"))

	a := gh.NewAdapter(fake)
	result, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

	require.NoError(t, err)
	assert.False(t, result.HasAccess)

	require.Len(t, fake.EnvCalls, 1)
	assert.Equal(t, "/tmp/gh-work", fake.EnvCalls[0]["GH_CONFIG_DIR"])
}

func TestProbeRepo_ErrorInStdout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		stdout string
		stderr error
	}{
		{
			name:   "404 in stdout only",
			stdout: `{"message":"Not Found","status":"404"}`,
			stderr: fmt.Errorf("exit status 1"),
		},
		{
			name:   "403 in stdout only",
			stdout: `{"message":"Forbidden","status":"403"}`,
			stderr: fmt.Errorf("exit status 1"),
		},
		{
			name:   "401 in stdout only",
			stdout: `{"message":"Bad credentials","status":"401"}`,
			stderr: fmt.Errorf("exit status 1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fake := testutil.NewFakeCommander()
			fake.Register("gh api repos/org/repo", tt.stdout, tt.stderr)

			a := gh.NewAdapter(fake)
			result, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

			require.NoError(t, err)
			assert.False(t, result.HasAccess)
		})
	}
}

func TestProbeRepo_UnexpectedError(t *testing.T) {
	t.Parallel()

	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", "", fmt.Errorf("network timeout"))

	a := gh.NewAdapter(fake)
	_, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "gh.ProbeRepo:")
}

func TestProbeRepo_InvalidJSON(t *testing.T) {
	t.Parallel()

	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", `not json`, nil)

	a := gh.NewAdapter(fake)
	_, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "gh.ProbeRepo: JSON 파싱 실패:")
}

func TestProbeRepo_SuppressesEnvTokens(t *testing.T) {
	t.Setenv("GH_TOKEN", "ghp_secret123")

	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", `{"permissions":{"push":true}}`, nil)

	a := gh.NewAdapter(fake)
	_, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

	require.NoError(t, err)
	require.Len(t, fake.EnvCalls, 1)
	assert.Equal(t, "/tmp/gh-work", fake.EnvCalls[0]["GH_CONFIG_DIR"])
	assert.Equal(t, "", fake.EnvCalls[0]["GH_TOKEN"])
}

func TestProbeAllProfiles(t *testing.T) {
	t.Parallel()

	fake := testutil.NewFakeCommander()
	fake.DefaultResponse = &testutil.Response{
		Output: []byte(`{"permissions":{"push":true}}`),
	}

	a := gh.NewAdapter(fake)
	profiles := map[string]string{
		"work":     "/tmp/gh-work",
		"personal": "/tmp/gh-personal",
	}
	results, err := a.ProbeAllProfiles(context.Background(), "org", "repo", profiles)

	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Each call should have used RunWithEnv with GH_CONFIG_DIR.
	require.Len(t, fake.EnvCalls, 2)
	envDirs := make(map[string]bool)
	for _, envCall := range fake.EnvCalls {
		envDirs[envCall["GH_CONFIG_DIR"]] = true
	}
	assert.True(t, envDirs["/tmp/gh-work"], "expected GH_CONFIG_DIR=/tmp/gh-work")
	assert.True(t, envDirs["/tmp/gh-personal"], "expected GH_CONFIG_DIR=/tmp/gh-personal")
}

func TestProbeAllProfiles_Error(t *testing.T) {
	t.Parallel()

	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", "", fmt.Errorf("network error"))

	a := gh.NewAdapter(fake)
	profiles := map[string]string{
		"work": "/tmp/gh-work",
	}
	_, err := a.ProbeAllProfiles(context.Background(), "org", "repo", profiles)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "gh.ProbeAllProfiles[work]:")
}

func TestSuppressEnvTokens_NoTokensSet(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	env := gh.SuppressEnvTokens()

	assert.Empty(t, env)
}

func TestSuppressEnvTokens_GHTokenSet(t *testing.T) {
	t.Setenv("GH_TOKEN", "ghp_test123")
	t.Setenv("GITHUB_TOKEN", "")

	env := gh.SuppressEnvTokens()

	assert.Len(t, env, 1)
	assert.Equal(t, "", env["GH_TOKEN"])
	_, hasGitHub := env["GITHUB_TOKEN"]
	assert.False(t, hasGitHub)
}

func TestSuppressEnvTokens_GitHubTokenSet(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "github_pat_abc")

	env := gh.SuppressEnvTokens()

	assert.Len(t, env, 1)
	assert.Equal(t, "", env["GITHUB_TOKEN"])
	_, hasGH := env["GH_TOKEN"]
	assert.False(t, hasGH)
}

func TestSuppressEnvTokens_BothTokensSet(t *testing.T) {
	t.Setenv("GH_TOKEN", "ghp_test123")
	t.Setenv("GITHUB_TOKEN", "github_pat_abc")

	env := gh.SuppressEnvTokens()

	assert.Len(t, env, 2)
	assert.Equal(t, "", env["GH_TOKEN"])
	assert.Equal(t, "", env["GITHUB_TOKEN"])
}

func TestDetectEnvTokenInterference(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	_, found := gh.DetectEnvTokenInterference()
	assert.False(t, found)

	t.Setenv("GH_TOKEN", "ghp_test123")
	key, found := gh.DetectEnvTokenInterference()
	assert.True(t, found)
	assert.Equal(t, "GH_TOKEN", key)
}
