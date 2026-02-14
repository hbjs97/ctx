package cli

import (
	"errors"
)

// ExitCode는 ctx의 종료 코드다. TECH_SPEC 참조.
type ExitCode int

const (
	// ExitSuccess는 정상 종료다.
	ExitSuccess ExitCode = 0
	// ExitGeneral는 일반 에러다.
	ExitGeneral ExitCode = 1
	// ExitGuardBlock는 guard에 의한 push 차단이다.
	ExitGuardBlock ExitCode = 2
	// ExitAmbiguous는 모호한 프로필 판정이다.
	ExitAmbiguous ExitCode = 3
	// ExitAuthFail는 인증 실패다.
	ExitAuthFail ExitCode = 4
	// ExitConfigError는 설정 파일 오류다.
	ExitConfigError ExitCode = 5
)

// MapExitCode는 sentinel error를 기반으로 적절한 종료 코드를 반환한다.
func MapExitCode(err error) ExitCode {
	if err == nil {
		return ExitSuccess
	}
	switch {
	case errors.Is(err, ErrGuardBlock):
		return ExitGuardBlock
	case errors.Is(err, ErrAmbiguous):
		return ExitAmbiguous
	case errors.Is(err, ErrAuthFail):
		return ExitAuthFail
	case errors.Is(err, ErrConfig):
		return ExitConfigError
	default:
		return ExitGeneral
	}
}
