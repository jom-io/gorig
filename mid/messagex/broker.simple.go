package messagex

import (
	"fmt"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
)

type subscription struct {
	id      uint64
	ch      chan *Message
	groupID string
	handler func(message *Message) *errors.Error
}

type SimpleMessageBroker struct {
	subscribers sync.Map // map[string][]*subscription
	nextID      uint64
}

func NewSimple() *SimpleMessageBroker {
	return &SimpleMessageBroker{}
}

func (mb *SimpleMessageBroker) Subscribe(topic string, handler func(message *Message) *errors.Error) (uint64, *errors.Error) {
	return mb.SubscribeGroup(topic, "", handler)
}

func (mb *SimpleMessageBroker) SubscribeGroup(topic string, groupID string, handler func(message *Message) *errors.Error) (uint64, *errors.Error) {
	newID := atomic.AddUint64(&mb.nextID, 1) // 为新订阅者生成唯一ID
	sub := &subscription{
		id:      newID,
		ch:      make(chan *Message, 200),
		groupID: groupID,
		handler: handler,
	}

	if value, ok := mb.subscribers.Load(topic); ok {
		subs := value.([]*subscription)
		subs = append(subs, sub)
		mb.subscribers.Store(topic, subs)
	} else {
		mb.subscribers.Store(topic, []*subscription{sub})
	}
	go mb.listen(sub)

	return newID, nil
}

func (mb *SimpleMessageBroker) UnSubscribe(topic string, subID uint64) *errors.Error {
	value, ok := mb.subscribers.Load(topic)
	if !ok {
		return errors.Sys("topic not found")
	}

	subs := value.([]*subscription)
	for i, sub := range subs {
		if sub.id == subID {
			close(sub.ch)                          // 关闭通道
			subs = append(subs[:i], subs[i+1:]...) // 移除订阅者
			if len(subs) == 0 {
				mb.subscribers.Delete(topic)
			} else {
				mb.subscribers.Store(topic, subs)
			}
			break
		}
	}

	return nil
}

func (mb *SimpleMessageBroker) Publish(topic string, message *Message) *errors.Error {
	return mb.PublishGroup(topic, "", message)
}

func (mb *SimpleMessageBroker) PublishGroup(topic string, groupID string, message *Message) *errors.Error {
	go func() {
		value, ok := mb.subscribers.Load(topic)
		if !ok {
			return
		}

		subs := value.([]*subscription)
		for _, sub := range subs {
			if sub.groupID == groupID {
				select {
				case sub.ch <- message:
				default:
					logger.Error(nil, fmt.Sprintf("topic %s message queue full", topic), zap.Any("message", message))
				}
			}
		}
	}()
	return nil
}

func (mb SimpleMessageBroker) listen(sub *subscription) {
	for message := range sub.ch {
		go func(msg *Message) {
			defer HandlePanic(msg)
			if err := sub.handler(msg); err != nil {
				HandleError(msg, err)
				//logger.Error(message.Ctx, "Error processing message", zap.Error(err))
			}
		}(message)
	}
}

func (mb *SimpleMessageBroker) TopicList() []string {
	var topics []string
	mb.subscribers.Range(func(key, value interface{}) bool {
		topics = append(topics, key.(string))
		return true
	})
	return topics
}

func (mb *SimpleMessageBroker) Startup(topic string) *errors.Error {
	// This method is simplified because listening starts automatically for each subscriber
	return nil
}

func (mb *SimpleMessageBroker) StartListening() {
	// Not needed as subscribers start listening upon subscription
}

func (mb *SimpleMessageBroker) StopListening() {
	mb.subscribers.Range(func(key, value interface{}) bool {
		subs := value.([]*subscription)
		for _, sub := range subs {
			close(sub.ch)
		}
		mb.subscribers.Delete(key)
		return true
	})
}
