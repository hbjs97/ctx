# ctx - GitHub 멀티계정 컨텍스트 매니저 CLI

## 빌드 / 테스트 / 린트

```bash
make build       # 바이너리 빌드
make test        # 전체 테스트 실행
make coverage    # 커버리지 리포트
make lint        # golangci-lint 실행
```

## 디렉토리 구조

```
cmd/ctx/          메인 엔트리포인트
internal/
  cli/            CLI 명령 파싱, 출력, 에러코드 관리
  config/         config.toml 로딩/검증, Profile Store (프로필 조회·매칭 포함)
  resolver/       계정 자동 판정 엔진 (5단계 파이프라인)
  cache/          리포-프로필 매핑 캐시 (cache.json)
  git/            Git Adapter (clone/remote/config/hook)
  gh/             GH Adapter (gh 기반 권한 probe, 인증 상태)
  guard/          Guard Engine (pre-push 컨텍스트 무결성 검사)
  doctor/         환경 진단 및 문제 원인 제시
  shell/          Shell Integration (activate, chpwd hook)
  testutil/       테스트 유틸리티 (Commander mock, 임시 디렉토리 등)
docs/             PRD, TECH_SPEC 등 설계 문서
```

## 핵심 컨벤션

### Go 스타일
- gofmt / goimports 필수 적용
- exported 함수에 GoDoc 주석 필수

### 에러 처리
- 에러는 반드시 처리 (`_` 무시 금지, 의도적 무시는 주석 명시)
- error wrapping: `fmt.Errorf("패키지.함수: %w", err)`

### 외부 명령 실행
- `git`, `gh`, `ssh` 등 외부 명령은 반드시 Commander interface를 통해 실행
- 테스트 시 mock 가능하도록 설계

### 패키지 의존성
- 방향: `cmd/ctx` -> `internal/cli` -> `internal/resolver`, `internal/config` 등
- `internal/resolver`는 `internal/gh`, `internal/git`을 사용 (adapter 패턴)
- `internal/gh`, `internal/git`은 서로 의존하지 않음
- 순환 의존성 절대 금지

### TDD (Kent Beck)
- RED → GREEN → REFACTOR 사이클 엄격 준수
- GREEN 단계: 테스트를 통과하는 **가장 단순한 구현** (Fake It → Triangulation → 일반화)
- YAGNI: 테스트가 요구하지 않는 코드를 작성하지 않는다
- table-driven tests 사용
- 커버리지 80% 이상 유지

## 커밋 메시지

Conventional Commits 형식을 따른다.

```
feat: 새로운 기능 추가
fix: 버그 수정
refactor: 리팩토링
test: 테스트 추가/수정
docs: 문서 변경
```
