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

func TestSetupCmd_HasForceFlag(t *testing.T) {
	t.Parallel()

	app := &cli.App{
		Commander: testutil.NewFakeCommander(),
		CfgPath:   "/tmp/test-config.toml",
	}
	cmd := app.NewRootCmd()

	setupCmd, _, err := cmd.Find([]string{"setup"})
	require.NoError(t, err)
	assert.NotNil(t, setupCmd)

	flag := setupCmd.Flags().Lookup("force")
	assert.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}

func TestSetupCmd_ForceRemovesExistingConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")

	// Create an existing config file
	require.NoError(t, os.WriteFile(cfgPath, []byte("existing content"), 0600))

	// Verify it exists
	_, err := os.Stat(cfgPath)
	require.NoError(t, err)

	// Run setup --force (it will fail at TUI prompt, but the file should be removed first)
	app := &cli.App{
		Commander: testutil.NewFakeCommander(),
		CfgPath:   cfgPath,
	}
	cmd := app.NewRootCmd()
	cmd.SetArgs([]string{"setup", "--force", "--config", cfgPath})

	// The command will error because HuhFormRunner can't run in test,
	// but we can verify the --force flag removed the file
	_ = cmd.Execute()

	// File should have been removed by --force before Runner.Run() was called
	_, err = os.Stat(cfgPath)
	assert.True(t, os.IsNotExist(err), "config file should have been removed by --force")
}
