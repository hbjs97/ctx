package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hbjs97/ctx/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSave_WritesValidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	promptOn := true
	pushGuard := false
	cfg := &config.Config{
		Version:           1,
		DefaultProfile:    "work",
		PromptOnAmbiguous: &promptOn,
		RequirePushGuard:  &pushGuard,
		CacheTTLDays:      30,
		Profiles: map[string]config.Profile{
			"work": {
				GHConfigDir: "/home/user/.config/gh-work",
				SSHHost:     "github-work",
				GitName:     "Work User",
				GitEmail:    "work@company.com",
				Owners:      []string{"company-org"},
			},
		},
	}

	err := config.Save(path, cfg)
	require.NoError(t, err)

	// 파일 권한 0600 확인
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Load로 round-trip 검증
	loaded, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Version)
	assert.Equal(t, "work", loaded.DefaultProfile)
	assert.Equal(t, "Work User", loaded.Profiles["work"].GitName)
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	promptOn := true
	pushGuard := true
	cfg := &config.Config{
		Version:               1,
		DefaultProfile:        "personal",
		PromptOnAmbiguous:     &promptOn,
		RequirePushGuard:      &pushGuard,
		AllowHTTPSManagedRepo: true,
		CacheTTLDays:          60,
		Profiles: map[string]config.Profile{
			"work": {
				GHConfigDir: "/home/user/.config/gh-work",
				SSHHost:     "github-work",
				GitName:     "Work User",
				GitEmail:    "work@company.com",
				Owners:      []string{"company-org", "company-team"},
			},
			"personal": {
				GHConfigDir: "/home/user/.config/gh-personal",
				SSHHost:     "github-personal",
				GitName:     "hbjs97",
				GitEmail:    "hbjs97@naver.com",
				Owners:      []string{"hbjs97", "sutefu23"},
			},
		},
	}

	err := config.Save(path, cfg)
	require.NoError(t, err)

	loaded, err := config.Load(path)
	require.NoError(t, err)

	// 모든 최상위 필드 보존 확인
	assert.Equal(t, cfg.Version, loaded.Version)
	assert.Equal(t, cfg.DefaultProfile, loaded.DefaultProfile)
	assert.Equal(t, cfg.IsPromptOnAmbiguous(), loaded.IsPromptOnAmbiguous())
	assert.Equal(t, cfg.IsRequirePushGuard(), loaded.IsRequirePushGuard())
	assert.Equal(t, cfg.AllowHTTPSManagedRepo, loaded.AllowHTTPSManagedRepo)
	assert.Equal(t, cfg.CacheTTLDays, loaded.CacheTTLDays)

	// 프로필 수 확인
	assert.Len(t, loaded.Profiles, 2)

	// work 프로필 필드 보존 확인
	work := loaded.Profiles["work"]
	assert.Equal(t, "/home/user/.config/gh-work", work.GHConfigDir)
	assert.Equal(t, "github-work", work.SSHHost)
	assert.Equal(t, "Work User", work.GitName)
	assert.Equal(t, "work@company.com", work.GitEmail)
	assert.Equal(t, []string{"company-org", "company-team"}, work.Owners)

	// personal 프로필 필드 보존 확인
	personal := loaded.Profiles["personal"]
	assert.Equal(t, "/home/user/.config/gh-personal", personal.GHConfigDir)
	assert.Equal(t, "github-personal", personal.SSHHost)
	assert.Equal(t, "hbjs97", personal.GitName)
	assert.Equal(t, "hbjs97@naver.com", personal.GitEmail)
	assert.Equal(t, []string{"hbjs97", "sutefu23"}, personal.Owners)
}

func TestSave_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	path := filepath.Join(nested, "config.toml")

	cfg := &config.Config{
		Version: 1,
		Profiles: map[string]config.Profile{
			"work": {
				GHConfigDir: "/tmp/gh",
				SSHHost:     "github-work",
				GitName:     "Test",
				GitEmail:    "t@t.com",
				Owners:      []string{"org"},
			},
		},
	}

	err := config.Save(path, cfg)
	require.NoError(t, err)

	// 디렉토리가 생성되었는지 확인
	info, err := os.Stat(nested)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// 파일이 정상적으로 로드되는지 확인
	loaded, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Version)
}
