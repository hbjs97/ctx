package cli

import (
	"context"
	"fmt"

	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/doctor"
	"github.com/spf13/cobra"
)

func (a *App) newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "환경 설정을 진단한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runDoctor(cmd.Context())
		},
	}
}

func (a *App) runDoctor(ctx context.Context) error {
	cfg, err := config.Load(a.CfgPath)
	if err != nil {
		fmt.Printf("[FAIL] config: %v\n", err)
		fmt.Println("      Fix: ctx setup 실행 또는 설정 파일 확인")
	}

	// Run diagnostics per profile if config loaded
	if cfg != nil {
		for name, profile := range cfg.Profiles {
			fmt.Printf("\n--- 프로필: %s ---\n", name)
			results := doctor.RunAll(ctx, a.Commander, profile.GHConfigDir, profile.SSHHost)
			printDiagResults(results)
		}
	} else {
		// Run basic binary checks without config
		results := doctor.CheckBinaries(ctx, a.Commander)
		printDiagResults(results)
	}
	return nil
}

// printDiagResults는 진단 결과 목록을 출력한다.
func printDiagResults(results []doctor.DiagResult) {
	for _, r := range results {
		icon := statusIcon(r.Status)
		fmt.Printf("  [%s] %s: %s\n", icon, r.Name, r.Message)
		if r.Fix != "" {
			fmt.Printf("      Fix: %s\n", r.Fix)
		}
	}
}

func statusIcon(s doctor.Status) string {
	switch s {
	case doctor.StatusOK:
		return "OK"
	case doctor.StatusWarn:
		return "!!"
	case doctor.StatusFail:
		return "FAIL"
	default:
		return "??"
	}
}
