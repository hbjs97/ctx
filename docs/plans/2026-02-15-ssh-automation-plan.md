# SSH Key Automation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** `ctx setup` 플로우에 SSH 키 감지/생성과 `~/.ssh/config` Host alias 자동 작성을 통합하여, 사용자가 수동 SSH 설정 없이 셋업을 완료할 수 있게 한다.

**Architecture:** `internal/setup/ssh.go`에 SSH 키 감지/생성/config 쓰기 함수를 집중시키고, 기존 detect.go의 SSH 관련 함수를 이동한다. Runner의 collectProfile 플로우에서 gh auth login 전에 SSH 키를 준비하고, 생성된 Host alias를 자동 선택한다.

**Tech Stack:** Go, `ssh-keygen` (Commander를 통해 실행), `charmbracelet/huh` (TUI), `os` (파일 I/O)

**Design doc:** `docs/plans/2026-02-15-ssh-automation-design.md`

---

### Task 1: SSHKeyInfo, SSHKeyChoice 타입 및 FormRunner 확장

**Files:**
- Modify: `internal/setup/types.go`

**Context:**
- 현재 types.go에는 Action enum, ProfileInput struct, FormRunner interface (7 메서드)가 있다.
- SSHKeyInfo는 감지된 SSH 키 정보, SSHKeyChoice는 사용자 선택 결과를 표현한다.
- FormRunner에 RunSSHKeySelect 메서드를 추가한다.

**Step 1: types.go에 타입과 메서드 추가**

`internal/setup/types.go`의 `FormRunner interface` 닫는 중괄호 앞에 다음을 추가:

```go
// SSHKeyInfo는 감지된 SSH 키 쌍 정보다.
type SSHKeyInfo struct {
	Name       string // 파일 이름 (예: "id_ed25519_work")
	PrivateKey string // 비밀키 전체 경로
	PublicKey  string // 공개키 전체 경로 (.pub)
}

// SSHKeyChoice는 SSH 키 선택 결과다.
type SSHKeyChoice struct {
	Action      string // "existing" | "generate"
	ExistingKey string // Action=="existing"일 때 비밀키 경로
}
```

FormRunner interface에 추가:

```go
	// RunSSHKeySelect는 기존 SSH 키 목록에서 선택하거나 새로 생성하는 UI를 표시한다.
	RunSSHKeySelect(existingKeys []SSHKeyInfo, profileName string) (SSHKeyChoice, error)
```

**Step 2: 빌드 확인**

Run: `go build ./internal/setup/...`
Expected: 컴파일 에러 — HuhFormRunner와 mockFormRunner가 RunSSHKeySelect를 구현하지 않아서 실패. 이 시점에서는 정상이다. 다음 태스크에서 해결한다.

**Step 3: Commit**

```bash
git add internal/setup/types.go
git commit -m "feat(setup): add SSHKeyInfo, SSHKeyChoice types and RunSSHKeySelect to FormRunner"
```

---

### Task 2: ssh.go 생성 및 기존 함수 이동

**Files:**
- Create: `internal/setup/ssh.go`
- Create: `internal/setup/ssh_test.go`
- Modify: `internal/setup/detect.go` — ParseSSHConfig, DefaultSSHConfigPath 제거
- Modify: `internal/setup/detect_test.go` — ParseSSHConfig 테스트 제거

**Context:**
- 현재 detect.go에 ParseSSHConfig(57줄), DefaultSSHConfigPath(6줄)이 있다.
- detect_test.go에 ParseSSHConfig 테스트 4개(NoFile, ParsesGitHubHosts, IgnoresNonGitHub, DetectsGitHubByHostName)가 있다.
- detect.go에는 DetectOrgs만 남기고, SSH 관련은 모두 ssh.go로 이동한다.

**Step 1: ssh.go 생성 — ParseSSHConfig, DefaultSSHConfigPath 이동**

`internal/setup/ssh.go` 생성:

```go
package setup

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
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
			if currentHost != "" && isGitHub {
				hosts = append(hosts, currentHost)
			}
			currentHost = strings.TrimPrefix(line, "Host ")
			currentHost = strings.TrimSpace(currentHost)
			isGitHub = strings.Contains(strings.ToLower(currentHost), "github")
		}

		if strings.HasPrefix(line, "HostName") {
			hostName := strings.TrimSpace(strings.TrimPrefix(line, "HostName"))
			if strings.Contains(hostName, "github.com") {
				isGitHub = true
			}
		}
	}

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
```

