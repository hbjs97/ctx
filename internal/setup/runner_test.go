package setup

import (
	"context"
	"fmt"
	"testing"

	"github.com/hbjs97/ctx/internal/config"
	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFormRunner는 테스트용 FormRunner다.
type mockFormRunner struct {
	profileInputs   []*ProfileInput
	profileIdx      int
	action          Action
	selectedProfile string
	confirms        []bool
	confirmIdx      int
	addMore         []bool
	addMoreIdx      int
	sshHost         string
	owners          []string
}

func (m *mockFormRunner) RunProfileForm(defaults *ProfileInput, existingNames []string) (*ProfileInput, error) {
	if m.profileIdx >= len(m.profileInputs) {
		return nil, fmt.Errorf("no more profile inputs")
	}
	p := m.profileInputs[m.profileIdx]
	m.profileIdx++
	return p, nil
}

func (m *mockFormRunner) RunActionSelect(profileNames []string) (Action, error) {
	return m.action, nil
}

func (m *mockFormRunner) RunProfileSelect(profileNames []string) (string, error) {
	return m.selectedProfile, nil
}

func (m *mockFormRunner) RunConfirm(message string) (bool, error) {
	if m.confirmIdx >= len(m.confirms) {
		return false, nil
	}
	c := m.confirms[m.confirmIdx]
	m.confirmIdx++
	return c, nil
}

func (m *mockFormRunner) RunAddMore() (bool, error) {
	if m.addMoreIdx >= len(m.addMore) {
		return false, nil
	}
	a := m.addMore[m.addMoreIdx]
	m.addMoreIdx++
	return a, nil
}

func (m *mockFormRunner) RunSSHHostSelect(hosts []string) (string, error) {
	return m.sshHost, nil
}

func (m *mockFormRunner) RunOwnersSelect(detected []string) ([]string, error) {
	return m.owners, nil
}

func (m *mockFormRunner) RunSSHKeySelect(existingKeys []SSHKeyInfo, profileName string) (SSHKeyChoice, error) {
	return SSHKeyChoice{Action: "skip"}, nil
}

// registerDoctorCommands는 doctor.RunAll이 호출하는 명령어 응답을 등록한다.
func registerDoctorCommands(fc *testutil.FakeCommander) {
	fc.Register("git --version", "git version 2.40.0", nil)
	fc.Register("gh --version", "gh version 2.40.0", nil)
	fc.Register("ssh -V", "OpenSSH_9.0", nil)
	fc.Register("gh auth status", "Logged in", nil)
	fc.Register("ssh -T", "Hi user!", fmt.Errorf("exit status 1: Hi user!"))
}

func TestRunner_FirstRun_SingleProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/config.toml"

	fc := testutil.NewFakeCommander()
	fc.Register("gh auth login --hostname github.com", "ok", nil)
	fc.Register("gh api user/orgs --jq .[].login", "my-org\n", nil)
	fc.Register("gh api user --jq .login", "myuser\n", nil)
	registerDoctorCommands(fc)

	mock := &mockFormRunner{
		profileInputs: []*ProfileInput{{
			Name: "work", GitName: "Test", GitEmail: "test@work.com",
		}},
		sshHost: "github.com-work",
		owners:  []string{"my-org", "myuser"},
		addMore: []bool{false},
	}

	r := &Runner{
		CfgPath:    cfgPath,
		Commander:  fc,
		FormRunner: mock,
	}

	err := r.Run(context.Background())
	require.NoError(t, err)

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 1)
	assert.Equal(t, "test@work.com", cfg.Profiles["work"].GitEmail)
	assert.Equal(t, "github.com-work", cfg.Profiles["work"].SSHHost)
	assert.Equal(t, []string{"my-org", "myuser"}, cfg.Profiles["work"].Owners)
}

func TestRunner_FirstRun_MultipleProfiles(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/config.toml"

	fc := testutil.NewFakeCommander()
	fc.Register("gh auth login --hostname github.com", "ok", nil)
	fc.Register("gh api user/orgs --jq .[].login", "org1\n", nil)
	fc.Register("gh api user --jq .login", "user1\n", nil)
	registerDoctorCommands(fc)

	mock := &mockFormRunner{
		profileInputs: []*ProfileInput{
			{Name: "work", GitName: "Work", GitEmail: "work@co.com"},
			{Name: "personal", GitName: "Personal", GitEmail: "me@personal.com"},
		},
		sshHost: "github.com-work",
		owners:  []string{"org1", "user1"},
		addMore: []bool{true, false}, // add one more, then stop
	}

	r := &Runner{
		CfgPath:    cfgPath,
		Commander:  fc,
		FormRunner: mock,
	}

	err := r.Run(context.Background())
	require.NoError(t, err)

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 2)
	assert.Equal(t, "work@co.com", cfg.Profiles["work"].GitEmail)
	assert.Equal(t, "me@personal.com", cfg.Profiles["personal"].GitEmail)
}

