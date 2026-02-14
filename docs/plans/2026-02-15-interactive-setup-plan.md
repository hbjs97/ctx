# Interactive Setup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** `ctx setup`을 TUI 기반 interactive wizard로 개선하여 프로필 CRUD, gh 인증, SSH host 자동 감지, 셸 hook 설치를 제공한다.

**Architecture:** 새 `internal/setup/` 패키지에 TUI 로직을 분리. `FormRunner` interface로 huh 의존성을 추상화하여 테스트 가능. `cli/setup.go`는 Runner를 호출만 한다.

**Tech Stack:** Go 1.26, charmbracelet/huh (TUI forms), BurntSushi/toml (config 직렬화)

**Design doc:** `docs/plans/2026-02-15-interactive-setup-design.md`

---

### Task 1: huh 의존성 추가

**Files:**
- Modify: `go.mod`

**Step 1: huh 라이브러리 설치**

Run: `go get github.com/charmbracelet/huh@latest`

**Step 2: go.mod 정리**

Run: `go mod tidy`

**Step 3: 빌드 확인**

Run: `go build ./...`
Expected: 성공 (huh를 아직 사용하지 않으므로)

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add charmbracelet/huh dependency for interactive setup"
```

---

### Task 2: config.Save() 함수 추가

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/config/save_test.go` (또는 기존 `config_test.go`에 추가)

**Step 1: 실패하는 테스트 작성**

`internal/config/save_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSave_WritesValidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	boolTrue := true
	cfg := &Config{
		Version:           1,
		DefaultProfile:    "work",
		PromptOnAmbiguous: &boolTrue,
		RequirePushGuard:  &boolTrue,
		CacheTTLDays:      90,
		Profiles: map[string]Profile{
			"work": {
				GHConfigDir: "/tmp/gh-work",
				SSHHost:     "github.com-work",
				GitName:     "HBJS",
				GitEmail:    "hbjs@company.com",
				Owners:      []string{"company-org"},
			},
		},
	}

	err := Save(path, cfg)
	require.NoError(t, err)

	// 파일이 0600 권한으로 생성되었는지 확인
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// 다시 로드하여 내용 확인
	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "work", loaded.DefaultProfile)
	assert.Equal(t, "hbjs@company.com", loaded.Profiles["work"].GitEmail)
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	boolTrue := true
	cfg := &Config{
		Version:           1,
		PromptOnAmbiguous: &boolTrue,
		RequirePushGuard:  &boolTrue,
		CacheTTLDays:      90,
		Profiles: map[string]Profile{
			"work": {
				GHConfigDir: "/tmp/gh-work",
				SSHHost:     "github.com-work",
				GitName:     "HBJS",
				GitEmail:    "hbjs@company.com",
				Owners:      []string{"company-org", "company-team"},
			},
			"personal": {
				GHConfigDir: "/tmp/gh-personal",
				SSHHost:     "github.com-personal",
				GitName:     "hbjs97",
				GitEmail:    "hbjs97@naver.com",
				Owners:      []string{"hbjs97"},
			},
		},
	}

	err := Save(path, cfg)
	require.NoError(t, err)

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, len(cfg.Profiles), len(loaded.Profiles))

	for name, p := range cfg.Profiles {
		lp, ok := loaded.Profiles[name]
		require.True(t, ok, "프로필 %s 없음", name)
		assert.Equal(t, p.GitEmail, lp.GitEmail)
		assert.Equal(t, p.SSHHost, lp.SSHHost)
		assert.Equal(t, p.Owners, lp.Owners)
	}
}

func TestSave_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "config.toml")

	boolTrue := true
	cfg := &Config{
		Version:          1,
		RequirePushGuard: &boolTrue,
		CacheTTLDays:     90,
		Profiles: map[string]Profile{
			"test": {
				GHConfigDir: "/tmp/gh-test",
				SSHHost:     "github.com-test",
				GitName:     "Test",
				GitEmail:    "test@test.com",
				Owners:      []string{"test-org"},
			},
		},
	}

	err := Save(path, cfg)
	require.NoError(t, err)

	_, err = os.Stat(path)
	assert.NoError(t, err)
}
```

