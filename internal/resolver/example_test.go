package resolver_test

import (
	"testing"
)

func TestResolve_ExplicitFlag(t *testing.T) {
	t.Skip("not implemented")

	// Given: --profile work flag is set
	// When: Resolve is called
	// Then: returns "work" profile immediately (Step 1)
	// And: no cache lookup or probe is performed
}

func TestResolve_ExplicitFlag_InvalidProfile(t *testing.T) {
	t.Skip("not implemented")

	// Given: --profile nonexistent flag is set
	// When: Resolve is called
	// Then: returns error with exit code 5
}

func TestResolve_CacheHit(t *testing.T) {
	t.Skip("not implemented")

	// Given: cache contains "company-org/api-server" -> "work", TTL valid, config_hash matches
	// When: Resolve("company-org/api-server") is called without --profile
	// Then: returns "work" from cache (Step 2)
}

func TestResolve_CacheTTLExpired(t *testing.T) {
	t.Skip("not implemented")

	// Given: cache entry exists but TTL has expired (>90 days)
	// When: Resolve is called
	// Then: cache miss, falls through to Step 3 (owner rule)
}

func TestResolve_CacheHashMismatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: cache entry exists but config_hash doesn't match current config
	// When: Resolve is called
	// Then: cache invalidated, falls through to Step 3
}

func TestResolve_OwnerRuleSingleMatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: repo owner "company-org" matches only "work" profile's owners
	// When: Resolve is called (no cache)
	// Then: returns "work" (Step 3)
}

func TestResolve_OwnerRuleNoMatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: repo owner "unknown-org" matches no profile
	// When: Resolve is called
	// Then: falls through to Step 4 (probe)
}

func TestResolve_OwnerRuleMultipleMatch(t *testing.T) {
	t.Skip("not implemented")

	// Given: repo owner matches 2+ profiles
	// When: Resolve is called
	// Then: falls through to Step 4 (probe)
}

func TestResolve_ProbeSinglePush(t *testing.T) {
	t.Skip("not implemented")

	// Given: gh api probe shows only "work" has push access
	// When: Resolve is called (no cache, no owner match)
	// Then: returns "work" (Step 4)
}

func TestResolve_ProbeNoPush(t *testing.T) {
	t.Skip("not implemented")

	// Given: no profile has push access (all 404 or push=false)
	// When: Resolve is called
	// Then: returns error with exit code 4
}

func TestResolve_ProbeMultiplePush_NonInteractive(t *testing.T) {
	t.Skip("not implemented")

	// Given: 2+ profiles have push access, non-interactive mode
	// When: Resolve is called
	// Then: returns error with exit code 3
}

func TestResolve_ProbeRateLimitWarning(t *testing.T) {
	t.Skip("not implemented")

	// Given: gh api response has X-RateLimit-Remaining < 10
	// When: probe is executed
	// Then: warning is output to stderr, probe continues
}

func TestResolve_FullPipeline(t *testing.T) {
	t.Skip("not implemented")

	// Integration: full 5-step pipeline with FakeCommander
	// Tests the transition between steps
}
