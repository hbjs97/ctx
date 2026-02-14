# 아키텍처 규칙

## 패키지 의존성 방향

```
cmd/ctx/
  └── internal/cli
        ├── internal/resolver
        │     ├── internal/gh      (GH Adapter)
        │     └── internal/git     (Git Adapter)
        ├── internal/config        (Profile Store)
        ├── internal/cache         (Cache Store)
        ├── internal/guard         (Guard Engine)
        ├── internal/doctor        (환경 진단)
        ├── internal/shell         (Shell Integration)
        └── internal/cmdexec       (Commander interface)
```

- `cmd/ctx` -> `internal/cli`: 엔트리포인트는 CLI 레이어만 호출한다.
- `internal/cli` -> 도메인/어댑터 패키지들: CLI가 각 기능 패키지를 조합한다.
- `internal/resolver`는 `internal/gh`, `internal/git`을 사용한다 (adapter 패턴).
- `internal/gh`와 `internal/git`은 서로 의존하지 않는다.
- `internal/cmdexec`는 Commander interface를 정의한다. 모든 adapter/engine 패키지가 의존한다.
- `internal/config`, `internal/cache`는 순수 데이터 패키지로 외부 명령에 의존하지 않는다.

## 순환 의존성 금지

- 패키지 간 순환 의존성은 절대 허용하지 않는다.
- 순환이 감지되면 interface 추출로 의존성 역전(DIP)을 적용한다.
- `go vet`이 순환 의존성을 보고하면 반드시 해결 후 커밋한다.

## 외부 명령 실행 규칙

- `git`, `gh`, `ssh` 등 외부 명령은 반드시 interface를 통해 실행한다.
- `os/exec`를 직접 호출하지 않고, Commander interface를 구현한 adapter를 사용한다.
- 테스트에서는 mock Commander를 주입하여 외부 명령 없이 검증한다.

```go
// 올바른 패턴: internal/cmdexec 패키지의 interface 사용
import "github.com/hbjs97/ctx/internal/cmdexec"

type Adapter struct {
    cmd cmdexec.Commander
}

// 금지: os/exec 직접 호출
// cmd := exec.Command("git", "clone", url)

// 테스트에서는 testutil.FakeCommander 주입
// fc := testutil.NewFakeCommander()
// adapter := NewAdapter(fc)
```

## 설계 원칙

- thin wrapper: `gh`, `git`, `ssh`를 직접 호출하되, 자체 구현은 최소화한다.
- 클린 아키텍처: `cli` -> `domain` <- `adapters` 방향을 유지한다.
- 각 패키지는 단일 책임 원칙(SRP)을 따른다.
