package doctor

import (
	"context"
	"fmt"
	"strings"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/hbjs97/ctx/internal/gh"
)

// Status는 진단 결과 상태다.
type Status string

const (
	// StatusOK는 정상 상태다.
	StatusOK Status = "OK"
	// StatusWarn는 경고 상태다.
	StatusWarn Status = "WARN"
	// StatusFail는 실패 상태다.
	StatusFail Status = "FAIL"
)

// DiagResult는 하나의 진단 결과다.
type DiagResult struct {
	Name    string
	Status  Status
	Message string
	Fix     string
}

// CheckBinaries는 필수 바이너리(git, gh, ssh) 존재 여부를 확인한다.
func CheckBinaries(ctx context.Context, cmd cmdexec.Commander) []DiagResult {
	binaries := []struct {
		name    string
		args    []string
		install string
	}{
		{"git", []string{"--version"}, "https://git-scm.com/downloads"},
		{"gh", []string{"--version"}, "https://cli.github.com/"},
		{"ssh", []string{"-V"}, "OpenSSH를 설치하세요"},
	}

	var results []DiagResult
	for _, b := range binaries {
		out, err := cmd.Run(ctx, b.name, b.args...)
		if err != nil {
			results = append(results, DiagResult{
				Name:    b.name,
				Status:  StatusFail,
				Message: fmt.Sprintf("%s 없음", b.name),
				Fix:     fmt.Sprintf("설치: %s", b.install),
			})
		} else {
			results = append(results, DiagResult{
				Name:    b.name,
				Status:  StatusOK,
				Message: strings.TrimSpace(string(out)),
			})
		}
	}
	return results
}

// CheckGHAuth는 gh CLI 인증 상태를 확인한다.
func CheckGHAuth(ctx context.Context, cmd cmdexec.Commander, ghConfigDir string) DiagResult {
	_, err := cmd.Run(ctx, "gh", "auth", "status")
	if err != nil {
		return DiagResult{
			Name:    "gh_auth",
			Status:  StatusFail,
			Message: "gh CLI 인증 안됨",
			Fix:     "gh auth login 실행",
		}
	}
	return DiagResult{
		Name:    "gh_auth",
		Status:  StatusOK,
		Message: "gh CLI 인증 완료",
	}
}

// CheckSSH는 SSH 연결을 확인한다.
func CheckSSH(ctx context.Context, cmd cmdexec.Commander, sshHost string) DiagResult {
	_, err := cmd.Run(ctx, "ssh", "-T", fmt.Sprintf("git@%s", sshHost))
	if err != nil {
		errStr := err.Error()
		// GitHub returns exit code 1 with "Hi user!" message — that's actually success
		if strings.Contains(errStr, "Hi ") {
			return DiagResult{
				Name:    fmt.Sprintf("ssh_%s", sshHost),
				Status:  StatusOK,
				Message: fmt.Sprintf("SSH %s 연결 성공", sshHost),
			}
		}
		return DiagResult{
			Name:    fmt.Sprintf("ssh_%s", sshHost),
			Status:  StatusFail,
			Message: fmt.Sprintf("SSH %s 연결 실패", sshHost),
			Fix:     fmt.Sprintf("ssh -T git@%s 로 연결 확인", sshHost),
		}
	}
	return DiagResult{
		Name:    fmt.Sprintf("ssh_%s", sshHost),
		Status:  StatusOK,
		Message: fmt.Sprintf("SSH %s 연결 성공", sshHost),
	}
}

// CheckEnvTokens는 환경변수 토큰 간섭을 확인한다.
func CheckEnvTokens() DiagResult {
	key, found := gh.DetectEnvTokenInterference()
	if found {
		return DiagResult{
			Name:    "env_tokens",
			Status:  StatusWarn,
			Message: fmt.Sprintf("환경변수 %s 설정됨 — 프로필 인증이 무시될 수 있음", key),
			Fix:     fmt.Sprintf("unset %s", key),
		}
	}
	return DiagResult{
		Name:    "env_tokens",
		Status:  StatusOK,
		Message: "토큰 환경변수 없음",
	}
}

// RunAll은 모든 진단을 실행한다.
func RunAll(ctx context.Context, cmd cmdexec.Commander, ghConfigDir, sshHost string) []DiagResult {
	var results []DiagResult
	results = append(results, CheckBinaries(ctx, cmd)...)
	results = append(results, CheckGHAuth(ctx, cmd, ghConfigDir))
	results = append(results, CheckSSH(ctx, cmd, sshHost))
	results = append(results, CheckEnvTokens())
	return results
}
