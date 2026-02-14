# Expert Review Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all BLOCKER/HIGH/MEDIUM issues identified by the 4-expert review to bring the project to production-ready quality.

**Architecture:** Fixes follow dependency order: Commander interface first (foundation), then adapters that use it, then CLI layer that wires everything. Sentinel errors replace fragile string matching. CLI gets dependency injection for testability.

**Tech Stack:** Go 1.26, cobra, testify, BurntSushi/toml

---

### Task 1: Commander RunWithEnv + FakeCommander support

**Why:** BLOCKER — `gh.ProbeRepo` needs to set `GH_CONFIG_DIR` env var when running `gh` commands. Current Commander interface has no env var support. This is the #1 critical bug: multi-account feature is completely broken without it.

**Files:**
- Modify: `internal/cmdexec/cmdexec.go`
- Modify: `internal/testutil/exec.go`
- Modify: `internal/testutil/exec_test.go`

**Step 1: Write failing test for RunWithEnv in FakeCommander**

Add to `internal/testutil/exec_test.go`:

```go
func TestFakeCommander_RunWithEnv(t *testing.T) {
	t.Parallel()
	fc := testutil.NewFakeCommander()
	fc.Register("gh api repos/owner/repo", `{"permissions":{"push":true}}`, nil)

	env := map[string]string{"GH_CONFIG_DIR": "/home/user/.config/gh-work"}
	out, err := fc.RunWithEnv(context.Background(), env, "gh", "api", "repos/owner/repo")

	assert.NoError(t, err)
	assert.Contains(t, string(out), "push")
	assert.True(t, fc.Called("gh api"))
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/testutil/ -run TestFakeCommander_RunWithEnv -v`
Expected: FAIL — `RunWithEnv` method does not exist

**Step 3: Add RunWithEnv to Commander interface and implementations**

In `internal/cmdexec/cmdexec.go`, add `RunWithEnv` to the interface and implement in `RealCommander`:

```go
package cmdexec

import (
	"context"
	"os/exec"
)

// Commander abstracts external command execution.
type Commander interface {
	// Run executes an external command and returns its combined output.
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	// RunWithEnv executes an external command with additional environment variables.
	RunWithEnv(ctx context.Context, env map[string]string, name string, args ...string) ([]byte, error)
}

// RealCommander executes actual external commands via os/exec.
type RealCommander struct{}

// Run executes the command using os/exec.CommandContext.
func (c *RealCommander) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// RunWithEnv executes the command with additional environment variables.
func (c *RealCommander) RunWithEnv(ctx context.Context, env map[string]string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), mapToEnvSlice(env)...)
	return cmd.CombinedOutput()
}

func mapToEnvSlice(env map[string]string) []string {
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}
```

Add `"os"` to imports.

In `internal/testutil/exec.go`, add `RunWithEnv` to FakeCommander:

```go
// RunWithEnv delegates to Run (env is recorded but not used in fake).
func (c *FakeCommander) RunWithEnv(_ context.Context, env map[string]string, name string, args ...string) ([]byte, error) {
	c.EnvCalls = append(c.EnvCalls, env)
	return c.Run(context.Background(), name, args...)
}
```

Add `EnvCalls []map[string]string` field to `FakeCommander` struct.

**Step 4: Run tests**

Run: `go test ./internal/testutil/ -v && go test ./internal/cmdexec/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cmdexec/cmdexec.go internal/testutil/exec.go internal/testutil/exec_test.go
git commit -m "feat(cmdexec): add RunWithEnv for environment variable injection"
```

---

### Task 2: Fix gh.ProbeRepo to use GH_CONFIG_DIR

**Why:** BLOCKER — ProbeRepo accepts `ghConfigDir` parameter but never uses it. The core multi-account feature is broken.

**Files:**
- Modify: `internal/gh/gh.go`
- Modify: `internal/gh/example_test.go`

**Step 1: Write failing test that verifies GH_CONFIG_DIR is passed**

Add to `internal/gh/example_test.go`:

