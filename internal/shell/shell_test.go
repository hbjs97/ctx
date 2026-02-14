package shell_test

import (
	"testing"

	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/shell"
	"github.com/stretchr/testify/assert"
)

func testProfile() *config.Profile {
	return &config.Profile{
		GHConfigDir: "/home/user/.config/gh-work",
		SSHHost:     "github-work",
		GitName:     "Test User",
		GitEmail:    "test@company.com",
	}
}

func TestActivate_PosixShell(t *testing.T) {
	output := shell.Activate("work", testProfile(), "zsh")
	assert.Contains(t, output, `export GH_CONFIG_DIR="/home/user/.config/gh-work"`)
	assert.Contains(t, output, `export CTX_PROFILE="work"`)
}

func TestActivate_Fish(t *testing.T) {
	output := shell.Activate("work", testProfile(), "fish")
	assert.Contains(t, output, `set -gx GH_CONFIG_DIR "/home/user/.config/gh-work"`)
	assert.Contains(t, output, `set -gx CTX_PROFILE "work"`)
}

func TestActivate_Bash(t *testing.T) {
	output := shell.Activate("work", testProfile(), "bash")
	assert.Contains(t, output, `export GH_CONFIG_DIR="/home/user/.config/gh-work"`)
	assert.Contains(t, output, `export CTX_PROFILE="work"`)
}

func TestDeactivate_PosixShell(t *testing.T) {
	output := shell.Deactivate("zsh")
	assert.Contains(t, output, "unset GH_CONFIG_DIR")
	assert.Contains(t, output, "unset CTX_PROFILE")
}

func TestDeactivate_Fish(t *testing.T) {
	output := shell.Deactivate("fish")
	assert.Contains(t, output, "set -e GH_CONFIG_DIR")
	assert.Contains(t, output, "set -e CTX_PROFILE")
}

func TestHookSnippet_Zsh(t *testing.T) {
	snippet := shell.HookSnippet("zsh")
	assert.Contains(t, snippet, "chpwd_functions")
	assert.Contains(t, snippet, "ctx activate")
}

func TestHookSnippet_Bash(t *testing.T) {
	snippet := shell.HookSnippet("bash")
	assert.Contains(t, snippet, "PROMPT_COMMAND")
	assert.Contains(t, snippet, "ctx activate")
}

func TestHookSnippet_Fish(t *testing.T) {
	snippet := shell.HookSnippet("fish")
	assert.Contains(t, snippet, "--on-variable PWD")
	assert.Contains(t, snippet, "ctx activate")
}

func TestHookSnippet_Unknown(t *testing.T) {
	snippet := shell.HookSnippet("unknown")
	assert.Empty(t, snippet)
}
