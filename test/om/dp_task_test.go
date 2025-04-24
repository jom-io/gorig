package om

import (
	"github.com/gin-gonic/gin"
	delpoy "github.com/jom-io/gorig/om/delploy/task"
	"testing"
)

func TestSaveTask(t *testing.T) {
	ctx := &gin.Context{}

	if e := delpoy.Task.Save(ctx, &delpoy.TaskOptions{
		Repo:   "git@github.com-jom:jom-io/gorig.git",
		Branch: "test",
		Auto:   true,
	}); e != nil {
		t.Errorf("Error: %v", e)
		return
	}
}