```go
func TestProbeRepo_UsesGHConfigDir(t *testing.T) {
	t.Parallel()
	fc := testutil.NewFakeCommander()
	fc.Register("gh api repos/owner/repo", `{"permissions":{"push":true}}`, nil)

	adapter := gh.NewAdapter(fc)
	_, err := adapter.ProbeRepo(context.Background(), "/home/user/.config/gh-work", "owner", "repo")

	assert.NoError(t, err)
	// Verify RunWithEnv was called (not Run)
	assert.Len(t, fc.EnvCalls, 1, "should call RunWithEnv exactly once")
	assert.Equal(t, "/home/user/.config/gh-work", fc.EnvCalls[0]["GH_CONFIG_DIR"])
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/gh/ -run TestProbeRepo_UsesGHConfigDir -v`
Expected: FAIL — EnvCalls is empty because ProbeRepo uses `Run` not `RunWithEnv`

**Step 3: Fix ProbeRepo to use RunWithEnv**

In `internal/gh/gh.go`, modify `ProbeRepo`:

```go
// ProbeRepo는 특정 프로필로 리포의 접근 권한을 확인한다.
func (a *Adapter) ProbeRepo(ctx context.Context, ghConfigDir, owner, repo string) (*ProbeResult, error) {
	env := map[string]string{"GH_CONFIG_DIR": ghConfigDir}
	out, err := a.cmd.RunWithEnv(ctx, env, "gh", "api",
		fmt.Sprintf("repos/%s/%s", owner, repo),
		"--hostname", "github.com")

	if err != nil {
		errStr := string(out) + err.Error()
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
```

Also fix `DetectEnvTokenInterference` to also unset tokens during probe. Add a helper:

```go
// SuppressEnvTokens는 probe 실행 시 GH_TOKEN/GITHUB_TOKEN을 임시 제거하는 env map을 반환한다.
func SuppressEnvTokens() map[string]string {
	suppress := make(map[string]string)
	for _, key := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if os.Getenv(key) != "" {
			suppress[key] = "" // empty value to override
		}
	}
	return suppress
}
```

Update `ProbeRepo` to merge suppress tokens into env:

```go
env := map[string]string{"GH_CONFIG_DIR": ghConfigDir}
// Suppress env tokens that would override profile auth
for k, v := range SuppressEnvTokens() {
	env[k] = v
}
```

**Step 4: Run all gh tests**

Run: `go test ./internal/gh/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/gh/gh.go internal/gh/example_test.go
git commit -m "fix(gh): inject GH_CONFIG_DIR via RunWithEnv in ProbeRepo"
```

---

### Task 3: Sentinel errors + MapExitCode refactor

**Why:** HIGH — Current MapExitCode uses fragile Korean string matching. Sentinel errors are type-safe and testable.

**Files:**
- Create: `internal/cli/errors.go`
- Modify: `internal/cli/exitcode.go`
- Modify: `internal/cli/example_test.go`
- Modify: `internal/resolver/resolver.go`
- Modify: `internal/guard/guard.go`

**Step 1: Write failing test for sentinel error matching**

Add to `internal/cli/example_test.go`:

```go
func TestMapExitCode_SentinelErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want cli.ExitCode
	}{
		{"nil", nil, cli.ExitSuccess},
		{"guard block", cli.ErrGuardBlock, cli.ExitGuardBlock},
		{"ambiguous", cli.ErrAmbiguous, cli.ExitAmbiguous},
		{"auth fail", cli.ErrAuthFail, cli.ExitAuthFail},
		{"config error", cli.ErrConfig, cli.ExitConfigError},
		{"no dependency", cli.ErrNoDependency, cli.ExitNoDependency},
		{"wrapped guard", fmt.Errorf("wrap: %w", cli.ErrGuardBlock), cli.ExitGuardBlock},
		{"wrapped config", fmt.Errorf("wrap: %w", cli.ErrConfig), cli.ExitConfigError},
		{"general", fmt.Errorf("unknown"), cli.ExitGeneral},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cli.MapExitCode(tt.err))
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestMapExitCode_SentinelErrors -v`
Expected: FAIL — sentinel errors not defined

**Step 3: Create sentinel errors and refactor MapExitCode**

Create `internal/cli/errors.go`:

