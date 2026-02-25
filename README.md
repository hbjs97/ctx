# ctx

GitHub 멀티계정 컨텍스트 매니저 CLI.

회사/개인 등 여러 GitHub 계정을 사용할 때, 리포지토리별로 올바른 SSH 호스트, git 사용자 정보, `gh` 인증을 자동으로 적용한다. 잘못된 계정으로 push하는 실수를 방지하는 pre-push guard를 제공한다.

## 주요 기능

- **자동 계정 판정** — 리포의 owner를 기반으로 프로필을 자동 매칭
- **pre-push guard** — push 전 SSH host, email, name 일치 여부를 검사하여 차단
- **셸 자동 전환** — 디렉토리 이동 시 `GH_CONFIG_DIR` 등 환경변수 자동 설정
- **환경 진단** — `ctx doctor`로 SSH, gh 인증, 설정 상태를 한눈에 확인

## 사전 요구사항

- Go 1.26+
- `git`, `gh`, `ssh` CLI 설치

### Go 설치

프로젝트는 Go 1.26 이상을 요구한다. 아직 Go가 설치되지 않았거나 버전이 낮다면 아래 방법 중 하나로 설치한다.

**macOS (Homebrew):**

```bash
brew install go && go version
```

**Linux:**

```bash
ARCH=$(dpkg --print-architecture 2>/dev/null || (uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')) && sudo rm -rf /usr/local/go && curl -fsSL "https://go.dev/dl/go1.26.0.linux-${ARCH}.tar.gz" | sudo tar -C /usr/local -xz && echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc && source ~/.bashrc && go version
```

그 외 설치 방법은 [go.dev/dl](https://go.dev/dl/) 참조.

## 설치

```bash
go install github.com/hbjs97/ctx/cmd/ctx@latest
```

`$GOBIN` (기본 `~/go/bin`)이 PATH에 포함되어 있어야 한다.

소스에서 직접 빌드하려면:

```bash
git clone https://github.com/hbjs97/ctx.git
cd ctx
make build
```

빌드된 바이너리를 PATH에 설치:

```bash
# macOS / Linux
sudo cp bin/ctx /usr/local/bin/
```

## 빠른 시작

### 1. 초기 설정

```bash
ctx setup
```

대화형 설정 마법사가 실행된다:

- 프로필 이름, git 사용자 정보 입력
- SSH 키 자동 감지 및 선택/생성 (기존 키 사용 또는 `id_ed25519_{프로필명}` 자동 생성)
- `~/.ssh/config`에 Host alias 자동 추가
- `gh auth login` 자동 실행
- `gh api`로 소속 조직 자동 조회
- 셸 hook 자동 설치

재실행하면 프로필 추가/수정/삭제가 가능하다. 기존 설정을 초기화하려면:

```bash
ctx setup --force
```

### 2. 리포 클론

```bash
ctx clone your-org/project
```

owner 규칙으로 프로필을 판정하고, SSH remote URL 설정, git user 설정, pre-push guard 설치를 자동으로 수행한다.

### 3. 기존 리포에 적용

```bash
cd ~/existing-repo
ctx init
```

### 4. 상태 확인

```bash
ctx status
```

### 5. 환경 진단

```bash
ctx doctor
```

## 명령어

| 명령 | 설명 |
|------|------|
| `ctx setup [--force]` | 대화형 설정 마법사 (프로필 CRUD) |
| `ctx clone <target>` | 리포 클론 + 프로필 자동 적용 |
| `ctx init` | 기존 리포에 프로필 적용 |
| `ctx status` | 현재 컨텍스트 확인 |
| `ctx doctor` | 환경 진단 (SSH, gh 인증, 설정 검증) |
| `ctx guard check` | pre-push hook이 호출하는 내부 명령 |
| `ctx activate` | 셸 hook이 호출하는 내부 명령 |

## 프로필 판정 방식

5단계 파이프라인으로 프로필을 자동 판정한다:

1. `--profile` 플래그로 명시 지정
2. 캐시 조회 (TTL + config hash 검증)
3. Owner 규칙 매칭 (`owners[]` 필드)
4. `gh api` 권한 probe
5. 사용자 대화형 선택

## 개발

```bash
make build      # 바이너리 빌드
make test       # 테스트 실행
make test-race  # Race condition 검사
make coverage   # 커버리지 리포트
make lint       # 정적 분석
```

## 라이선스

MIT
