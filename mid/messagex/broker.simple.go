package messagex

import (
	"context"
	"fmt"
	"github.com/jom-io/gorig/cache"
	"github.com/jom-io/gorig/global/consts"
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
	BRPopCtx(ctx context.Context, timeout time.Duration, queue string) (value T, err error)
}

type SimpleMessageBroker struct {
	brokerType  BrokerType
	subscribers sync.Map
	store       store[*Message]
	nextID      uint64
	topicOnce   sync.Map
	topicLock   sync.Mutex
	stopCtxs    sync.Map
}

func NewSimple() *SimpleMessageBroker {
	return &SimpleMessageBroker{}
}

func NewSimpleByType(brokerType BrokerType) *SimpleMessageBroker {
	simpleBroker := &SimpleMessageBroker{
		brokerType: brokerType,
	}
	if brokerType == Redis {
		simpleBroker.store = cache.GetRedisInstance[*Message](context.Background())
	}
	return simpleBroker
}

func (mb *SimpleMessageBroker) StartStoreListener(topic string) {
	if mb.store == nil {
		logger.Error(nil, "store is not initialized, cannot start listener", zap.String("topic", topic))
		return
	}
	onceVal, _ := mb.topicOnce.LoadOrStore(topic, new(sync.Once))
	once := onceVal.(*sync.Once)

	once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		mb.stopCtxs.Store(topic, cancel)

		go func() {
			for {
				select {
				case <-ctx.Done():
					logger.Info(nil, "Stopping listener for topic", zap.String("topic", topic))
					return
				default:
					msg, err := mb.store.BRPopCtx(ctx, 0, topic)
					if err != nil {
						logger.Error(nil, "redis BRPop error", zap.String("topic", topic), zap.Error(err))
						time.Sleep(2 * time.Second) // Retry after a short delay
						continue
					}
					if msg.GroupID != "" {
						msg.Ctx = context.WithValue(context.Background(), consts.TraceIDKey, msg.GroupID)
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

	if mb.brokerType == Redis {
		mb.StartStoreListener(topic)
	}

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
				mb.stopCtxs.Range(func(key, value interface{}) bool {
					if key == topic {
						if cancelFunc, ok := value.(context.CancelFunc); ok {
							cancelFunc() // Cancel the context to stop the listener
						}
						mb.stopCtxs.Delete(key)
					}
					return true
				})
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
		defer func() {
			if r := recover(); r != nil {
				logger.Error(nil, "message publish panic", zap.Any("panic", r), zap.String("topic", topic), zap.Any("message", message))
			}
		}()
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
			if msg.Ctx == nil {
				msg.Ctx = context.Background()
			}
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
		if cancelFunc, ok := mb.stopCtxs.Load(key); ok {
			if cancelFunc, ok := cancelFunc.(context.CancelFunc); ok {
				cancelFunc() // Cancel the context to stop the listener
			}
			mb.stopCtxs.Delete(key)
		}
		return true
	})
}