**Step 2: 테스트 실패 확인**

Run: `go test ./internal/config/ -run TestSave -v`
Expected: FAIL — `Save` 함수 미정의

**Step 3: Save 구현**

`internal/config/config.go`에 추가:

```go
import (
	"bytes"
	// 기존 import...
)

// Save는 Config를 TOML 형식으로 파일에 저장한다.
func Save(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}

	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("config.Save: %w", err)
	}
	return nil
}
```

**Step 4: 테스트 통과 확인**

Run: `go test ./internal/config/ -run TestSave -v`
Expected: PASS

**Step 5: 전체 테스트 확인**

Run: `go test ./...`
Expected: 전체 PASS

**Step 6: Commit**

```bash
git add internal/config/config.go internal/config/save_test.go
git commit -m "feat(config): add Save function for TOML serialization"
```

---

### Task 3: setup 패키지 타입 및 인터페이스 정의

**Files:**
- Create: `internal/setup/types.go`
- Create: `internal/setup/types_test.go`

**Step 1: 타입 파일 작성**

`internal/setup/types.go`:

```go
package setup

// Action은 재실행 시 사용자가 선택하는 작업이다.
type Action string

const (
	ActionAdd    Action = "add"
	ActionEdit   Action = "edit"
	ActionDelete Action = "delete"
)

// ProfileInput은 프로필 생성/수정 시 사용자 입력 값이다.
type ProfileInput struct {
	Name     string
	GitName  string
	GitEmail string
	SSHHost  string
	Owners   []string
}

// FormRunner는 TUI 폼 실행을 추상화하는 interface다.
// 프로덕션에서는 huh 기반 구현, 테스트에서는 mock을 사용한다.
type FormRunner interface {
	// RunProfileForm은 프로필 입력 폼을 실행한다.
	// defaults가 nil이 아니면 기존 값을 기본값으로 표시한다 (수정 모드).
	RunProfileForm(defaults *ProfileInput, existingNames []string) (*ProfileInput, error)

	// RunActionSelect는 작업 선택 UI를 표시한다.
	RunActionSelect(profileNames []string) (Action, error)

	// RunProfileSelect는 프로필 선택 UI를 표시한다.
	RunProfileSelect(profileNames []string) (string, error)

	// RunConfirm은 확인 프롬프트를 표시한다.
	RunConfirm(message string) (bool, error)

	// RunAddMore는 "프로필을 더 추가하시겠습니까?" 프롬프트를 표시한다.
	RunAddMore() (bool, error)

	// RunSSHHostSelect는 감지된 SSH host 목록에서 선택 UI를 표시한다.
	// hosts가 비어있으면 직접 입력 폼으로 fallback한다.
	RunSSHHostSelect(hosts []string) (string, error)

	// RunOwnersSelect는 조직/사용자 체크리스트를 표시한다.
	// detected가 비어있으면 직접 입력 폼으로 fallback한다.
	RunOwnersSelect(detected []string) ([]string, error)
}
```

**Step 2: 빌드 확인**

Run: `go build ./internal/setup/...`
Expected: 성공

**Step 3: Commit**

```bash
git add internal/setup/types.go
git commit -m "feat(setup): define types and FormRunner interface"
```

---

### Task 4: SSH host 및 조직 자동 감지

**Files:**
- Create: `internal/setup/detect.go`
- Create: `internal/setup/detect_test.go`

**Step 1: SSH host 감지 실패 테스트**

`internal/setup/detect_test.go`:

```go
package setup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSSHConfig_NoFile(t *testing.T) {
	hosts := ParseSSHConfig("/nonexistent/path")
	assert.Empty(t, hosts)
}

func TestParseSSHConfig_ParsesGitHubHosts(t *testing.T) {
	content := `Host github.com-work
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519_work

Host github.com-personal
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519_personal

Host other-server
    HostName example.com
