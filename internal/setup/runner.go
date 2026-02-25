package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

	// SSH 키 감지 + 사용 중인 키 필터링
	sshDirPath := r.sshDir()
	sshConfigPath := r.SSHConfigPath
	if sshConfigPath == "" {
		sshConfigPath = DefaultSSHConfigPath()
	}

	allKeys := DetectSSHKeys(sshDirPath)
	usedPaths := r.usedIdentityFiles(cfg, sshConfigPath)
	availableKeys := FilterUsedSSHKeys(allKeys, usedPaths)
	keyChoice, err := r.FormRunner.RunSSHKeySelect(availableKeys, input.Name)
	if err != nil {
		return nil, err
	}

	var identityFile string

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
	ghLoginArgs := []string{"auth", "login", "--hostname", "github.com", "--git-protocol", "ssh"}
	if identityFile != "" {
		ghLoginArgs = append(ghLoginArgs, "--ssh-key", identityFile+".pub")
	}
	err = r.Commander.RunInteractiveWithEnv(ctx, env, "gh", ghLoginArgs...)
	if err != nil {
		// gh auth login이 SSH 키 업로드 실패(422) 등으로 에러를 반환해도
		// 인증 자체는 성공했을 수 있다. gh auth status로 실제 인증 상태를 확인한다.
		_, statusErr := r.Commander.RunWithEnv(ctx, env, "gh", "auth", "status", "--hostname", "github.com")
		if statusErr != nil {
			fmt.Fprintf(os.Stderr, "경고: gh 인증 실패 — 나중에 직접 인증하세요\n")
		}
	}

	// SSH 키가 GitHub에 등록되었는지 확인 및 보정
	if identityFile != "" {
		r.ensureSSHKeyRegistered(ctx, env, input.SSHHost, identityFile+".pub", input.Name)
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

// ensureSSHKeyRegistered는 SSH 키가 GitHub에 등록되었는지 확인하고, 필요하면 등록을 시도한다.
// gh auth login --ssh-key 플래그가 키 업로드에 실패했을 때(scope 부족 등) fallback으로 동작한다.
func (r *Runner) ensureSSHKeyRegistered(ctx context.Context, env map[string]string, sshHost, pubKeyPath, profileName string) {
	// SSH 연결 확인 — 이미 키가 등록되어 있으면 skip
	out, err := r.Commander.Run(ctx, "ssh", "-T", fmt.Sprintf("git@%s", sshHost))
	if err == nil || strings.Contains(string(out), "successfully authenticated") {
		return
	}

	// gh ssh-key add로 직접 등록 시도
	title := fmt.Sprintf("ctx-%s", profileName)
	addOut, addErr := r.Commander.RunWithEnv(ctx, env, "gh", "ssh-key", "add", pubKeyPath, "--title", title)
	if addErr == nil {
		fmt.Println("SSH 키가 GitHub에 등록되었습니다.")
		return
	}

	// 키가 다른 GitHub 계정에 이미 등록된 경우 — scope refresh로 해결 불가
	if strings.Contains(string(addOut), "already in use") {
		fmt.Fprintf(os.Stderr, "경고: 이 SSH 키는 다른 GitHub 계정에 이미 등록되어 있습니다.\n")
		fmt.Fprintf(os.Stderr, "  다른 계정에서 키를 제거하거나 새 키를 생성하세요.\n")
		return
	}

	// scope 부족(404)일 수 있음 — admin:public_key 권한 추가
	fmt.Fprintf(os.Stderr, "SSH 키 등록에 admin:public_key 권한이 필요합니다.\n")
	refreshErr := r.Commander.RunInteractiveWithEnv(ctx, env, "gh", "auth", "refresh", "-h", "github.com", "-s", "admin:public_key")
	if refreshErr != nil {
		r.printSSHKeyManualFix(pubKeyPath, title)
		return
	}

	// 권한 추가 후 재시도
	_, retryErr := r.Commander.RunWithEnv(ctx, env, "gh", "ssh-key", "add", pubKeyPath, "--title", title)
	if retryErr != nil {
		r.printSSHKeyManualFix(pubKeyPath, title)
		return
	}
	fmt.Println("SSH 키가 GitHub에 등록되었습니다.")
}

// printSSHKeyManualFix는 SSH 키 수동 등록 안내를 출력한다.
func (r *Runner) printSSHKeyManualFix(pubKeyPath, title string) {
	fmt.Fprintf(os.Stderr, "경고: SSH 키를 GitHub에 등록하지 못했습니다.\n")
	fmt.Fprintf(os.Stderr, "  수동 등록: gh ssh-key add %s --title %s\n", pubKeyPath, title)
}

// usedIdentityFiles는 기존 프로필들이 사용 중인 SSH IdentityFile 경로를 수집한다.
func (r *Runner) usedIdentityFiles(cfg *config.Config, sshConfigPath string) map[string]bool {
	hostToKey := ParseSSHConfigIdentityFiles(sshConfigPath)
	used := make(map[string]bool)
	for _, p := range cfg.Profiles {
		if idFile, ok := hostToKey[p.SSHHost]; ok {
			// ~ 경로를 절대 경로로 확장
			if strings.HasPrefix(idFile, "~/") {
				home, err := os.UserHomeDir()
				if err == nil {
					idFile = filepath.Join(home, idFile[2:])
				}
			}
			used[idFile] = true
		}
	}
	return used
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
