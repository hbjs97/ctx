package setup

import (
	"os"
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
