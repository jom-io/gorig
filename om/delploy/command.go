package delploy

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"os/exec"
	"strings"
)

func RunCommand(ctx *gin.Context, cmd string, args ...string) (string, *errors.Error) {
	logger.Info(ctx, fmt.Sprintf("Running command: %s %s", cmd, args))
	command := exec.Command(cmd, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &out
	command.Stderr = &stderr

	err := command.Run()
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			logger.Warn(ctx, "pkill: no process found")
		} else {
			return "", errors.Verify(fmt.Sprintf("Command failed: %s", err.Error()), err)
		}
	}

	result := out.String()
	//if result == "" {
	//	return "", errors.Verify("Command returned empty result")
	//}
	output := strings.Split(result, "\n")
	for _, line := range output {
		if strings.TrimSpace(line) != "" {
			logger.Info(ctx, line)
		}
	}
	return result, nil
}
