package messagex

import (
	"fmt"
	"github.com/jom-io/gorig/cache"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
	"time"
)

type subscription struct {
	id      uint64
	ch      chan *Message
	groupID string
	handler func(message *Message) *errors.Error
}

type store[T any] interface {
	RPush(topic string, message *Message) error
	BRPop(timeout time.Duration, queue string) (value T, err error)
}

type SimpleMessageBroker struct {
	subscribers sync.Map
	store       store[*Message]
	nextID      uint64
	topicOnce   sync.Map
	topicLock   sync.Mutex
	stopChan    sync.Map
}

func NewSimple() *SimpleMessageBroker {
	return &SimpleMessageBroker{}
}

func NewSimpleByType(brokerType BrokerType) *SimpleMessageBroker {
	if brokerType == Redis {
		return &SimpleMessageBroker{
			store: cache.GetRedisInstance[*Message](),
		}
	}
	return &SimpleMessageBroker{}
}

func (mb *SimpleMessageBroker) StartStoreListener(topic string) {
	if mb.store == nil {
		logger.Error(nil, "store is not initialized, cannot start listener", zap.String("topic", topic))
		return
	}
	onceVal, _ := mb.topicOnce.LoadOrStore(topic, new(sync.Once))
	once := onceVal.(*sync.Once)

	once.Do(func() {
		stop := make(chan struct{})
		mb.stopChan.Store(topic, stop)
		go func() {
			for {
				select {
				case <-stop:
					return
				default:
					msg, err := mb.store.BRPop(0, topic)
					if err != nil {
						logger.Error(nil, "redis BRPop error", zap.String("topic", topic), zap.Error(err))
						time.Sleep(2 * time.Second) // Retry after a short delay
						continue
					}
					mb.publish(topic, "", msg)
				}
			}
		}()
	})
}

func (mb *SimpleMessageBroker) Subscribe(topic string, handler func(message *Message) *errors.Error) (uint64, *errors.Error) {
	return mb.SubscribeGroup(topic, "", handler)
}

func (mb *SimpleMessageBroker) SubscribeGroup(topic string, groupID string, handler func(message *Message) *errors.Error) (uint64, *errors.Error) {
	mb.topicLock.Lock()
	defer mb.topicLock.Unlock()

	mb.StartStoreListener(topic)

	newID := atomic.AddUint64(&mb.nextID, 1)
	sub := &subscription{
		id:      newID,
		ch:      make(chan *Message, 20000),
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
			close(sub.ch)
			subs = append(subs[:i], subs[i+1:]...)
			if len(subs) == 0 {
				mb.subscribers.Delete(topic)
				mb.topicOnce.Delete(topic)
				if stop, ok := mb.stopChan.Load(topic); ok {
					close(stop.(chan struct{}))
					mb.stopChan.Delete(topic)
				}
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

func (mb *SimpleMessageBroker) publish(topic string, groupID string, message *Message) {
	go func() {
		value, ok := mb.subscribers.Load(topic)
		if !ok {
			return
		}

		subs := value.([]*subscription)
		for _, sub := range subs {
			if groupID == "" || sub.groupID == groupID {
				select {
				case sub.ch <- message:
				default:
					logger.Error(nil, fmt.Sprintf("topic %s message queue full", topic), zap.Any("message", message))
				}
			}
		}
	}()
}

func (mb *SimpleMessageBroker) PublishGroup(topic string, groupID string, message *Message) *errors.Error {
	if mb.store != nil {
		if err := mb.store.RPush(topic, message); err != nil {
			logger.Error(nil, "store RPush error", zap.String("topic", topic), zap.Error(err))
			return errors.Sys(fmt.Sprintf("store RPush error for topic %s: %v", topic, err))
		}
		return nil
	}
	mb.publish(topic, groupID, message)
	return nil
}

func (mb *SimpleMessageBroker) listen(sub *subscription) {
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
		mb.topicOnce.Delete(key.(string))
		if stop, ok := mb.stopChan.Load(key); ok {
			close(stop.(chan struct{}))
			mb.stopChan.Delete(key)
		}
		return true
	})
}
