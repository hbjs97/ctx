# SSH Key Automation Design

**Goal:** `ctx setup` 플로우에 SSH 키 생성, `~/.ssh/config` Host alias 작성을 자동화하여, 사용자가 수동으로 SSH 키를 생성하거나 config를 편집할 필요 없이 셋업을 완료할 수 있게 한다.

**접근 방식:** A+C 혼합 — `internal/setup/ssh.go`에 SSH 관련 함수를 집중시키고, Runner의 collectProfile 플로우에 통합한다.

---

## 1. 새 파일: `internal/setup/ssh.go`

기존 `detect.go`에서 `ParseSSHConfig`, `DefaultSSHConfigPath`를 이동하고, 다음 함수를 추가한다:

| 함수 | 역할 |
|------|------|
| `ParseSSHConfig(path)` | (기존) SSH config에서 GitHub Host alias 목록 추출 |
| `DefaultSSHConfigPath()` | (기존) `~/.ssh/config` 경로 반환 |
| `DetectSSHKeys(sshDir)` | `~/.ssh/`에서 `id_*` 키 쌍(공개+비밀) 목록 반환 |
| `GenerateSSHKey(cmd, email, path)` | `ssh-keygen -t ed25519 -C {email} -f {path} -N ""` 실행 |
| `WriteSSHConfigEntry(configPath, host, identityFile)` | Host 블록 추가 (중복 시 스킵) |

### SSHKeyInfo 타입

```go
type SSHKeyInfo struct {
    Name       string // 예: "id_ed25519_work"
    PrivateKey string // 전체 경로
    PublicKey  string // 전체 경로 (.pub)
}
```

### SSHKeyChoice 타입

```go
type SSHKeyChoice struct {
    Action      string // "existing" | "generate"
    ExistingKey string // Action=="existing"일 때 비밀키 경로
}
```

## 2. FormRunner 확장

```go
// RunSSHKeySelect는 기존 키 목록에서 선택하거나 새로 생성하는 UI를 표시한다.
RunSSHKeySelect(existingKeys []SSHKeyInfo, profileName string) (SSHKeyChoice, error)
```

- 기존 키가 있으면: 목록 표시 + "새로 생성" 옵션
- 기존 키가 없으면: "SSH 키가 없습니다. 생성합니까?" 확인 프롬프트

## 3. Runner collectProfile 플로우

**변경 전:**
```
ProfileForm → gh auth login → ParseSSHConfig → SSHHostSelect → DetectOrgs
```

**변경 후:**
```
ProfileForm → DetectSSHKeys → RunSSHKeySelect
  → (GenerateSSHKey if needed)
  → WriteSSHConfigEntry
  → gh auth login
  → SSHHost 자동결정 (방금 생성한 host alias)
  → DetectOrgs → RunOwnersSelect
```

### Host alias 네이밍

`github.com-{profileName}` (예: `github.com-work`, `github.com-personal`)

### SSH config 엔트리 형식

```
Host github.com-{profileName}
  HostName github.com
  User git
  IdentityFile ~/.ssh/id_ed25519_{profileName}
  IdentitiesOnly yes
```

`IdentitiesOnly yes` — TECH_SPEC doctor 항목 #4 충족 (다중 키 혼선 방지).

## 4. 에러 처리

- SSH 키 생성 실패 → 경고 출력, 기존 수동 플로우로 fallback (차단하지 않음)
- SSH config 쓰기 실패 → 경고 출력, 수동 SSHHostSelect로 fallback
- 동일 Host가 이미 config에 존재 → 스킵하고 기존 엔트리 사용
- 기존 키 선택 시 → 해당 키의 IdentityFile로 config 엔트리 작성

## 5. 테스트 전략

- `ssh_test.go`: DetectSSHKeys (tmpdir), WriteSSHConfigEntry (파일 I/O), GenerateSSHKey (FakeCommander)
- Runner 테스트: mockFormRunner에 RunSSHKeySelect 추가, 플로우 통합 테스트
- 기존 `detect_test.go`의 ParseSSHConfig 테스트는 `ssh_test.go`로 이동

## 6. 영향 범위

- `internal/setup/ssh.go` — 새 파일 (기존 함수 이동 + 신규 함수)
- `internal/setup/ssh_test.go` — 새 파일 (기존 테스트 이동 + 신규 테스트)
- `internal/setup/detect.go` — ParseSSHConfig, DefaultSSHConfigPath 제거 (ssh.go로 이동)
- `internal/setup/detect_test.go` — ParseSSHConfig 테스트 제거 (ssh_test.go로 이동)
- `internal/setup/types.go` — SSHKeyInfo, SSHKeyChoice 타입 + FormRunner에 RunSSHKeySelect 추가
- `internal/setup/form.go` — HuhFormRunner에 RunSSHKeySelect 구현
- `internal/setup/runner.go` — collectProfile 플로우 변경
- `internal/setup/runner_test.go` — mockFormRunner 업데이트, 신규 테스트
- `README.md` — 사전 요구사항에서 SSH 수동 설정 항목 제거