`
	dir := t.TempDir()
	path := dir + "/config"
	os.WriteFile(path, []byte(content), 0600)

	hosts := ParseSSHConfig(path)
	assert.Equal(t, []string{"github.com-work", "github.com-personal"}, hosts)
}

func TestParseSSHConfig_IgnoresNonGitHub(t *testing.T) {
	content := `Host example.com
    HostName example.com
`
	dir := t.TempDir()
	path := dir + "/config"
	os.WriteFile(path, []byte(content), 0600)

	hosts := ParseSSHConfig(path)
	assert.Empty(t, hosts)
}
```

**Step 2: 테스트 실패 확인**

Run: `go test ./internal/setup/ -run TestParseSSHConfig -v`
Expected: FAIL

**Step 3: ParseSSHConfig 구현**

`internal/setup/detect.go`:

```go
package setup

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/hbjs97/ctx/internal/gh"
)

// ParseSSHConfig는 ~/.ssh/config에서 GitHub 관련 Host alias 목록을 추출한다.
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
```

**Step 4: SSH 테스트 통과 확인**

Run: `go test ./internal/setup/ -run TestParseSSHConfig -v`
Expected: PASS

**Step 5: 조직 조회 테스트 작성**

`internal/setup/detect_test.go`에 추가:

```go
func TestDetectOrgs_Success(t *testing.T) {
	fc := testutil.NewFakeCommander()
	fc.Register("gh api user/orgs --jq .[].login", testutil.Response{
		Output: []byte("company-org\ncompany-team\n"),
	})
	fc.Register("gh api user --jq .login", testutil.Response{
		Output: []byte("hbjs97\n"),
	})

	orgs := DetectOrgs(context.Background(), fc, "/tmp/gh-work")
	assert.Equal(t, []string{"company-org", "company-team", "hbjs97"}, orgs)
}

func TestDetectOrgs_Failure(t *testing.T) {
	fc := testutil.NewFakeCommander()
	fc.Register("gh api user/orgs --jq .[].login", testutil.Response{
		Err: fmt.Errorf("auth required"),
	})

	orgs := DetectOrgs(context.Background(), fc, "/tmp/gh-work")
	assert.Empty(t, orgs)
}
```

**Step 6: DetectOrgs 구현**

`internal/setup/detect.go`에 추가:

```go
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
```

**Step 7: 테스트 통과 확인**

Run: `go test ./internal/setup/ -run TestDetect -v`
Expected: PASS

**Step 8: Commit**

```bash
git add internal/setup/detect.go internal/setup/detect_test.go
git commit -m "feat(setup): SSH host detection and org auto-query"
```

---

### Task 5: huh 기반 FormRunner 구현

**Files:**
- Create: `internal/setup/form.go`

이 파일은 huh 라이브러리에 의존하는 프로덕션 코드다.
TUI 폼은 `WithInput()`으로 테스트할 수 있지만, 복잡한 상호작용 테스트보다는
Runner 레벨에서 mock FormRunner로 테스트하는 것이 효과적이다.

**Step 1: HuhFormRunner 구현**

`internal/setup/form.go`:

