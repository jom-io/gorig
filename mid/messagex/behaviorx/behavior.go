package behaviorx

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"gorig/mid/messagex"
	"gorig/utils/errors"
	"gorig/utils/logger"
	"gorig/utils/sys"
	"strings"
)

// Action and Event are the two main types of behaviors in the system.
type Action struct {
	Topic     string                                                              `json:"name"`    // 用于标记Topic的名称
	Deliver   map[string]interface{}                                              `json:"deliver"` // 用于传递消息的内容
	Assembler func(msg *messagex.Message) (map[string]interface{}, *errors.Error) // 用于组装消息内容
}

type Result struct {
	Topic     string                                            `json:"name"`    // 用于标记Topic的名称
	Filters   map[string]interface{}                            `json:"filters"` // 用于过滤消息的内容
	Validator func(msg *messagex.Message) (bool, *errors.Error) // 用于校验消息内容
}

type Behavior struct {
	SubID  uint64    // 标记订阅的ID
	Name   string    `json:"name"` // 用于标记行为的名称
	First  *Behavior // 用于标记链路的第一个行为
	Action Action    `json:"action"` // 用于标记行为的动作
	Result Result    `json:"result"` // 用于标记行为的结果
	Next   *Behavior // 用于标记链路的下一个行为
	//Message *messagex.Message // 用于传递消息的内容
}

// copy Deliver
func (a *Action) DeliverCopy() map[string]interface{} {
	if a == nil {
		return nil
	}
	deliver := map[string]interface{}{}
	for k, v := range a.Deliver {
		deliver[k] = v
	}
	return deliver
}

func (b *Behavior) IsFirst() bool {
	return b.First == nil
}

// DeepCopy 创建当前Behavior的深拷贝，包括它的整个Next链
func (b *Behavior) DeepCopy() *Behavior {
	if b == nil {
		return nil
	}
	// 创建当前Behavior的深拷贝
	behavior := *b
	if b.Next != nil {
		// 创建Next链的深拷贝
		behavior.Next = b.Next.DeepCopy()
	}
	return &behavior
}

func (b *Behavior) Execute() {
	if !b.IsFirst() {
		logger.Error(nil, "behavior Execute: behavior is not first", zap.String("name", b.Name))
		return
	}
	if b.SubID != 0 {
		logger.Error(nil, "behavior Execute: subID is not 0", zap.String("name", b.Name), zap.Uint64("subID", b.SubID))
		return
	}
	if b.Result.Topic != "" {
		handler := func(message *messagex.Message) *errors.Error {
			sys.Info("subId ", b.SubID, " topic ", b.Result.Topic, " message ", message.Content)
			if message == nil {
				return errors.Sys(fmt.Sprintf("topic %s message is nil", b.Result.Topic))
			}
			// 当行为链首次触发的时候 创建一个行为链的深拷贝，以便并发安全地处理
			behavior := b.DeepCopy()
			// 设置SubID=0 表示这是行为链首次分身触发 不需要取消订阅 因为没有进行过订阅
			behavior.SubID = 0
			// 注意这个地方必须传值，否则会出现并发问题
			go behavior.processMessage(message.DeepCopy())
			return nil
		}
		subID, e := messagex.RegisterTopic(b.Result.Topic, handler)
		if e != nil {
			logger.Error(nil, "register topic failed", zap.Error(e))
			return
		}
		b.SubID = subID
	}
	if b.Action.Topic != "" {
		go b.publishMessage(&messagex.Message{})
	}
}