**Step 2: ssh_test.go 생성 — ParseSSHConfig 테스트 이동**

기존 detect_test.go에서 ParseSSHConfig 관련 테스트 4개를 ssh_test.go로 이동한다. 테스트 내용은 동일하되 파일만 변경한다.

**Step 3: detect.go에서 ParseSSHConfig, DefaultSSHConfigPath 제거**

detect.go에서 해당 함수와 사용하지 않는 import(`bufio`, `os`, `path/filepath`)를 제거한다. DetectOrgs 함수만 남는다.

**Step 4: detect_test.go에서 ParseSSHConfig 테스트 제거**

ParseSSHConfig 관련 테스트 4개(TestParseSSHConfig_*)를 제거한다.

**Step 5: 전체 테스트 확인**

Run: `go test ./internal/setup/... -v -count=1`
Expected: 기존 테스트 모두 PASS (함수 이동이므로 동일 패키지 내에서 호출 경로 변경 없음)

**Step 6: Commit**

```bash
git add internal/setup/ssh.go internal/setup/ssh_test.go internal/setup/detect.go internal/setup/detect_test.go
git commit -m "refactor(setup): move SSH functions from detect.go to ssh.go"
```

---

### Task 3: DetectSSHKeys 함수

**Files:**
- Modify: `internal/setup/ssh.go`
- Modify: `internal/setup/ssh_test.go`

**Context:**
- `~/.ssh/` 디렉토리에서 `id_*` 패턴의 키 쌍(비밀키 + .pub)을 찾아 SSHKeyInfo 목록으로 반환한다.
- 비밀키 파일이 있고, 대응하는 .pub 파일이 있어야 유효한 키 쌍이다.

**Step 1: 실패하는 테스트 작성**

ssh_test.go에 추가:

```go
func TestDetectSSHKeys_FindsKeyPairs(t *testing.T) {
	dir := t.TempDir()

	// 유효한 키 쌍 생성
	os.WriteFile(filepath.Join(dir, "id_ed25519_work"), []byte("private"), 0600)
	os.WriteFile(filepath.Join(dir, "id_ed25519_work.pub"), []byte("public"), 0644)

	// 공개키만 있는 경우 (무시해야 함)
	os.WriteFile(filepath.Join(dir, "id_rsa_orphan.pub"), []byte("public"), 0644)

	// 키가 아닌 파일 (무시해야 함)
	os.WriteFile(filepath.Join(dir, "config"), []byte("config"), 0644)
	os.WriteFile(filepath.Join(dir, "known_hosts"), []byte("hosts"), 0644)

	keys := DetectSSHKeys(dir)

	if len(keys) != 1 {
		t.Fatalf("expected 1 key pair, got %d", len(keys))
	}
	if keys[0].Name != "id_ed25519_work" {
		t.Errorf("expected name id_ed25519_work, got %s", keys[0].Name)
	}
	if keys[0].PrivateKey != filepath.Join(dir, "id_ed25519_work") {
		t.Errorf("unexpected private key path: %s", keys[0].PrivateKey)
	}
	if keys[0].PublicKey != filepath.Join(dir, "id_ed25519_work.pub") {
		t.Errorf("unexpected public key path: %s", keys[0].PublicKey)
	}
}

func TestDetectSSHKeys_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	keys := DetectSSHKeys(dir)
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(keys))
	}
}

func TestDetectSSHKeys_NonExistentDir(t *testing.T) {
	keys := DetectSSHKeys("/nonexistent/path")
	if keys != nil {
		t.Fatalf("expected nil, got %v", keys)
	}
}

func TestDetectSSHKeys_MultipleKeys(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "id_ed25519"), []byte("p"), 0600)
	os.WriteFile(filepath.Join(dir, "id_ed25519.pub"), []byte("p"), 0644)
	os.WriteFile(filepath.Join(dir, "id_rsa_work"), []byte("p"), 0600)
	os.WriteFile(filepath.Join(dir, "id_rsa_work.pub"), []byte("p"), 0644)

	keys := DetectSSHKeys(dir)
	if len(keys) != 2 {
		t.Fatalf("expected 2 key pairs, got %d", len(keys))
	}
}
```