```go
package setup

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
)

// HuhFormRunner는 charmbracelet/huh 기반의 FormRunner 구현이다.
type HuhFormRunner struct{}

var _ FormRunner = (*HuhFormRunner)(nil)

var profileNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)

// RunProfileForm은 프로필 입력 폼을 실행한다.
func (h *HuhFormRunner) RunProfileForm(defaults *ProfileInput, existingNames []string) (*ProfileInput, error) {
	input := &ProfileInput{}
	if defaults != nil {
		*input = *defaults
	}

	nameValidate := func(s string) error {
		if s == "" {
			return fmt.Errorf("프로필 이름을 입력하세요")
		}
		if !profileNameRegex.MatchString(s) {
			return fmt.Errorf("영문, 숫자, 하이픈만 사용 가능합니다")
		}
		for _, n := range existingNames {
			if n == s && (defaults == nil || defaults.Name != s) {
				return fmt.Errorf("이미 존재하는 프로필 이름입니다: %s", s)
			}
		}
		return nil
	}

	emailValidate := func(s string) error {
		if !strings.Contains(s, "@") {
			return fmt.Errorf("올바른 이메일 형식이 아닙니다")
		}
		return nil
	}

	fields := []huh.Field{
		huh.NewInput().Title("프로필 이름").Value(&input.Name).Validate(nameValidate),
		huh.NewInput().Title("git user.name").Value(&input.GitName).Validate(huh.ValidateNotEmpty()),
		huh.NewInput().Title("git user.email").Value(&input.GitEmail).Validate(emailValidate),
	}

	form := huh.NewForm(huh.NewGroup(fields...))
	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("setup.RunProfileForm: %w", err)
	}

	return input, nil
}

// RunActionSelect는 작업 선택 UI를 표시한다.
func (h *HuhFormRunner) RunActionSelect(profileNames []string) (Action, error) {
	var action Action
	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[Action]().
			Title("작업을 선택하세요").
			Options(
				huh.NewOption("프로필 추가", ActionAdd),
				huh.NewOption("프로필 수정", ActionEdit),
				huh.NewOption("프로필 삭제", ActionDelete),
			).
			Value(&action),
	))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("setup.RunActionSelect: %w", err)
	}
	return action, nil
}

// RunProfileSelect는 프로필 선택 UI를 표시한다.
func (h *HuhFormRunner) RunProfileSelect(profileNames []string) (string, error) {
	var selected string
	options := make([]huh.Option[string], len(profileNames))
	for i, name := range profileNames {
		options[i] = huh.NewOption(name, name)
	}

	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("프로필을 선택하세요").
			Options(options...).
			Value(&selected),
	))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("setup.RunProfileSelect: %w", err)
	}
	return selected, nil
}

// RunConfirm은 확인 프롬프트를 표시한다.
func (h *HuhFormRunner) RunConfirm(message string) (bool, error) {
	var confirm bool
	form := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(message).Value(&confirm),
	))
	if err := form.Run(); err != nil {
		return false, fmt.Errorf("setup.RunConfirm: %w", err)
	}
	return confirm, nil
}

// RunAddMore는 "프로필을 더 추가하시겠습니까?" 프롬프트를 표시한다.
func (h *HuhFormRunner) RunAddMore() (bool, error) {
	return h.RunConfirm("프로필을 더 추가하시겠습니까?")
}

// RunSSHHostSelect는 SSH host 선택 UI를 표시한다.
func (h *HuhFormRunner) RunSSHHostSelect(hosts []string) (string, error) {
	if len(hosts) == 0 {
		var host string
		form := huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("SSH host alias").
				Description("~/.ssh/config의 Host 값 (예: github.com-work)").
				Value(&host).
				Validate(huh.ValidateNotEmpty()),
		))
		if err := form.Run(); err != nil {
			return "", fmt.Errorf("setup.RunSSHHostSelect: %w", err)
		}
		return host, nil
	}

	var selected string
	options := make([]huh.Option[string], 0, len(hosts)+1)
	for _, h := range hosts {
		options = append(options, huh.NewOption(h, h))
	}
	options = append(options, huh.NewOption("직접 입력...", "__manual__"))

	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("SSH host를 선택하세요").
			Options(options...).
			Value(&selected),
	))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("setup.RunSSHHostSelect: %w", err)
	}

	if selected == "__manual__" {
		return h.RunSSHHostSelect(nil)
	}
	return selected, nil
}

// RunOwnersSelect는 owners 선택 UI를 표시한다.
func (h *HuhFormRunner) RunOwnersSelect(detected []string) ([]string, error) {
	if len(detected) == 0 {
		var input string
		form := huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("owners (콤마 구분)").
				Description("GitHub 조직명 또는 사용자명").
				Value(&input).
				Validate(huh.ValidateNotEmpty()),
		))
		if err := form.Run(); err != nil {
			return nil, fmt.Errorf("setup.RunOwnersSelect: %w", err)
		}
		var owners []string
		for _, o := range strings.Split(input, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				owners = append(owners, o)
			}
		}
		return owners, nil
	}

	selected := make([]string, len(detected))
	copy(selected, detected)

	form := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("이 계정으로 접근 가능한 조직/사용자").
			Options(func() []huh.Option[string] {
				opts := make([]huh.Option[string], len(detected))
				for i, d := range detected {
					opts[i] = huh.NewOption(d, d).Selected(true)
				}
				return opts
			}()...).
			Value(&selected),
	))
	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("setup.RunOwnersSelect: %w", err)
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("최소 1개 이상 선택해야 합니다")
	}
	return selected, nil
}
```

