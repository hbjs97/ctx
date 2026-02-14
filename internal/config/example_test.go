package config_test

import (
	"os"
	"testing"

	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_ValidTOML(t *testing.T) {
	content := `version = 1
default_profile = "personal"

[profiles.work]
gh_config_dir = "/home/test/.config/gh-work"
ssh_host = "github-work"
git_name = "Test User"
git_email = "test@company.com"
email_domain = "company.com"
owners = ["company-org", "company-team"]

[profiles.personal]
gh_config_dir = "/home/test/.config/gh-personal"
ssh_host = "github-personal"
git_name = "testuser"
git_email = "test@example.com"
email_domain = "example.com"
owners = ["testuser"]`

	path := testutil.TempConfigFile(t, content)
	cfg, err := config.Load(path)

	require.NoError(t, err)
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, "personal", cfg.DefaultProfile)
	assert.True(t, cfg.IsPromptOnAmbiguous())
	assert.True(t, cfg.IsRequirePushGuard())
	assert.Equal(t, 90, cfg.CacheTTLDays)
	assert.Len(t, cfg.Profiles, 2)

	work := cfg.Profiles["work"]
	assert.Equal(t, "/home/test/.config/gh-work", work.GHConfigDir)
	assert.Equal(t, "github-work", work.SSHHost)
	assert.Equal(t, "Test User", work.GitName)
	assert.Equal(t, "test@company.com", work.GitEmail)
	assert.Equal(t, "company.com", work.EmailDomain)
	assert.Equal(t, []string{"company-org", "company-team"}, work.Owners)
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.toml")
	assert.Error(t, err)
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	path := testutil.TempConfigFile(t, "invalid toml [[[")
	_, err := config.Load(path)
	assert.Error(t, err)
}

func TestLoadConfig_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "no profiles",
			content: `version = 1`,
		},
		{
			name: "missing gh_config_dir",
			content: `version = 1
[profiles.work]
ssh_host = "h"
git_name = "n"
git_email = "e"
owners = ["o"]`,
		},
		{
			name: "missing git_email",
			content: `version = 1
[profiles.work]
gh_config_dir = "/tmp"
ssh_host = "h"
git_name = "n"
owners = ["o"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := testutil.TempConfigFile(t, tt.content)
			_, err := config.Load(path)
			assert.Error(t, err)
		})
	}
}

func TestLoadConfig_DefaultValues(t *testing.T) {
	content := `version = 1
[profiles.work]
gh_config_dir = "/tmp/gh"
ssh_host = "github-work"
git_name = "Test"
git_email = "t@t.com"
owners = ["org"]`

	path := testutil.TempConfigFile(t, content)
	cfg, err := config.Load(path)

	require.NoError(t, err)
	assert.True(t, cfg.IsPromptOnAmbiguous())
	assert.True(t, cfg.IsRequirePushGuard())
	assert.False(t, cfg.AllowHTTPSManagedRepo)
	assert.Equal(t, 90, cfg.CacheTTLDays)
}

func TestLoadConfig_ExplicitFalse(t *testing.T) {
	content := `version = 1
prompt_on_ambiguous = false
require_push_guard = false
cache_ttl_days = 30
[profiles.work]
gh_config_dir = "/tmp/gh"
ssh_host = "h"
git_name = "n"
git_email = "e"
owners = ["o"]`

	path := testutil.TempConfigFile(t, content)
	cfg, err := config.Load(path)

	require.NoError(t, err)
	assert.False(t, cfg.IsPromptOnAmbiguous())
	assert.False(t, cfg.IsRequirePushGuard())
	assert.Equal(t, 30, cfg.CacheTTLDays)
}

func TestValidateFilePermissions(t *testing.T) {
	path := testutil.TempConfigFile(t, `version = 1`)

	// 0600 — no error
	err := config.ValidateFilePermissions(path)
	assert.NoError(t, err)

	// 0644 — error
	os.Chmod(path, 0644)
	err = config.ValidateFilePermissions(path)
	assert.Error(t, err)
}

func TestMatchOwner_SingleMatch(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			"work":     {Owners: []string{"company-org", "company-team"}},
			"personal": {Owners: []string{"hbjs97", "sutefu23"}},
		},
	}
	matches := cfg.MatchOwner("company-org")
	assert.Equal(t, []string{"work"}, matches)
}

func TestMatchOwner_NoMatch(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			"work": {Owners: []string{"company-org"}},
		},
	}
	matches := cfg.MatchOwner("unknown-org")
	assert.Empty(t, matches)
}

func TestMatchOwner_MultipleMatch(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			"work":     {Owners: []string{"shared-org"}},
			"personal": {Owners: []string{"shared-org"}},
		},
	}
	matches := cfg.MatchOwner("shared-org")
	assert.Len(t, matches, 2)
}

func TestGetProfile_Exists(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			"work": {GHConfigDir: "/tmp/gh-work", GitEmail: "w@co.com"},
		},
	}
	p, err := cfg.GetProfile("work")
	require.NoError(t, err)
	assert.Equal(t, "/tmp/gh-work", p.GHConfigDir)
}

func TestGetProfile_NotExists(t *testing.T) {
	cfg := &config.Config{Profiles: map[string]config.Profile{}}
	_, err := cfg.GetProfile("nonexistent")
	assert.Error(t, err)
}

func TestConfigHash(t *testing.T) {
	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			"work": {GHConfigDir: "/tmp", GitEmail: "a@b.com"},
		},
	}

	hash1 := cfg.ConfigHash()
	assert.NotEmpty(t, hash1)

	// 동일 설정 → 동일 해시
	hash2 := cfg.ConfigHash()
	assert.Equal(t, hash1, hash2)

	// 프로필 변경 → 해시 변경
	cfg.Profiles["work"] = config.Profile{GHConfigDir: "/tmp2", GitEmail: "c@d.com"}
	hash3 := cfg.ConfigHash()
	assert.NotEqual(t, hash1, hash3)
}
