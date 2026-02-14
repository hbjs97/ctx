package setup

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/hbjs97/ctx/internal/gh"
)

// ParseSSHConfig는 SSH config 파일에서 GitHub 관련 Host alias 목록을 추출한다.
// 파일이 없거나 파싱 실패 시 빈 슬라이스를 반환한다.
func ParseSSHConfig(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var hosts []string
	var currentHost string
	var isGitHub bool

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "Host ") && !strings.Contains(line, "*") {
			// 이전 Host 블록 처리
			if currentHost != "" && isGitHub {
				hosts = append(hosts, currentHost)
			}
			currentHost = strings.TrimPrefix(line, "Host ")
			currentHost = strings.TrimSpace(currentHost)
			// Host 이름에 github이 포함되면 GitHub으로 간주
			isGitHub = strings.Contains(strings.ToLower(currentHost), "github")
		}

		if strings.HasPrefix(line, "HostName") {
			hostName := strings.TrimSpace(strings.TrimPrefix(line, "HostName"))
			if strings.Contains(hostName, "github.com") {
				isGitHub = true
			}
		}
	}

	// 마지막 Host 블록 처리
	if currentHost != "" && isGitHub {
		hosts = append(hosts, currentHost)
	}

	return hosts
}

// DefaultSSHConfigPath는 기본 SSH config 경로를 반환한다.
func DefaultSSHConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

// DetectOrgs는 gh api로 인증된 사용자의 조직 목록과 사용자명을 조회한다.
// 조회 실패 시 빈 슬라이스를 반환한다 (에러로 차단하지 않음).
func DetectOrgs(ctx context.Context, cmd cmdexec.Commander, ghConfigDir string) []string {
	env := gh.SuppressEnvTokens()
	env["GH_CONFIG_DIR"] = ghConfigDir

	var orgs []string

	// 조직 목록 조회
	out, err := cmd.RunWithEnv(ctx, env, "gh", "api", "user/orgs", "--jq", ".[].login")
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				orgs = append(orgs, line)
			}
		}
	}

	// 사용자명 조회
	out, err = cmd.RunWithEnv(ctx, env, "gh", "api", "user", "--jq", ".login")
	if err == nil {
		login := strings.TrimSpace(string(out))
		if login != "" {
			orgs = append(orgs, login)
		}
	}

	return orgs
}
