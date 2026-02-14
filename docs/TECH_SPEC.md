# TECH_SPEC.md

## 1. 문서 정보

- 문서명: GitHub 멀티계정 컨텍스트 매니저(TECH SPEC)
- 버전: v1.0
- 작성일: 2026-02-14
- 연관 PRD: PRD.md
- 구현 가정: CLI 바이너리 `ctx`(Go 권장)

## 2. 기술 목표

1. 기존 `gh`, `git`, `ssh`를 감싸는 thin wrapper 구현
2. 계정 컨텍스트 자동 판정 엔진 제공
3. 오동작 방지용 pre-push guard 제공
4. 설명 가능한 상태 진단 제공

## 3. 시스템 아키텍처

| 컴포넌트      | 역할                                   |
| ------------- | -------------------------------------- |
| CLI Layer     | `ctx` 명령 파싱, 출력, 에러코드 관리   |
| Profile Store | 회사/개인 프로필 로딩/검증             |
| Resolver      | 계정 자동 판정 로직 수행               |
| Git Adapter   | clone/remote/config/hook 작업          |
| GH Adapter    | `gh` 기반 권한 probe 및 인증 상태 조회 |
| Guard Engine  | pre-push 시 컨텍스트 무결성 검사       |
| Cache Store   | 리포-프로필 매핑 저장                  |
| Doctor        | 환경 진단 및 문제 원인 제시            |

## 4. 외부 의존성

1. `git` (필수)
2. `gh` (필수)
3. `ssh` 클라이언트 (필수)
4. macOS Keychain 또는 OS credential store (간접 사용)
5. 선택: `direnv` (자동 환경 주입 보조)

## 5. 파일/데이터 모델

### 5.1 설정 파일

경로: `~/.config/ctx/config.toml`  
권한: `0600`

```toml
version = 1
default_profile = "personal"
prompt_on_ambiguous = true
require_push_guard = true
allow_https_managed_repo = false

[profiles.work]
gh_config_dir = "/Users/hbjs/.config/gh-company"
ssh_host = "github-company"
git_name = "HBJS"
git_email = "hbjs@company.com"
email_domain = "company.com"
owners = ["company-org", "company-team"]

[profiles.personal]
gh_config_dir = "/Users/hbjs/.config/gh-personal"
ssh_host = "github-personal"
git_name = "hbjs97"
git_email = "hbjs97@naver.com"
email_domain = "naver.com"
owners = ["hbjs97", "sutefu23"]
```

### 5.2 캐시 파일

경로: `~/.config/ctx/cache.json`
권한: `0600`

```json
{
  "version": 1,
  "entries": {
    "company-org/api-server": {
      "profile": "work",
      "reason": "owner_rule",
      "resolved_at": "2026-02-14T10:30:00Z",
      "config_hash": "a1b2c3"
    }
  }
}
```

| 필드 | 설명 |
|------|------|
| `reason` | 판정 근거: `explicit`, `cache`, `owner_rule`, `probe`, `user_select` |
| `resolved_at` | ISO 8601 타임스탬프 |
| `config_hash` | config.toml profiles 섹션의 SHA-256 해시. 불일치 시 캐시 무효화 |

기본 TTL: 90일. `cache_ttl_days`로 설정 가능.

### 5.3 리포 메타데이터

경로: `<repo>/.git/ctx-profile`
내용: 프로필명 1줄 (예: `work`)

Guard Engine과 `ctx status`, shell hook(`ctx activate`)이 참조하는 로컬 앵커.
`ctx clone`, `ctx init` 실행 시 자동 생성.

## 6. Resolver 상세 로직

5단계 판정 파이프라인. 각 단계에서 확정되면 즉시 반환, 실패 시 다음 단계로 전이.

```
명시 플래그 → 캐시 조회 → Owner 규칙 → 권한 Probe → 사용자 선택
   ↓확정      ↓확정       ↓확정        ↓확정         ↓확정/에러
```

### 6.1 Step 1: 명시 플래그

- **입력**: `--profile <name>` CLI 플래그
- **확정 조건**: 플래그 존재 시 무조건 확정
- **실패 전이**: 플래그 없음 → Step 2
- **에러**: 지정된 프로필이 config.toml에 없으면 exit code 5

