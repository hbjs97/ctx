package cli

import (
	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/guard"
	"github.com/hbjs97/ctx/internal/resolver"
)

// 각 도메인 패키지의 sentinel error를 CLI 레이어에서 편의상 re-export한다.
var (
	// ErrGuardBlock는 guard 검사 실패로 push가 차단될 때의 sentinel error다.
	ErrGuardBlock = guard.ErrGuardBlock
	// ErrAmbiguous는 복수 프로필이 매칭되어 자동 판정이 불가능할 때의 sentinel error다.
	ErrAmbiguous = resolver.ErrAmbiguous
	// ErrAuthFail는 접근 가능한 프로필이 없을 때의 sentinel error다.
	ErrAuthFail = resolver.ErrAuthFail
	// ErrConfig는 설정 파일 오류를 나타내는 sentinel error다.
	ErrConfig = config.ErrConfig
)
