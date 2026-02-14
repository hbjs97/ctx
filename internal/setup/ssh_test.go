package setup

import (
	"os"
	"path/filepath"
	"testing"

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