### 6.2 Step 2: 캐시 조회

- **입력**: `owner/repo` (URL 또는 remote에서 추출)
- **조회**: `cache.json`에서 키 매칭
- **확정 조건**: 캐시 히트 + TTL 이내 + `config_hash` 일치
- **실패 전이**: 캐시 미스 / TTL 초과 / config_hash 불일치 → Step 3

### 6.3 Step 3: Owner 규칙 매칭

- **입력**: repo owner (예: `company-org`)
- **매칭**: 모든 프로필의 `owners[]`에서 owner 포함 여부 검사
- **확정 조건**: 정확히 1개 프로필 매칭
- **실패 전이**: 0개 매칭 또는 2개 이상 매칭 → Step 4

### 6.4 Step 4: 권한 Probe

- **입력**: `owner/repo`
- **동작**: 모든 프로필에 대해 `gh api repos/{owner}/{repo}` 호출 (`GH_CONFIG_DIR` 주입)
- **응답 파싱**:
  - HTTP 200 + `permissions.push == true` → push 가능
  - HTTP 200 + `permissions.push == false` → read only
  - HTTP 404 / 403 → 접근 불가
- **확정 조건**: push 가능 프로필이 정확히 1개
- **실패 전이**:
  - push 가능 0개 → 에러 (exit code 4), read-only 프로필 정보 + 권한 확보 안내 출력
  - push 가능 2개 이상 → Step 5
- **Rate limit**: `X-RateLimit-Remaining < 10` 시 경고 출력

### 6.5 Step 5: 사용자 선택

- **대화형 모드**: probe 결과와 함께 선택지 제시, 사용자 입력 대기
- **비대화형 모드**: 에러 종료 (exit code 3), `--profile` 사용 안내 출력
- **선택 후**: 결과를 캐시에 저장 (`reason: "user_select"`)

## 7. CLI 명령 스펙

### 7.0 명령 체계

**사용자 명령 (Porcelain)** — 사용자가 직접 실행하는 5개 명령:

| 명령 | 스코프 | 용도 | 빈도 |
|------|--------|------|------|
| `ctx setup` | 글로벌 | 프로필 추가/관리. 재실행으로 계정 추가 | 1회 + 필요시 |
| `ctx clone` | 리포 | 클론 + 자동 프로필 적용 | 일상 |
| `ctx init` | 리포 | 기존 리포에 프로필 적용 | 필요시 |
| `ctx status` | 리포 | 현재 컨텍스트 확인 | 필요시 |
| `ctx doctor` | 글로벌 | 환경 진단 | 문제시 |

**내부 명령 (Plumbing)** — hook이 자동 호출하며 사용자가 직접 실행할 필요 없음:

| 명령 | 호출 주체 |
|------|----------|
| `ctx guard check` | pre-push hook |
| `ctx activate` | shell chpwd hook |

### 7.1 `ctx setup`

프로필 추가/관리. `gh auth login`처럼 몇 번이든 재실행 가능.

```
ctx setup [--force]
```

**첫 실행** (config.toml 없음):

1. `~/.config/ctx/config.toml` 생성 (권한 `0600`)
2. 프로필 이름 입력 (예: `work`)
3. 프로필 설정:
   - `git_name`, `git_email` 입력
   - `gh` 인증: `GH_CONFIG_DIR={path} gh auth login` 실행
   - `gh_config_dir` 자동 생성 (`~/.config/gh-{profile_name}/`)
   - SSH Host alias 입력 → `~/.ssh/config` 설정 존재 여부 검증
   - `owners[]` 입력 (GitHub 조직/사용자명)
4. "프로필을 더 추가하시겠습니까?" → 반복 또는 완료
5. 셸 hook 설정 (10절 참조) — 사용 중인 셸 자동 감지
6. `ctx doctor` 자동 실행으로 설정 검증

**재실행** (config.toml 있음):

1. 기존 프로필 목록 표시
2. "새 프로필을 추가합니다" → 동일 플로우로 프로필 추가
3. config.toml에 신규 프로필 append
4. `ctx doctor` 자동 실행

`--force`: 기존 config.toml을 무시하고 전체 재설정.

### 7.2 `ctx clone`

