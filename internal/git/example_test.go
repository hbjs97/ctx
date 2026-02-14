package git_test

import (
	"context"
	"testing"

	"github.com/hbjs97/ctx/internal/git"
	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    git.RepoRef
		wantErr bool
	}{
		{name: "ssh custom host", input: "git@github-company:company-org/api-server.git",
			want: git.RepoRef{Owner: "company-org", Repo: "api-server", Host: "github-company"}},
		{name: "ssh github.com", input: "git@github.com:hbjs97/dotfiles.git",
			want: git.RepoRef{Owner: "hbjs97", Repo: "dotfiles", Host: "github.com"}},
		{name: "https with .git", input: "https://github.com/hbjs97/dotfiles.git",
			want: git.RepoRef{Owner: "hbjs97", Repo: "dotfiles", Host: "github.com"}},
		{name: "https without .git", input: "https://github.com/hbjs97/dotfiles",
			want: git.RepoRef{Owner: "hbjs97", Repo: "dotfiles", Host: "github.com"}},
		{name: "shorthand", input: "company-org/api-server",
			want: git.RepoRef{Owner: "company-org", Repo: "api-server"}},
		{name: "invalid single word", input: "not-a-url", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.ParseRepoURL(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildSSHRemoteURL(t *testing.T) {
	got := git.BuildSSHRemoteURL("github-company", "company-org", "api-server")
	assert.Equal(t, "git@github-company:company-org/api-server.git", got)
}

func TestIsHTTPSRemote(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"https://github.com/o/r.git", true},
		{"http://github.com/o/r.git", true},
		{"git@github.com:o/r.git", false},
		{"git@github-work:o/r.git", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, git.IsHTTPSRemote(tt.input))
		})
	}
}

func TestAdapter_Clone(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("git clone", "", nil)

	a := git.NewAdapter(fake)
	err := a.Clone(context.Background(), "git@host:o/r.git", "/tmp/dest")

	assert.NoError(t, err)
	assert.True(t, fake.Called("git clone"))
}

func TestAdapter_SetLocalConfig(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("git -C", "", nil)

	a := git.NewAdapter(fake)
	err := a.SetLocalConfig(context.Background(), "/tmp/repo", "user.name", "Test")

	assert.NoError(t, err)
	assert.True(t, fake.Called("git -C"))
}

func TestAdapter_GetRemoteURL(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("git -C", "git@github-work:org/repo.git\n", nil)

	a := git.NewAdapter(fake)
	u, err := a.GetRemoteURL(context.Background(), "/tmp/repo", "origin")

	require.NoError(t, err)
	assert.Equal(t, "git@github-work:org/repo.git", u)
}

func TestAdapter_SetRemoteURL(t *testing.T) {
	fake := testutil.NewFakeCommander()
	fake.Register("git -C", "", nil)

	a := git.NewAdapter(fake)
	err := a.SetRemoteURL(context.Background(), "/tmp/repo", "origin", "git@new:o/r.git")

	assert.NoError(t, err)
	assert.True(t, fake.Called("git -C"))
}
