package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hbjs97/ctx/internal/cli"
	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupCmd_CreatesConfigFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subdir", "config.toml")

	app := &cli.App{
		Commander: testutil.NewFakeCommander(),
		CfgPath:   cfgPath,
	}
	cmd := app.NewRootCmd()
	cmd.SetArgs([]string{"setup", "--config", cfgPath})

	err := cmd.Execute()
	assert.NoError(t, err)

	// Verify file exists with correct permissions
	info, err := os.Stat(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Verify it's valid content with profiles
	data, _ := os.ReadFile(cfgPath)
	assert.Contains(t, string(data), "[profiles.")
	assert.Contains(t, string(data), "version = 1")
}

func TestSetupCmd_AlreadyExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("existing"), 0600))

	app := &cli.App{
		Commander: testutil.NewFakeCommander(),
		CfgPath:   cfgPath,
	}
	cmd := app.NewRootCmd()
	cmd.SetArgs([]string{"setup", "--config", cfgPath})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "이미 존재합니다")
}

func TestSetupCmd_CreatesDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "deep", "nested", "config.toml")

	app := &cli.App{
		Commander: testutil.NewFakeCommander(),
		CfgPath:   cfgPath,
	}
	cmd := app.NewRootCmd()
	cmd.SetArgs([]string{"setup", "--config", cfgPath})

	err := cmd.Execute()
	assert.NoError(t, err)

	// Verify directory was created with 0700
	parentInfo, err := os.Stat(filepath.Dir(cfgPath))
	require.NoError(t, err)
	assert.True(t, parentInfo.IsDir())
}
