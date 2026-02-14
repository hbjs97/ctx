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

// ParseSSHConfigIdentityFiles는 SSH config 파일에서 Host → IdentityFile 매핑을 추출한다.
// 모든 Host 블록을 파싱하며, IdentityFile이 없는 블록은 무시한다.
func ParseSSHConfigIdentityFiles(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	result := make(map[string]string)
	var currentHost string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "Host ") && !strings.Contains(line, "*") {
			currentHost = strings.TrimSpace(strings.TrimPrefix(line, "Host "))
		}

		if currentHost != "" && strings.HasPrefix(line, "IdentityFile") {
			idFile := strings.TrimSpace(strings.TrimPrefix(line, "IdentityFile"))
			result[currentHost] = idFile
		}
	}

	return result
}

// FilterUsedSSHKeys는 이미 다른 프로필에서 사용 중인 키를 제외한다.
func FilterUsedSSHKeys(keys []SSHKeyInfo, usedPaths map[string]bool) []SSHKeyInfo {
	var filtered []SSHKeyInfo
	for _, k := range keys {
		if !usedPaths[k.PrivateKey] {
			filtered = append(filtered, k)
		}
	}
	return filtered
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

// WriteSSHConfigEntry는 SSH config 파일에 GitHub Host alias 블록을 추가한다.
// 동일한 Host가 이미 존재하면 스킵한다. 파일이 없으면 생성한다.
func WriteSSHConfigEntry(configPath, host, identityFile string) error {
	existing, _ := os.ReadFile(configPath)
	hostLine := "Host " + host
	if strings.Contains(string(existing), hostLine) {
		return nil // 이미 존재
	}

	entry := fmt.Sprintf("\nHost %s\n  HostName github.com\n  User git\n  IdentityFile %s\n  IdentitiesOnly yes\n", host, identityFile)

	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("setup.WriteSSHConfigEntry: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("setup.WriteSSHConfigEntry: %w", err)
	}
	return nil
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