**Step 2: 테스트 실패 확인**

Run: `go test ./internal/setup/... -run TestDetectSSHKeys -v`
Expected: FAIL — DetectSSHKeys 함수가 없음

**Step 3: 구현**

ssh.go에 추가:

```go
// DetectSSHKeys는 주어진 디렉토리에서 SSH 키 쌍을 감지한다.
// id_* 패턴의 비밀키 파일과 대응하는 .pub 파일이 모두 존재해야 유효한 키 쌍이다.
func DetectSSHKeys(sshDir string) []SSHKeyInfo {
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil
	}

	var keys []SSHKeyInfo
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasSuffix(name, ".pub") || !strings.HasPrefix(name, "id_") {
			continue
		}
		pubPath := filepath.Join(sshDir, name+".pub")
		if _, err := os.Stat(pubPath); err != nil {
			continue
		}
		keys = append(keys, SSHKeyInfo{
			Name:       name,
			PrivateKey: filepath.Join(sshDir, name),
			PublicKey:  pubPath,
		})
	}
	return keys
}
```

**Step 4: 테스트 통과 확인**

Run: `go test ./internal/setup/... -run TestDetectSSHKeys -v`
Expected: 4/4 PASS

**Step 5: Commit**

```bash
git add internal/setup/ssh.go internal/setup/ssh_test.go
git commit -m "feat(setup): add DetectSSHKeys for SSH key pair discovery"
```

---

### Task 4: GenerateSSHKey 함수

**Files:**
- Modify: `internal/setup/ssh.go`
- Modify: `internal/setup/ssh_test.go`

**Context:**
- Commander interface를 통해 `ssh-keygen` 실행. 테스트에서는 FakeCommander 사용.
- `ssh-keygen -t ed25519 -C {email} -f {path} -N ""` 형식.
- `testutil.FakeCommander`의 Register 시그니처: `Register(key string, output string, err error)`. 3인자.

**Step 1: 실패하는 테스트 작성**

ssh_test.go에 추가 (import에 `"context"`, `"github.com/hbjs97/ctx/internal/testutil"` 필요):

```go
func TestGenerateSSHKey_Success(t *testing.T) {
	fc := testutil.NewFakeCommander()
	fc.Register("ssh-keygen", "", nil)

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519_test")

	err := GenerateSSHKey(context.Background(), fc, "test@example.com", keyPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !fc.Called("ssh-keygen") {
		t.Fatal("ssh-keygen was not called")
	}

	// 명령어 인자 확인
	call := fc.Calls[0]
	if !strings.Contains(call, "-t ed25519") {
		t.Errorf("expected -t ed25519 in command, got: %s", call)
	}
	if !strings.Contains(call, "-C test@example.com") {
		t.Errorf("expected -C email in command, got: %s", call)
	}
	if !strings.Contains(call, "-f "+keyPath) {
		t.Errorf("expected -f keyPath in command, got: %s", call)
	}
}

func TestGenerateSSHKey_Failure(t *testing.T) {
	fc := testutil.NewFakeCommander()
	fc.Register("ssh-keygen", "error output", fmt.Errorf("exit status 1"))

	err := GenerateSSHKey(context.Background(), fc, "test@example.com", "/tmp/key")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

**Step 2: 테스트 실패 확인**

Run: `go test ./internal/setup/... -run TestGenerateSSHKey -v`
Expected: FAIL — GenerateSSHKey 함수가 없음

**Step 3: 구현**

ssh.go에 추가 (import에 `"context"`, `"fmt"`, `"github.com/hbjs97/ctx/internal/cmdexec"` 필요):

```go
// GenerateSSHKey는 ssh-keygen으로 ed25519 키 쌍을 생성한다.
// 빈 passphrase로 생성하며, Commander를 통해 실행한다.
func GenerateSSHKey(ctx context.Context, cmd cmdexec.Commander, email, keyPath string) error {
	_, err := cmd.Run(ctx, "ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", "")
	if err != nil {
		return fmt.Errorf("setup.GenerateSSHKey: %w", err)
	}
	return nil
}
```

**Step 4: 테스트 통과 확인**

Run: `go test ./internal/setup/... -run TestGenerateSSHKey -v`
Expected: 2/2 PASS

**Step 5: Commit**

```bash
git add internal/setup/ssh.go internal/setup/ssh_test.go
git commit -m "feat(setup): add GenerateSSHKey via Commander"
```

---

### Task 5: WriteSSHConfigEntry 함수

**Files:**
- Modify: `internal/setup/ssh.go`
- Modify: `internal/setup/ssh_test.go`

**Context:**
- `~/.ssh/config`에 Host 블록을 추가한다. 이미 동일 Host가 있으면 스킵한다.
- 파일이 없으면 생성한다 (0600 권한).
- IdentitiesOnly yes를 포함한다.

**Step 1: 실패하는 테스트 작성**

ssh_test.go에 추가:

```go
func TestWriteSSHConfigEntry_NewFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	err := WriteSSHConfigEntry(configPath, "github.com-work", "~/.ssh/id_ed25519_work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(configPath)
	s := string(content)
	if !strings.Contains(s, "Host github.com-work") {
		t.Error("missing Host line")
	}
	if !strings.Contains(s, "HostName github.com") {
		t.Error("missing HostName line")
	}
	if !strings.Contains(s, "IdentityFile ~/.ssh/id_ed25519_work") {
		t.Error("missing IdentityFile line")
	}
	if !strings.Contains(s, "IdentitiesOnly yes") {
		t.Error("missing IdentitiesOnly line")
	}
}

