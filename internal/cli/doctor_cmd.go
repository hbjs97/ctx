package cli

import (
	"fmt"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "환경 설정을 진단한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd)
		},
	}
}

func runDoctor(cmd *cobra.Command) error {
	commander := &cmdexec.RealCommander{}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Printf("[FAIL] config: %v\n", err)
		fmt.Println("      Fix: ctx setup 실행 또는 설정 파일 확인")
	}

	// Run diagnostics per profile if config loaded
	if cfg != nil {
		for name, profile := range cfg.Profiles {
			fmt.Printf("\n--- 프로필: %s ---\n", name)
			results := doctor.RunAll(cmd.Context(), commander, profile.GHConfigDir, profile.SSHHost)
			for _, r := range results {
				icon := statusIcon(r.Status)
				fmt.Printf("  [%s] %s: %s\n", icon, r.Name, r.Message)
				if r.Fix != "" {
					fmt.Printf("      Fix: %s\n", r.Fix)
				}
			}
		}
	} else {
		// Run basic binary checks without config
		results := doctor.CheckBinaries(cmd.Context(), commander)
		for _, r := range results {
			icon := statusIcon(r.Status)
			fmt.Printf("  [%s] %s: %s\n", icon, r.Name, r.Message)
			if r.Fix != "" {
				fmt.Printf("      Fix: %s\n", r.Fix)
			}
		}
	}
	return nil
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
