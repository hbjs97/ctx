package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/spf13/cobra"
)

// App holds CLI dependencies for testability.
type App struct {
	Commander cmdexec.Commander
	CfgPath   string
	Verbose   bool
}

// NewApp creates an App with default production dependencies.
func NewApp() *App {
	return &App{
		Commander: &cmdexec.RealCommander{},
	}
}

// NewRootCmd는 ctx CLI의 루트 명령을 생성한다. (App 메서드)
func (a *App) NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "ctx",
		Short:        "GitHub 멀티계정 컨텍스트 매니저",
		SilenceUsage: true,
	}

	defaultCfg := filepath.Join(homeDir(), ".config", "ctx", "config.toml")
	cmd.PersistentFlags().StringVar(&a.CfgPath, "config", defaultCfg, "설정 파일 경로")
	cmd.PersistentFlags().BoolVar(&a.Verbose, "verbose", false, "상세 출력")

	cmd.AddCommand(
		a.newCloneCmd(),
		a.newInitCmd(),
		a.newStatusCmd(),
		a.newDoctorCmd(),
		a.newGuardCmd(),
		a.newActivateCmd(),
		a.newSetupCmd(),
	)
	return cmd
}

// NewRootCmd는 ctx CLI의 루트 명령을 생성한다. (하위 호환용 free function)
func NewRootCmd() *cobra.Command {
	return NewApp().NewRootCmd()
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "경고: 홈 디렉토리 확인 실패: %v\n", err)
		return "."
	}
	return home
}

// cachePath는 설정 파일 경로의 디렉토리를 기준으로 캐시 파일 경로를 반환한다.
func (a *App) cachePath() string {
	if a.CfgPath != "" {
		return filepath.Join(filepath.Dir(a.CfgPath), "cache.json")
	}
	return filepath.Join(homeDir(), ".config", "ctx", "cache.json")
}