func TestWriteSSHConfigEntry_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	os.WriteFile(configPath, []byte("Host existing\n  HostName example.com\n"), 0600)

	err := WriteSSHConfigEntry(configPath, "github.com-personal", "~/.ssh/id_ed25519_personal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(configPath)
	s := string(content)
	if !strings.Contains(s, "Host existing") {
		t.Error("existing entry was lost")
	}
	if !strings.Contains(s, "Host github.com-personal") {
		t.Error("new entry not appended")
	}
}

func TestWriteSSHConfigEntry_SkipsDuplicate(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	existing := "Host github.com-work\n  HostName github.com\n  User git\n  IdentityFile ~/.ssh/id_ed25519_work\n"
	os.WriteFile(configPath, []byte(existing), 0600)

	err := WriteSSHConfigEntry(configPath, "github.com-work", "~/.ssh/id_ed25519_work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(configPath)
	if count := strings.Count(string(content), "Host github.com-work"); count != 1 {
		t.Errorf("expected 1 occurrence, got %d", count)
	}
}
```

**Step 2: 테스트 실패 확인**

Run: `go test ./internal/setup/... -run TestWriteSSHConfigEntry -v`
Expected: FAIL — WriteSSHConfigEntry 함수가 없음

**Step 3: 구현**

ssh.go에 추가:

```go
// WriteSSHConfigEntry는 SSH config 파일에 GitHub Host alias 블록을 추가한다.
// 동일한 Host가 이미 존재하면 스킵한다. 파일이 없으면 생성한다.
func WriteSSHConfigEntry(configPath, host, identityFile string) error {
	existing, _ := os.ReadFile(configPath)
	hostLine := "Host " + host
	if strings.Contains(string(existing), hostLine) {
		return nil // 이미 존재
	}

	entry := fmt.Sprintf("\n%s\n  HostName github.com\n  User git\n  IdentityFile %s\n  IdentitiesOnly yes\n", hostLine, identityFile)

	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("setup.WriteSSHConfigEntry: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("setup.WriteSSHConfigEntry: %w", err)
	}
	return nil
}
```

**Step 4: 테스트 통과 확인**

Run: `go test ./internal/setup/... -run TestWriteSSHConfigEntry -v`
Expected: 3/3 PASS

**Step 5: Commit**

```bash
git add internal/setup/ssh.go internal/setup/ssh_test.go
git commit -m "feat(setup): add WriteSSHConfigEntry for SSH config automation"
```

---

### Task 6: HuhFormRunner.RunSSHKeySelect 및 mockFormRunner 업데이트

**Files:**
- Modify: `internal/setup/form.go`
- Modify: `internal/setup/runner_test.go` — mockFormRunner에 RunSSHKeySelect 추가

**Context:**
- HuhFormRunner는 charmbracelet/huh 기반 TUI 구현이다.
- mockFormRunner는 runner_test.go에서 테스트용으로 사용한다.
- 기존 키가 있으면 목록 + "새로 생성" 옵션, 없으면 생성 확인 프롬프트.

**Step 1: HuhFormRunner에 RunSSHKeySelect 구현**

form.go에 추가:

```go
// RunSSHKeySelect는 SSH 키 선택 UI를 표시한다.
func (h *HuhFormRunner) RunSSHKeySelect(existingKeys []SSHKeyInfo, profileName string) (SSHKeyChoice, error) {
	if len(existingKeys) == 0 {
		var generate bool
		form := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title("SSH 키가 없습니다. 새로 생성할까요?").
				Description(fmt.Sprintf("~/.ssh/id_ed25519_%s 키 쌍을 생성합니다", profileName)).
				Value(&generate),
		))
		if err := form.Run(); err != nil {
			return SSHKeyChoice{}, fmt.Errorf("setup.RunSSHKeySelect: %w", err)
		}
		if generate {
			return SSHKeyChoice{Action: "generate"}, nil
		}
		return SSHKeyChoice{Action: "skip"}, nil
	}

	options := make([]huh.Option[string], 0, len(existingKeys)+1)
	for _, k := range existingKeys {
		options = append(options, huh.NewOption(k.Name+" ("+k.PublicKey+")", k.PrivateKey))
	}
	options = append(options, huh.NewOption("새 키 생성 (id_ed25519_"+profileName+")", "__generate__"))

	var selected string
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("SSH 키를 선택하세요").
			Options(options...).
			Value(&selected),
	))
	if err := form.Run(); err != nil {
		return SSHKeyChoice{}, fmt.Errorf("setup.RunSSHKeySelect: %w", err)
	}

	if selected == "__generate__" {
		return SSHKeyChoice{Action: "generate"}, nil
	}
	return SSHKeyChoice{Action: "existing", ExistingKey: selected}, nil
}
```

**Step 2: mockFormRunner에 RunSSHKeySelect 추가**

runner_test.go의 mockFormRunner struct에 필드 및 메서드 추가:

```go
// struct에 추가:
sshKeyChoice SSHKeyChoice