**Step 2: 빌드 확인**

Run: `go build ./internal/setup/...`
Expected: 성공

**Step 3: Commit**

```bash
git add internal/setup/form.go
git commit -m "feat(setup): implement HuhFormRunner with charmbracelet/huh"
```

---

### Task 6: Runner — 첫 실행 플로우

**Files:**
- Create: `internal/setup/runner.go`
- Create: `internal/setup/runner_test.go`

**Step 1: mock FormRunner 작성 (테스트 헬퍼)**

`internal/setup/runner_test.go` 상단에 mock 정의:

```go
package setup

import (
	"context"
	"testing"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFormRunner는 테스트용 FormRunner다.
type mockFormRunner struct {
	profileInputs []*ProfileInput // RunProfileForm에서 순서대로 반환
	profileIdx    int
	action        Action
	selectedProfile string
	confirms      []bool
	confirmIdx    int
	addMore       []bool
	addMoreIdx    int
	sshHost       string
	owners        []string
}

func (m *mockFormRunner) RunProfileForm(defaults *ProfileInput, existingNames []string) (*ProfileInput, error) {
	if m.profileIdx >= len(m.profileInputs) {
		return nil, fmt.Errorf("no more profile inputs")
	}
	p := m.profileInputs[m.profileIdx]
	m.profileIdx++
	return p, nil
}

func (m *mockFormRunner) RunActionSelect(profileNames []string) (Action, error) {
	return m.action, nil
}

func (m *mockFormRunner) RunProfileSelect(profileNames []string) (string, error) {
	return m.selectedProfile, nil
}

func (m *mockFormRunner) RunConfirm(message string) (bool, error) {
	if m.confirmIdx >= len(m.confirms) {
		return false, nil
	}
	c := m.confirms[m.confirmIdx]
	m.confirmIdx++
	return c, nil
}

func (m *mockFormRunner) RunAddMore() (bool, error) {
	if m.addMoreIdx >= len(m.addMore) {
		return false, nil
	}
	a := m.addMore[m.addMoreIdx]
	m.addMoreIdx++
	return a, nil
}

func (m *mockFormRunner) RunSSHHostSelect(hosts []string) (string, error) {
	return m.sshHost, nil
}

func (m *mockFormRunner) RunOwnersSelect(detected []string) ([]string, error) {
	return m.owners, nil
}
```

**Step 2: 첫 실행 테스트 작성**

```go
func TestRunner_FirstRun_SingleProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/config.toml"

	fc := testutil.NewFakeCommander()
	// gh auth login 성공
	fc.Register("gh auth login --hostname github.com", testutil.Response{Output: []byte("ok")})
	// gh api 조직 조회
	fc.Register("gh api user/orgs --jq .[].login", testutil.Response{Output: []byte("my-org\n")})
	fc.Register("gh api user --jq .login", testutil.Response{Output: []byte("myuser\n")})

	mock := &mockFormRunner{
		profileInputs: []*ProfileInput{{
			Name: "work", GitName: "Test", GitEmail: "test@work.com",
		}},
		sshHost: "github.com-work",
		owners:  []string{"my-org", "myuser"},
		addMore: []bool{false},
	}

	r := &Runner{
		CfgPath:    cfgPath,
		Commander:  fc,
		FormRunner: mock,
	}

	err := r.Run(context.Background())
	require.NoError(t, err)

	// config가 올바르게 저장되었는지 확인
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 1)
	assert.Equal(t, "test@work.com", cfg.Profiles["work"].GitEmail)
	assert.Equal(t, "github.com-work", cfg.Profiles["work"].SSHHost)
	assert.Equal(t, []string{"my-org", "myuser"}, cfg.Profiles["work"].Owners)
}
```

