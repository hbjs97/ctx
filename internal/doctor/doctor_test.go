package doctor_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hbjs97/ctx/internal/doctor"
	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckBinaries_AllPresent(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("git --version", "git version 2.40.0", nil)
	fake.Register("gh --version", "gh version 2.30.0", nil)
	fake.Register("ssh -V", "OpenSSH_9.0", nil)

	results := doctor.CheckBinaries(context.Background(), fake)
	for _, r := range results {
		assert.Equal(t, doctor.StatusOK, r.Status, "check %s should be OK", r.Name)
	}
}

func TestCheckBinaries_GitMissing(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("git --version", "", fmt.Errorf("not found"))
	fake.Register("gh --version", "gh version 2.30.0", nil)
	fake.Register("ssh -V", "OpenSSH_9.0", nil)

	results := doctor.CheckBinaries(context.Background(), fake)
	var gitResult *doctor.DiagResult
	for _, r := range results {
		if r.Name == "git" {
			gitResult = &r
			break
		}
	}
	require.NotNil(t, gitResult)
	assert.Equal(t, doctor.StatusFail, gitResult.Status)
	assert.NotEmpty(t, gitResult.Fix)
}

func TestCheckGHAuth_Authenticated(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("gh auth status", "Logged in to github.com", nil)

	result := doctor.CheckGHAuth(context.Background(), fake, "/tmp/gh-config")
	assert.Equal(t, doctor.StatusOK, result.Status)
}

func TestCheckGHAuth_NotAuthenticated(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("gh auth status", "", fmt.Errorf("not logged in"))

	result := doctor.CheckGHAuth(context.Background(), fake, "/tmp/gh-config")
	assert.Equal(t, doctor.StatusFail, result.Status)
	assert.Contains(t, result.Fix, "gh auth login")
}

func TestCheckSSHConnection_Success(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("ssh -T", "Hi user!", nil)

	result := doctor.CheckSSH(context.Background(), fake, "github-work")
	assert.Equal(t, doctor.StatusOK, result.Status)
}

func TestCheckSSHConnection_Failure(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("ssh -T", "", fmt.Errorf("connection refused"))

	result := doctor.CheckSSH(context.Background(), fake, "github-work")
	assert.Equal(t, doctor.StatusFail, result.Status)
}

func TestCheckEnvTokens_None(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	result := doctor.CheckEnvTokens()
	assert.Equal(t, doctor.StatusOK, result.Status)
}

func TestCheckEnvTokens_Set(t *testing.T) {
	t.Setenv("GH_TOKEN", "ghp_test123")

	result := doctor.CheckEnvTokens()
	assert.Equal(t, doctor.StatusWarn, result.Status)
	assert.Contains(t, result.Message, "GH_TOKEN")
}

func TestRunAll(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.DefaultResponse = &testutil.Response{Output: []byte("ok"), Err: nil}

	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	results := doctor.RunAll(context.Background(), fake, "/tmp/gh-config", "github-work")
	assert.NotEmpty(t, results)
	for _, r := range results {
		assert.NotEmpty(t, r.Name)
	}
}