```go
package cli

import "errors"

// Sentinel errors for exit code mapping.
var (
	// ErrGuardBlock indicates guard blocked the push.
	ErrGuardBlock = errors.New("guard 검사 실패 — push 차단")
	// ErrAmbiguous indicates ambiguous profile resolution.
	ErrAmbiguous = errors.New("모호한 판정, --profile 플래그 필요")
	// ErrAuthFail indicates no accessible profile found.
	ErrAuthFail = errors.New("접근 가능한 프로필 없음")
	// ErrConfig indicates a configuration error.
	ErrConfig = errors.New("설정 오류")
	// ErrNoDependency indicates missing required tools.
	ErrNoDependency = errors.New("필수 도구 미설치")
)
```

Modify `internal/cli/exitcode.go` — replace string matching with `errors.Is`:

```go
package cli

import "errors"

// ExitCode는 ctx의 종료 코드다. TECH_SPEC 참조.
type ExitCode int

const (
	ExitSuccess      ExitCode = 0
	ExitGeneral      ExitCode = 1
	ExitGuardBlock   ExitCode = 2
	ExitAmbiguous    ExitCode = 3
	ExitAuthFail     ExitCode = 4
	ExitConfigError  ExitCode = 5
	ExitNoDependency ExitCode = 6
)

// MapExitCode는 에러를 기반으로 적절한 종료 코드를 반환한다.
func MapExitCode(err error) ExitCode {
	if err == nil {
		return ExitSuccess
	}
	switch {
	case errors.Is(err, ErrGuardBlock):
		return ExitGuardBlock
	case errors.Is(err, ErrAmbiguous):
		return ExitAmbiguous
	case errors.Is(err, ErrAuthFail):
		return ExitAuthFail
	case errors.Is(err, ErrConfig):
		return ExitConfigError
	case errors.Is(err, ErrNoDependency):
		return ExitNoDependency
	default:
		return ExitGeneral
	}
}
```

Update `internal/resolver/resolver.go` to use sentinel errors:

```go
// At step 4 — no pushable profiles:
return nil, fmt.Errorf("resolver.Resolve: %w", cli.ErrAuthFail)

// At step 5 — ambiguous:
return nil, fmt.Errorf("resolver.Resolve: %w", cli.ErrAmbiguous)
```

Wait — this would create a circular dependency (resolver -> cli). Instead, define the sentinel errors in a shared location. The cleanest approach: keep sentinel errors in `cli` package but have resolver return plain errors, then have the CLI layer wrap them. Actually, the simplest approach is to define domain errors that both packages can use.

**Better approach:** Define errors where they originate:
- `ErrGuardBlock` in `guard` package
- `ErrAmbiguous`, `ErrAuthFail` in `resolver` package
- `ErrConfig` in `config` package
- `ErrNoDependency` in `doctor` package

Then `cli/exitcode.go` imports them and maps them.

Create error vars in each package:

In `internal/resolver/resolver.go`:
```go
var (
	ErrAmbiguous = errors.New("모호한 판정, --profile 플래그 필요")
	ErrAuthFail  = errors.New("접근 가능한 프로필 없음")
)
```

In `internal/guard/guard.go`:
```go
var ErrGuardBlock = errors.New("guard 검사 실패 — push 차단")
```

In `internal/config/config.go`:
```go
var ErrConfig = errors.New("설정 오류")
```

Then `cli/errors.go` re-exports for convenience and adds cli-only errors:
```go
package cli

import (
	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/guard"
	"github.com/hbjs97/ctx/internal/resolver"
)

// Re-export sentinel errors for MapExitCode.
var (
	ErrGuardBlock  = guard.ErrGuardBlock
	ErrAmbiguous   = resolver.ErrAmbiguous
	ErrAuthFail    = resolver.ErrAuthFail
	ErrConfig      = config.ErrConfig
)
```

And `MapExitCode` uses `errors.Is` with the domain package errors.

**Step 4: Update resolver and guard to use sentinel errors**

In `internal/resolver/resolver.go`:
- `return nil, fmt.Errorf("resolver.Resolve: %w", ErrAuthFail)` (was: `fmt.Errorf("resolver.Resolve: 접근 가능한 프로필 없음")`)
- `return nil, fmt.Errorf("resolver.Resolve: %w", ErrAmbiguous)` (was: `fmt.Errorf("resolver.Resolve: 모호한 판정, --profile 플래그 필요")`)