**Step 3: 테스트 실패 확인**

Run: `go test ./internal/setup/ -run TestRunner_FirstRun -v`
Expected: FAIL

**Step 4: Runner 첫 실행 구현**

`internal/setup/runner.go`:

```go
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
	CfgPath      string
	Commander    cmdexec.Commander
	FormRunner   FormRunner
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
```

**Step 5: 테스트 통과 확인**

Run: `go test ./internal/setup/ -run TestRunner_FirstRun -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/setup/runner.go internal/setup/runner_test.go
git commit -m "feat(setup): implement first-run setup flow with profile collection"
```

---

### Task 7: Runner — 재실행 CRUD 플로우

**Files:**
- Modify: `internal/setup/runner.go`
- Modify: `internal/setup/runner_test.go`

**Step 1: CRUD 테스트 작성**

`internal/setup/runner_test.go`에 추가:

```go
func TestRunner_Existing_AddProfile(t *testing.T) {
	cfgPath := testutil.SetupTestProfiles(t)

	fc := testutil.NewFakeCommander()
	fc.Register("gh auth login --hostname github.com", testutil.Response{Output: []byte("ok")})
	fc.Register("gh api user/orgs --jq .[].login", testutil.Response{Output: []byte("new-org\n")})
	fc.Register("gh api user --jq .login", testutil.Response{Output: []byte("newuser\n")})

	mock := &mockFormRunner{
		action: ActionAdd,
		profileInputs: []*ProfileInput{{
			Name: "freelance", GitName: "Freelance", GitEmail: "free@example.com",
		}},
		sshHost: "github.com-freelance",
		owners:  []string{"new-org"},
		addMore: []bool{false},
	}

	r := &Runner{CfgPath: cfgPath, Commander: fc, FormRunner: mock}
	err := r.Run(context.Background())
	require.NoError(t, err)

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 3) // work + personal + freelance
	assert.Equal(t, "free@example.com", cfg.Profiles["freelance"].GitEmail)
}

func TestRunner_Existing_EditProfile(t *testing.T) {
	cfgPath := testutil.SetupTestProfiles(t)

	fc := testutil.NewFakeCommander()

	mock := &mockFormRunner{
		action:          ActionEdit,
		selectedProfile: "work",
		profileInputs: []*ProfileInput{{
			Name: "work", GitName: "New Name", GitEmail: "new@company.com",
			SSHHost: "github-company", Owners: []string{"company-org"},
		}},
	}

	r := &Runner{CfgPath: cfgPath, Commander: fc, FormRunner: mock}
	err := r.Run(context.Background())
	require.NoError(t, err)

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "new@company.com", cfg.Profiles["work"].GitEmail)
	assert.Equal(t, "New Name", cfg.Profiles["work"].GitName)
}

func TestRunner_Existing_DeleteProfile(t *testing.T) {
	cfgPath := testutil.SetupTestProfiles(t)

	fc := testutil.NewFakeCommander()

	mock := &mockFormRunner{
		action:          ActionDelete,
		selectedProfile: "work",
		confirms:        []bool{true},
	}

	r := &Runner{CfgPath: cfgPath, Commander: fc, FormRunner: mock}
	err := r.Run(context.Background())
	require.NoError(t, err)

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 1) // personal만 남음
	_, exists := cfg.Profiles["work"]
	assert.False(t, exists)
}

func TestRunner_Existing_DeleteLastProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/config.toml"
	boolTrue := true
	cfg := &config.Config{
		Version: 1, RequirePushGuard: &boolTrue, CacheTTLDays: 90,
		Profiles: map[string]config.Profile{
			"only": {
				GHConfigDir: "/tmp/gh-only", SSHHost: "github.com-only",
				GitName: "Only", GitEmail: "only@test.com", Owners: []string{"only-org"},
			},
		},
	}
	require.NoError(t, config.Save(cfgPath, cfg))

	fc := testutil.NewFakeCommander()
	mock := &mockFormRunner{
		action:          ActionDelete,
		selectedProfile: "only",
		confirms:        []bool{true},
	}

	r := &Runner{CfgPath: cfgPath, Commander: fc, FormRunner: mock}
	err := r.Run(context.Background())
	assert.Error(t, err) // 마지막 프로필 삭제 불가
}
```

