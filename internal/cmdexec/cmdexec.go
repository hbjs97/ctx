// Package cmdexec abstracts external command execution for testability.
// Production code uses Commander interface; tests inject FakeCommander from testutil.
package cmdexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Commander abstracts external command execution.
type Commander interface {
	// Run executes an external command and returns its combined output.
	Run(ctx context.Context, name string, args ...string) ([]byte, error)

	// RunWithEnv executes an external command with additional environment variables
	// merged on top of the current process environment.
	RunWithEnv(ctx context.Context, env map[string]string, name string, args ...string) ([]byte, error)
}

// RealCommander executes actual external commands via os/exec.
type RealCommander struct{}

// Run executes the command using os/exec.CommandContext.
func (c *RealCommander) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// RunWithEnv executes the command with additional environment variables.
// The provided env map is merged on top of the current process environment.
func (c *RealCommander) RunWithEnv(ctx context.Context, env map[string]string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), mapToEnvSlice(env)...)
	return cmd.CombinedOutput()
}

// mapToEnvSlice converts a map of environment variables to a slice of "KEY=VALUE" strings.
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
