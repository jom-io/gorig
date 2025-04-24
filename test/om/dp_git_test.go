package om

import (
	"github.com/gin-gonic/gin"
	delpoy "github.com/jom-io/gorig/om/delploy/git"
	"testing"
)

func TestCheckVersion(t *testing.T) {
	ctx := gin.Context{}

	git, e := delpoy.Git.CheckGit(&ctx)
	if e != nil {
		t.Errorf("Error: %v", e)
		return
	}
	if !git {
		t.Error("Git is not available")
		return
	}
	t.Log("Git is available")
}

// SetRepo
func TestSetRepo(t *testing.T) {
	ctx := &gin.Context{}
	if e := delpoy.Git.SetRepo(ctx, "git@github.com-jom:jom-io/gorig.git"); e != nil {
		t.Errorf("Error: %v", e)
		return
	}
}

// GetSSHKey
func TestGetSSHKey(t *testing.T) {
	ctx := &gin.Context{}

	if sshKey, e := delpoy.Git.GetSSHKey(ctx); e != nil {
		t.Errorf("Error: %v", e)
		return
	} else if sshKey != "" {
		t.Logf("SSH Key: %s", sshKey)
		return
	} else {
		t.Log("SSH Key is empty")
		return
	}
}

// ListBranches
func TestListBranches(t *testing.T) {
	ctx := &gin.Context{}

	if branches, e := delpoy.Git.ListBranches(ctx); e != nil {
		t.Errorf("Error: %v", e)
		return
	} else if len(branches) == 0 {
		t.Log("No branches found")
		return
	} else {
		t.Logf("Branches: %v", branches)
		return
	}
}
