package om

import (
	"github.com/gin-gonic/gin"
	delpoy "github.com/jom-io/gorig/om/delploy/app"
	"testing"
)

func TestRestartApp(t *testing.T) {
	ctx := &gin.Context{}

	defer func() {
		if e := delpoy.App.Clean(ctx); e != nil {
			t.Errorf("Error: %v", e)
			return
		}
	}()

	if e := delpoy.App.Restart(ctx); e != nil {
		t.Errorf("Error: %v", e)
		return
	}

}

func TestStopApp(t *testing.T) {
	ctx := &gin.Context{}

	defer func() {
		if e := delpoy.App.Clean(ctx); e != nil {
			t.Errorf("Error: %v", e)
			return
		}
	}()

	if e := delpoy.App.Stop(ctx); e != nil {
		t.Errorf("Error: %v", e)
		return
	}
}