In `internal/guard/guard.go` (in `cli/guard_cmd.go` actually — the error is returned from CLI):
- `return fmt.Errorf("guard: %w", guard.ErrGuardBlock)` (was: `fmt.Errorf("guard 검사 실패 — push 차단")`)

In `internal/config/config.go`, wrap validation errors:
- `return nil, fmt.Errorf("config.Load: %w", ErrConfig)` for validation failures

**Step 5: Run all tests**

Run: `go test ./... -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/cli/errors.go internal/cli/exitcode.go internal/cli/example_test.go internal/resolver/resolver.go internal/guard/guard.go internal/config/config.go
git commit -m "refactor: replace string-based exit code mapping with sentinel errors"
```

---

### Task 4: Security fixes (file permissions, CTX_SKIP_GUARD warning)

**Why:** HIGH — ctx-profile written with 0644, ValidateFilePermissions never called, CTX_SKIP_GUARD bypasses silently.

**Files:**
- Modify: `internal/cli/clone.go:74` (0644 → 0600)
- Modify: `internal/cli/init_cmd.go:85` (0644 → 0600)
- Modify: `internal/guard/guard.go:40-42` (add warning for skip)
- Modify: `internal/config/config.go` (call ValidateFilePermissions in Load)
- Modify: tests for each

**Step 1: Write failing tests**

Test ctx-profile permission in `internal/cli/example_test.go` — this requires CLI DI (Task 6). Instead, test at the unit level.

Test CTX_SKIP_GUARD warning in `internal/guard/example_test.go`:

```go
func TestCheck_SkipGuardWarnsToStderr(t *testing.T) {
	// CTX_SKIP_GUARD=1 should pass but also return a warning violation
	t.Setenv("CTX_SKIP_GUARD", "1")

	profile := &config.Profile{SSHHost: "gh-work", GitName: "user", GitEmail: "user@work.com"}
	fc := testutil.NewFakeCommander()

	result, err := guard.Check(context.Background(), "/tmp/repo", profile, fc)
	assert.NoError(t, err)
	assert.True(t, result.Pass)
	assert.True(t, result.Skipped, "should indicate guard was skipped")
}
```

**Step 2: Implement fixes**

Fix file permissions in `internal/cli/clone.go:74`:
```go
_ = os.WriteFile(profilePath, []byte(result.Profile+"\n"), 0600) // 보안: 0600 권한
```

Fix in `internal/cli/init_cmd.go:85`:
```go
_ = os.WriteFile(profilePath, []byte(result.Profile+"\n"), 0600) // 보안: 0600 권한
```

Fix CTX_SKIP_GUARD in `internal/guard/guard.go`:
```go
// CheckResult에 Skipped 필드 추가
type CheckResult struct {
	Pass       bool
	Skipped    bool
	Violations []Violation
}

func Check(...) (*CheckResult, error) {
	if os.Getenv("CTX_SKIP_GUARD") == "1" {
		fmt.Fprintln(os.Stderr, "경고: CTX_SKIP_GUARD=1 — guard 검사를 건너뜁니다")
		return &CheckResult{Pass: true, Skipped: true}, nil
	}
	// ...
}
```