**Step 2: 테스트 실패 확인**

Run: `go test ./internal/setup/ -run TestRunner_Existing -v`
Expected: FAIL

**Step 3: runExisting 구현**

`internal/setup/runner.go`에서 `runExisting` 교체:

```go
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
	return config.Save(r.CfgPath, cfg)
}

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
	return nil
}

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
```

**Step 4: 테스트 통과 확인**

Run: `go test ./internal/setup/ -run TestRunner -v`
Expected: 전체 PASS

**Step 5: 전체 테스트 확인**

Run: `go test ./...`
Expected: 전체 PASS

**Step 6: Commit**

```bash
git add internal/setup/runner.go internal/setup/runner_test.go
git commit -m "feat(setup): implement CRUD flow for existing config"
```

---

### Task 8: 셸 hook 자동 설치

**Files:**
- Create: `internal/setup/shell_hook.go`
- Create: `internal/setup/shell_hook_test.go`
- Modify: `internal/setup/runner.go` (첫 실행에 hook 설치 연결)

**Step 1: 셸 감지 테스트 작성**

`internal/setup/shell_hook_test.go`:

```go
package setup

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectShell_Zsh(t *testing.T) {
	os.Setenv("SHELL", "/bin/zsh")
	defer os.Unsetenv("SHELL")
	assert.Equal(t, "zsh", DetectShell())
}

func TestDetectShell_Bash(t *testing.T) {
	os.Setenv("SHELL", "/usr/bin/bash")
	defer os.Unsetenv("SHELL")
	assert.Equal(t, "bash", DetectShell())
}

func TestDetectShell_Fish(t *testing.T) {
	os.Setenv("SHELL", "/usr/local/bin/fish")
	defer os.Unsetenv("SHELL")
	assert.Equal(t, "fish", DetectShell())
}

func TestInstallShellHook_Zsh(t *testing.T) {
	dir := t.TempDir()
	rcPath := dir + "/.zshrc"

	err := InstallShellHook("zsh", rcPath)
	assert.NoError(t, err)

	content, _ := os.ReadFile(rcPath)
	assert.Contains(t, string(content), "ctx shell integration")
	assert.Contains(t, string(content), "ctx activate")
}

func TestInstallShellHook_AlreadyInstalled(t *testing.T) {
	dir := t.TempDir()
	rcPath := dir + "/.zshrc"
	os.WriteFile(rcPath, []byte("# ctx shell integration (zsh)\nexisting"), 0600)

	err := InstallShellHook("zsh", rcPath)
	assert.NoError(t, err) // 이미 설치됨 — 중복 설치 안 함
}
```

**Step 2: 구현**

`internal/setup/shell_hook.go`:

```go
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
	base := filepath.Base(sh)
	switch base {
	case "zsh":
		return "zsh"
	case "bash":
		return "bash"
	case "fish":
		return "fish"
	default:
		return base
	}
}

// ShellRCPath는 셸별 RC 파일 경로를 반환한다.
func ShellRCPath(shellType string) string {
	home, _ := os.UserHomeDir()
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

	existing, _ := os.ReadFile(rcPath)
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
```

**Step 3: 테스트 통과 확인**

Run: `go test ./internal/setup/ -run TestDetectShell -v && go test ./internal/setup/ -run TestInstallShellHook -v`
Expected: PASS

