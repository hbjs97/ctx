# 보안 규칙

## 토큰/키 로그 출력 금지

- 토큰, 키, 비밀번호를 로그에 절대 출력하지 않는다.
- 디버그 출력(`--verbose`) 시 다음 패턴을 자동 마스킹한다:
  - `ghp_*` (GitHub Personal Access Token)
  - `gho_*` (GitHub OAuth Token)
  - `github_pat_*` (GitHub Fine-grained PAT)
  - `ghs_*` (GitHub App Server-to-Server Token)
  - `ghu_*` (GitHub App User-to-Server Token)
- 마스킹 예시: `ghp_xxxxxxxxxxxx` -> `ghp_****`

## 파일 권한

- `config.toml`과 `cache.json`은 생성 시 반드시 `0600` 권한으로 설정한다.
- 파일 읽기 전에 권한을 검증하고, `0600`보다 넓은 권한이면 경고를 출력한다.
- 디렉토리(`~/.config/ctx/`)는 `0700` 권한으로 생성한다.

```go
// 올바른 파일 생성
os.WriteFile(path, data, 0600)
os.MkdirAll(dir, 0700)
```

## 환경변수 간섭 감지

- `GH_TOKEN` 또는 `GITHUB_TOKEN` 환경변수가 설정되어 있으면 반드시 감지하고 경고한다.
- 이 환경변수들은 `GH_CONFIG_DIR` 기반 인증을 덮어쓰므로, probe 실행 시 임시 unset한다.
- 경고 메시지: "환경변수 GH_TOKEN이 설정되어 프로필 인증이 무시됩니다"

## 토큰 비저장 원칙

- ctx는 자체적으로 토큰을 저장하지 않는다.
- 모든 인증은 `gh auth`에 위임한다.
- 캐시(`cache.json`)에는 프로필명과 판정 근거만 저장하며, 인증 정보는 포함하지 않는다.

## 캐시 안전성

- 캐시에 저장되는 정보: 프로필명, 판정 근거(`reason`), 타임스탬프, config 해시
- 캐시에 저장하면 안 되는 정보: 토큰, 비밀번호, SSH 키, API 응답 본문
