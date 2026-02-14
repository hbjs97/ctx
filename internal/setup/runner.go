package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/doctor"
	"github.com/hbjs97/ctx/internal/gh"
)

// Runner는 interactive setup의 진입점이다.
type Runner struct {
	CfgPath       string
	Commander     cmdexec.Commander
	FormRunner    FormRunner
	SSHConfigPath string // 테스트용. 비어있으면 기본 경로.
	SSHDir        string // 테스트용. 비어있으면 ~/.ssh.
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

	// 셸 hook 설치
	shellType := DetectShell()
	if shellType != "" {
		rcPath := ShellRCPath(shellType)
		if rcPath != "" {
			if err := InstallShellHook(shellType, rcPath); err != nil {
				fmt.Fprintf(os.Stderr, "경고: 셸 hook 설치 실패: %v\n", err)
			} else {
				fmt.Printf("셸 hook이 설치되었습니다: %s\n", rcPath)
			}
		}
	}

	r.runDoctor(ctx, cfg)
	return nil
}

func (r *Runner) sshDir() string {
	if r.SSHDir != "" {
		return r.SSHDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh")
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

	// SSH 키 감지 + 선택/생성
	sshDirPath := r.sshDir()
	existingKeys := DetectSSHKeys(sshDirPath)
	keyChoice, err := r.FormRunner.RunSSHKeySelect(existingKeys, input.Name)
	if err != nil {
		return nil, err
	}

	var identityFile string
	sshConfigPath := r.SSHConfigPath
	if sshConfigPath == "" {
		sshConfigPath = DefaultSSHConfigPath()
	}

	switch keyChoice.Action {
	case "generate":
		keyPath := filepath.Join(sshDirPath, fmt.Sprintf("id_ed25519_%s", input.Name))
		if err := GenerateSSHKey(ctx, r.Commander, input.GitEmail, keyPath); err != nil {
			return nil, err
		}
		identityFile = keyPath
		// SSH config에 Host 엔트리 자동 추가
		sshHost := fmt.Sprintf("github.com-%s", input.Name)
		if err := WriteSSHConfigEntry(sshConfigPath, sshHost, identityFile); err != nil {
			return nil, err
		}
		input.SSHHost = sshHost
	case "existing":
		identityFile = keyChoice.ExistingKey
		// SSH config에 Host 엔트리 자동 추가
		sshHost := fmt.Sprintf("github.com-%s", input.Name)
		if err := WriteSSHConfigEntry(sshConfigPath, sshHost, identityFile); err != nil {
			return nil, err
		}
		input.SSHHost = sshHost
	default:
		// "skip" — 기존 SSH host 선택 플로우로 fallback
		hosts := ParseSSHConfig(sshConfigPath)
		sshHost, err := r.FormRunner.RunSSHHostSelect(hosts)
		if err != nil {
			return nil, err
		}
		input.SSHHost = sshHost
	}

	// gh auth login 실행
	ghDir := r.ghConfigDir(input.Name)
	if err := os.MkdirAll(ghDir, 0700); err != nil {
		return nil, fmt.Errorf("setup: gh config 디렉토리 생성 실패: %w", err)
	}

	env := gh.SuppressEnvTokens()
	env["GH_CONFIG_DIR"] = ghDir
	err = r.Commander.RunInteractiveWithEnv(ctx, env, "gh", "auth", "login", "--hostname", "github.com", "--git-protocol", "ssh")
	if err != nil {
		fmt.Fprintf(os.Stderr, "경고: gh 인증 실패 — 나중에 직접 인증하세요\n")
	}

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

// runDoctor는 설정 완료 후 환경 진단을 실행한다.
func (r *Runner) runDoctor(ctx context.Context, cfg *config.Config) {
	fmt.Println("\n환경 진단 실행 중...")
	for name, p := range cfg.Profiles {
		fmt.Printf("\n[%s] 프로필 진단:\n", name)
		results := doctor.RunAll(ctx, r.Commander, p.GHConfigDir, p.SSHHost)
		for _, res := range results {
			icon := "✓"
			if res.Status == doctor.StatusFail {
				icon = "✗"
			} else if res.Status == doctor.StatusWarn {
				icon = "!"
			}
			fmt.Printf("  [%s] %s: %s\n", icon, res.Name, res.Message)
			if res.Fix != "" {
				fmt.Printf("      Fix: %s\n", res.Fix)
			}
		}
	}
}

// runExisting는 기존 config가 있을 때의 CRUD 플로우다.
func (r *Runner) runExisting(ctx context.Context) error {
	cfg, err := config.Load(r.CfgPath)
	if err != nil {
		return err
	}

	profileNames := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		profileNames = append(profileNames, name)
	}
	sort.Strings(profileNames)

	fmt.Println("기존 프로필:")
	for _, name := range profileNames {
		p := cfg.Profiles[name]
		fmt.Printf("  - %s (%s)\n", name, p.GitEmail)
	}

	action, err := r.FormRunner.RunActionSelect(profileNames)
	if err != nil {
		return err
	}

	switch action {
	case ActionAdd:
		return r.addProfile(ctx, cfg)
	case ActionEdit:
		return r.editProfile(ctx, cfg, profileNames)
	case ActionDelete:
		return r.deleteProfile(cfg, profileNames)
	default:
		return fmt.Errorf("setup: 알 수 없는 작업: %s", action)
	}
}

// addProfile은 새 프로필을 추가한다.
func (r *Runner) addProfile(ctx context.Context, cfg *config.Config) error {
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
	if err := config.Save(r.CfgPath, cfg); err != nil {
		return err
	}
	r.runDoctor(ctx, cfg)
	return nil
}

// editProfile은 기존 프로필을 수정한다.
func (r *Runner) editProfile(ctx context.Context, cfg *config.Config, profileNames []string) error {
	selected, err := r.FormRunner.RunProfileSelect(profileNames)
	if err != nil {
		return err
	}

	existing := cfg.Profiles[selected]
	defaults := &ProfileInput{
		Name:     selected,
		GitName:  existing.GitName,
		GitEmail: existing.GitEmail,
		SSHHost:  existing.SSHHost,
		Owners:   existing.Owners,
	}

	input, err := r.FormRunner.RunProfileForm(defaults, profileNames)
	if err != nil {
		return err
	}

	// 이름이 변경된 경우
	if input.Name != selected {
		delete(cfg.Profiles, selected)
	}

	cfg.Profiles[input.Name] = config.Profile{
		GHConfigDir: existing.GHConfigDir,
		SSHHost:     input.SSHHost,
		GitName:     input.GitName,
		GitEmail:    input.GitEmail,
		Owners:      input.Owners,
	}

	if err := config.Save(r.CfgPath, cfg); err != nil {
		return err
	}

	fmt.Println("기존 리포에 반영하려면 ctx init --refresh를 실행하세요.")
	r.runDoctor(ctx, cfg)
	return nil
}

// deleteProfile은 프로필을 삭제한다.
func (r *Runner) deleteProfile(cfg *config.Config, profileNames []string) error {
	if len(cfg.Profiles) <= 1 {
		return fmt.Errorf("setup: 마지막 프로필은 삭제할 수 없습니다")
	}

	selected, err := r.FormRunner.RunProfileSelect(profileNames)
	if err != nil {
		return err
	}

	confirmed, err := r.FormRunner.RunConfirm(
		fmt.Sprintf("프로필 %q을 정말 삭제하시겠습니까?", selected))
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("삭제가 취소되었습니다.")
		return nil
	}

	delete(cfg.Profiles, selected)
	return config.Save(r.CfgPath, cfg)
}