**Step 4: Runner에 hook 설치 연결**

`runner.go`의 `runFirstTime`에서 config 저장 후 추가:

```go
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
```

**Step 5: 전체 테스트 확인**

Run: `go test ./...`
Expected: 전체 PASS

**Step 6: Commit**

```bash
git add internal/setup/shell_hook.go internal/setup/shell_hook_test.go internal/setup/runner.go
git commit -m "feat(setup): shell detection and hook auto-installation"
```

---

### Task 9: CLI 연결 및 --force 플래그

**Files:**
- Modify: `internal/cli/setup.go`
- Modify: `internal/cli/root.go` (필요시)

**Step 1: setup.go 리팩토링**

`internal/cli/setup.go`를 교체:

```go
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/hbjs97/ctx/internal/setup"
	"github.com/spf13/cobra"
)

func (a *App) newSetupCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "ctx 초기 설정을 시작한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runSetup(cmd.Context(), force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "기존 설정을 무시하고 재설정")
	return cmd
}

// runSetup는 interactive setup wizard를 실행한다.
func (a *App) runSetup(ctx context.Context, force bool) error {
	if force {
		if err := os.Remove(a.CfgPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("cli.setup: 기존 설정 파일 제거 실패: %w", err)
		}
	}

	r := &setup.Runner{
		CfgPath:    a.CfgPath,
		Commander:  a.Commander,
		FormRunner: &setup.HuhFormRunner{},
	}

	return r.Run(ctx)
}
```

**Step 2: 기존 setup 테스트 업데이트**

기존 `internal/cli/setup_test.go`를 확인하고, 변경된 시그니처에 맞게 업데이트한다. setup 명령의 cobra 등록이 올바른지 확인.

**Step 3: 빌드 및 전체 테스트**

Run: `go build ./... && go test ./...`
Expected: 전체 PASS

**Step 4: 바이너리로 수동 확인**

Run: `./bin/ctx setup --help`
Expected: `--force` 플래그 표시

**Step 5: Commit**

```bash
git add internal/cli/setup.go
git commit -m "feat(cli): wire interactive setup with --force flag"
```

---

### Task 10: doctor 자동 실행 연결

**Files:**
- Modify: `internal/setup/runner.go`

**Step 1: 첫 실행과 재실행 완료 후 doctor 호출 추가**

Runner에 doctor 실행 로직 추가. config의 모든 프로필에 대해 `doctor.RunAll`을 호출하고 결과를 출력.

```go
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
```

`runFirstTime`과 `runExisting`의 각 작업 완료 후 `r.runDoctor(ctx, cfg)` 호출.

**Step 2: 테스트에 doctor 명령 mock 추가**

기존 테스트의 FakeCommander에 git/gh/ssh 바이너리 체크 응답을 등록하여 doctor가 패닉하지 않도록 한다.

**Step 3: 전체 테스트 확인**

Run: `go test ./...`
Expected: 전체 PASS

**Step 4: Commit**

```bash
git add internal/setup/runner.go internal/setup/runner_test.go
git commit -m "feat(setup): auto-run doctor after setup completion"
```

---

### Task 11: 최종 검증 및 정리

**Step 1: 전체 빌드 + vet + 테스트**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: 전체 PASS

**Step 2: 커버리지 확인**

Run: `go test -coverprofile=coverage.out ./internal/setup/ && go tool cover -func=coverage.out`
Expected: setup 패키지 80%+

**Step 3: README 업데이트**

`ctx setup`의 설명을 interactive wizard로 업데이트:

```markdown
### 1. 초기 설정

\```bash
ctx setup
\```

대화형 설정 마법사가 실행된다:
- 프로필 이름, git 사용자 정보 입력
- gh 인증 자동 실행
- SSH host 자동 감지
- 소속 조직 자동 조회
- 셸 hook 자동 설치

재실행하면 프로필 추가/수정/삭제가 가능하다.
```

**Step 4: Commit**

```bash
git add -A
git commit -m "docs: update README for interactive setup"
```
