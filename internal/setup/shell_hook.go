package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hbjs97/ctx/internal/shell"
)

// DetectShell은 현재 사용자의 셸을 감지한다.
func DetectShell() string {
	sh := os.Getenv("SHELL")
	return filepath.Base(sh)
}

// ShellRCPath는 셸별 RC 파일 경로를 반환한다.
func ShellRCPath(shellType string) string {
	home, _ := os.UserHomeDir() // 홈 디렉토리 조회 실패 시 빈 문자열
	switch shellType {
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "bash":
		return filepath.Join(home, ".bashrc")
	case "fish":
		return filepath.Join(home, ".config", "fish", "conf.d", "ctx.fish")
	default:
		return ""
	}
}

// InstallShellHook은 셸 RC 파일에 ctx hook을 추가한다.
// 이미 설치되어 있으면 건너뛴다.
func InstallShellHook(shellType, rcPath string) error {
	snippet := shell.HookSnippet(shellType)
	if snippet == "" {
		return fmt.Errorf("setup.InstallShellHook: 지원하지 않는 셸: %s", shellType)
	}

	existing, _ := os.ReadFile(rcPath) // 파일이 없으면 빈 바이트
	if strings.Contains(string(existing), "ctx shell integration") {
		return nil // 이미 설치됨
	}

	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("setup.InstallShellHook: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "\n%s", snippet); err != nil {
		return fmt.Errorf("setup.InstallShellHook: %w", err)
	}

	return nil
}
