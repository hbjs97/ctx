// Package cmdexec는 테스트 가능성을 위해 외부 명령 실행을 추상화한다.
// 프로덕션 코드는 Commander interface를 사용하고, 테스트는 testutil.FakeCommander를 주입한다.
package cmdexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Commander는 외부 명령 실행을 추상화하는 interface다.
type Commander interface {
	// Run은 외부 명령을 실행하고 combined output을 반환한다.
	Run(ctx context.Context, name string, args ...string) ([]byte, error)

	// RunWithEnv는 추가 환경변수를 현재 프로세스 환경에 병합하여 외부 명령을 실행한다.
	RunWithEnv(ctx context.Context, env map[string]string, name string, args ...string) ([]byte, error)

	// RunInteractiveWithEnv는 터미널 stdin/stdout/stderr를 직접 연결하여 대화형 명령을 실행한다.
	// gh auth login 등 사용자 입력이 필요한 명령에 사용한다.
	RunInteractiveWithEnv(ctx context.Context, env map[string]string, name string, args ...string) error
}

// RealCommander는 os/exec를 통해 실제 외부 명령을 실행한다.
type RealCommander struct{}

// Run은 os/exec.CommandContext를 사용하여 명령을 실행한다.
func (c *RealCommander) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// RunWithEnv는 추가 환경변수를 병합하여 명령을 실행한다.
func (c *RealCommander) RunWithEnv(ctx context.Context, env map[string]string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), mapToEnvSlice(env)...)
	return cmd.CombinedOutput()
}

// RunInteractiveWithEnv는 터미널에 직접 연결하여 대화형 명령을 실행한다.
func (c *RealCommander) RunInteractiveWithEnv(ctx context.Context, env map[string]string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), mapToEnvSlice(env)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// mapToEnvSlice는 환경변수 맵을 "KEY=VALUE" 문자열 슬라이스로 변환한다.
func mapToEnvSlice(env map[string]string) []string {
	if env == nil {
		return nil
	}
	result := make([]string, 0, len(env))
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}
