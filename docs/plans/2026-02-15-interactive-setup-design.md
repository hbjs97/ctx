# Interactive Setup 설계

- 작성일: 2026-02-15
- 상태: 승인됨

## 1. 목표

`ctx setup`을 template 파일 생성에서 **TUI 기반 interactive setup wizard**로 개선한다.
프로필 CRUD, gh 인증, SSH host 자동 감지, 셸 hook 설치를 한 번에 수행한다.

## 2. TUI 라이브러리

**`charmbracelet/huh`** 사용.

- Form 중심 API — setup wizard에 최적
- Group 단위 페이지 분리, 그 사이에 외부 명령(`gh auth login`) 실행 가능
- 테스트: `WithInput(io.Reader)`로 프로그래밍적 입력 주입
- bubbletea 위에 구축 — 필요시 확장 가능
- Go 1.23+ 호환 (프로젝트 Go 1.26)

## 3. 전체 플로우

### 3.1 첫 실행 (config.toml 없음)

```
[환영 메시지]
→ 프로필 추가 플로우 (반복)
  → 이름 입력
  → git_name, git_email 입력
  → gh_config_dir 자동 생성 (~/.config/gh-{프로필명}/)
  → GH_CONFIG_DIR={path} gh auth login 실행
  → ~/.ssh/config에서 GitHub SSH Host 자동 감지 → 선택
  → gh api로 소속 조직/사용자 자동 조회 → 체크리스트 선택 (owners)
  → "프로필을 더 추가하시겠습니까?"
→ 셸 hook 자동 설치 (감지된 셸 표시)
→ config.toml 저장 (0600)
→ ctx doctor 자동 실행
```

### 3.2 재실행 (config.toml 있음)

```
[기존 프로필 목록 표시]
→ 작업 선택: 추가 / 수정 / 삭제
  - 추가: 3.1과 동일한 프로필 추가 플로우
  - 수정: 프로필 선택 → 필드별 현재값 표시 + 변경 입력
  - 삭제: 프로필 선택 → 확인 → 제거 (마지막 프로필 삭제 불가)
→ config.toml 저장
→ ctx doctor 자동 실행
```

### 3.3 --force 플래그

기존 config.toml을 무시하고 첫 실행 플로우로 진입.

## 4. 아키텍처

### 4.1 패키지 구조

```
internal/
  setup/              ← 새 패키지
    setup.go          - Runner (진입점, 플로우 제어)
    profile_form.go   - 프로필 입력 폼 (huh 기반)
    shell_hook.go     - 셸 감지 + hook 설치
    detect.go         - SSH host / org 자동 감지
    setup_test.go     - 테스트
  cli/
    setup.go          - cobra 명령 정의 (Runner 호출만)
```

### 4.2 의존성 방향

```
cli/setup.go → setup.Runner
setup.Runner → config (로드/저장)
setup.Runner → cmdexec.Commander (gh auth login, gh api 실행)
setup.Runner → doctor (자동 진단)
setup.Runner → huh (TUI 폼)
```

### 4.3 핵심 인터페이스

```go
// Runner는 interactive setup의 진입점이다.
type Runner struct {
    CfgPath   string
    Commander cmdexec.Commander
    FormRunner FormRunner
}

// FormRunner는 TUI 폼 실행을 추상화한다.
type FormRunner interface {
    RunProfileForm(defaults *ProfileInput) (*ProfileInput, error)
    RunActionSelect(profiles []string) (Action, error)
    RunConfirm(message string) (bool, error)
}
```

테스트에서는 FormRunner를 mock하여 huh 의존성 없이 플로우 로직을 검증한다.

## 5. 프로필 CRUD 상세

### 5.1 추가 (Create)

1. 프로필 이름, git_name, git_email 입력
2. `gh_config_dir` 자동 생성 (`~/.config/gh-{프로필명}/`)
3. `GH_CONFIG_DIR={path} gh auth login --hostname github.com` 실행
4. `~/.ssh/config`에서 GitHub SSH Host 자동 감지 → 선택 UI
5. `gh api user/orgs`로 조직 목록 조회 + 본인 사용자명 포함 → 체크리스트
6. "프로필을 더 추가하시겠습니까?"

### 5.2 수정 (Update)

1. 프로필 선택 (이름 + email 표시)
2. 각 필드의 현재값을 기본값으로 표시, 변경할 부분만 수정
3. SSH host/owners도 동일한 자동 감지 UI 재사용
4. 변경 후 안내: "기존 리포에 반영하려면 `ctx init --refresh`를 실행하세요."

### 5.3 삭제 (Delete)

1. 프로필 선택
2. 확인 프롬프트
3. 마지막 프로필은 삭제 불가 (config에 최소 1개 프로필 필요)

### 5.4 Validation

| 필드 | 검증 |
|------|------|
| 프로필 이름 | 비어있지 않음, 중복 없음, 영문+숫자+하이픈 |
| git_email | `@` 포함 |
| ssh_host | `~/.ssh/config`에 미존재 시 경고 (차단 안 함) |
| owners | 최소 1개 선택 |

## 6. 자동 감지

### 6.1 SSH Host 감지

`~/.ssh/config` 파싱 → `Host github*` 또는 `HostName github.com` 항목 추출.
감지 실패 시 직접 입력 폼으로 fallback.

### 6.2 조직/사용자 자동 조회

`GH_CONFIG_DIR={path} gh api user/orgs --jq '.[].login'`으로 조직 목록 조회.
인증된 사용자명도 `gh api user --jq '.login'`으로 조회하여 목록에 포함.
조회 실패 시 직접 입력 폼으로 fallback.

## 7. 에러 처리

| 상황 | 동작 |
|------|------|
| `gh` 미설치 | 안내 메시지 + 종료 |
| `gh auth login` 실패/취소 | "인증을 건너뛰시겠습니까?" → gh_config_dir만 설정 |
| `~/.ssh/config` 없음 | SSH host 자동 감지 건너뛰고 직접 입력 fallback |
| `gh api` 실패 | owners 자동 조회 건너뛰고 직접 입력 fallback |
| config.toml 쓰기 실패 | 에러 메시지 + exit code 1 |
| 사용자 Ctrl+C | "설정이 취소되었습니다." 출력, 변경사항 저장 안 함 |

핵심 원칙: **자동 감지 실패는 차단하지 않고 수동 입력으로 fallback**.

## 8. 테스트 전략

### 8.1 플로우 로직 테스트 (mock FormRunner)

- 첫 실행 / 재실행 분기
- CRUD 각 작업 후 config 상태 검증
- `gh auth login` 실패 시 건너뛰기 플로우
- 마지막 프로필 삭제 차단

### 8.2 폼 UI 테스트 (huh WithInput)

- validation 규칙 (이름 중복, email 형식 등)
- 기본값 표시 검증

### 8.3 자동 감지 테스트 (mock Commander)

- `~/.ssh/config` 파싱 → Host 목록 추출
- `gh api user/orgs` → 조직 목록 추출
- 감지 실패 시 빈 목록 반환 (에러 아님)

커버리지 목표: setup 패키지 80%+.
