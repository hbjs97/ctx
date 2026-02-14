package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/guard"
	"github.com/spf13/cobra"
)

func newGuardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "guard",
		Short: "pre-push guard 관리",
	}
	cmd.AddCommand(newGuardCheckCmd())
	return cmd
}

func newGuardCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "현재 리포의 컨텍스트 무결성을 검사한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGuardCheck(cmd)
		},
	}
}

func runGuardCheck(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cli.guard: %w", err)
	}

	profilePath := filepath.Join(cwd, ".git", "ctx-profile")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return fmt.Errorf("cli.guard: ctx-profile 없음 — 'ctx init' 실행 필요")
	}
	profileName := strings.TrimSpace(string(data))

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		return err
	}

	commander := &cmdexec.RealCommander{}
	result, err := guard.Check(cmd.Context(), cwd, profile, commander)
	if err != nil {
		return err
	}

	if !result.Pass {
		for _, v := range result.Violations {
			fmt.Printf("[%s] %s: 기대=%s, 실제=%s\n", v.Severity, v.Field, v.Expected, v.Actual)
		}
		return fmt.Errorf("guard: %w", guard.ErrGuardBlock)
	}

	for _, v := range result.Violations {
		if v.Severity == "warning" {
			fmt.Printf("[경고] %s: 기대=%s, 실제=%s\n", v.Field, v.Expected, v.Actual)
		}
	}

	fmt.Println("guard 검사 통과")
	return nil
}