// processMessage 处理接收到的消息，此方法应在拷贝的行为链上调用
func (b *Behavior) processMessage(message *messagex.Message) {
	var err *errors.Error
	defer messagex.HandlePanic(message)
	defer func() { messagex.HandleError(message, err) }()

	if message == nil {
		err = errors.Sys(fmt.Sprintf("name %s topic %s message is nil", b.Name, b.Result.Topic))
		return
	}

	message.LowerContentKey()
	if b.Result.Topic != message.Topic {
		err = errors.Sys(fmt.Sprintf("name %s topic %s not match message topic %s", b.Name, b.Result.Topic, message.Topic))
		return
	}
	// 优先执行校验器
	if b.Result.Validator != nil {
		result, e := b.Result.Validator(message)
		if e != nil || !result {
			if b.IsFirst() {
				logger.Info(message.ToNewGinCtx(), fmt.Sprintf("name %s topic %s validator failed", b.Name, b.Result.Topic), zap.Error(e))
			} else {
				err = errors.Sys(fmt.Sprintf("name %s topic %s validator failed", b.Name, b.Result.Topic), e)
			}
			return
		}
	}
	for k, v := range b.Result.Filters {
		k = strings.ToLower(k)
		if _, ok := message.Content[k]; !ok {
			errText := fmt.Sprintf("name %s topic %s filter %s not found", b.Name, b.Result.Topic, k)
			if b.IsFirst() {
				err = errors.Verify(errText)
			} else {
				err = errors.Sys(errText)
			}
			return
		}
		if message.Content[k] != v {
			//errText := fmt.Sprintf("name %s topic %s filter %s not match v:%s message: %v", b.Name, b.Result.Topic, k, v, message.Content)
			//if b.IsFirst() {
			//	err = errors.Verify(errText)
			//} else {
			//	err = errors.Sys(errText)
			//}
			return
		}
	}
	if b.Next != nil {
		if b.IsFirst() {
			b.Next.First = b
		} else {
			b.Next.First = b.First
		}
		b.Next.Name = b.Name
		subID, _ := messagex.RegisterTopic(b.Next.Result.Topic, func(nextMessage *messagex.Message) *errors.Error {
			logger.Info(nextMessage.ToNewGinCtx(), fmt.Sprintf("%s subID %d receive message", b.Name, b.SubID))
			go b.Next.processMessage(nextMessage.DeepCopy())
			return nil
		})
		b.Next.SubID = subID
		go b.Next.publishMessage(message)
	} else {
		logger.Info(message.ToNewGinCtx(), fmt.Sprintf("%s subID %d execute completed", b.Name, b.SubID))
	}
	if b.SubID != 0 {
		if subscribeErr := messagex.UnSubscribe(b.Result.Topic, b.SubID); subscribeErr != nil {
			err = errors.Sys(fmt.Sprintf("%s subID %d unsubscribe topic %s failed", b.Name, b.SubID, b.Result.Topic), subscribeErr)
		}
	}
}

// publishMessage 发布消息
func (b *Behavior) publishMessage(message *messagex.Message) {
	defer messagex.HandlePanic(message)
	if b.Action.Topic == "" {
		logger.Error(message.ToNewGinCtx(), "action topic is empty", zap.String("name", b.Name))
		return
	}
	if message == nil {
		logger.Error(message.ToNewGinCtx(), "message is nil", zap.String("name", b.Name))
		return
	}

	// 表示处理后的新消息内容
	if message.Content == nil {
		message.Content = map[string]interface{}{}
	}

	// 代表上一次传递的历史消息内容
	if b.Action.Deliver == nil {
		b.Action.Deliver = map[string]interface{}{}
	}

	deliver := b.Action.DeliverCopy()

	// 合并替换Message的内容
	//for k, v := range b.Action.Deliver {
	//	k = strings.ToLower(k)
	//	message.SetValue(k, v)
	//}

	// 合并替换Deliver的内容
	message.LowerContentKey()
	for k, v := range message.Content {
		deliver[k] = v
	}
	for k, v := range deliver {
		message.SetValue(k, v)
	}

	if b.Action.Assembler != nil {
		if newMessage, e := b.Action.Assembler(message); e != nil {
			messagex.HandleError(message, e)
		} else {
			// 将组装的消息内容合并到Deliver中
			for k, v := range newMessage {
				k = strings.ToLower(k)
				deliver[k] = v
			}
		}
	}

	if b.Next != nil {
		b.Next.Action.Deliver = deliver
	}

	messagex.PublishNewMsg(nil, b.Action.Topic, deliver, message.GroupID)
}

func Create(b *Behavior) *Behavior {
	return b
}

func (b *Behavior) SetNext(behavior *Behavior) *Behavior {
	if b.IsFirst() {
		behavior.First = b
	} else {
		behavior.First = b.First
	}
	b.Next = behavior
	return b.Next
}

func (b *Behavior) Register(key string) {
	b.SubID = 0
	first := b.First
	if first == nil {
		first = b
	}
	RegisterBehavior(key, first)
}

//var behaviors []Behavior

// 分类行为链 使用map存储 key为分类名称 value为数组
var behaviorMap = map[string][]Behavior{}

func RegisterBehavior(key string, behavior *Behavior) {
	if _, ok := behaviorMap[key]; !ok {
		behaviorMap[key] = []Behavior{}
	}
	behaviorMap[key] = append(behaviorMap[key], *behavior)
	//logger.Info(nil, "behavior registered", zap.Any("behavior action", behavior.Action), zap.Any("behavior result", behavior.Result))
	//logger.Info(nil, "all behaviors length", zap.Int("length", len(Behaviors)))
}

func ExecuteBehaviors(key string) {
	if behaviors, ok := behaviorMap[key]; ok {
		for i, _ := range behaviors {
			behaviors[i].Execute()
		}
	}
}

func Startup(code, port string) error {
	ExecuteBehaviors(code)
	return nil
}

func Shutdown(code string, context context.Context) error {
	delete(behaviorMap, code)
	return nil
}
