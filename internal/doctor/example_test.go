package doctor_test

import (
	"testing"
)

func TestCheckBinaries_AllPresent(t *testing.T) {
	t.Skip("not implemented")

	// Given: git, gh, ssh are all installed
	// When: CheckBinaries is called
	// Then: returns OK status for all three
}

func TestCheckBinaries_Missing(t *testing.T) {
	t.Skip("not implemented")

	// Given: gh is not installed
	// When: CheckBinaries is called
	// Then: returns FAIL for gh with install guidance
}

func TestCheckGHAuth_Authenticated(t *testing.T) {
	t.Skip("not implemented")

	// Given: profile with valid gh auth
	// When: CheckGHAuth(profile) is called
	// Then: returns OK status
}

func TestCheckGHAuth_NotAuthenticated(t *testing.T) {
	t.Skip("not implemented")

	// Given: profile with no gh auth
	// When: CheckGHAuth(profile) is called
	// Then: returns FAIL with "gh auth login" guidance
}

func TestCheckSSHConnection(t *testing.T) {
	t.Skip("not implemented")

	// Given: profile with ssh_host configured
	// When: CheckSSHConnection(profile) is called via FakeCommander
	// Then: returns OK when ssh -T succeeds, FAIL when it fails
}

func TestCheckIdentitiesOnly(t *testing.T) {
	t.Skip("not implemented")

	// Given: SSH config with/without IdentitiesOnly
	// When: CheckIdentitiesOnly is called
	// Then: returns OK when set, WARN when not set
}

func TestCheckEnvTokens(t *testing.T) {
	t.Skip("not implemented")

	// Given: GH_TOKEN or GITHUB_TOKEN is set in environment
	// When: CheckEnvTokens is called
	// Then: returns WARN about profile auth being overridden
}

func TestCheckHTTPSCredentialHelper(t *testing.T) {
	t.Skip("not implemented")

	// Given: osxkeychain credential helper for github.com
	// When: CheckHTTPSCredentialHelper is called
	// Then: returns FAIL about credential helper conflict
}

func TestCheckConfigValidity(t *testing.T) {
	t.Skip("not implemented")

	// Given: a valid config.toml
	// When: CheckConfigValidity is called
	// Then: returns OK
}

func TestCheckShellHook(t *testing.T) {
	t.Skip("not implemented")

	// Given: shell hook is/isn't installed in zshrc
	// When: CheckShellHook is called
	// Then: returns OK when installed, WARN when not with "ctx setup" guidance
}

func TestRunAll(t *testing.T) {
	t.Skip("not implemented")

	// Integration: run all doctor checks with FakeCommander
	// Verify the complete diagnostic report structure
}
