package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/gh"
)

// Runner는 interactive setup의 진입점이다.
type Runner struct {
	CfgPath       string
	Commander     cmdexec.Commander
	FormRunner    FormRunner
	SSHConfigPath string // 테스트용. 비어있으면 기본 경로.
}

// Run은 setup 플로우를 실행한다.
func (r *Runner) Run(ctx context.Context) error {
	_, err := os.Stat(r.CfgPath)
	if os.IsNotExist(err) {
		return r.runFirstTime(ctx)
	}
	if err != nil {
		return fmt.Errorf("setup.Run: %w", err)
	}
	return r.runExisting(ctx)
}

func (r *Runner) runFirstTime(ctx context.Context) error {
	fmt.Println("ctx 초기 설정을 시작합니다.")

	cfg := &config.Config{
		Version:  1,
		Profiles: make(map[string]config.Profile),
	}

	for {
		profile, err := r.collectProfile(ctx, cfg, nil)
		if err != nil {
			return err
		}
		cfg.Profiles[profile.Name] = config.Profile{
			GHConfigDir: r.ghConfigDir(profile.Name),
			SSHHost:     profile.SSHHost,
			GitName:     profile.GitName,
			GitEmail:    profile.GitEmail,
			Owners:      profile.Owners,
		}

		more, err := r.FormRunner.RunAddMore()
		if err != nil || !more {
			break
		}
	}

	if err := config.Save(r.CfgPath, cfg); err != nil {
		return err
	}

	fmt.Printf("설정 파일이 저장되었습니다: %s\n", r.CfgPath)
	return nil
}

func (r *Runner) collectProfile(ctx context.Context, cfg *config.Config, defaults *ProfileInput) (*ProfileInput, error) {
	existingNames := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		existingNames = append(existingNames, name)
	}

	input, err := r.FormRunner.RunProfileForm(defaults, existingNames)
	if err != nil {
		return nil, err
	}

	// gh auth login 실행
	ghDir := r.ghConfigDir(input.Name)
	if err := os.MkdirAll(ghDir, 0700); err != nil {
		return nil, fmt.Errorf("setup: gh config 디렉토리 생성 실패: %w", err)
	}

	env := gh.SuppressEnvTokens()
	env["GH_CONFIG_DIR"] = ghDir
	_, err = r.Commander.RunWithEnv(ctx, env, "gh", "auth", "login", "--hostname", "github.com")
	if err != nil {
		fmt.Fprintf(os.Stderr, "경고: gh 인증 실패 — 나중에 직접 인증하세요\n")
	}

	// SSH host 감지 + 선택
	sshPath := r.SSHConfigPath
	if sshPath == "" {
		sshPath = DefaultSSHConfigPath()
	}
	hosts := ParseSSHConfig(sshPath)
	sshHost, err := r.FormRunner.RunSSHHostSelect(hosts)
	if err != nil {
		return nil, err
	}
	input.SSHHost = sshHost

	// 조직 조회 + 선택
	detected := DetectOrgs(ctx, r.Commander, ghDir)
	owners, err := r.FormRunner.RunOwnersSelect(detected)
	if err != nil {
		return nil, err
	}
	input.Owners = owners

	return input, nil
}

func (r *Runner) ghConfigDir(profileName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", fmt.Sprintf("gh-%s", profileName))
}

// runExisting는 기존 config가 있을 때의 플로우다.
// Task 7에서 구현.
func (r *Runner) runExisting(ctx context.Context) error {
	return fmt.Errorf("setup.runExisting: 미구현")
}
