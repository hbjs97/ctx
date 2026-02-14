package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hbjs97/ctx/internal/cli"
	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTestConfig creates a test config.toml with work and personal profiles.
// Returns the config file path.
func writeTestConfig(t *testing.T, dir string) string {
	t.Helper()
	cfg := `version = 1
require_push_guard = true

[profiles.work]
gh_config_dir = "/tmp/gh-work"
ssh_host = "gh-work"
git_name = "Test User"
git_email = "test@work.com"
owners = ["myorg"]

[profiles.personal]
gh_config_dir = "/tmp/gh-personal"
ssh_host = "gh-personal"
git_name = "Personal User"
git_email = "me@personal.com"
owners = ["myuser"]
`
	cfgPath := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfg), 0600))
	return cfgPath
}

// writeTestConfigNoGuard creates a test config with require_push_guard = false.
func writeTestConfigNoGuard(t *testing.T, dir string) string {
	t.Helper()
	cfg := `version = 1
require_push_guard = false

[profiles.work]
gh_config_dir = "/tmp/gh-work"
ssh_host = "gh-work"
git_name = "Test User"
git_email = "test@work.com"
owners = ["myorg"]

[profiles.personal]
gh_config_dir = "/tmp/gh-personal"
ssh_host = "gh-personal"
git_name = "Personal User"
git_email = "me@personal.com"
owners = ["myuser"]
`
	cfgPath := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfg), 0600))
	return cfgPath
}

// writeTestConfigWithDefault creates a config with a default_profile.
func writeTestConfigWithDefault(t *testing.T, dir string) string {
	t.Helper()
	cfg := `version = 1
default_profile = "personal"

[profiles.work]
gh_config_dir = "/tmp/gh-work"
ssh_host = "gh-work"
git_name = "Test User"
git_email = "test@work.com"
owners = ["myorg"]

[profiles.personal]
gh_config_dir = "/tmp/gh-personal"
ssh_host = "gh-personal"
git_name = "Personal User"
git_email = "me@personal.com"
owners = ["myuser"]
`
	cfgPath := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfg), 0600))
	return cfgPath
}

// newTestApp creates an App with a FakeCommander and the given config path.
func newTestApp(t *testing.T, fc *testutil.FakeCommander, cfgPath string) *cli.App {
	t.Helper()
	return &cli.App{
		Commander: fc,
		CfgPath:   cfgPath,
	}
}

// --- Clone command tests ---

func TestCloneCmd_OwnerRuleMatch(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	// clone calls git clone and git -C (for setting local config)
	fc.Register("git clone", "", nil)
	fc.Register("git -C", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "clone", "myorg/myrepo"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify git clone was called with SSH URL using work profile's ssh_host
	assert.True(t, fc.Called("git clone"))
	cloneCall := ""
	for _, c := range fc.Calls {
		if len(c) > 9 && c[:9] == "git clone" {
			cloneCall = c
			break
		}
	}
	assert.Contains(t, cloneCall, "git@gh-work:myorg/myrepo.git")
}

func TestCloneCmd_ExplicitProfile(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git clone", "", nil)
	fc.Register("git -C", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "clone", "--profile", "personal", "myorg/myrepo"})

	err := cmd.Execute()
	require.NoError(t, err)

	// With explicit personal profile, SSH host should be gh-personal
	cloneCall := ""
	for _, c := range fc.Calls {
		if len(c) > 9 && c[:9] == "git clone" {
			cloneCall = c
			break
		}
	}
	assert.Contains(t, cloneCall, "git@gh-personal:myorg/myrepo.git")
}

func TestCloneCmd_InvalidTarget(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", cfgPath, "clone", "invalid-no-slash"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "owner/repo")
}

func TestCloneCmd_NoArgs(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", cfgPath, "clone"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestCloneCmd_InvalidProfile(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", cfgPath, "clone", "--profile", "nonexistent", "myorg/myrepo"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestCloneCmd_BadConfig(t *testing.T) {
	t.Parallel()

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, "/nonexistent/config.toml")
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", "/nonexistent/config.toml", "clone", "myorg/myrepo"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestCloneCmd_CloneFails(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git clone", "", fmt.Errorf("clone failed: network error"))

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", cfgPath, "clone", "myorg/myrepo"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "clone failed")
}