```
ctx clone <target> [flags]

target:
  owner/repo
  https://github.com/owner/repo.git
  git@github.com:owner/repo.git

flags:
  --profile <name>   프로필 명시 (Resolver Step 1)
  --dir <path>       클론 대상 디렉토리
  --no-guard         pre-push guard 설치 생략
```

동작 순서:

1. target에서 `owner/repo` 추출 (URL 파싱)
2. Resolver 실행 → 프로필 확정
3. remote URL 생성: `git@{ssh_host}:{owner}/{repo}.git`
4. `git clone` 실행
5. `.git/config`에 `user.name`, `user.email` 설정
6. `.git/ctx-profile`에 프로필명 기록
7. pre-push guard 설치 (`--no-guard` 미사용 시)
8. 캐시 저장
9. 판정 결과 요약 출력

### 7.3 `ctx init`

기존 리포에 프로필 적용. `git init`처럼 리포 안에서 실행.

```
ctx init [flags]

flags:
  --profile <name>   프로필 명시
  --yes              확인 프롬프트 생략
  --refresh          캐시 무시하고 재판정
```

동작 순서:

1. 현재 디렉토리가 git 리포인지 확인
2. remote URL에서 `owner/repo` 추출
3. Resolver 실행 → 프로필 확정
4. remote URL이 HTTPS인 경우:
   - `allow_https_managed_repo = false`(기본값): SSH 변환 제안 + 사용자 확인 후 `git remote set-url` 실행
   - `allow_https_managed_repo = true`: HTTPS 유지, 경고 출력
5. `.git/config`에 `user.name`, `user.email` 설정
6. `.git/ctx-profile` 기록
7. pre-push guard 설치
8. 캐시 저장

`--refresh`: 기존 캐시를 무효화하고 Resolver를 처음부터 재실행. 프로필 변경이나 권한 변경 시 사용.

### 7.4 `ctx status`

```
ctx status [flags]

flags:
  --json    JSON 출력
```

출력 항목:

| 항목 | 출처 |
|------|------|
| 현재 프로필명 | `.git/ctx-profile` |
| 판정 근거 | `cache.json` |
| remote URL / SSH host | `git remote get-url origin` |
| git user.name / user.email | `git config user.name/email` |
| gh 활성 계정 | `GH_CONFIG_DIR` 기준 `gh auth status` |
| guard 설치 상태 | hook 파일 검사 |
| 불일치 항목 | 위 항목 간 교차 검증 |

캐시 히트 시 300ms 이내 응답 (NFR-03).

ctx 미관리 리포에서 실행 시: "이 리포는 ctx로 관리되지 않습니다. `ctx init`을 실행하세요." 안내.

### 7.5 `ctx doctor`

```
ctx doctor [flags]

flags:
  --json    JSON 출력
```

점검 항목:

| # | 점검 | OK | WARN | FAIL |
|---|------|----|------|------|
| 1 | `git` / `gh` / `ssh` 바이너리 | 존재 | — | 미설치 |
| 2 | 프로필별 `gh` 인증 상태 | 인증됨 | 토큰 만료 임박 | 미인증 |
| 3 | 프로필별 SSH 연결 | `ssh -T git@{ssh_host}` 성공 | — | 실패 |
| 4 | SSH `IdentitiesOnly` 설정 | 설정됨 | — | 미설정 (다중 키 혼선 위험) |
| 5 | `GH_TOKEN`/`GITHUB_TOKEN` 간섭 | 미설정 | 설정됨 (프로필 우회 경고) | — |
| 6 | HTTPS credential helper 충돌 | 없음 | — | `osxkeychain`이 github.com에 등록됨 |
| 7 | config.toml 유효성 | 파싱 성공 | — | 파싱 실패 / 필수 필드 누락 |
| 8 | 셸 hook 설치 상태 | 설정됨 | — | 미설정 (`ctx setup` 안내) |

각 항목에 대해 상태 + 수정 안내 출력.

### 7.6 Internal: `ctx guard check`

pre-push hook이 호출하는 내부 명령. 사용자가 직접 실행할 필요 없음.
상세 동작은 8절 Guard Engine 참조.

### 7.7 Internal: `ctx activate`

shell chpwd hook이 호출하는 내부 명령. `ctx setup` 시 자동 설정됨.

```
ctx activate [--shell {bash|zsh|fish}]
```

