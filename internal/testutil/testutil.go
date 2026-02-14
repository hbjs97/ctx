// Package testutil은 ctx 프로젝트의 공통 테스트 헬퍼를 제공한다.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TempGitRepo는 임시 git 리포를 생성하고 경로를 반환한다.
// 테스트 종료 시 자동으로 정리된다.
func TempGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.name", "test-user"},
		{"git", "config", "user.email", "test@example.com"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("TempGitRepo: %s failed: %v\n%s", args[0], err, out)
		}
	}

	return dir
}

// TempGitRepoWithRemote는 지정된 URL로 remote가 설정된 임시 git 리포를 생성한다.
func TempGitRepoWithRemote(t *testing.T, remoteURL string) string {
	t.Helper()

	dir := TempGitRepo(t)

	cmd := exec.Command("git", "remote", "add", "origin", remoteURL)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("TempGitRepoWithRemote: git remote add failed: %v\n%s", err, out)
	}

	return dir
}

// TempBareRepo는 임시 bare git 리포를 생성하고 경로를 반환한다.
// push 대상이 필요한 E2E 테스트에 유용하다.
func TempBareRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("TempBareRepo: git init --bare failed: %v\n%s", err, out)
	}

	return dir
}

// TempConfigFile은 주어진 내용으로 임시 config.toml을 생성하고 경로를 반환한다.
// 파일은 자동으로 정리된다.
func TempConfigFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("TempConfigFile: write failed: %v", err)
	}

	return path
}

// TempCacheFile은 주어진 내용으로 임시 cache.json을 생성하고 경로를 반환한다.
func TempCacheFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("TempCacheFile: write failed: %v", err)
	}

	return path
}

// SetupTestProfiles는 work/personal 2개 프로필이 설정된 임시 config.toml을 생성한다.
// config 파일 경로를 반환한다.
func SetupTestProfiles(t *testing.T) string {
	t.Helper()

	content := `version = 1
default_profile = "personal"
prompt_on_ambiguous = true
require_push_guard = true
allow_https_managed_repo = false

[profiles.work]
gh_config_dir = "/tmp/gh-work"
ssh_host = "github-company"
git_name = "HBJS"
git_email = "hbjs@company.com"
owners = ["company-org", "company-team"]

[profiles.personal]
gh_config_dir = "/tmp/gh-personal"
ssh_host = "github-personal"
git_name = "hbjs97"
git_email = "hbjs97@naver.com"
owners = ["hbjs97", "sutefu23"]
`
	return TempConfigFile(t, content)
}

// WriteCtxProfile은 주어진 git 리포의 .git 디렉토리에 ctx-profile 파일을 기록한다.
func WriteCtxProfile(t *testing.T, repoDir string, profileName string) {
	t.Helper()

	profilePath := filepath.Join(repoDir, ".git", "ctx-profile")
	if err := os.WriteFile(profilePath, []byte(profileName+"\n"), 0600); err != nil {
		t.Fatalf("WriteCtxProfile: write failed: %v", err)
	}
}

// ReadCtxProfile은 주어진 git 리포의 .git 디렉토리에서 ctx-profile을 읽는다.
func ReadCtxProfile(t *testing.T, repoDir string) string {
	t.Helper()

	profilePath := filepath.Join(repoDir, ".git", "ctx-profile")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("ReadCtxProfile: read failed: %v", err)
	}

	return string(data)
}