func TestCloneCmd_NoGuardFlag(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git clone", "", nil)
	fc.Register("git -C", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "clone", "--no-guard", "myorg/myrepo"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.True(t, fc.Called("git clone"))
}

// --- Init command tests ---

func TestInitCmd_Success(t *testing.T) {
	// init uses os.Getwd() so we need t.Chdir
	repoDir := testutil.TempGitRepoWithRemote(t, "git@gh-work:myorg/myrepo.git")
	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	// init calls GetRemoteURL via git adapter
	fc.Register("git -C "+repoDir+" remote get-url origin", "git@gh-work:myorg/myrepo.git", nil)
	// init calls SetLocalConfig
	fc.Register("git -C "+repoDir+" config --local", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "init"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify ctx-profile was written
	profilePath := filepath.Join(repoDir, ".git", "ctx-profile")
	data, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "work")
}

func TestInitCmd_ExplicitProfile(t *testing.T) {
	repoDir := testutil.TempGitRepoWithRemote(t, "git@gh-personal:myuser/myrepo.git")
	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git -C "+repoDir+" remote get-url origin", "git@gh-personal:myuser/myrepo.git", nil)
	fc.Register("git -C "+repoDir+" config --local", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "init", "--profile", "personal"})

	err := cmd.Execute()
	require.NoError(t, err)

	profilePath := filepath.Join(repoDir, ".git", "ctx-profile")
	data, err := os.ReadFile(profilePath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "personal")
}

func TestInitCmd_NoRemote(t *testing.T) {
	// repo without remote
	repoDir := testutil.TempGitRepo(t)
	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git -C "+repoDir+" remote get-url origin", "", fmt.Errorf("fatal: no such remote 'origin'"))

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", cfgPath, "init"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "origin remote")
}

func TestInitCmd_HTTPSRemoteConverted(t *testing.T) {
	repoDir := testutil.TempGitRepoWithRemote(t, "https://github.com/myorg/myrepo.git")
	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir) // allow_https_managed_repo defaults to false

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git -C "+repoDir+" remote get-url origin", "https://github.com/myorg/myrepo.git", nil)
	fc.Register("git -C "+repoDir+" remote set-url", "", nil)
	fc.Register("git -C "+repoDir+" config --local", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "init"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify remote set-url was called
	assert.True(t, fc.Called("git -C "+repoDir+" remote set-url"))
}

func TestInitCmd_BadConfig(t *testing.T) {
	repoDir := testutil.TempGitRepoWithRemote(t, "git@gh-work:myorg/myrepo.git")
	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git -C "+repoDir+" remote get-url origin", "git@gh-work:myorg/myrepo.git", nil)

	app := newTestApp(t, fc, "/nonexistent/config.toml")
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", "/nonexistent/config.toml", "init"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestInitCmd_NoGuardFlag(t *testing.T) {
	repoDir := testutil.TempGitRepoWithRemote(t, "git@gh-work:myorg/myrepo.git")
	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git -C "+repoDir+" remote get-url origin", "git@gh-work:myorg/myrepo.git", nil)
	fc.Register("git -C "+repoDir+" config --local", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "init", "--no-guard"})

	err := cmd.Execute()
	require.NoError(t, err)
}

// --- Status command tests ---

func TestStatusCmd_NoProfile(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "status"})

	err := cmd.Execute()
	require.NoError(t, err)
	// No ctx-profile => should suggest running ctx init (printed to stdout)
}

func TestStatusCmd_WithProfile(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	testutil.WriteCtxProfile(t, repoDir, "work")

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git -C "+repoDir+" remote get-url origin", "git@gh-work:myorg/myrepo.git", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "status"})

	err := cmd.Execute()
	require.NoError(t, err)
	// Status prints profile info to stdout (via fmt.Printf, not cmd.SetOut)
}

func TestStatusCmd_ProfileNotInConfig(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	testutil.WriteCtxProfile(t, repoDir, "nonexistent")

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", cfgPath, "status"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestStatusCmd_BadConfig(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	testutil.WriteCtxProfile(t, repoDir, "work")

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, "/nonexistent/config.toml")
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", "/nonexistent/config.toml", "status"})

	err := cmd.Execute()
	assert.Error(t, err)
}

// --- Doctor command tests ---

func TestDoctorCmd_AllBinariesPresent(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	// doctor runs CheckBinaries then RunAll per profile
	fc.Register("git --version", "git version 2.44.0", nil)
	fc.Register("gh --version", "gh version 2.45.0", nil)
	fc.Register("ssh -V", "OpenSSH_9.6", nil)
	fc.Register("gh auth status", "Logged in", nil)
	fc.Register("ssh -T", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "doctor"})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestDoctorCmd_NoConfig(t *testing.T) {
	t.Parallel()

	fc := testutil.NewFakeCommander()
	// Without config, doctor runs CheckBinaries only
	fc.Register("git --version", "git version 2.44.0", nil)
	fc.Register("gh --version", "gh version 2.45.0", nil)
	fc.Register("ssh -V", "OpenSSH_9.6", nil)

	app := newTestApp(t, fc, "/nonexistent/config.toml")
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", "/nonexistent/config.toml", "doctor"})

	err := cmd.Execute()
	// doctor does not return error even when config is missing
	require.NoError(t, err)
}