// 메서드 추가:
func (m *mockFormRunner) RunSSHKeySelect(existingKeys []SSHKeyInfo, profileName string) (SSHKeyChoice, error) {
	return m.sshKeyChoice, nil
}
```

**Step 3: 빌드 확인**

Run: `go build ./internal/setup/...`
Expected: 성공 — 모든 FormRunner 구현체가 인터페이스를 만족

**Step 4: 기존 테스트 통과 확인**

Run: `go test ./internal/setup/... -v -count=1`
Expected: 기존 테스트 모두 PASS

**Step 5: Commit**

```bash
git add internal/setup/form.go internal/setup/runner_test.go
git commit -m "feat(setup): implement RunSSHKeySelect in HuhFormRunner and mock"
```

---

### Task 7: Runner collectProfile 플로우 변경

**Files:**
- Modify: `internal/setup/runner.go`
- Modify: `internal/setup/runner_test.go`

**Context:**
- 현재 collectProfile 순서: ProfileForm → gh auth login → ParseSSHConfig → SSHHostSelect → DetectOrgs
- 변경 후: ProfileForm → DetectSSHKeys → RunSSHKeySelect → (GenerateSSHKey) → WriteSSHConfigEntry → gh auth login → SSHHost 자동결정 → DetectOrgs
- Host alias 네이밍: `github.com-{profileName}`
- 키 경로: `~/.ssh/id_ed25519_{profileName}`
- SSH 키/config 실패 시 경고만 출력하고 기존 수동 플로우로 fallback

**Step 1: runner.go collectProfile 수정**

기존 collectProfile 함수를 다음으로 교체:

```go
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
	sshDir := r.sshDir()
	sshHost := r.setupSSHKey(ctx, input.Name, input.GitEmail, sshDir)

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

	// SSH host 결정
	if sshHost != "" {
		input.SSHHost = sshHost
	} else {
		// fallback: 기존 수동 선택 플로우
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

// sshDir는 SSH 디렉토리 경로를 반환한다.
func (r *Runner) sshDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh")
}

