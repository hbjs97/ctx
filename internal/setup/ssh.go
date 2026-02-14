package setup

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hbjs97/ctx/internal/cmdexec"
)

// ParseSSHConfig는 SSH config 파일에서 GitHub 관련 Host alias 목록을 추출한다.
// 파일이 없거나 파싱 실패 시 빈 슬라이스를 반환한다.
func ParseSSHConfig(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var hosts []string
	var currentHost string
	var isGitHub bool

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "Host ") && !strings.Contains(line, "*") {
			// 이전 Host 블록 처리
			if currentHost != "" && isGitHub {
				hosts = append(hosts, currentHost)
			}
			currentHost = strings.TrimPrefix(line, "Host ")
			currentHost = strings.TrimSpace(currentHost)
			// Host 이름에 github이 포함되면 GitHub으로 간주
			isGitHub = strings.Contains(strings.ToLower(currentHost), "github")
		}

		if strings.HasPrefix(line, "HostName") {
			hostName := strings.TrimSpace(strings.TrimPrefix(line, "HostName"))
			if strings.Contains(hostName, "github.com") {
				isGitHub = true
			}
		}
	}

	// 마지막 Host 블록 처리
	if currentHost != "" && isGitHub {
		hosts = append(hosts, currentHost)
	}

	return hosts
}

// DetectSSHKeys는 주어진 디렉토리에서 SSH 키 쌍을 감지한다.
// id_* 패턴의 비밀키 파일과 대응하는 .pub 파일이 모두 존재해야 유효한 키 쌍이다.
func DetectSSHKeys(sshDir string) []SSHKeyInfo {
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil
	}

	var keys []SSHKeyInfo
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasSuffix(name, ".pub") || !strings.HasPrefix(name, "id_") {
			continue
		}
		pubPath := filepath.Join(sshDir, name+".pub")
		if _, err := os.Stat(pubPath); err != nil {
			continue
		}
		keys = append(keys, SSHKeyInfo{
			Name:       name,
			PrivateKey: filepath.Join(sshDir, name),
			PublicKey:  pubPath,
		})
	}
	return keys
}

// DefaultSSHConfigPath는 기본 SSH config 경로를 반환한다.
func DefaultSSHConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

// GenerateSSHKey는 ssh-keygen으로 ed25519 키 쌍을 생성한다.
// 빈 passphrase로 생성하며, Commander를 통해 실행한다.
func GenerateSSHKey(ctx context.Context, cmd cmdexec.Commander, email, keyPath string) error {
	_, err := cmd.Run(ctx, "ssh-keygen", "-t", "ed25519", "-C", email, "-f", keyPath, "-N", "")
	if err != nil {
		return fmt.Errorf("setup.GenerateSSHKey: %w", err)
	}
	return nil
}