현재 리포의 프로필에 맞는 환경변수를 셸 eval 형식으로 출력.

```bash
# 출력 예시
export GH_CONFIG_DIR="/Users/hbjs/.config/gh-company"
export CTX_PROFILE="work"
```

리포 밖에서는 `default_profile`의 환경변수를 출력.
상세 동작은 10절 Shell Integration 참조.

## 8. Guard Engine 상세

### 8.1 검사 항목

pre-push hook 실행 시 다음을 순서대로 검사:

| 검사 | 기대값 소스 | 실제값 소스 | 불일치 시 |
|------|------------|------------|----------|
| 프로필 존재 | `.git/ctx-profile` | config.toml | 에러: 프로필 미등록 |
| remote SSH host | 프로필의 `ssh_host` | `git remote get-url origin` 파싱 | 차단 |
| git user.email | 프로필의 `git_email` | `git config user.email` | 차단 |
| git user.name | 프로필의 `git_name` | `git config user.name` | 경고 (차단은 선택) |

### 8.2 차단 시 출력

```
[ctx] push 차단: 컨텍스트 불일치 감지

  기대 프로필: work
  remote host: github-company (기대) ≠ github.com (실제)
  user.email:  hbjs@company.com (기대) ≠ hbjs97@naver.com (실제)

  수정 방법:
    ctx init --profile work    # 프로필 재적용
    git push --no-verify       # 가드 우회 (비권장)
```

### 8.3 우회

- `git push --no-verify`: git 내장 hook 우회 (ctx 제어 밖)
- `CTX_SKIP_GUARD=1 git push`: 환경변수로 ctx guard만 우회
- 두 경우 모두 stderr에 경고 메시지 출력

### 8.4 Hook 공존 전략

설치 시 환경 탐지 순서:

1. `core.hooksPath` 설정 확인
   - **설정됨** (husky, lefthook 등): 해당 경로의 `pre-push` 파일에 ctx guard 호출을 삽입
     ```bash
     # ctx-guard-start
     command -v ctx >/dev/null 2>&1 && ctx guard check || exit 1
     # ctx-guard-end
     ```
   - **미설정**: `.git/hooks/pre-push`에 직접 설치
2. 기존 `pre-push` hook 존재 시:
   - 원본을 `.git/hooks/pre-push.ctx-backup`으로 백업
   - 새 `pre-push`에서 ctx guard 실행 후 원본 체이닝
3. `ctx guard uninstall` 시:
   - 삽입된 마커(`ctx-guard-start` / `ctx-guard-end`) 사이 제거
   - 또는 백업에서 원본 복원

## 9. GH Adapter 상세

### 9.1 권한 Probe

```bash
GH_CONFIG_DIR={profile.gh_config_dir} gh api repos/{owner}/{repo} \
  --hostname github.com --jq '.permissions'
```

응답 처리:

| HTTP 상태 | 해석 | 후속 동작 |
|-----------|------|----------|
| 200 | 접근 가능. `.permissions.push`로 push 여부 판단 | 결과 수집 |
| 404 | 접근 불가 (private + 권한 없음) | 해당 프로필 제외 |
| 403 | 인증됐으나 권한 부족 | 응답 헤더 `X-GitHub-SSO` 존재 시 SSO authorize URL 안내 |
| 401 | 토큰 만료/무효 | 해당 프로필 제외 + `gh auth refresh` 안내 |

### 9.2 Rate Limit 대응

- `X-RateLimit-Remaining` 확인
- 잔여 10 미만: stderr에 경고 출력, probe 계속 진행
- 잔여 0: 에러 + `X-RateLimit-Reset` 시각 안내
- probe 결과는 5분간 인메모리 캐시 (동일 세션 내 중복 호출 방지)

### 9.3 환경변수 간섭 감지

probe 실행 전 환경변수 확인:

- `GH_TOKEN` 또는 `GITHUB_TOKEN`이 설정되어 있으면:
  - 해당 토큰이 `GH_CONFIG_DIR`의 인증 정보를 덮어씀
  - stderr에 경고 출력: "환경변수 GH_TOKEN이 설정되어 프로필 인증이 무시됩니다"
  - probe 시 해당 환경변수를 임시 unset (`env -u GH_TOKEN gh api ...`)

