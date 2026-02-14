package cli_test

import (
	"testing"
)

func TestParseCloneTarget_ShortForm(t *testing.T) {
	t.Skip("not implemented")

	// Given: "owner/repo" as clone target
	// When: ParseCloneTarget is called
	// Then: returns owner="owner", repo="repo"
}

func TestParseCloneTarget_SSHURL(t *testing.T) {
	t.Skip("not implemented")

	// Given: "git@github.com:owner/repo.git" as clone target
	// When: ParseCloneTarget is called
	// Then: returns owner="owner", repo="repo"
}

func TestParseCloneTarget_HTTPSURL(t *testing.T) {
	t.Skip("not implemented")

	// Given: "https://github.com/owner/repo.git" as clone target
	// When: ParseCloneTarget is called
	// Then: returns owner="owner", repo="repo"
}

func TestExitCodeMapping(t *testing.T) {
	t.Skip("not implemented")

	// Given: various error types from resolver, guard, config
	// When: MapExitCode is called
	// Then: returns correct exit codes:
	//   0: success
	//   1: general error
	//   2: guard block
	//   3: ambiguous resolution
	//   4: auth/permission failure
	//   5: config error
	//   6: missing dependency
}

func TestStatusOutput_ManagedRepo(t *testing.T) {
	t.Skip("not implemented")

	// Given: a ctx-managed repo with work profile
	// When: status command is executed
	// Then: outputs profile, reason, remote, identity, guard status
}

func TestStatusOutput_UnmanagedRepo(t *testing.T) {
	t.Skip("not implemented")

	// Given: a repo without .git/ctx-profile
	// When: status command is executed
	// Then: outputs "not managed by ctx" message with "ctx init" suggestion
}

func TestStatusOutput_JSON(t *testing.T) {
	t.Skip("not implemented")

	// Given: --json flag
	// When: status command is executed
	// Then: outputs valid JSON with all status fields
}
