package guard_test

import (
	"testing"
)

func TestCheck_AllMatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: repo with ctx-profile="work", remote SSH host matches, email matches
	// When: Check(repoDir) is called
	// Then: returns nil (guard pass)
}

func TestCheck_RemoteHostMismatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: ctx-profile="work" (ssh_host="github-company") but remote is "github.com"
	// When: Check is called
	// Then: returns GuardError with remote host mismatch details
}

func TestCheck_EmailMismatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: ctx-profile="work" (email="hbjs@company.com") but git email is "hbjs97@naver.com"
	// When: Check is called
	// Then: returns GuardError with email mismatch details
}

func TestCheck_NameMismatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: ctx-profile="work" (name="HBJS") but git name is "hbjs97"
	// When: Check is called
	// Then: returns GuardWarning (warning, not block by default)
}

func TestCheck_ProfileNotFound(t *testing.T) {
	t.Skip("not implemented")

	// Given: ctx-profile references a profile not in config.toml
	// When: Check is called
	// Then: returns error about unregistered profile
}

func TestCheck_NoCtxProfile(t *testing.T) {
	t.Skip("not implemented")

	// Given: repo without .git/ctx-profile
	// When: Check is called
	// Then: returns error suggesting "ctx init"
}

func TestCheck_SkipGuardEnv(t *testing.T) {
	t.Skip("not implemented")

	// Given: CTX_SKIP_GUARD=1 is set
	// When: Check is called
	// Then: returns nil (bypassed) with warning on stderr
}

func TestInstallHook_NoExistingHook(t *testing.T) {
	t.Skip("not implemented")

	// Given: repo with no pre-push hook
	// When: InstallHook(repoDir) is called
	// Then: .git/hooks/pre-push is created with ctx guard check invocation
}

func TestInstallHook_ExistingHook(t *testing.T) {
	t.Skip("not implemented")

	// Given: repo with existing pre-push hook
	// When: InstallHook is called
	// Then: original hook is backed up, new hook chains ctx guard + original
}

func TestInstallHook_HooksPath(t *testing.T) {
	t.Skip("not implemented")

	// Given: repo with core.hooksPath set (husky/lefthook)
	// When: InstallHook is called
	// Then: ctx guard is inserted into the hooksPath pre-push file
}

func TestUninstallHook(t *testing.T) {
	t.Skip("not implemented")

	// Given: repo with ctx guard installed in pre-push hook
	// When: UninstallHook(repoDir) is called
	// Then: ctx guard markers are removed, backup is restored if exists
}
