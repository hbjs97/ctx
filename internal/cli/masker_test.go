package cli_test

import (
	"testing"

	"github.com/hbjs97/ctx/internal/cli"
	"github.com/stretchr/testify/assert"
)

func TestMaskTokens(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"ghp token", "token: ghp_abc123def456ghi789", "token: ghp_****"},
		{"gho token", "auth gho_secretvalue here", "auth gho_**** here"},
		{"github_pat", "github_pat_abcdef1234567890", "github_pat_****"},
		{"ghs token", "ghs_servertoken123", "ghs_****"},
		{"ghu token", "ghu_usertoken456", "ghu_****"},
		{"no token", "hello world", "hello world"},
		{"multiple tokens", "ghp_aaa and gho_bbb", "ghp_**** and gho_****"},
		{"empty string", "", ""},
		{"prefix only at boundary", "ghp_ next", "ghp_ next"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, cli.MaskTokens(tt.input))
		})
	}
}
