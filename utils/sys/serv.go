package sys

import (
	configure "github.com/jom-io/gorig/utils/cofigure"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/rs/xid"
	"os"
	"strings"
)

var ServerID string
var RunMode Mode
var WorkingDirectory string

func Exit(err *errors.Error) {
	if err != nil {
		Error("Crash error: ", err.Error())
	}
	os.Exit(0)
}

func init() {
	ServerID = strings.ToUpper(xid.New().String())
	RunMode = ModeValueOf(configure.GetString("sys.mode"))
	wd, nErr := os.Getwd()
	if nErr != nil {
		Error("Get Current Working Directory Failed: ", nErr.Error())
		Exit(errors.Sys("Get Current Working Directory Failed!"))
		return
	}
	WorkingDirectory = wd
	Info("# Server ID: ", ServerID)
	Info("# Run Mode: ", strings.ToUpper(string(RunMode)))
}
