package cli_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/hbjs97/ctx/internal/cli"
	"github.com/stretchr/testify/assert"
)

func TestNewRootCmd_Help(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "GitHub")
}

func TestNewRootCmd_SubCommands(t *testing.T) {
	cmd := cli.NewRootCmd()
	subCmds := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subCmds[sub.Name()] = true
	}
	assert.True(t, subCmds["clone"])
	assert.True(t, subCmds["init"])
	assert.True(t, subCmds["status"])
	assert.True(t, subCmds["doctor"])
	assert.True(t, subCmds["guard"])
	assert.True(t, subCmds["activate"])
	assert.True(t, subCmds["setup"])
}

func TestMapExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want cli.ExitCode
	}{
		{"nil error", nil, cli.ExitSuccess},
		{"guard block", cli.ErrGuardBlock, cli.ExitGuardBlock},
		{"wrapped guard", fmt.Errorf("wrap: %w", cli.ErrGuardBlock), cli.ExitGuardBlock},
		{"ambiguous", cli.ErrAmbiguous, cli.ExitAmbiguous},
		{"wrapped ambiguous", fmt.Errorf("resolver: %w", cli.ErrAmbiguous), cli.ExitAmbiguous},
		{"auth fail", cli.ErrAuthFail, cli.ExitAuthFail},
		{"wrapped auth fail", fmt.Errorf("resolver: %w", cli.ErrAuthFail), cli.ExitAuthFail},
		{"config error", cli.ErrConfig, cli.ExitConfigError},
		{"wrapped config", fmt.Errorf("load: %w", cli.ErrConfig), cli.ExitConfigError},
		{"general", fmt.Errorf("unknown"), cli.ExitGeneral},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cli.MapExitCode(tt.err))
		})
	}
}
