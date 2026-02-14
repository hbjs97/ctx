package git_test

import (
	"testing"
)

func TestParseRepoURL_SSH(t *testing.T) {
	t.Skip("not implemented")

	// Given: SSH URL "git@github-company:company-org/api-server.git"
	// When: ParseRepoURL is called
	// Then: returns RepoRef{Owner: "company-org", Repo: "api-server", Host: "github-company"}
}

func TestParseRepoURL_HTTPS(t *testing.T) {
	t.Skip("not implemented")

	// Given: HTTPS URL "https://github.com/hbjs97/dotfiles.git"
	// When: ParseRepoURL is called
	// Then: returns RepoRef{Owner: "hbjs97", Repo: "dotfiles", Host: "github.com"}
}

func TestParseRepoURL_Shorthand(t *testing.T) {
	t.Skip("not implemented")

	// Given: shorthand "company-org/api-server"
	// When: ParseRepoURL is called
	// Then: returns RepoRef{Owner: "company-org", Repo: "api-server"}
}

func TestParseRepoURL_Invalid(t *testing.T) {
	t.Skip("not implemented")

	// Given: invalid input "not-a-url"
	// When: ParseRepoURL is called
	// Then: returns error
}

func TestBuildSSHRemoteURL(t *testing.T) {
	t.Skip("not implemented")

	// Given: ssh_host "github-company", owner "company-org", repo "api-server"
	// When: BuildSSHRemoteURL is called
	// Then: returns "git@github-company:company-org/api-server.git"
}

func TestSetLocalGitConfig(t *testing.T) {
	t.Skip("not implemented")

	// Given: a git repository
	// When: SetLocalGitConfig(repo, "user.name", "HBJS") is called
	// Then: git config --local user.name returns "HBJS"
}

func TestGetRemoteURL(t *testing.T) {
	t.Skip("not implemented")

	// Given: a git repository with origin remote
	// When: GetRemoteURL(repo, "origin") is called
	// Then: returns the configured remote URL
}

func TestIsHTTPSRemote(t *testing.T) {
	t.Skip("not implemented")

	// Given: various remote URLs
	// When: IsHTTPSRemote is called
	// Then: correctly identifies HTTPS vs SSH URLs
}
