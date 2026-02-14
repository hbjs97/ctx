# TESTING.md

## 1. 테스팅 전략 개요

ctx 프로젝트는 **3계층 테스트 전략**을 채택한다.

| 계층 | 범위 | 속도 | 외부 의존 | 실행 조건 |
|------|------|------|----------|----------|
| 단위 테스트 | 함수/메서드 단위 | < 1s | 없음 (mock) | 항상 |
| 통합 테스트 | Resolver 5단계 파이프라인 전체 흐름 | < 5s | mock git/gh | 항상 |
| E2E 테스트 | 실제 git repo 생성, clone/init/guard 검증 | < 30s | git (실제) | `go test -tags=e2e` |

커버리지 목표: **80%+** (핵심 로직인 Resolver, Guard는 90%+)

## 2. 단위 테스트

### 2.1 원칙

- 각 `internal/` 패키지에 `*_test.go` 파일 배치
- **Table-driven tests** 패턴을 기본으로 사용
- `github.com/stretchr/testify/assert` 활용
- 외부 명령 실행은 `Commander` interface로 추상화하여 mock

### 2.2 패키지별 테스트 대상

| 패키지 | 핵심 테스트 대상 | 파일 |
|--------|-----------------|------|
| `config` | TOML 파싱, 유효성 검증, 기본값 적용, 파일 권한 검사 | `config_test.go` |
| `profile` | 프로필 로드, owners 매칭, config_hash 생성 | `profile_test.go` |
| `resolver` | 5단계 판정 각 단계별 로직, 파이프라인 전이 | `resolver_test.go` |
| `cache` | 캐시 읽기/쓰기, TTL 검증, config_hash 무효화 | `cache_test.go` |
| `git` | URL 파싱 (SSH/HTTPS/shorthand), remote 조작, config 설정 | `git_test.go` |
| `gh` | API 응답 파싱, 권한 판정, rate limit 감지, env 간섭 감지 | `gh_test.go` |
| `guard` | 컨텍스트 불일치 검사, hook 설치/제거/체이닝 | `guard_test.go` |
| `doctor` | 각 점검 항목별 OK/WARN/FAIL 판정 | `doctor_test.go` |
| `cli` | 명령 파싱, exit code 매핑 | `cli_test.go` |

### 2.3 Table-Driven Test 예시

```go
func TestParseRepoURL(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    RepoRef
        wantErr bool
    }{
        {
            name:  "ssh url",
            input: "git@github-company:company-org/api-server.git",
            want:  RepoRef{Owner: "company-org", Repo: "api-server"},
        },
        {
            name:  "https url",
            input: "https://github.com/hbjs97/dotfiles.git",
            want:  RepoRef{Owner: "hbjs97", Repo: "dotfiles"},
        },
        {
            name:  "shorthand",
            input: "company-org/api-server",
            want:  RepoRef{Owner: "company-org", Repo: "api-server"},
        },
        {
            name:    "invalid",
            input:   "not-a-url",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseRepoURL(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## 3. 통합 테스트

### 3.1 범위

Resolver 5단계 판정 파이프라인의 **전체 흐름**을 검증한다.

```
명시 플래그 -> 캐시 조회 -> Owner 규칙 -> 권한 Probe -> 사용자 선택
```

### 3.2 Mock 전략

- `Commander` interface를 통해 `git`, `gh`, `ssh` 명령을 mock
- `FakeCommander`에 미리 정의한 응답을 등록
- 파일 시스템은 실제 임시 디렉토리 사용 (`t.TempDir()`)

### 3.3 테스트 시나리오

| # | 시나리오 | 기대 결과 |
|---|---------|----------|
| 1 | `--profile work` 명시 | Step 1에서 즉시 확정, 이후 단계 미실행 |
| 2 | 캐시에 `company-org/api-server` -> `work` 존재, TTL 유효 | Step 2에서 확정 |
| 3 | 캐시 TTL 초과 | Step 2 실패, Step 3으로 전이 |
| 4 | 캐시 config_hash 불일치 | Step 2 실패, Step 3으로 전이 |
| 5 | owner `company-org`가 work 프로필에만 매칭 | Step 3에서 확정 |
| 6 | owner가 어느 프로필에도 매칭되지 않음 | Step 3 실패, Step 4로 전이 |
| 7 | owner가 2개 프로필에 매칭 | Step 3 실패, Step 4로 전이 |
| 8 | push 가능 프로필 1개 (probe) | Step 4에서 확정 |
| 9 | push 가능 프로필 0개 | exit code 4 에러 |
| 10 | push 가능 프로필 2개 이상, 대화형 | Step 5 사용자 선택 |
| 11 | push 가능 프로필 2개 이상, 비대화형 | exit code 3 에러 |
| 12 | 미존재 프로필 명시 (`--profile nonexist`) | exit code 5 에러 |

### 3.4 Guard 통합 테스트

| # | 시나리오 | 기대 결과 |
|---|---------|----------|
| 1 | 프로필/remote/email 모두 일치 | guard pass (exit 0) |
| 2 | remote SSH host 불일치 | guard 차단 (exit 2) |
| 3 | git user.email 불일치 | guard 차단 (exit 2) |
| 4 | `CTX_SKIP_GUARD=1` 설정 | guard 우회 + 경고 출력 |
| 5 | `.git/ctx-profile` 누락 | 에러 메시지 출력 |

## 4. E2E 테스트

### 4.1 빌드 태그

E2E 테스트는 실제 `git` 바이너리를 사용하므로 빌드 태그로 분리한다.

```go
//go:build e2e

