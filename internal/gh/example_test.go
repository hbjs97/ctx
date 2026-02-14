package gh_test

import (
	"testing"
)

func TestProbeRepo_PushAccess(t *testing.T) {
	t.Skip("not implemented")

	// Given: gh api returns 200 with permissions.push=true
	// When: ProbeRepo("owner/repo", profile) is called
	// Then: returns ProbeResult with CanPush=true
}

func TestProbeRepo_ReadOnly(t *testing.T) {
	t.Skip("not implemented")

	// Given: gh api returns 200 with permissions.push=false
	// When: ProbeRepo is called
	// Then: returns ProbeResult with CanPush=false, CanRead=true
}

func TestProbeRepo_NotFound(t *testing.T) {
	t.Skip("not implemented")

	// Given: gh api returns 404
	// When: ProbeRepo is called
	// Then: returns ProbeResult with CanPush=false, CanRead=false
}

func TestProbeRepo_Forbidden_SSO(t *testing.T) {
	t.Skip("not implemented")

	// Given: gh api returns 403 with X-GitHub-SSO header
	// When: ProbeRepo is called
	// Then: returns ProbeResult with SSORequired=true and authorize URL
}

func TestProbeRepo_Unauthorized(t *testing.T) {
	t.Skip("not implemented")

	// Given: gh api returns 401
	// When: ProbeRepo is called
	// Then: returns ProbeResult with TokenExpired=true
}

func TestProbeAllProfiles(t *testing.T) {
	t.Skip("not implemented")

	// Given: 2 profiles, one with push access, one without
	// When: ProbeAllProfiles("owner/repo") is called
	// Then: returns results for both profiles
}

func TestDetectRateLimitWarning(t *testing.T) {
	t.Skip("not implemented")

	// Given: gh api response with X-RateLimit-Remaining < 10
	// When: response is processed
	// Then: rate limit warning is flagged
}

func TestDetectEnvTokenInterference(t *testing.T) {
	t.Skip("not implemented")

	// Given: GH_TOKEN environment variable is set
	// When: DetectEnvTokenInterference is called
	// Then: returns warning about env token overriding profile auth
}