func TestDoctorCmd_MissingBinaries(t *testing.T) {
	t.Parallel()

	fc := testutil.NewFakeCommander()
	fc.Register("git --version", "", fmt.Errorf("not found"))
	fc.Register("gh --version", "", fmt.Errorf("not found"))
	fc.Register("ssh -V", "", fmt.Errorf("not found"))

	app := newTestApp(t, fc, "/nonexistent/config.toml")
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", "/nonexistent/config.toml", "doctor"})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestDoctorCmd_WithConfigAndFailedAuth(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git --version", "git version 2.44.0", nil)
	fc.Register("gh --version", "gh version 2.45.0", nil)
	fc.Register("ssh -V", "OpenSSH_9.6", nil)
	fc.Register("gh auth status", "", fmt.Errorf("not logged in"))
	fc.Register("ssh -T", "", fmt.Errorf("Permission denied"))

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "doctor"})

	err := cmd.Execute()
	require.NoError(t, err)
}

// --- Guard check command tests ---

func TestGuardCheckCmd_Pass(t *testing.T) {
	repoDir := testutil.TempGitRepoWithRemote(t, "git@gh-work:myorg/myrepo.git")
	testutil.WriteCtxProfile(t, repoDir, "work")

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git -C "+repoDir+" remote get-url origin", "git@gh-work:myorg/myrepo.git", nil)
	fc.Register("git -C "+repoDir+" config --local user.email", "test@work.com", nil)
	fc.Register("git -C "+repoDir+" config --local user.name", "Test User", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "guard", "check"})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestGuardCheckCmd_Fail_EmailMismatch(t *testing.T) {
	repoDir := testutil.TempGitRepoWithRemote(t, "git@gh-work:myorg/myrepo.git")
	testutil.WriteCtxProfile(t, repoDir, "work")

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git -C "+repoDir+" remote get-url origin", "git@gh-work:myorg/myrepo.git", nil)
	fc.Register("git -C "+repoDir+" config --local user.email", "wrong@other.com", nil)
	fc.Register("git -C "+repoDir+" config --local user.name", "Test User", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", cfgPath, "guard", "check"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "guard")
}

func TestGuardCheckCmd_Fail_HostMismatch(t *testing.T) {
	repoDir := testutil.TempGitRepoWithRemote(t, "git@wrong-host:myorg/myrepo.git")
	testutil.WriteCtxProfile(t, repoDir, "work")

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git -C "+repoDir+" remote get-url origin", "git@wrong-host:myorg/myrepo.git", nil)
	fc.Register("git -C "+repoDir+" config --local user.email", "test@work.com", nil)
	fc.Register("git -C "+repoDir+" config --local user.name", "Test User", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", cfgPath, "guard", "check"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "guard")
}

func TestGuardCheckCmd_NoProfile(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", cfgPath, "guard", "check"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ctx-profile")
}

func TestGuardCheckCmd_ProfileNotInConfig(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	testutil.WriteCtxProfile(t, repoDir, "nonexistent")

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", cfgPath, "guard", "check"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestGuardCheckCmd_BadConfig(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	testutil.WriteCtxProfile(t, repoDir, "work")

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, "/nonexistent/config.toml")
	cmd := app.NewRootCmd()

	errBuf := new(bytes.Buffer)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"--config", "/nonexistent/config.toml", "guard", "check"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestGuardCheckCmd_WarningOnly_NameMismatch(t *testing.T) {
	repoDir := testutil.TempGitRepoWithRemote(t, "git@gh-work:myorg/myrepo.git")
	testutil.WriteCtxProfile(t, repoDir, "work")

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git -C "+repoDir+" remote get-url origin", "git@gh-work:myorg/myrepo.git", nil)
	fc.Register("git -C "+repoDir+" config --local user.email", "test@work.com", nil)
	fc.Register("git -C "+repoDir+" config --local user.name", "Different Name", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "guard", "check"})

	// Name mismatch is only a warning, should still pass
	err := cmd.Execute()
	require.NoError(t, err)
}

// --- Activate command tests ---

func TestActivateCmd_HookOnly_Zsh(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "activate", "--hook", "--shell", "zsh"})

	err := cmd.Execute()
	require.NoError(t, err)
	// Hook snippet is printed to stdout (via fmt.Print)
}

func TestActivateCmd_HookOnly_Bash(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "activate", "--hook", "--shell", "bash"})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestActivateCmd_HookOnly_Fish(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "activate", "--hook", "--shell", "fish"})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestActivateCmd_WithProfile(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	testutil.WriteCtxProfile(t, repoDir, "work")

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "activate", "--shell", "zsh"})

	err := cmd.Execute()
	require.NoError(t, err)
	// Activate should output export commands (via fmt.Print to stdout)
}

