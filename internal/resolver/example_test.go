package resolver_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hbjs97/ctx/internal/cache"
	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/gh"
	"github.com/hbjs97/ctx/internal/git"
	"github.com/hbjs97/ctx/internal/resolver"
	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig() *config.Config {
	return &config.Config{
		CacheTTLDays: 90,
		Profiles: map[string]config.Profile{
			"work": {
				GHConfigDir: "/tmp/gh-work", SSHHost: "github-work",
				GitName: "W", GitEmail: "w@co.com", Owners: []string{"company-org"},
			},
			"personal": {
				GHConfigDir: "/tmp/gh-personal", SSHHost: "github-personal",
				GitName: "P", GitEmail: "p@me.com", Owners: []string{"hbjs97"},
			},
		},
	}
}

// Step 1: Explicit flag
func TestResolve_ExplicitFlag(t *testing.T) {
	cfg := testConfig()
	fake := testutil.NewFakeCommander()
	r := resolver.New(cfg, cache.New(), git.NewAdapter(fake), gh.NewAdapter(fake), false)

	result, err := r.Resolve(context.Background(), "any/repo", "work")
	require.NoError(t, err)
	assert.Equal(t, "work", result.Profile)
	assert.Equal(t, "explicit", result.Reason)
}

func TestResolve_ExplicitFlag_NotExists(t *testing.T) {
	cfg := testConfig()
	fake := testutil.NewFakeCommander()
	r := resolver.New(cfg, cache.New(), git.NewAdapter(fake), gh.NewAdapter(fake), false)

	_, err := r.Resolve(context.Background(), "any/repo", "nonexist")
	assert.Error(t, err)
}

// Step 2: Cache
func TestResolve_CacheHit(t *testing.T) {
	cfg := testConfig()
	c := cache.New()
	c.Set("company-org/api", cache.Entry{
		Profile: "work", Reason: "owner_rule",
		ResolvedAt: time.Now().Format(time.RFC3339), ConfigHash: cfg.ConfigHash(),
	})
	fake := testutil.NewFakeCommander()
	r := resolver.New(cfg, c, git.NewAdapter(fake), gh.NewAdapter(fake), false)

	result, err := r.Resolve(context.Background(), "company-org/api", "")
	require.NoError(t, err)
	assert.Equal(t, "work", result.Profile)
	assert.Equal(t, "cache", result.Reason)
}

func TestResolve_CacheTTLExpired(t *testing.T) {
	cfg := testConfig()
	c := cache.New()
	c.Set("company-org/api", cache.Entry{
		Profile: "work", Reason: "owner_rule",
		ResolvedAt: time.Now().Add(-91 * 24 * time.Hour).Format(time.RFC3339),
		ConfigHash: cfg.ConfigHash(),
	})
	fake := testutil.NewFakeCommander()
	r := resolver.New(cfg, c, git.NewAdapter(fake), gh.NewAdapter(fake), false)

	result, err := r.Resolve(context.Background(), "company-org/api", "")
	require.NoError(t, err)
	assert.Equal(t, "owner_rule", result.Reason)
}

func TestResolve_CacheHashMismatch(t *testing.T) {
	cfg := testConfig()
	c := cache.New()
	c.Set("company-org/api", cache.Entry{
		Profile: "work", Reason: "owner_rule",
		ResolvedAt: time.Now().Format(time.RFC3339), ConfigHash: "stale_hash",
	})
	fake := testutil.NewFakeCommander()
	r := resolver.New(cfg, c, git.NewAdapter(fake), gh.NewAdapter(fake), false)

	result, err := r.Resolve(context.Background(), "company-org/api", "")
	require.NoError(t, err)
	assert.Equal(t, "owner_rule", result.Reason)
}

// Step 3: Owner rule
func TestResolve_OwnerRuleSingleMatch(t *testing.T) {
	cfg := testConfig()
	fake := testutil.NewFakeCommander()
	r := resolver.New(cfg, cache.New(), git.NewAdapter(fake), gh.NewAdapter(fake), false)

	result, err := r.Resolve(context.Background(), "company-org/api", "")
	require.NoError(t, err)
	assert.Equal(t, "work", result.Profile)
	assert.Equal(t, "owner_rule", result.Reason)
}

func TestResolve_OwnerRuleNoMatch(t *testing.T) {
	cfg := testConfig()
	fake := testutil.NewFakeCommander()
	fake.DefaultResponse = &testutil.Response{Err: fmt.Errorf("HTTP 404")}
	r := resolver.New(cfg, cache.New(), git.NewAdapter(fake), gh.NewAdapter(fake), false)

	_, err := r.Resolve(context.Background(), "unknown-org/repo", "")
	assert.Error(t, err)
}

func TestResolve_OwnerRuleMultipleMatch(t *testing.T) {
	cfg := testConfig()
	cfg.Profiles["personal"] = config.Profile{
		GHConfigDir: "/tmp/gh-p", SSHHost: "gh-p",
		GitName: "P", GitEmail: "p@p.com",
		Owners: []string{"company-org"},
	}
	fake := testutil.NewFakeCommander()
	fake.DefaultResponse = &testutil.Response{Err: fmt.Errorf("HTTP 404")}
	r := resolver.New(cfg, cache.New(), git.NewAdapter(fake), gh.NewAdapter(fake), false)

	_, err := r.Resolve(context.Background(), "company-org/repo", "")
	assert.Error(t, err)
}

// Step 4: Probe
func TestResolve_ProbeSinglePush(t *testing.T) {
	cfg := &config.Config{
		CacheTTLDays: 90,
		Profiles: map[string]config.Profile{
			"work": {
				GHConfigDir: "/tmp/gh-work", SSHHost: "github-work",
				GitName: "W", GitEmail: "w@co.com", Owners: []string{"company-org"},
			},
		},
	}
	fake := testutil.NewFakeCommander()
	fake.DefaultResponse = &testutil.Response{
		Output: []byte(`{"permissions":{"push":true}}`),
	}

	r := resolver.New(cfg, cache.New(), git.NewAdapter(fake), gh.NewAdapter(fake), false)
	result, err := r.Resolve(context.Background(), "unknown/repo", "")

	require.NoError(t, err)
	assert.Equal(t, "work", result.Profile)
	assert.Equal(t, "probe", result.Reason)
}

func TestResolve_ProbeNoPush(t *testing.T) {
	cfg := testConfig()
	fake := testutil.NewFakeCommander()
	fake.DefaultResponse = &testutil.Response{Err: fmt.Errorf("HTTP 404")}
	r := resolver.New(cfg, cache.New(), git.NewAdapter(fake), gh.NewAdapter(fake), false)

	_, err := r.Resolve(context.Background(), "private-org/repo", "")
	assert.Error(t, err)
}

// Step 5: Non-interactive ambiguous
func TestResolve_ProbeMultiplePush_NonInteractive(t *testing.T) {
	cfg := testConfig()
	fake := testutil.NewFakeCommander()
	fake.DefaultResponse = &testutil.Response{
		Output: []byte(`{"permissions":{"push":true}}`),
	}
	r := resolver.New(cfg, cache.New(), git.NewAdapter(fake), gh.NewAdapter(fake), false)

	_, err := r.Resolve(context.Background(), "shared/repo", "")
	assert.Error(t, err)
}
