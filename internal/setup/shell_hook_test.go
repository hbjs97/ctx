package setup

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectShell_Zsh(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	assert.Equal(t, "zsh", DetectShell())
}

func TestDetectShell_Bash(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/bash")
	assert.Equal(t, "bash", DetectShell())
}

func TestDetectShell_Fish(t *testing.T) {
	t.Setenv("SHELL", "/usr/local/bin/fish")
	assert.Equal(t, "fish", DetectShell())
}

func TestDetectShell_Unknown(t *testing.T) {
	t.Setenv("SHELL", "/bin/tcsh")
	assert.Equal(t, "tcsh", DetectShell())
}

func TestInstallShellHook_Zsh(t *testing.T) {
	dir := t.TempDir()
	rcPath := dir + "/.zshrc"

	err := InstallShellHook("zsh", rcPath)
	require.NoError(t, err)

	content, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "ctx shell integration")
	assert.Contains(t, string(content), "ctx activate")
}

func TestInstallShellHook_Bash(t *testing.T) {
	dir := t.TempDir()
	rcPath := dir + "/.bashrc"

	err := InstallShellHook("bash", rcPath)
	require.NoError(t, err)

	content, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "ctx shell integration")
}

func TestInstallShellHook_AlreadyInstalled(t *testing.T) {
	dir := t.TempDir()
	rcPath := dir + "/.zshrc"
	os.WriteFile(rcPath, []byte("# ctx shell integration (zsh)\nexisting content"), 0600)

	err := InstallShellHook("zsh", rcPath)
	require.NoError(t, err)

	content, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	// Should NOT have duplicate installations
	assert.Equal(t, "# ctx shell integration (zsh)\nexisting content", string(content))
}

func TestInstallShellHook_UnsupportedShell(t *testing.T) {
	dir := t.TempDir()
	rcPath := dir + "/.tcshrc"

	err := InstallShellHook("tcsh", rcPath)
	assert.Error(t, err)
}

func TestInstallShellHook_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	rcPath := dir + "/.zshrc"
	os.WriteFile(rcPath, []byte("# existing content\n"), 0600)

	err := InstallShellHook("zsh", rcPath)
	require.NoError(t, err)

	content, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# existing content")
	assert.Contains(t, string(content), "ctx shell integration")
}