// setupSSHKey는 SSH 키 감지/생성 + config 작성을 수행한다.
// 성공 시 생성된 Host alias를 반환한다. 실패/스킵 시 빈 문자열을 반환한다.
func (r *Runner) setupSSHKey(ctx context.Context, profileName, email, sshDir string) string {
	if sshDir == "" {
		return ""
	}

	keys := DetectSSHKeys(sshDir)
	choice, err := r.FormRunner.RunSSHKeySelect(keys, profileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "경고: SSH 키 선택 실패: %v\n", err)
		return ""
	}

	host := fmt.Sprintf("github.com-%s", profileName)
	var identityFile string

	switch choice.Action {
	case "generate":
		keyPath := filepath.Join(sshDir, fmt.Sprintf("id_ed25519_%s", profileName))
		if err := GenerateSSHKey(ctx, r.Commander, email, keyPath); err != nil {
			fmt.Fprintf(os.Stderr, "경고: SSH 키 생성 실패: %v\n", err)
			return ""
		}
		identityFile = keyPath
		fmt.Printf("SSH 키가 생성되었습니다: %s\n", keyPath)
	case "existing":
		identityFile = choice.ExistingKey
	default:
		return ""
	}

	sshConfigPath := r.SSHConfigPath
	if sshConfigPath == "" {
		sshConfigPath = DefaultSSHConfigPath()
	}
	if err := WriteSSHConfigEntry(sshConfigPath, host, identityFile); err != nil {
		fmt.Fprintf(os.Stderr, "경고: SSH config 작성 실패: %v\n", err)
		return ""
	}
	fmt.Printf("SSH config에 Host alias가 추가되었습니다: %s\n", host)

	return host
}
```

**Step 2: runner_test.go 업데이트**

기존 테스트에서 mockFormRunner에 `sshKeyChoice` 설정 추가. 새 테스트 추가:

```go
func TestRunner_FirstRun_WithSSHKeyGeneration(t *testing.T) {
	// SSH 키 생성 + config 작성 + gh auth login 순서 테스트
	// sshKeyChoice: {Action: "generate"}
	// SSH dir에 키가 없는 상태에서 시작
	// 기대: ssh-keygen 호출됨, SSH config 작성됨, SSHHost가 github.com-{profileName}
}

func TestRunner_FirstRun_WithExistingSSHKey(t *testing.T) {
	// 기존 키 선택 테스트
	// sshKeyChoice: {Action: "existing", ExistingKey: "/path/to/key"}
	// 기대: ssh-keygen 호출 안 됨, SSH config 작성됨
}

func TestRunner_FirstRun_SSHKeySkip(t *testing.T) {
	// SSH 키 생성 스킵 테스트
	// sshKeyChoice: {Action: "skip"}
	// 기대: fallback으로 RunSSHHostSelect 호출됨
}
```

기존 테스트의 mockFormRunner에도 `sshKeyChoice`를 적절히 설정:
- 기존 테스트들은 `sshKeyChoice: SSHKeyChoice{Action: "generate"}` + ssh-keygen mock 등록으로 업데이트

**Step 3: 테스트 통과 확인**

Run: `go test ./internal/setup/... -v -count=1`
Expected: 기존 + 신규 테스트 모두 PASS

**Step 4: Commit**

```bash
git add internal/setup/runner.go internal/setup/runner_test.go
git commit -m "feat(setup): integrate SSH key generation into collectProfile flow"
```

---

### Task 8: README 업데이트

**Files:**
- Modify: `README.md`

**Context:**
- 사전 요구사항에서 SSH 수동 설정 항목 제거
- 빠른 시작 섹션에 SSH 자동화 설명 추가

**Step 1: README 수정**

사전 요구사항 섹션:

```markdown
## 사전 요구사항

- Go 1.26+
- `git`, `gh`, `ssh` CLI 설치
```

(기존의 "GitHub 계정별 `gh auth login` 완료" 및 "GitHub 계정별 SSH Host alias 설정" 항목 제거)

빠른 시작 > 초기 설정 섹션 업데이트:

```markdown
### 1. 초기 설정

\`\`\`bash
ctx setup
\`\`\`

대화형 설정 마법사가 실행된다:

- 프로필 이름, git 사용자 정보 입력
- SSH 키 자동 감지 (기존 키 선택 또는 새로 생성)
- `~/.ssh/config`에 Host alias 자동 추가
- `gh auth login` 자동 실행 (생성된 SSH 키 업로드 포함)
- `gh api`로 소속 조직 자동 조회
- 셸 hook 자동 설치
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: update README to reflect SSH key automation"
```
