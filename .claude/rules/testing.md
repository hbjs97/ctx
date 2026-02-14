# 테스트 규칙

## TDD 철학 (Kent Beck)

TDD는 단순한 테스트 작성 기법이 아니라 **설계 방법론**이다.

### 핵심 원칙

1. **RED → GREEN → REFACTOR** 사이클을 엄격히 따른다.
   - RED: 실패하는 테스트를 먼저 작성한다.
   - GREEN: 테스트를 통과하는 **가장 단순한 구현**을 작성한다.
   - REFACTOR: 테스트를 유지하며 코드를 개선한다.

2. **Baby Steps** - 한 번에 하나의 작은 변경만 한다.
   - 한 번에 하나의 테스트만 RED 상태로 유지한다.
   - 큰 기능은 작은 테스트 여러 개로 분해한다.

3. **YAGNI (You Aren't Gonna Need It)** - 지금 필요하지 않은 기능은 구현하지 않는다.
   - 테스트가 요구하지 않는 코드를 작성하지 않는다.
   - "나중에 필요할 것 같은" 추상화를 미리 만들지 않는다.
   - interface는 두 번째 구현체가 필요할 때 추출한다 (Commander는 예외 - 테스트 가능성을 위해 처음부터 interface).

4. **Fake It Till You Make It** - 하드코딩으로 시작하여 점진적으로 일반화한다.
   ```go
   // 첫 번째 테스트: 하드코딩으로 통과
   func Resolve() string { return "work" }

   // 두 번째 테스트 추가 후: 일반화
   func Resolve(owner string) string { ... }
   ```

5. **Triangulation** - 테스트 케이스를 추가하여 일반화를 유도한다.
   - 하나의 테스트만으로는 하드코딩으로 통과할 수 있다.
   - 두 번째, 세 번째 테스트를 추가하면서 자연스럽게 일반화된 구현이 나온다.

6. **Simple Design** - 동작하는 가장 단순한 것을 선택한다.
   - 중복 제거와 의도 표현에 집중한다.
   - 코드 라인 수가 적다고 단순한 것이 아니다. 읽기 쉬운 것이 단순한 것이다.

## 테스트 스타일

- table-driven tests를 기본으로 사용한다.
  ```go
  func TestResolve_OwnerRuleMatch(t *testing.T) {
      tests := []struct {
          name     string
          owner    string
          profiles []Profile
          want     string
          wantErr  bool
      }{
          // 테스트 케이스들...
      }
      for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) {
              // ...
          })
      }
  }
  ```

## 테스트 함수 네이밍

- 형식: `Test{Function}_{Scenario}`
- 예시:
  - `TestResolve_OwnerRuleMatch` - owner 규칙으로 매칭되는 경우
  - `TestResolve_AmbiguousProbe` - probe 결과가 모호한 경우
  - `TestLoadConfig_InvalidTOML` - 잘못된 TOML 파싱 실패
  - `TestGuardCheck_EmailMismatch` - 이메일 불일치 감지

## 테스트 유틸리티

- `testutil` 패키지의 헬퍼를 활용한다.
- mock Commander, 임시 디렉토리 생성 등 공통 기능은 testutil에 둔다.
- 테스트 간 상태를 공유하지 않는다. 각 테스트는 독립적이어야 한다.

## 커버리지

- 전체 커버리지 80% 이상을 유지한다.
- `make coverage`로 확인한다.
- 새 기능 추가 시 해당 패키지의 커버리지가 80% 미만이면 커밋하지 않는다.

## 외부 명령 테스트

- 외부 명령(`git`, `gh`, `ssh`)을 실행하는 코드는 Commander interface를 mock하여 테스트한다.
- 실제 외부 명령을 호출하는 통합 테스트는 별도 build tag(`//go:build integration`)로 분리한다.
