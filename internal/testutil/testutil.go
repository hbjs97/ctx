// Package testutil provides common test helpers for the ctx project.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TempGitRepo creates a temporary git repository and returns its path.
// The repository is automatically cleaned up when the test finishes.
func TempGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.name", "test-user"},
		{"git", "config", "user.email", "test@example.com"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("TempGitRepo: %s failed: %v\n%s", args[0], err, out)
		}
	}

	return dir
}

// TempGitRepoWithRemote creates a temporary git repository with a remote
// configured to the given URL.
func TempGitRepoWithRemote(t *testing.T, remoteURL string) string {
	t.Helper()

	dir := TempGitRepo(t)

	cmd := exec.Command("git", "remote", "add", "origin", remoteURL)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("TempGitRepoWithRemote: git remote add failed: %v\n%s", err, out)
	}

	return dir
}

// TempBareRepo creates a temporary bare git repository and returns its path.
// Useful for E2E tests that need a push target.
func TempBareRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("TempBareRepo: git init --bare failed: %v\n%s", err, out)
	}

	return dir
}

// TempConfigFile creates a temporary config.toml with the given content
// and returns its path. The file is automatically cleaned up.
func TempConfigFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("TempConfigFile: write failed: %v", err)
	}

	return path
}

// TempCacheFile creates a temporary cache.json with the given content
// and returns its path.
func TempCacheFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("TempCacheFile: write failed: %v", err)
	}

	return path
}

// SetupTestProfiles creates a temporary config.toml with work and personal
// profiles pre-configured. Returns the config file path.
func SetupTestProfiles(t *testing.T) string {
	t.Helper()

	content := `version = 1
default_profile = "personal"
prompt_on_ambiguous = true
require_push_guard = true
allow_https_managed_repo = false

[profiles.work]
gh_config_dir = "/tmp/gh-work"
ssh_host = "github-company"
git_name = "HBJS"
git_email = "hbjs@company.com"
email_domain = "company.com"
owners = ["company-org", "company-team"]

[profiles.personal]
gh_config_dir = "/tmp/gh-personal"
ssh_host = "github-personal"
git_name = "hbjs97"
git_email = "hbjs97@naver.com"
email_domain = "naver.com"
owners = ["hbjs97", "sutefu23"]
`
	return TempConfigFile(t, content)
}

// WriteCtxProfile writes a ctx-profile file in the given git repo's .git directory.
func WriteCtxProfile(t *testing.T, repoDir string, profileName string) {
	t.Helper()

	profilePath := filepath.Join(repoDir, ".git", "ctx-profile")
	if err := os.WriteFile(profilePath, []byte(profileName+"\n"), 0600); err != nil {
		t.Fatalf("WriteCtxProfile: write failed: %v", err)
	}
}

// ReadCtxProfile reads the ctx-profile from the given git repo's .git directory.
func ReadCtxProfile(t *testing.T, repoDir string) string {
	t.Helper()

	profilePath := filepath.Join(repoDir, ".git", "ctx-profile")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("ReadCtxProfile: read failed: %v", err)
	}

	return string(data)
}
