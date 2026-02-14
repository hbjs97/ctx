package git

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hbjs97/ctx/internal/cmdexec"
)

// RepoRef는 파싱된 리포지토리 참조다.
type RepoRef struct {
	Owner string
	Repo  string
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
	return RepoRef{Owner: owner, Repo: repo}, nil
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
	return RepoRef{Owner: owner, Repo: repo}, nil
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

// Adapter는 git CLI를 Commander를 통해 실행한다.
type Adapter struct {
	cmd cmdexec.Commander
}

// NewAdapter는 새 Git Adapter를 생성한다.
func NewAdapter(cmd cmdexec.Commander) *Adapter {
	return &Adapter{cmd: cmd}
}

// Clone은 리포를 클론한다.
func (a *Adapter) Clone(ctx context.Context, remoteURL, dir string) error {
	if _, err := a.cmd.Run(ctx, "git", "clone", remoteURL, dir); err != nil {
		return fmt.Errorf("git.Clone: %w", err)
	}
	return nil
}

// SetLocalConfig는 리포 로컬 git config를 설정한다.
func (a *Adapter) SetLocalConfig(ctx context.Context, repoDir, key, value string) error {
	if _, err := a.cmd.Run(ctx, "git", "-C", repoDir, "config", "--local", key, value); err != nil {
		return fmt.Errorf("git.SetLocalConfig: %w", err)
	}
	return nil
}

// GetRemoteURL은 remote URL을 반환한다.
func (a *Adapter) GetRemoteURL(ctx context.Context, repoDir, remoteName string) (string, error) {
	out, err := a.cmd.Run(ctx, "git", "-C", repoDir, "remote", "get-url", remoteName)
	if err != nil {
		return "", fmt.Errorf("git.GetRemoteURL: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// SetRemoteURL은 remote URL을 변경한다.
func (a *Adapter) SetRemoteURL(ctx context.Context, repoDir, remoteName, newURL string) error {
	if _, err := a.cmd.Run(ctx, "git", "-C", repoDir, "remote", "set-url", remoteName, newURL); err != nil {
		return fmt.Errorf("git.SetRemoteURL: %w", err)
	}
	return nil
}
