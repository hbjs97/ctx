package guard_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/guard"
	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testProfile() *config.Profile {
	return &config.Profile{
		GHConfigDir: "/tmp/gh-work",
		SSHHost:     "github-work",
		GitName:     "Test User",
		GitEmail:    "test@company.com",
	}
}

func TestCheck_AllMatch(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("git -C", "git@github-work:org/repo.git\n", nil)
	// For user.email query
	fake.Responses["git -C /tmp/repo config --local user.email"] = testutil.Response{Output: []byte("test@company.com\n")}
	fake.Responses["git -C /tmp/repo config --local user.name"] = testutil.Response{Output: []byte("Test User\n")}
	fake.Responses["git -C /tmp/repo remote get-url origin"] = testutil.Response{Output: []byte("git@github-work:org/repo.git\n")}

	result, err := guard.Check(context.Background(), "/tmp/repo", testProfile(), fake)
	require.NoError(t, err)
	assert.True(t, result.Pass)
	assert.Empty(t, result.Violations)
}

func TestCheck_RemoteHostMismatch(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Responses["git -C /tmp/repo remote get-url origin"] = testutil.Response{Output: []byte("git@github-personal:org/repo.git\n")}
	fake.Responses["git -C /tmp/repo config --local user.email"] = testutil.Response{Output: []byte("test@company.com\n")}
	fake.Responses["git -C /tmp/repo config --local user.name"] = testutil.Response{Output: []byte("Test User\n")}

	result, err := guard.Check(context.Background(), "/tmp/repo", testProfile(), fake)
	require.NoError(t, err)
	assert.False(t, result.Pass)
	assert.Equal(t, "error", result.Violations[0].Severity)
	assert.Equal(t, "remote_host", result.Violations[0].Field)
}

func TestCheck_EmailMismatch(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Responses["git -C /tmp/repo remote get-url origin"] = testutil.Response{Output: []byte("git@github-work:org/repo.git\n")}
	fake.Responses["git -C /tmp/repo config --local user.email"] = testutil.Response{Output: []byte("wrong@email.com\n")}
	fake.Responses["git -C /tmp/repo config --local user.name"] = testutil.Response{Output: []byte("Test User\n")}

	result, err := guard.Check(context.Background(), "/tmp/repo", testProfile(), fake)
	require.NoError(t, err)
	assert.False(t, result.Pass)
	assert.Equal(t, "user_email", result.Violations[0].Field)
}

func TestCheck_NameMismatch(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Responses["git -C /tmp/repo remote get-url origin"] = testutil.Response{Output: []byte("git@github-work:org/repo.git\n")}
	fake.Responses["git -C /tmp/repo config --local user.email"] = testutil.Response{Output: []byte("test@company.com\n")}
	fake.Responses["git -C /tmp/repo config --local user.name"] = testutil.Response{Output: []byte("Wrong Name\n")}

	result, err := guard.Check(context.Background(), "/tmp/repo", testProfile(), fake)
	require.NoError(t, err)
	// Name mismatch is WARNING, not error — so Pass is still true
	assert.True(t, result.Pass)
	assert.Len(t, result.Violations, 1)
	assert.Equal(t, "warning", result.Violations[0].Severity)
	assert.Equal(t, "user_name", result.Violations[0].Field)
}

func TestCheck_SkipGuardEnv(t *testing.T) {
	t.Setenv("CTX_SKIP_GUARD", "1")
	fake := testutil.NewFakeCommander()

	result, err := guard.Check(context.Background(), "/tmp/repo", testProfile(), fake)
	require.NoError(t, err)
	assert.True(t, result.Pass)
}

func TestInstallHook_NoExisting(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	hookPath := filepath.Join(repoDir, ".git", "hooks", "pre-push")

	err := guard.InstallHook(repoDir)
	require.NoError(t, err)

	data, err := os.ReadFile(hookPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "# ctx-guard-start")
	assert.Contains(t, string(data), "# ctx-guard-end")
	assert.Contains(t, string(data), "command -v ctx")
	assert.Contains(t, string(data), "ctx guard check")

	info, _ := os.Stat(hookPath)
	assert.True(t, info.Mode()&0111 != 0, "hook should be executable")
}

func TestInstallHook_ContainsPathCheck(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(gitDir, 0755))

	err := guard.InstallHook(dir)
	require.NoError(t, err)

	hookPath := filepath.Join(gitDir, "pre-push")
	data, err := os.ReadFile(hookPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "command -v ctx")
	assert.Contains(t, content, "ctx guard check || exit 1")
	assert.Contains(t, content, "ctx-guard-start")
	assert.Contains(t, content, "ctx-guard-end")
}

func TestInstallHook_ExistingHook(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	hookDir := filepath.Join(repoDir, ".git", "hooks")
	os.MkdirAll(hookDir, 0755)
	hookPath := filepath.Join(hookDir, "pre-push")
	os.WriteFile(hookPath, []byte("#!/bin/sh\necho existing\n"), 0755)

	err := guard.InstallHook(repoDir)
	require.NoError(t, err)

	data, _ := os.ReadFile(hookPath)
	content := string(data)
	assert.Contains(t, content, "echo existing")
	assert.Contains(t, content, "# ctx-guard-start")
}

func TestUninstallHook(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)

	// First install
	guard.InstallHook(repoDir)

	// Then uninstall
	err := guard.UninstallHook(repoDir)
	require.NoError(t, err)

	hookPath := filepath.Join(repoDir, ".git", "hooks", "pre-push")
	data, err := os.ReadFile(hookPath)
	if err == nil {
		content := string(data)
		assert.NotContains(t, content, "# ctx-guard-start")
		assert.NotContains(t, content, "ctx guard check")
	}
}
