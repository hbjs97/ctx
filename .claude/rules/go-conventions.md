# Go 코딩 컨벤션

## 포매팅
- gofmt 필수 적용. 모든 코드는 커밋 전 gofmt를 통과해야 한다.
- goimports를 사용하여 import 정리. 표준 라이브러리 / 외부 패키지 / 내부 패키지 순서로 그룹핑한다.

## GoDoc 주석
- exported 함수, 타입, 상수에는 반드시 GoDoc 주석을 작성한다.
- 주석은 해당 식별자 이름으로 시작한다. (예: `// Resolve는 리포에 대한 프로필을 판정한다.`)

## 에러 처리
- 에러는 반드시 처리한다. `_`로 무시하지 않는다.
- 의도적으로 에러를 무시해야 하는 경우, 반드시 주석으로 이유를 명시한다.
  ```go
  _ = file.Close() // 읽기 전용이므로 close 에러 무시
  ```
- error wrapping은 `fmt.Errorf`와 `%w` verb를 사용한다.
  ```go
  return fmt.Errorf("config.Load: %w", err)
  ```
- wrapping 형식: `"패키지.함수: %w"` 패턴을 따른다.

## 네이밍
- Go 표준 네이밍을 따른다: camelCase (unexported), PascalCase (exported).
- 약어는 대문자를 유지한다: `URL`, `SSH`, `GH`, `HTTP`, `JSON`, `API`, `TTL`, `ID`.
  ```go
  // 올바름
  func ParseURL(rawURL string) (*URL, error)
  type SSHConfig struct{}
  var ghConfigDir string

  // 틀림
  func ParseUrl(rawUrl string) (*Url, error)
  type SshConfig struct{}
  ```
- 인터페이스는 행위 기반으로 명명한다: `Commander`, `Resolver`, `Prober`.
- 단일 메서드 인터페이스는 `-er` 접미사를 사용한다.
