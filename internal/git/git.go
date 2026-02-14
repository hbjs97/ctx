package git

import (
	"fmt"
	"net/url"
	"strings"
)

// RepoRef는 파싱된 리포지토리 참조다.
type RepoRef struct {
	Owner string
	Repo  string
	Host  string
}

// ParseRepoURL은 SSH/HTTPS/shorthand 형식의 리포 URL을 파싱한다.
func ParseRepoURL(raw string) (RepoRef, error) {
	if raw == "" {
		return RepoRef{}, fmt.Errorf("git.ParseRepoURL: 빈 입력")
	}
	if strings.HasPrefix(raw, "git@") {
		return parseSSH(raw)
	}
	if strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://") {
		return parseHTTPS(raw)
	}
	return parseShorthand(raw)
}

func parseSSH(raw string) (RepoRef, error) {
	withoutPrefix := strings.TrimPrefix(raw, "git@")
	parts := strings.SplitN(withoutPrefix, ":", 2)
	if len(parts) != 2 {
		return RepoRef{}, fmt.Errorf("git.ParseRepoURL: 잘못된 SSH URL: %s", raw)
	}
	owner, repo, err := splitOwnerRepo(strings.TrimSuffix(parts[1], ".git"))
	if err != nil {
		return RepoRef{}, err
	}
	return RepoRef{Owner: owner, Repo: repo, Host: parts[0]}, nil
}

func parseHTTPS(raw string) (RepoRef, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return RepoRef{}, fmt.Errorf("git.ParseRepoURL: %w", err)
	}
	path := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	owner, repo, err := splitOwnerRepo(path)
	if err != nil {
		return RepoRef{}, err
	}
	return RepoRef{Owner: owner, Repo: repo, Host: u.Host}, nil
}

func parseShorthand(raw string) (RepoRef, error) {
	owner, repo, err := splitOwnerRepo(raw)
	if err != nil {
		return RepoRef{}, err
	}
	return RepoRef{Owner: owner, Repo: repo}, nil
}

func splitOwnerRepo(s string) (string, string, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("git.ParseRepoURL: owner/repo 형식 아님: %s", s)
	}
	return parts[0], parts[1], nil
}

// BuildSSHRemoteURL은 SSH remote URL을 생성한다.
func BuildSSHRemoteURL(sshHost, owner, repo string) string {
	return fmt.Sprintf("git@%s:%s/%s.git", sshHost, owner, repo)
}

// IsHTTPSRemote는 URL이 HTTPS/HTTP인지 확인한다.
func IsHTTPSRemote(remoteURL string) bool {
	return strings.HasPrefix(remoteURL, "https://") || strings.HasPrefix(remoteURL, "http://")
}
