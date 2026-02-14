package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	cfgPath string
	verbose bool
)

// NewRootCmd는 ctx CLI의 루트 명령을 생성한다.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "ctx",
		Short:        "GitHub 멀티계정 컨텍스트 매니저",
		SilenceUsage: true,
	}

	defaultCfg := filepath.Join(homeDir(), ".config", "ctx", "config.toml")
	cmd.PersistentFlags().StringVar(&cfgPath, "config", defaultCfg, "설정 파일 경로")
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "상세 출력")

	cmd.AddCommand(
		newCloneCmd(),
		newInitCmd(),
		newStatusCmd(),
		newDoctorCmd(),
		newGuardCmd(),
		newActivateCmd(),
		newSetupCmd(),
	)
	return cmd
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "경고: 홈 디렉토리 확인 실패: %v\n", err)
		return "."
	}
	return home
}

func cachePath() string {
	return filepath.Join(homeDir(), ".config", "ctx", "cache.json")
}
