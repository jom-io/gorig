package messagex

import (
	"fmt"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/jom-io/gorig/utils/notify/dingding"
	"runtime/debug"
)

func HandlePanic(msg *Message) {
	if r := recover(); r != nil {
		if msg == nil {
			log := fmt.Sprintf("Message execution crash: \nPanic: %v, \nStack: %s", r, string(debug.Stack()))
			logger.Logger.DPanic(log)
			go dingding.PanicNotifyDefault(log)
			return
		}
		debug.PrintStack()
		log := fmt.Sprintf("Message execution crash: %s,\nPanic: %v, \ntopic: %s, \ncontent: %v, \nStack: %s", r, msg.Topic, msg.Content, string(debug.Stack()))
		logger.DPanic(msg.Ctx, log)
		go dingding.PanicNotifyDefault(log)
	}
}

func HandleError(msg *Message, error *errors.Error) {
	if error == nil {
		return
	}
	if msg == nil {
		log := fmt.Sprintf("Message execution exception: \nError: %s, \nStack: %s", error, string(debug.Stack()))
		logger.Logger.Error(log)
		go dingding.ErrNotifyDefault(log)
		return
	}
	errText := fmt.Sprintf("Message execution exception: \nError: %v, \ntopic: %s, \ncontent: %v", error, msg.Topic, msg.Content)
	if error.Type == errors.System {
		log := fmt.Sprintf("%s, \nStack: %s", errText, string(debug.Stack()))
		logger.Error(msg.Ctx, error.Error())
		go dingding.ErrNotifyDefault(log)
		return
	}
	if error.Type == errors.Application {
		logger.Error(msg.Ctx, errText)
	}
}
