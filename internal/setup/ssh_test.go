package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestParseSSHConfig_NoFile(t *testing.T) {
	hosts := ParseSSHConfig("/nonexistent/path")
	assert.Empty(t, hosts)
}

func TestParseSSHConfig_ParsesGitHubHosts(t *testing.T) {
	content := `Host github.com-work
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519_work

Host github.com-personal
    HostName github.com
    User git
    IdentityFile ~/.ssh/id_ed25519_personal

Host other-server
    HostName example.com
`
	dir := t.TempDir()
	path := dir + "/config"
	os.WriteFile(path, []byte(content), 0600)

	hosts := ParseSSHConfig(path)
	assert.Equal(t, []string{"github.com-work", "github.com-personal"}, hosts)
}

func TestParseSSHConfig_IgnoresNonGitHub(t *testing.T) {
	content := `Host example.com
    HostName example.com
`
	dir := t.TempDir()
	path := dir + "/config"
	os.WriteFile(path, []byte(content), 0600)

	hosts := ParseSSHConfig(path)
	assert.Empty(t, hosts)
}

func TestParseSSHConfig_DetectsGitHubByHostName(t *testing.T) {
	content := `Host my-custom-alias
    HostName github.com
    User git
`
	dir := t.TempDir()
	path := dir + "/config"
	os.WriteFile(path, []byte(content), 0600)

	hosts := ParseSSHConfig(path)
	assert.Equal(t, []string{"my-custom-alias"}, hosts)
}

func TestDetectSSHKeys_FindsKeyPairs(t *testing.T) {
	dir := t.TempDir()

	// Valid key pair
	os.WriteFile(filepath.Join(dir, "id_ed25519_work"), []byte("private"), 0600)
	os.WriteFile(filepath.Join(dir, "id_ed25519_work.pub"), []byte("public"), 0644)

	// Orphan public key (should be ignored)
	os.WriteFile(filepath.Join(dir, "id_rsa_orphan.pub"), []byte("public"), 0644)

	// Non-key files (should be ignored)
	os.WriteFile(filepath.Join(dir, "config"), []byte("config"), 0644)
	os.WriteFile(filepath.Join(dir, "known_hosts"), []byte("hosts"), 0644)

	keys := DetectSSHKeys(dir)

	if len(keys) != 1 {
		t.Fatalf("expected 1 key pair, got %d", len(keys))
	}
	if keys[0].Name != "id_ed25519_work" {
		t.Errorf("expected name id_ed25519_work, got %s", keys[0].Name)
	}
	if keys[0].PrivateKey != filepath.Join(dir, "id_ed25519_work") {
		t.Errorf("unexpected private key path: %s", keys[0].PrivateKey)
	}
	if keys[0].PublicKey != filepath.Join(dir, "id_ed25519_work.pub") {
		t.Errorf("unexpected public key path: %s", keys[0].PublicKey)
	}
}

func TestDetectSSHKeys_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	keys := DetectSSHKeys(dir)
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys, got %d", len(keys))
	}
}

func TestDetectSSHKeys_NonExistentDir(t *testing.T) {
	keys := DetectSSHKeys("/nonexistent/path")
	if keys != nil {
		t.Fatalf("expected nil, got %v", keys)
	}
}

func TestDetectSSHKeys_MultipleKeys(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "id_ed25519"), []byte("p"), 0600)
	os.WriteFile(filepath.Join(dir, "id_ed25519.pub"), []byte("p"), 0644)
	os.WriteFile(filepath.Join(dir, "id_rsa_work"), []byte("p"), 0600)
	os.WriteFile(filepath.Join(dir, "id_rsa_work.pub"), []byte("p"), 0644)

	keys := DetectSSHKeys(dir)
	if len(keys) != 2 {
		t.Fatalf("expected 2 key pairs, got %d", len(keys))
	}
}

func TestGenerateSSHKey_Success(t *testing.T) {
	fc := testutil.NewFakeCommander()
	fc.Register("ssh-keygen", "", nil)

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519_test")

	err := GenerateSSHKey(context.Background(), fc, "test@example.com", keyPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !fc.Called("ssh-keygen") {
		t.Fatal("ssh-keygen was not called")
	}

	call := fc.Calls[0]
	if !strings.Contains(call, "-t ed25519") {
		t.Errorf("expected -t ed25519 in command, got: %s", call)
	}
	if !strings.Contains(call, "-C test@example.com") {
		t.Errorf("expected -C email in command, got: %s", call)
	}
	if !strings.Contains(call, "-f "+keyPath) {
		t.Errorf("expected -f keyPath in command, got: %s", call)
	}
}

func TestGenerateSSHKey_Failure(t *testing.T) {
	fc := testutil.NewFakeCommander()
	fc.Register("ssh-keygen", "error output", fmt.Errorf("exit status 1"))

	err := GenerateSSHKey(context.Background(), fc, "test@example.com", "/tmp/key")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWriteSSHConfigEntry_NewFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")

	err := WriteSSHConfigEntry(configPath, "github.com-work", "~/.ssh/id_ed25519_work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(configPath)
	s := string(content)
	if !strings.Contains(s, "Host github.com-work") {
		t.Error("missing Host line")
	}
	if !strings.Contains(s, "HostName github.com") {
		t.Error("missing HostName line")
	}
	if !strings.Contains(s, "User git") {
		t.Error("missing User line")
	}
	if !strings.Contains(s, "IdentityFile ~/.ssh/id_ed25519_work") {
		t.Error("missing IdentityFile line")
	}
	if !strings.Contains(s, "IdentitiesOnly yes") {
		t.Error("missing IdentitiesOnly line")
	}
}

func TestWriteSSHConfigEntry_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	os.WriteFile(configPath, []byte("Host existing\n  HostName example.com\n"), 0600)

	err := WriteSSHConfigEntry(configPath, "github.com-personal", "~/.ssh/id_ed25519_personal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(configPath)
	s := string(content)
	if !strings.Contains(s, "Host existing") {
		t.Error("existing entry was lost")
	}
	if !strings.Contains(s, "Host github.com-personal") {
		t.Error("new entry not appended")
	}
}

func TestWriteSSHConfigEntry_SkipsDuplicate(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	existing := "Host github.com-work\n  HostName github.com\n  User git\n  IdentityFile ~/.ssh/id_ed25519_work\n"
	os.WriteFile(configPath, []byte(existing), 0600)

	err := WriteSSHConfigEntry(configPath, "github.com-work", "~/.ssh/id_ed25519_work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(configPath)
	if count := strings.Count(string(content), "Host github.com-work"); count != 1 {
		t.Errorf("expected 1 occurrence, got %d", count)
	}
}
