package guard

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/hbjs97/ctx/internal/config"
)

// ErrGuardBlock는 guard 검사 실패로 push가 차단될 때 반환된다.
var ErrGuardBlock = errors.New("guard 검사 실패 — push 차단")

const (
	hookStartMarker = "# ctx-guard-start"
	hookEndMarker   = "# ctx-guard-end"
	hookScript      = `# ctx-guard-start
# Installed by ctx — do not edit this block manually.
if ! command -v ctx >/dev/null 2>&1; then
  echo "ctx: command not found — skipping guard check" >&2
  exit 0
fi
ctx guard check || exit 1
# ctx-guard-end`
)

// CheckResult는 guard 검사 결과다.
type CheckResult struct {
	Pass       bool
	Skipped    bool
	Violations []Violation
}

// Violation은 검사 위반 항목이다.
type Violation struct {
	Field    string // "remote_host", "user_email", "user_name"
	Expected string
	Actual   string
	Severity string // "error", "warning"
}

// Check는 리포의 컨텍스트 무결성을 검사한다.
func Check(ctx context.Context, repoDir string, profile *config.Profile, cmd cmdexec.Commander) (*CheckResult, error) {
	// CTX_SKIP_GUARD 환경변수로 우회
	if os.Getenv("CTX_SKIP_GUARD") == "1" {
		fmt.Fprintln(os.Stderr, "경고: CTX_SKIP_GUARD=1 — guard 검사를 건너뜁니다")
		return &CheckResult{Pass: true, Skipped: true}, nil
	}

	result := &CheckResult{Pass: true}

	// Remote host 검사
	remoteOut, err := cmd.Run(ctx, "git", "-C", repoDir, "remote", "get-url", "origin")
	if err != nil {
		return nil, fmt.Errorf("guard.Check: %w", err)
	}
	remoteURL := strings.TrimSpace(string(remoteOut))
	if profile.SSHHost != "" && strings.HasPrefix(remoteURL, "git@") {
		hostPart := strings.TrimPrefix(remoteURL, "git@")
		if idx := strings.Index(hostPart, ":"); idx > 0 {
			actualHost := hostPart[:idx]
			if actualHost != profile.SSHHost {
				result.Pass = false
				result.Violations = append(result.Violations, Violation{
					Field: "remote_host", Expected: profile.SSHHost,
					Actual: actualHost, Severity: "error",
				})
			}
		}
	}

	// Email 검사
	emailOut, err := cmd.Run(ctx, "git", "-C", repoDir, "config", "--local", "user.email")
	if err != nil {
		return nil, fmt.Errorf("guard.Check: %w", err)
	}
	actualEmail := strings.TrimSpace(string(emailOut))
	if actualEmail != profile.GitEmail {
		result.Pass = false
		result.Violations = append(result.Violations, Violation{
			Field: "user_email", Expected: profile.GitEmail,
			Actual: actualEmail, Severity: "error",
		})
	}

	// Name 검사 (warning only)
	nameOut, err := cmd.Run(ctx, "git", "-C", repoDir, "config", "--local", "user.name")
	if err != nil {
		return nil, fmt.Errorf("guard.Check: %w", err)
	}
	actualName := strings.TrimSpace(string(nameOut))
	if actualName != profile.GitName {
		result.Violations = append(result.Violations, Violation{
			Field: "user_name", Expected: profile.GitName,
			Actual: actualName, Severity: "warning",
		})
	}

	return result, nil
}

// InstallHook은 pre-push hook에 guard 스크립트를 설치한다.
func InstallHook(repoDir string) error {
	hookDir := filepath.Join(repoDir, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		return fmt.Errorf("guard.InstallHook: %w", err)
	}
	hookPath := filepath.Join(hookDir, "pre-push")

	existing, err := os.ReadFile(hookPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("guard.InstallHook: %w", err)
	}

	var content string
	if len(existing) > 0 {
		existingStr := string(existing)
		if strings.Contains(existingStr, hookStartMarker) {
			return nil // already installed
		}
		content = existingStr + "\n" + hookScript + "\n"
	} else {
		content = "#!/bin/sh\n" + hookScript + "\n"
	}

	return os.WriteFile(hookPath, []byte(content), 0755)
}

// UninstallHook은 pre-push hook에서 guard 스크립트를 제거한다.
func UninstallHook(repoDir string) error {
	hookPath := filepath.Join(repoDir, ".git", "hooks", "pre-push")
	data, err := os.ReadFile(hookPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("guard.UninstallHook: %w", err)
	}

	content := string(data)
	startIdx := strings.Index(content, hookStartMarker)
	endIdx := strings.Index(content, hookEndMarker)
	if startIdx == -1 || endIdx == -1 {
		return nil
	}

	before := content[:startIdx]
	after := content[endIdx+len(hookEndMarker):]
	cleaned := strings.TrimSpace(before + after)

	if cleaned == "" || cleaned == "#!/bin/sh" {
		return os.Remove(hookPath)
	}
	return os.WriteFile(hookPath, []byte(cleaned+"\n"), 0755)
}
