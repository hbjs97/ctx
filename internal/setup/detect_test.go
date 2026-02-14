package setup

import (
	"context"
	"fmt"
	"testing"

	"github.com/hbjs97/ctx/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestDetectOrgs_Success(t *testing.T) {
	fc := testutil.NewFakeCommander()
	fc.Register("gh api user/orgs --jq .[].login", "company-org\ncompany-team\n", nil)
	fc.Register("gh api user --jq .login", "hbjs97\n", nil)

	orgs := DetectOrgs(context.Background(), fc, "/tmp/gh-work")
	assert.Equal(t, []string{"company-org", "company-team", "hbjs97"}, orgs)
}

func TestDetectOrgs_Failure(t *testing.T) {
	fc := testutil.NewFakeCommander()
	fc.Register("gh api user/orgs --jq .[].login", "", fmt.Errorf("auth required"))
	fc.Register("gh api user --jq .login", "", fmt.Errorf("auth required"))

	orgs := DetectOrgs(context.Background(), fc, "/tmp/gh-work")
	assert.Empty(t, orgs)
}

func TestDetectOrgs_PartialFailure(t *testing.T) {
	fc := testutil.NewFakeCommander()
	fc.Register("gh api user/orgs --jq .[].login", "", fmt.Errorf("forbidden"))
	fc.Register("gh api user --jq .login", "myuser\n", nil)

	orgs := DetectOrgs(context.Background(), fc, "/tmp/gh-work")
	assert.Equal(t, []string{"myuser"}, orgs)
}
