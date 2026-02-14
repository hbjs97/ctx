package setup

import (
	"context"
	"strings"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/hbjs97/ctx/internal/gh"
)

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
