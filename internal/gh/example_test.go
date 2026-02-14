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
	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", `{"permissions":{"push":true,"admin":false}}`, nil)

	a := gh.NewAdapter(fake)
	result, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

	require.NoError(t, err)
	assert.True(t, result.HasAccess)
	assert.True(t, result.CanPush)
}

func TestProbeRepo_ReadOnly(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", `{"permissions":{"push":false}}`, nil)

	a := gh.NewAdapter(fake)
	result, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

	require.NoError(t, err)
	assert.True(t, result.HasAccess)
	assert.False(t, result.CanPush)
}

func TestProbeRepo_NotFound(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", "", fmt.Errorf("HTTP 404: Not Found"))

	a := gh.NewAdapter(fake)
	result, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

	require.NoError(t, err)
	assert.False(t, result.HasAccess)
}

func TestProbeRepo_Unauthorized(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("gh api repos/org/repo", "", fmt.Errorf("HTTP 401"))

	a := gh.NewAdapter(fake)
	result, err := a.ProbeRepo(context.Background(), "/tmp/gh-work", "org", "repo")

	require.NoError(t, err)
	assert.False(t, result.HasAccess)
}

func TestProbeAllProfiles(t *testing.T) {
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
