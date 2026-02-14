package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/git"
	"github.com/spf13/cobra"
)

func (a *App) newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "현재 리포의 ctx 프로필 상태를 표시한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runStatus(cmd.Context())
		},
	}
}

func (a *App) runStatus(ctx context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cli.status: %w", err)
	}

	// Read ctx-profile
	profilePath := filepath.Join(cwd, ".git", "ctx-profile")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		fmt.Println("ctx 프로필이 설정되지 않았습니다. 'ctx init'을 실행하세요.")
		return nil
	}
	profileName := strings.TrimSpace(string(data))

	cfg, err := config.Load(a.CfgPath)
	if err != nil {
		return err
	}

	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		return fmt.Errorf("cli.status: %w", err)
	}

	gitAdapter := git.NewAdapter(a.Commander)

	fmt.Printf("프로필: %s\n", profileName)
	fmt.Printf("  git name:  %s\n", profile.GitName)
	fmt.Printf("  git email: %s\n", profile.GitEmail)
	fmt.Printf("  SSH host:  %s\n", profile.SSHHost)

	// Show current remote URL
	if remote, err := gitAdapter.GetRemoteURL(ctx, cwd, "origin"); err == nil {
		fmt.Printf("  remote:    %s\n", strings.TrimSpace(remote))
	}

	return nil
}
