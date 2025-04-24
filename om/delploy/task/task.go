package delpoy

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/cache"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
)

var Task taskService

type taskService struct {
}

func init() {
	Task = taskService{}
}

const Key = "dp_task"

type TaskOptions struct {
	Repo   string `json:"repo" binding:"required"`
	Branch string `json:"branch" binding:"required"`
	Auto   bool   `json:"auto"`
}

func (t taskService) Save(ctx *gin.Context, opts *TaskOptions) *errors.Error {
	logger.Info(ctx, fmt.Sprintf("Saving task with options: %v", opts))
	err := cache.New[TaskOptions](cache.JSON).Set(opts.Repo, *opts, 0)
	if err != nil {
		return errors.Verify(err.Error())
	}
	return nil
}