Wire `ValidateFilePermissions` in `config.Load` (warn, don't fail):
```go
func Load(path string) (*Config, error) {
	if err := ValidateFilePermissions(path); err != nil {
		fmt.Fprintf(os.Stderr, "경고: %v\n", err)
	}
	// ... rest of Load
}
```

**Step 3: Run all tests**

Run: `go test ./internal/guard/ ./internal/cli/ ./internal/config/ -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/cli/clone.go internal/cli/init_cmd.go internal/guard/guard.go internal/config/config.go
git commit -m "fix(security): enforce 0600 permissions, warn on CTX_SKIP_GUARD bypass"
```

---

### Task 5: Guard hook PATH safety check

**Why:** BLOCKER — If `ctx` binary is not in PATH, the hook script will fail silently or error cryptically.

**Files:**
- Modify: `internal/guard/guard.go` (hookScript constant)
- Modify: `internal/guard/example_test.go`

**Step 1: Write failing test**

```go
func TestInstallHook_ContainsPathCheck(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(gitDir, 0755))

	err := guard.InstallHook(dir)
	require.NoError(t, err)

	hookPath := filepath.Join(gitDir, "pre-push")
	data, err := os.ReadFile(hookPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "command -v ctx")
}
```

**Step 2: Fix hookScript to include PATH check**

```go
const hookScript = `# ctx-guard-start
# Installed by ctx — do not edit this block manually.
if ! command -v ctx >/dev/null 2>&1; then
  echo "ctx: command not found — skipping guard check" >&2
  exit 0
fi
ctx guard check || exit 1
# ctx-guard-end`
```

**Step 3: Run tests**

Run: `go test ./internal/guard/ -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/guard/guard.go internal/guard/example_test.go
git commit -m "fix(guard): add PATH safety check in pre-push hook script"
```

---

### Task 6: CLI dependency injection refactor

**Why:** HIGH — All CLI commands hard-code `&cmdexec.RealCommander{}`, making them untestable. Package-level `cfgPath`/`verbose` globals prevent concurrent testing.

**Files:**
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/clone.go`
- Modify: `internal/cli/init_cmd.go`
- Modify: `internal/cli/status.go`
- Modify: `internal/cli/doctor_cmd.go`
- Modify: `internal/cli/guard_cmd.go`
- Modify: `internal/cli/activate.go`
- Modify: `internal/cli/setup.go`

**Step 1: Create App struct to hold dependencies**

Replace package-level globals with an `App` struct in `internal/cli/root.go`:

```go
package cli

import (
	"os"
	"path/filepath"

	"github.com/hbjs97/ctx/internal/cmdexec"
	"github.com/spf13/cobra"
)

// App holds CLI dependencies for testability.
type App struct {
	Commander cmdexec.Commander
	CfgPath   string
	Verbose   bool
}

// NewApp creates an App with default production dependencies.
func NewApp() *App {
	return &App{
		Commander: &cmdexec.RealCommander{},
	}
}

// NewRootCmd는 ctx CLI의 루트 명령을 생성한다.
func (a *App) NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "ctx",
		Short:        "GitHub 멀티계정 컨텍스트 매니저",
		SilenceUsage: true,
	}

	defaultCfg := filepath.Join(homeDir(), ".config", "ctx", "config.toml")
	cmd.PersistentFlags().StringVar(&a.CfgPath, "config", defaultCfg, "설정 파일 경로")
	cmd.PersistentFlags().BoolVar(&a.Verbose, "verbose", false, "상세 출력")

	cmd.AddCommand(
		a.newCloneCmd(),
		a.newInitCmd(),
		a.newStatusCmd(),
		a.newDoctorCmd(),
		a.newGuardCmd(),
		a.newActivateCmd(),
		a.newSetupCmd(),
	)
	return cmd
}
```

Also keep backward-compatible `NewRootCmd()` free function:
```go
// NewRootCmd는 기본 의존성으로 루트 명령을 생성한다.
func NewRootCmd() *cobra.Command {
	return NewApp().NewRootCmd()
}
```

**Step 2: Convert all sub-commands to methods on App**

Each file changes `func newXxxCmd()` → `func (a *App) newXxxCmd()` and replaces `&cmdexec.RealCommander{}` with `a.Commander`, `cfgPath` with `a.CfgPath`.

Example for clone.go:
```go
func (a *App) newCloneCmd() *cobra.Command { ... }
func (a *App) runClone(ctx context.Context, target, profileFlag string, noGuard bool) error {
	// ...
	commander := a.Commander  // was: &cmdexec.RealCommander{}
	cfg, err := config.Load(a.CfgPath)  // was: cfgPath
	// ...
}
```

Apply same pattern to: init_cmd.go, status.go, doctor_cmd.go, guard_cmd.go, activate.go, setup.go.

**Step 3: Update main.go**

```go
func main() {
	app := cli.NewApp()
	cmd := app.NewRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(int(cli.MapExitCode(err)))
	}
}
```

**Step 4: Run all tests**

Run: `go test ./... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/ cmd/ctx/main.go
git commit -m "refactor(cli): inject Commander via App struct for testability"
```

---

### Task 7: Token masking implementation

**Why:** HIGH — Security rules require masking tokens in `--verbose` output. No implementation exists.

**Files:**
- Create: `internal/cli/masker.go`
- Create: `internal/cli/masker_test.go`

**Step 1: Write failing test**

```go
func TestMaskTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"ghp token", "token: ghp_abc123def456ghi789", "token: ghp_****"},
		{"gho token", "auth gho_secretvalue here", "auth gho_**** here"},
		{"github_pat", "github_pat_abcdef1234567890", "github_pat_****"},
		{"ghs token", "ghs_servertoken123", "ghs_****"},
		{"ghu token", "ghu_usertoken456", "ghu_****"},
		{"no token", "hello world", "hello world"},
		{"multiple tokens", "ghp_aaa and gho_bbb", "ghp_**** and gho_****"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cli.MaskTokens(tt.input))
		})
	}
}
```

**Step 2: Implement MaskTokens**

```go
package cli

import "regexp"

var tokenPattern = regexp.MustCompile(`(ghp_|gho_|github_pat_|ghs_|ghu_)\S+`)

// MaskTokens는 GitHub 토큰 패턴을 마스킹한다.
func MaskTokens(s string) string {
	return tokenPattern.ReplaceAllStringFunc(s, func(match string) string {
		for _, prefix := range []string{"ghp_", "gho_", "github_pat_", "ghs_", "ghu_"} {
			if len(match) >= len(prefix) && match[:len(prefix)] == prefix {
				return prefix + "****"
			}
		}
		return match
	})
}
```

**Step 3: Run test**

Run: `go test ./internal/cli/ -run TestMaskTokens -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/cli/masker.go internal/cli/masker_test.go
git commit -m "feat(cli): implement token masking for verbose output"
```

---

### Task 8: ctx setup implementation

**Why:** BLOCKER — Currently a placeholder that prints "not implemented".

**Files:**
- Modify: `internal/cli/setup.go`
- Add tests in `internal/cli/example_test.go`

**Step 1: Write failing test**

```go
func TestSetupCmd_CreatesConfigFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	app := &cli.App{
		Commander: testutil.NewFakeCommander(),
		CfgPath:   cfgPath,
	}
	cmd := app.NewRootCmd()
	cmd.SetArgs([]string{"setup", "--config", cfgPath})

	// Setup writes a template config
	err := cmd.Execute()
	assert.NoError(t, err)

	// Verify config file exists with correct permissions
	info, err := os.Stat(cfgPath)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Verify it's valid TOML with example profiles
	data, _ := os.ReadFile(cfgPath)
	assert.Contains(t, string(data), "[profiles.")
}
```

**Step 2: Implement setup command**

`internal/cli/setup.go`:

```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const configTemplate = `# ctx configuration file
# See: https://github.com/hbjs97/ctx

version = 1
# default_profile = "work"
# prompt_on_ambiguous = true
# require_push_guard = true
# cache_ttl_days = 90

[profiles.work]
gh_config_dir = "~/.config/gh-work"
ssh_host = "github.com-work"
git_name = "Your Name"
git_email = "you@work.com"
owners = ["your-org"]

[profiles.personal]
gh_config_dir = "~/.config/gh-personal"
ssh_host = "github.com-personal"
git_name = "Your Name"
git_email = "you@personal.com"
owners = ["your-username"]
`

func (a *App) newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "ctx 초기 설정을 시작한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runSetup()
		},
	}
}

func (a *App) runSetup() error {
	cfgPath := a.CfgPath

	// Check if config already exists
	if _, err := os.Stat(cfgPath); err == nil {
		return fmt.Errorf("cli.setup: 설정 파일이 이미 존재합니다: %s", cfgPath)
	}

	// Create directory
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("cli.setup: %w", err)
	}

	// Write template config with 0600 permissions
	if err := os.WriteFile(cfgPath, []byte(configTemplate), 0600); err != nil {
		return fmt.Errorf("cli.setup: %w", err)
	}

	fmt.Printf("설정 파일 생성: %s\n", cfgPath)
	fmt.Println("프로필을 수정한 후 'ctx doctor'로 설정을 확인하세요.")
	return nil
}
```

**Step 3: Run tests**

Run: `go test ./internal/cli/ -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/cli/setup.go internal/cli/example_test.go
git commit -m "feat(cli): implement ctx setup with template config generation"
```

---

### Task 9: CLI test coverage to 80%+

**Why:** HIGH — CLI coverage is 18.9%, well below the 80% target.

**Files:**
- Modify: `internal/cli/example_test.go`

**Step 1: Add comprehensive CLI tests**

Test each command with injected FakeCommander. Example patterns:

```go
func TestCloneCmd_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := testutil.TempConfigFile(t, dir, `
version = 1
[profiles.work]
gh_config_dir = "/tmp/gh-work"
ssh_host = "gh-work"
git_name = "Test User"
git_email = "test@work.com"
owners = ["myorg"]
`)

	fc := testutil.NewFakeCommander()
	fc.Register("gh api repos/myorg/myrepo", `{"permissions":{"push":true}}`, nil)
	fc.Register("git clone", "", nil)
	fc.Register("git -C", "", nil)
	fc.DefaultResponse = &testutil.Response{Output: []byte(""), Err: nil}

	app := &cli.App{Commander: fc, CfgPath: cfgPath}
	cmd := app.NewRootCmd()
	cmd.SetArgs([]string{"clone", "myorg/myrepo"})
	// ... test execution and assertions
}
```

Tests needed (target 80%+):
- `TestCloneCmd_Success` — full clone flow
- `TestCloneCmd_InvalidRepo` — bad repo format
- `TestCloneCmd_NoMatchingProfile` — auth fail
- `TestInitCmd_Success` — init in existing repo
- `TestStatusCmd_NoCtxProfile` — no ctx-profile file
- `TestStatusCmd_WithProfile` — normal status output
- `TestDoctorCmd_Basic` — doctor with config
- `TestGuardCheckCmd_Pass` — guard passes
- `TestGuardCheckCmd_Fail` — guard blocks
- `TestActivateCmd_Hook` — --hook flag
- `TestActivateCmd_WithProfile` — activation
- `TestSetupCmd_AlreadyExists` — config exists error
- `TestMaskTokens_*` — already in Task 7

**Step 2: Run tests with coverage**

Run: `go test ./internal/cli/ -v -coverprofile=cli.cov && go tool cover -func=cli.cov`
Expected: cli coverage >= 80%

**Step 3: Commit**

```bash
git add internal/cli/example_test.go
git commit -m "test(cli): comprehensive command tests, coverage 80%+"
```

---

### Task 10: go mod tidy + final cleanup

**Why:** HIGH — cobra is marked `// indirect` incorrectly. Also rename example_test.go files.

**Files:**
- Modify: `go.mod`
- Rename: test files (example_test.go → *_test.go with conventional names)

**Step 1: Run go mod tidy**

```bash
go mod tidy
```

This should move `cobra` from indirect to direct dependencies.

**Step 2: Verify build and tests**

```bash
go build ./... && go vet ./... && go test ./...
```

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: go mod tidy, fix cobra dependency classification"
```

---

## Summary of Changes

| # | Task | Severity | Impact |
|---|------|----------|--------|
| 1 | Commander RunWithEnv | BLOCKER | Enables env var injection for all adapters |
| 2 | Fix gh.ProbeRepo GH_CONFIG_DIR | BLOCKER | Core multi-account feature works |
| 3 | Sentinel errors | HIGH | Type-safe exit code mapping |
| 4 | Security fixes (perms, skip warning) | HIGH | File permissions enforced |
| 5 | Guard hook PATH check | BLOCKER | Hook fails gracefully |
| 6 | CLI dependency injection | HIGH | CLI becomes testable |
| 7 | Token masking | HIGH | Security compliance |
| 8 | ctx setup implementation | BLOCKER | Setup command works |
| 9 | CLI test coverage 80%+ | HIGH | Quality gate met |
| 10 | go mod tidy + cleanup | HIGH | Correct dependency tree |

**Execution order matters:** Tasks 1→2 (commander first, then gh fix), Task 3 and 5 (independent), Task 6 (CLI refactor), Tasks 7-8 (use new CLI structure), Task 9 (tests use DI), Task 10 (final).