package e2e_test
```

실행: `go test -tags=e2e ./test/e2e/...`

### 4.2 테스트 시나리오

| # | 시나리오 | 검증 항목 |
|---|---------|----------|
| 1 | `ctx clone owner/repo` | git repo 생성됨, `.git/ctx-profile` 존재, `user.name/email` 설정됨 |
| 2 | `ctx init` (기존 리포) | `.git/ctx-profile` 생성됨, pre-push hook 설치됨 |
| 3 | pre-push guard 차단 | 불일치 상태에서 push 시도 시 exit code 2 |
| 4 | pre-push guard 통과 | 일치 상태에서 push 시도 시 exit code 0 |
| 5 | `ctx status` 출력 | 프로필, remote, identity 정보 올바르게 출력 |
| 6 | `ctx doctor` 점검 | 각 점검 항목 상태 출력 |

### 4.3 E2E 환경 설정

- `TempGitRepo()` 헬퍼로 bare repo + working repo 쌍 생성
- 실제 `git init`, `git remote add` 사용
- `gh` 호출은 PATH에 fake 스크립트 삽입하여 mock
- 테스트 완료 후 `t.Cleanup()`으로 자동 정리

## 5. 테스트 유틸리티 (`internal/testutil/`)

### 5.1 testutil.go

| 함수 | 용도 |
|------|------|
| `TempGitRepo(t *testing.T) string` | 임시 git repo 생성, t.Cleanup 자동 정리 |
| `TempGitRepoWithRemote(t *testing.T, remoteURL string) string` | remote 설정된 임시 git repo 생성 |
| `TempBareRepo(t *testing.T) string` | 임시 bare git repo 생성 (push 대상) |
| `TempConfigFile(t *testing.T, content string) string` | 임시 config.toml 생성 |
| `TempCacheFile(t *testing.T, content string) string` | 임시 cache.json 생성 |
| `SetupTestProfiles(t *testing.T) string` | work/personal 2개 프로필 config 생성 |
| `WriteCtxProfile(t *testing.T, repoDir, profileName string)` | .git/ctx-profile 파일 기록 |
| `ReadCtxProfile(t *testing.T, repoDir string) string` | .git/ctx-profile 파일 읽기 |

### 5.2 exec.go - Commander Interface

```go
type Commander interface {
    Run(ctx context.Context, name string, args ...string) ([]byte, error)
    RunWithEnv(ctx context.Context, env map[string]string, name string, args ...string) ([]byte, error)
}
```

| 구현체 | 용도 |
|--------|------|
| `RealCommander` | 실제 외부 명령 실행 (프로덕션/E2E) |
| `FakeCommander` | 미리 등록된 응답 반환 (단위/통합 테스트) |

`FakeCommander`는 `map[string]Response` 구조로 명령별 응답을 등록한다.
키 형식: `"git clone"`, `"gh api repos/owner/repo"` 등.

## 6. 테스트 실행 명령

```bash
# 단위 + 통합 테스트
go test ./...

# 특정 패키지
go test ./internal/resolver/...

# E2E 테스트 (git 필요)
go test -tags=e2e ./test/e2e/...

# 커버리지 리포트
go test -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html

# Race detector
go test -race ./...

# 짧은 테스트만 (CI 캐시 워밍)
go test -short ./...

# verbose 모드
go test -v ./internal/resolver/...
```

## 7. Makefile 타겟

```makefile
test:           ## 단위 + 통합 테스트
test-e2e:       ## E2E 테스트
test-cover:     ## 커버리지 리포트 생성
test-race:      ## Race condition 검사
test-all:       ## 전체 테스트 (단위 + 통합 + E2E)
lint:           ## 정적 분석 (golangci-lint)
```

## 8. CI 통합

### 8.1 GitHub Actions 파이프라인

```
push/PR -> lint -> test (unit+integration) -> test-e2e -> coverage upload
```

### 8.2 커버리지 게이트

- PR 머지 조건: 전체 커버리지 80% 이상
- 핵심 패키지 (resolver, guard): 90% 이상
- 커버리지 하락 PR 차단

## 9. 테스트 작성 컨벤션

1. **파일 명명**: `{대상}_test.go` (같은 패키지) 또는 `{대상}_integration_test.go`
2. **함수 명명**: `Test{Function}_{Scenario}` (예: `TestResolve_CacheHit`)
3. **Table-driven**: 2개 이상 케이스가 있으면 반드시 table-driven 사용
4. **t.Helper()**: 테스트 헬퍼 함수에는 반드시 `t.Helper()` 호출
5. **t.Parallel()**: 독립적인 테스트에는 `t.Parallel()` 적용
6. **t.Cleanup()**: 리소스 정리는 defer 대신 `t.Cleanup()` 사용
7. **assert vs require**: 계속 실행 가능하면 `assert`, 실패 시 중단 필요하면 `require`
8. **Golden files**: 복잡한 출력 검증은 `testdata/` 디렉토리에 golden file 사용
