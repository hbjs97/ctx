package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config는 ctx 설정 파일의 최상위 구조체다.
type Config struct {
	Version               int                `toml:"version"`
	DefaultProfile        string             `toml:"default_profile"`
	PromptOnAmbiguous     *bool              `toml:"prompt_on_ambiguous"`
	RequirePushGuard      *bool              `toml:"require_push_guard"`
	AllowHTTPSManagedRepo bool               `toml:"allow_https_managed_repo"`
	CacheTTLDays          int                `toml:"cache_ttl_days"`
	Profiles              map[string]Profile `toml:"profiles"`
}

// Profile은 하나의 GitHub 계정 프로필이다.
type Profile struct {
	GHConfigDir string   `toml:"gh_config_dir"`
	SSHHost     string   `toml:"ssh_host"`
	GitName     string   `toml:"git_name"`
	GitEmail    string   `toml:"git_email"`
	EmailDomain string   `toml:"email_domain"`
	Owners      []string `toml:"owners"`
}

// Load는 config.toml을 파싱하여 Config를 반환한다.
func Load(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("config.Load: %w", err)
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// IsPromptOnAmbiguous는 prompt_on_ambiguous 설정값을 반환한다.
func (c *Config) IsPromptOnAmbiguous() bool {
	if c.PromptOnAmbiguous == nil {
		return true
	}
	return *c.PromptOnAmbiguous
}

// IsRequirePushGuard는 require_push_guard 설정값을 반환한다.
func (c *Config) IsRequirePushGuard() bool {
	if c.RequirePushGuard == nil {
		return true
	}
	return *c.RequirePushGuard
}

// ValidateFilePermissions는 파일 권한이 0600보다 넓으면 에러를 반환한다.
func ValidateFilePermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("config.ValidateFilePermissions: %w", err)
	}
	perm := info.Mode().Perm()
	if perm&0077 != 0 {
		return fmt.Errorf("config.ValidateFilePermissions: %s 권한이 %o (0600 필요)", path, perm)
	}
	return nil
}

func (c *Config) applyDefaults() {
	if c.PromptOnAmbiguous == nil {
		t := true
		c.PromptOnAmbiguous = &t
	}
	if c.RequirePushGuard == nil {
		t := true
		c.RequirePushGuard = &t
	}
	if c.CacheTTLDays == 0 {
		c.CacheTTLDays = 90
	}
}

func (c *Config) validate() error {
	if len(c.Profiles) == 0 {
		return fmt.Errorf("config.Load: 프로필이 정의되지 않았습니다")
	}
	for name, p := range c.Profiles {
		if p.GHConfigDir == "" {
			return fmt.Errorf("config.Load: profiles.%s.gh_config_dir 필수", name)
		}
		if p.SSHHost == "" {
			return fmt.Errorf("config.Load: profiles.%s.ssh_host 필수", name)
		}
		if p.GitName == "" {
			return fmt.Errorf("config.Load: profiles.%s.git_name 필수", name)
		}
		if p.GitEmail == "" {
			return fmt.Errorf("config.Load: profiles.%s.git_email 필수", name)
		}
	}
	return nil
}
