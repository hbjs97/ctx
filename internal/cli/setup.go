package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/hbjs97/ctx/internal/setup"
	"github.com/spf13/cobra"
)

func (a *App) newSetupCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "ctx 초기 설정을 시작한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runSetup(cmd.Context(), force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "기존 설정을 무시하고 재설정")
	return cmd
}

// runSetup는 interactive setup wizard를 실행한다.
func (a *App) runSetup(ctx context.Context, force bool) error {
	if force {
		if err := os.Remove(a.CfgPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("cli.setup: 기존 설정 파일 제거 실패: %w", err)
		}
	}

	r := &setup.Runner{
		CfgPath:    a.CfgPath,
		Commander:  a.Commander,
		FormRunner: &setup.HuhFormRunner{},
	}

	return r.Run(ctx)
}