func TestActivateCmd_NoProfile_NoDefault(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir) // no default_profile

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "activate", "--shell", "zsh"})

	err := cmd.Execute()
	require.NoError(t, err)
	// Should deactivate (unset)
}

func TestActivateCmd_NoProfile_WithDefault(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	cfgDir := t.TempDir()
	cfgPath := writeTestConfigWithDefault(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "activate", "--shell", "zsh"})

	err := cmd.Execute()
	require.NoError(t, err)
	// Should activate with default profile
}

func TestActivateCmd_BadConfig(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, "/nonexistent/config.toml")
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", "/nonexistent/config.toml", "activate", "--shell", "zsh"})

	// activate does not error on bad config, just deactivates
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestActivateCmd_ProfileNotInConfig(t *testing.T) {
	repoDir := testutil.TempGitRepo(t)
	testutil.WriteCtxProfile(t, repoDir, "nonexistent")

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	t.Chdir(repoDir)

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "activate", "--shell", "zsh"})

	// activate does not error on unknown profile, just deactivates
	err := cmd.Execute()
	require.NoError(t, err)
}

// --- Root command tests ---

func TestRootCmd_VerboseFlag(t *testing.T) {
	t.Parallel()

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, "/tmp/config.toml")
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--verbose", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)
}

func TestRootCmd_ConfigFlag(t *testing.T) {
	t.Parallel()

	fc := testutil.NewFakeCommander()
	app := newTestApp(t, fc, "/tmp/config.toml")
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", "/custom/path.toml", "--help"})

	err := cmd.Execute()
	require.NoError(t, err)
}

// --- Guard subcommand structure test ---

func TestGuardCmd_NoSubcommand(t *testing.T) {
	t.Parallel()

	fc := testutil.NewFakeCommander()
	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)
	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"guard"})

	// guard without subcommand should show help (no error from cobra for parent commands)
	err := cmd.Execute()
	require.NoError(t, err)
}

// --- statusIcon coverage ---

func TestDoctorCmd_StatusIconCoverage(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	// Mix of ok, fail results
	fc.Register("git --version", "git version 2.44.0", nil)
	fc.Register("gh --version", "", fmt.Errorf("not found"))
	fc.Register("ssh -V", "OpenSSH_9.6", nil)
	// For profiles, auth fail and ssh fail
	fc.Register("gh auth status", "", fmt.Errorf("not logged in"))
	fc.Register("ssh -T", "", fmt.Errorf("connection refused"))

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "doctor"})

	err := cmd.Execute()
	require.NoError(t, err)
}

// --- Clone with AmbiguousOwner (owner matches no profile -> goes to probe) ---

func TestCloneCmd_UnknownOwner_ProbeSuccess(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	// "unknownorg" is not in any profile's owners, so resolver goes to probe
	// Probe for work profile => push:true, personal => push:false
	fc.Register("gh api repos/unknownorg/somerepo --hostname github.com", `{"permissions":{"push":true}}`, nil)
	fc.Register("git clone", "", nil)
	fc.Register("git -C", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "clone", "unknownorg/somerepo"})

	// This will call ProbeAllProfiles via RunWithEnv.
	// FakeCommander uses Run for RunWithEnv, so we need to register the gh api call.
	// The probe calls for both profiles. Since FakeCommander prefix-matches,
	// both will match the same "gh api" response.
	// Both profiles will get push:true, resulting in ambiguous.
	err := cmd.Execute()
	// With both profiles having push access, this is ambiguous
	assert.Error(t, err)
}

// --- NewApp coverage ---

func TestNewApp(t *testing.T) {
	t.Parallel()

	app := cli.NewApp()
	assert.NotNil(t, app)
	assert.NotNil(t, app.Commander)
}

// --- cachePath coverage ---

func TestCloneCmd_CacheIsSaved(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git clone", "", nil)
	fc.Register("git -C", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "clone", "myorg/myrepo"})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify cache.json was created in the same directory as config
	cachePath := filepath.Join(cfgDir, "cache.json")
	_, err = os.Stat(cachePath)
	assert.NoError(t, err)
}

// --- Clone with SSH URL format ---

func TestCloneCmd_SSHUrlFormat(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git clone", "", nil)
	fc.Register("git -C", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "clone", "git@github.com:myorg/myrepo.git"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.True(t, fc.Called("git clone"))
}

func TestCloneCmd_HTTPSUrlFormat(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	cfgPath := writeTestConfig(t, cfgDir)

	fc := testutil.NewFakeCommander()
	fc.Register("git clone", "", nil)
	fc.Register("git -C", "", nil)

	app := newTestApp(t, fc, cfgPath)
	cmd := app.NewRootCmd()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--config", cfgPath, "clone", "https://github.com/myorg/myrepo.git"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.True(t, fc.Called("git clone"))
}
