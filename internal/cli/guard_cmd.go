package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/guard"
	"github.com/spf13/cobra"
)

func (a *App) newGuardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guard",
		Short: "pre-push guard 관리",
	}
	cmd.AddCommand(a.newGuardCheckCmd())
	return cmd
}

func (a *App) newGuardCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "현재 리포의 컨텍스트 무결성을 검사한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runGuardCheck(cmd)
		},
	}
}

func (a *App) runGuardCheck(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cli.guard: %w", err)
	}

	profilePath := filepath.Join(cwd, ".git", "ctx-profile")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return fmt.Errorf("cli.guard: ctx-profile 읽기 실패: %w", err)
	}
	profileName := strings.TrimSpace(string(data))

	cfg, err := config.Load(a.CfgPath)
	if err != nil {
		return err
	}

	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		return err
	}

	result, err := guard.Check(cmd.Context(), cwd, profile, a.Commander)
	if err != nil {
		return err
	}

	if !result.Pass {
		for _, v := range result.Violations {
			fmt.Printf("[%s] %s: 기대=%s, 실제=%s\n", v.Severity, v.Field, v.Expected, v.Actual)
		}
		return fmt.Errorf("cli.guard: %w", guard.ErrGuardBlock)
	}

	if result.Skipped {
		fmt.Fprintln(os.Stderr, "guard 검사 건너뜀 (CTX_SKIP_GUARD=1)")
		return nil
	}

	for _, v := range result.Violations {
		if v.Severity == "warning" {
			fmt.Fprintf(os.Stderr, "[경고] %s: 기대=%s, 실제=%s\n", v.Field, v.Expected, v.Actual)
		}
	}

	fmt.Println("guard 검사 통과")
	return nil
}