func TestRunner_Existing_AddProfile(t *testing.T) {
	cfgPath := testutil.SetupTestProfiles(t)

	fc := testutil.NewFakeCommander()
	fc.Register("gh auth login --hostname github.com", "ok", nil)
	fc.Register("gh api user/orgs --jq .[].login", "new-org\n", nil)
	fc.Register("gh api user --jq .login", "newuser\n", nil)
	registerDoctorCommands(fc)

	mock := &mockFormRunner{
		action: ActionAdd,
		profileInputs: []*ProfileInput{{
			Name: "freelance", GitName: "Freelance", GitEmail: "free@example.com",
		}},
		sshHost: "github.com-freelance",
		owners:  []string{"new-org"},
	}

	r := &Runner{CfgPath: cfgPath, Commander: fc, FormRunner: mock}
	err := r.Run(context.Background())
	require.NoError(t, err)

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 3) // work + personal + freelance
	assert.Equal(t, "free@example.com", cfg.Profiles["freelance"].GitEmail)
}

func TestRunner_Existing_EditProfile(t *testing.T) {
	cfgPath := testutil.SetupTestProfiles(t)

	fc := testutil.NewFakeCommander()
	registerDoctorCommands(fc)

	mock := &mockFormRunner{
		action:          ActionEdit,
		selectedProfile: "work",
		profileInputs: []*ProfileInput{{
			Name: "work", GitName: "New Name", GitEmail: "new@company.com",
			SSHHost: "github-company", Owners: []string{"company-org"},
		}},
	}

	r := &Runner{CfgPath: cfgPath, Commander: fc, FormRunner: mock}
	err := r.Run(context.Background())
	require.NoError(t, err)

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "new@company.com", cfg.Profiles["work"].GitEmail)
	assert.Equal(t, "New Name", cfg.Profiles["work"].GitName)
}

func TestRunner_Existing_DeleteProfile(t *testing.T) {
	cfgPath := testutil.SetupTestProfiles(t)

	fc := testutil.NewFakeCommander()

	mock := &mockFormRunner{
		action:          ActionDelete,
		selectedProfile: "work",
		confirms:        []bool{true},
	}

	r := &Runner{CfgPath: cfgPath, Commander: fc, FormRunner: mock}
	err := r.Run(context.Background())
	require.NoError(t, err)

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 1) // personal만 남음
	_, exists := cfg.Profiles["work"]
	assert.False(t, exists)
}

func TestRunner_Existing_DeleteLastProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/config.toml"
	cfg := &config.Config{
		Version: 1,
		Profiles: map[string]config.Profile{
			"only": {
				GHConfigDir: "/tmp/gh-only", SSHHost: "github.com-only",
				GitName: "Only", GitEmail: "only@test.com", Owners: []string{"only-org"},
			},
		},
	}
	require.NoError(t, config.Save(cfgPath, cfg))

	fc := testutil.NewFakeCommander()
	mock := &mockFormRunner{
		action:          ActionDelete,
		selectedProfile: "only",
		confirms:        []bool{true},
	}

	r := &Runner{CfgPath: cfgPath, Commander: fc, FormRunner: mock}
	err := r.Run(context.Background())
	assert.Error(t, err) // 마지막 프로필 삭제 불가
}

func TestRunner_FirstRun_GHAuthFails(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/config.toml"

	fc := testutil.NewFakeCommander()
	fc.Register("gh auth login --hostname github.com", "", fmt.Errorf("auth failed"))
	fc.Register("gh api user/orgs --jq .[].login", "", fmt.Errorf("no auth"))
	fc.Register("gh api user --jq .login", "", fmt.Errorf("no auth"))
	registerDoctorCommands(fc)

	mock := &mockFormRunner{
		profileInputs: []*ProfileInput{{
			Name: "work", GitName: "Test", GitEmail: "test@work.com",
		}},
		sshHost: "github.com-work",
		owners:  []string{"manual-org"},
		addMore: []bool{false},
	}

	r := &Runner{
		CfgPath:    cfgPath,
		Commander:  fc,
		FormRunner: mock,
	}

	err := r.Run(context.Background())
	require.NoError(t, err) // gh auth failure is non-fatal

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 1)
}

func TestRunner_FirstRun_RunsDoctorAfterSetup(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/config.toml"

	fc := testutil.NewFakeCommander()
	fc.Register("gh auth login --hostname github.com", "ok", nil)
	fc.Register("gh api user/orgs --jq .[].login", "my-org\n", nil)
	fc.Register("gh api user --jq .login", "myuser\n", nil)
	registerDoctorCommands(fc)

	mock := &mockFormRunner{
		profileInputs: []*ProfileInput{{
			Name: "work", GitName: "Test", GitEmail: "test@work.com",
		}},
		sshHost: "github.com-work",
		owners:  []string{"my-org"},
		addMore: []bool{false},
	}

	r := &Runner{
		CfgPath:    cfgPath,
		Commander:  fc,
		FormRunner: mock,
	}

	err := r.Run(context.Background())
	require.NoError(t, err)

	// doctor 명령이 실행되었는지 확인
	assert.True(t, fc.Called("git --version"))
	assert.True(t, fc.Called("gh auth status"))
}