## 10. Shell Integration 상세

### 10.1 `ctx activate` 동작

1. 현재 디렉토리가 git 리포인지 확인
2. `.git/ctx-profile`에서 프로필명 읽기
3. config.toml에서 해당 프로필의 설정 로드
4. 환경변수 export 명령 출력

### 10.2 셸별 연동 방법

**Zsh** (`~/.zshrc`):
```bash
ctx_chpwd() {
  if [ -f "$(git rev-parse --show-toplevel 2>/dev/null)/.git/ctx-profile" ]; then
    eval "$(ctx activate 2>/dev/null)"
  fi
}
chpwd_functions+=(ctx_chpwd)
```

**Bash** (`~/.bashrc`):
```bash
ctx_prompt_command() {
  if [ -f "$(git rev-parse --show-toplevel 2>/dev/null)/.git/ctx-profile" ]; then
    eval "$(ctx activate 2>/dev/null)"
  fi
}
PROMPT_COMMAND="ctx_prompt_command;$PROMPT_COMMAND"
```

**Fish** (`~/.config/fish/conf.d/ctx.fish`):
```fish
function __ctx_chpwd --on-variable PWD
  set -l root (git rev-parse --show-toplevel 2>/dev/null)
  if test -n "$root" -a -f "$root/.git/ctx-profile"
    ctx activate --shell fish | source
  end
end
```

**direnv** (`ctx init` 시 `.envrc` 생성 옵션 제공):
```bash
# .envrc
eval "$(ctx activate)"
```

### 10.3 비리포 디렉토리 동작

리포 밖에서 `ctx activate` 실행 시:

- `default_profile`이 설정되어 있으면 해당 프로필의 환경변수 출력
- 미설정이면 `unset GH_CONFIG_DIR` 등 환경변수 정리 명령 출력

## 11. 캐시 무효화 정책

| 트리거 | 동작 |
|--------|------|
| TTL 초과 (기본 90일) | 다음 Resolver 호출 시 캐시 무시, 재판정 후 갱신 |
| config.toml 변경 (`config_hash` 불일치) | 해당 항목 무효화, Resolver 재실행 |
| `ctx init --refresh` | 해당 리포 캐시 무효화 + Resolver 재실행 |
| push 인증 실패 (git exit 128) | 해당 리포 캐시 무효화 + Resolver 재실행 제안 출력 |
| `ctx init --profile <name>` | 해당 리포 캐시를 새 프로필로 덮어쓰기 |
| 프로필 삭제 (config.toml에서 제거) | 해당 프로필 참조 캐시 전체 무효화 |

## 12. 에러 코드 체계

| 코드 | 의미 | 대표 시나리오 |
|------|------|-------------|
| 0 | 성공 | 정상 clone, status, guard pass |
| 1 | 일반 에러 | git/gh 실행 실패, 네트워크 오류 |
| 2 | Guard 차단 | pre-push 시 컨텍스트 불일치 |
| 3 | 모호 판정 | 비대화형 모드에서 Resolver 확정 실패 |
| 4 | 권한/인증 실패 | 모든 프로필에서 접근 불가 |
| 5 | 설정 오류 | config.toml 파싱 실패, 프로필 미존재 |
| 6 | 의존성 없음 | git / gh / ssh 미설치 |

스크립팅 연동: exit code 2(guard 차단)와 3(모호 판정)은 의도적 방어 동작이므로 에러와 구분하여 처리.

## 13. 보안 고려사항

1. **토큰 비저장**: ctx는 자체적으로 토큰을 저장하지 않음. `gh auth`에 위임
2. **파일 권한**: `config.toml`, `cache.json` 모두 `0600`. 생성 시 권한 검증, 권한 초과 시 경고
3. **로그 마스킹**: 디버그 출력(`--verbose`) 시 토큰 패턴 자동 마스킹 (`ghp_*`, `gho_*`, `github_pat_*`)
4. **캐시 안전성**: 캐시에는 프로필명/판정 근거만 저장. 인증 정보 미포함
5. **`.git/ctx-profile`**: 프로필명만 포함. `.git/` 하위이므로 커밋 대상 아님
6. **환경변수 격리**: probe 시 `GH_TOKEN`/`GITHUB_TOKEN` 임시 unset으로 간섭 방지
