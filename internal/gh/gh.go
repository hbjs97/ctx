package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hbjs97/ctx/internal/cmdexec"
)

// ProbeResult는 권한 probe 결과다.
type ProbeResult struct {
	Profile   string
	HasAccess bool
	CanPush   bool
}

// Adapter는 gh CLI를 Commander를 통해 실행한다.
type Adapter struct {
	cmd cmdexec.Commander
}

// NewAdapter는 새 GH Adapter를 생성한다.
func NewAdapter(cmd cmdexec.Commander) *Adapter {
	return &Adapter{cmd: cmd}
}

// ProbeRepo는 특정 프로필로 리포의 접근 권한을 확인한다.
func (a *Adapter) ProbeRepo(ctx context.Context, ghConfigDir, owner, repo string) (*ProbeResult, error) {
	out, err := a.cmd.Run(ctx, "gh", "api",
		fmt.Sprintf("repos/%s/%s", owner, repo),
		"--hostname", "github.com")

	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "404") || strings.Contains(errStr, "403") || strings.Contains(errStr, "401") {
			return &ProbeResult{HasAccess: false}, nil
		}
		return nil, fmt.Errorf("gh.ProbeRepo: %w", err)
	}

	var resp struct {
		Permissions struct {
			Push  bool `json:"push"`
			Admin bool `json:"admin"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("gh.ProbeRepo: JSON 파싱 실패: %w", err)
	}
	return &ProbeResult{HasAccess: true, CanPush: resp.Permissions.Push}, nil
}

// ProbeAllProfiles는 모든 프로필로 리포를 probe한다.
// profiles: map[profileName]ghConfigDir
func (a *Adapter) ProbeAllProfiles(ctx context.Context, owner, repo string, profiles map[string]string) ([]ProbeResult, error) {
	var results []ProbeResult
	for name, dir := range profiles {
		result, err := a.ProbeRepo(ctx, dir, owner, repo)
		if err != nil {
			return nil, fmt.Errorf("gh.ProbeAllProfiles[%s]: %w", name, err)
		}
		result.Profile = name
		results = append(results, *result)
	}
	return results, nil
}

// DetectEnvTokenInterference는 GH_TOKEN/GITHUB_TOKEN 환경변수를 감지한다.
func DetectEnvTokenInterference() (string, bool) {
	for _, key := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if os.Getenv(key) != "" {
			return key, true
		}
	}
	return "", false
}
