package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/shell"
	"github.com/spf13/cobra"
)

func (a *App) newActivateCmd() *cobra.Command {
	var shellType string
	var hookOnly bool

	cmd := &cobra.Command{
		Use:   "activate",
		Short: "현재 디렉토리에 맞는 프로필을 활성화한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if hookOnly {
				fmt.Print(shell.HookSnippet(shellType))
				return nil
			}
			return a.runActivate(shellType)
		},
	}
	cmd.Flags().StringVar(&shellType, "shell", "zsh", "셸 유형 (bash, zsh, fish)")
	cmd.Flags().BoolVar(&hookOnly, "hook", false, "hook 스니펫만 출력")
	return cmd
}

func (a *App) runActivate(shellType string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cli.activate: %w", err)
	}

	cfg, err := config.Load(a.CfgPath)
	if err != nil {
		// config 로드 실패 시 deactivate
		fmt.Print(shell.Deactivate(shellType))
		return nil
	}

	// Check for ctx-profile
	profilePath := filepath.Join(cwd, ".git", "ctx-profile")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		// Not a ctx-managed repo — use default profile or deactivate
		if cfg.DefaultProfile != "" {
			profile, err := cfg.GetProfile(cfg.DefaultProfile)
			if err == nil {
				fmt.Print(shell.Activate(cfg.DefaultProfile, profile, shellType))
				return nil
			}
		}
		fmt.Print(shell.Deactivate(shellType))
		return nil
	}

	profileName := strings.TrimSpace(string(data))
	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		fmt.Print(shell.Deactivate(shellType))
		return nil
	}

	fmt.Print(shell.Activate(profileName, profile, shellType))
	return nil
}
