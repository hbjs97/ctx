package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// setupTemplate는 ctx setup이 생성하는 기본 config.toml 내용이다.
const setupTemplate = `# ctx configuration file
# See: https://github.com/hbjs97/ctx

version = 1
# default_profile = "work"
# prompt_on_ambiguous = true
# require_push_guard = true
# cache_ttl_days = 90

[profiles.work]
gh_config_dir = "~/.config/gh-work"
ssh_host = "github.com-work"
git_name = "Your Name"
git_email = "you@work.com"
owners = ["your-org"]

[profiles.personal]
gh_config_dir = "~/.config/gh-personal"
ssh_host = "github.com-personal"
git_name = "Your Name"
git_email = "you@personal.com"
owners = ["your-username"]
`

func (a *App) newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "ctx 초기 설정을 시작한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runSetup()
		},
	}
}

// runSetup는 설정 파일 템플릿을 생성한다.
func (a *App) runSetup() error {
	// Check if config file already exists
	if _, err := os.Stat(a.CfgPath); err == nil {
		return fmt.Errorf("cli.setup: 설정 파일이 이미 존재합니다: %s", a.CfgPath)
	}

	// Create parent directory
	dir := filepath.Dir(a.CfgPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("cli.setup: 디렉토리 생성 실패: %w", err)
	}

	// Write template config file
	if err := os.WriteFile(a.CfgPath, []byte(setupTemplate), 0600); err != nil {
		return fmt.Errorf("cli.setup: 설정 파일 생성 실패: %w", err)
	}

	fmt.Printf("설정 파일이 생성되었습니다: %s\n", a.CfgPath)
	fmt.Println("프로필을 수정한 후 ctx doctor로 환경을 확인하세요.")
	return nil
}
