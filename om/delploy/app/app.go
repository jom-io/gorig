package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/global/variable"
	"github.com/jom-io/gorig/om/delploy"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/sys"
	"os"
	"strings"
	"time"
)

var App appService

type appService struct {
}

func init() {
	App = appService{}
}

func getRunFileName() (string, *errors.Error) {
	runFile := fmt.Sprintf("%s-%s.linux64", variable.SysName, sys.RunMode)
	runFile = strings.ToLower(runFile)
	runFile = strings.ReplaceAll(runFile, "_", "-")
	if _, err := os.Stat(runFile); os.IsNotExist(err) {
		return "", errors.Verify("Run file not found", err)
	}

	fileInfo, err := os.Stat(runFile)
	if err != nil {
		return "", errors.Verify("Failed to get file info", err)
	}
	if fileInfo.Mode()&0111 == 0 {
		if err := os.Chmod(runFile, 0755); err != nil {
			return "", errors.Verify("Failed to change file permissions", err)
		}
		logger.Info(nil, "File permissions changed to executable")
	} else {
		logger.Info(nil, "File is already executable")
	}

	return runFile, nil
}

func (a appService) Restart(ctx *gin.Context) *errors.Error {
	logger.Info(ctx, "Restarting application...")

	runFile, e := getRunFileName()
	if e != nil {
		return e
	}

	restartFile := "restart.sh"
	//if _, err := os.Stat(restartFile); os.IsNotExist(err) {
	//file, errC := os.Create(restartFile)
	//if errC != nil {
	//	return "", errors.Verify("Failed to create restart.sh file", errC)
	//}
	//defer file.Close()

	content := fmt.Sprintf(`#!/bin/bash
echo "Service restarting..."
echo "Stopping service..."
pkill -15 -f %s
timeout=0
while pgrep -f %s.linux64 > /dev/null; do
    echo "Waiting for the service to stop..."
    timeout=$(($timeout+1))
    if [ $timeout -gt 10 ]; then
        echo "Service stop failed. Force stop."
        pkill -9 -f %s
        break
    fi
    sleep 1
done
echo "Service stopped successfully."
echo "Starting service..."
export GORIG_SYS_MODE=%s
nohup ./%s > nohup.out 2>&1 &
pid=$!
echo "Service started with PID: $pid"
echo "Service restarted."`, runFile, runFile, runFile, sys.RunMode, runFile)

	if errW := os.WriteFile(restartFile, []byte(content), 0755); errW != nil {
		return errors.Verify("Failed to write to restart.sh file", errW)
	}

	//if _, errW := file.WriteString(content); errW != nil {
	//	return "", errors.Verify("Failed to write to restart.sh file", errW)
	//}
	//// chmod +x
	//if errCh := os.Chmod(restartFile, 0755); err != nil {
	//	return "", errors.Verify("Failed to change permissions of restart.sh file", errCh)
	//}
	//}

	// Create watchdog script
	watchdogFile := fmt.Sprintf("%s_%s.sh", "watchdog", sys.RunMode)
	//if _, err := os.Stat(watchdogFile); os.IsNotExist(err) {
	//file, errC := os.Create(watchdogFile)
	//if errC != nil {
	//	return "", errors.Verify("Failed to create watchdog file", errC)
	//}
	//defer file.Close()

	content = fmt.Sprintf(`#!/bin/bash
echo "Watchdog service started at: $(date)"
while true; do
    echo "Checking at: $(date)"
    if ! pgrep -f %s > /dev/null; then
        echo "Service is not running. Restarting..."
        mkdir -p restart_logs
        cp nohup.out restart_logs/auto_restart_$(date +%%Y%%m%%d%%H%%M%%S).log
        ./restart.sh
    fi
    sleep 5
done`, runFile)

	if errW := os.WriteFile(watchdogFile, []byte(content), 0755); errW != nil {
		return errors.Verify("Failed to write to watchdog file", errW)
	}

	//if _, errW := file.WriteString(content); errW != nil {
	//	return "", errors.Verify("Failed to write to watchdog file", errW)
	//}
	//// chmod +x
	//if errCh := os.Chmod(watchdogFile, 0755); errCh != nil {
	//	return "", errors.Verify("Failed to change permissions of watchdog file", errCh)
	//}
	//}

	if _, rErr := delploy.RunCommand(ctx, "echo", "Stopping watchdog service..."); rErr != nil {
		return rErr
	}
	if _, rErr := delploy.RunCommand(ctx, "pkill", "-9", "-f", fmt.Sprintf("watchdog_%s.sh", sys.RunMode)); rErr != nil {
		return rErr
	}

	if _, rErr := delploy.RunCommand(ctx, "./restart.sh"); rErr != nil {
		return rErr
	}

	time.Sleep(3 * time.Second)
	if r, rErr := delploy.RunCommand(ctx, "bash", "-c", fmt.Sprintf("ps -ef | grep %s | grep -v grep", runFile)); rErr != nil {
		return rErr
	} else {
		if len(r) == 0 {
			logger.Error(ctx, "Service is not running")
			goto end
		}
	}

	if _, rErr := delploy.RunCommand(ctx, "echo", "Starting watchdog service..."); rErr != nil {
		return rErr
	}
	if _, rErr := delploy.RunCommand(ctx, "bash", "-c", fmt.Sprintf("nohup ./%s > watchdog.out 2>&1 &", watchdogFile)); rErr != nil {
		return rErr
	}

end:
	if _, rErr := delploy.RunCommand(ctx, "cat", "nohup.out"); rErr != nil {
		return rErr
	}

	return nil
}

func (a appService) Stop(ctx *gin.Context) *errors.Error {
	logger.Info(ctx, "Stopping application...")

	stopFile := "stop.sh"
	//if _, err := os.Stat(stopFile); os.IsNotExist(err) {
	//file, errC := os.Create(stopFile)
	//if errC != nil {
	//	return "", errors.Verify("Failed to create stop.sh file", errC)
	//}
	//defer file.Close()

	runFile, _ := getRunFileName()

	content := fmt.Sprintf(`#!/bin/bash
echo "Stopping watchdog service..."
pkill -9 -f watchdog_%s.sh
echo "Stopping service..."
pkill -15 -f %s
echo "Service stopped successfully."`, sys.RunMode, runFile)

	if errW := os.WriteFile(stopFile, []byte(content), 0755); errW != nil {
		return errors.Verify("Failed to write to stop.sh file", errW)
	}

	//if _, errW := file.WriteString(content); errW != nil {
	//	return "", errors.Verify("Failed to write to stop.sh file", errW)
	//}
	//if errCh := os.Chmod(stopFile, 0755); errCh != nil {
	//	return "", errors.Verify("Failed to change permissions of stop.sh file", errCh)
	//}
	//}

	if _, err := delploy.RunCommand(ctx, "./stop.sh"); err != nil {
		return errors.Verify("Failed to execute stop.sh", err)
	}

	return nil
}

func (a appService) Clean(ctx *gin.Context) *errors.Error {
	logger.Info(ctx, "Cleaning files...")

	files := []string{"restart.sh", "stop.sh", fmt.Sprintf("watchdog_%s.sh", sys.RunMode), "nohup.out", "restart_logs", "watchdog.out"}
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue
		}

		if err := os.RemoveAll(file); err != nil {
		}
		logger.Info(ctx, fmt.Sprintf("Removed file: %s", file))
	}

	return nil
}
