package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "ctx 초기 설정을 시작한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("ctx setup은 아직 구현 중입니다.")
			fmt.Println("수동으로 ~/.config/ctx/config.toml을 생성하세요.")
			fmt.Println("참고: docs/PRD.md")
			return nil
		},
	}
}
