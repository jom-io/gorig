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

var defaultRetryIntervals = []time.Duration{
	2 * time.Second,
	time.Minute,
	10 * time.Minute,
	time.Hour,
	6 * time.Hour,
	24 * time.Hour,
}

type subscription struct {
	id         uint64
	ch         chan *Message
	groupID    string
	topic      string
	sequential bool
	maxRetry   int
	dlqTopic   string
	retryItv   []time.Duration
	handler    func(message *Message) *errors.Error
}

type store[T any] interface {
	RPush(topic string, message T) error
	BRPopCtx(ctx context.Context, timeout time.Duration, queue string) (value T, err error)
	LPop(queue string) (value T, err error)
	AddDelayed(queue string, message T, score float64) error
	PopDueDelayed(queue string, now float64, limit int) ([]T, error)
}

type seqConfig struct {
	maxRetry int
	dlqTopic string
	retryItv []time.Duration
}

type SeqOption func(*seqConfig)

func WithMaxRetry(max int) SeqOption {
	return func(cfg *seqConfig) {
		if max >= 0 {
			cfg.maxRetry = max
		}
	}
}

func WithDLQTopic(dlq string) SeqOption {
	return func(cfg *seqConfig) {
		if dlq != "" {
			cfg.dlqTopic = dlq
		}
	}
}

func WithRetryIntervals(itv ...time.Duration) SeqOption {
	return func(cfg *seqConfig) {
		if len(itv) > 0 {
			cfg.retryItv = itv
		}
	}
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

		go func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					mb.promoteDelayed(topic)
				}
			}
		}()
	})
}

func (mb *SimpleMessageBroker) Subscribe(topic string, handler func(message *Message) *errors.Error) (uint64, *errors.Error) {
	return mb.subscribe(topic, "", handler, false, nil)
}

func (mb *SimpleMessageBroker) SubscribeGroup(topic string, groupID string, handler func(message *Message) *errors.Error) (uint64, *errors.Error) {
	return mb.subscribe(topic, groupID, handler, false, nil)
}

func (mb *SimpleMessageBroker) SubscribeSeq(topic string, handler func(message *Message) *errors.Error, opts ...SeqOption) (uint64, *errors.Error) {
	return mb.subscribe(topic, "", handler, true, opts)
}

func (mb *SimpleMessageBroker) subscribe(topic string, groupID string, handler func(message *Message) *errors.Error, sequential bool, opts []SeqOption) (uint64, *errors.Error) {
	mb.topicLock.Lock()
	defer mb.topicLock.Unlock()

	if mb.brokerType == Redis {
		mb.StartStoreListener(topic)
	}

	cfg := seqConfig{
		maxRetry: 0,
		dlqTopic: topic + ".dlq",
		retryItv: defaultRetryIntervals,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	newID := atomic.AddUint64(&mb.nextID, 1)
	chCap := 20000
	if sequential {
		chCap = 1
	}
	sub := &subscription{
		id:         newID,
		ch:         make(chan *Message, chCap),
		groupID:    groupID,
		topic:      topic,
		sequential: sequential,
		maxRetry:   cfg.maxRetry,
		dlqTopic:   cfg.dlqTopic,
		retryItv:   cfg.retryItv,
		handler:    handler,
	}

	if value, ok := mb.subscribers.Load(topic); ok {
		subs := value.([]*subscription)
		subs = append(subs, sub)
		mb.subscribers.Store(topic, subs)
	} else {
		mb.subscribers.Store(topic, []*subscription{sub})
	}
	if sequential {
		go mb.listenSequential(sub)
	} else {
		go mb.listen(sub)
	}

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
	value, ok := mb.subscribers.Load(topic)
	if !ok {
		return
	}

	subs := value.([]*subscription)
	seqSubs := make([]*subscription, 0)
	asyncSubs := make([]*subscription, 0)
	for _, sub := range subs {
		if groupID != "" && sub.groupID != groupID {
			continue
		}
		if sub.sequential {
			seqSubs = append(seqSubs, sub)
		} else {
			asyncSubs = append(asyncSubs, sub)
		}
	}

	// 顺序订阅同步发送，保证发布顺序
	for _, sub := range seqSubs {
		sub.ch <- message
	}

	// 异步订阅保持原有非阻塞语义
	if len(asyncSubs) > 0 {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error(nil, "message publish panic", zap.Any("panic", r), zap.String("topic", topic), zap.Any("message", message))
				}
			}()
			for _, sub := range asyncSubs {
				select {
				case sub.ch <- message:
				default:
					logger.Error(nil, fmt.Sprintf("topic %s message queue full", topic), zap.Any("message", message))
				}
			}
		}()
	}
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
			if msg.GroupID != "" {
				msg.Ctx = context.WithValue(msg.Ctx, consts.TraceIDKey, msg.GroupID)
			}
			if err := sub.handler(msg); err != nil {
				HandleError(msg, err)
				//logger.Error(message.Ctx, "Error processing message", zap.Error(err))
			}
		}(message)
	}
}

func (mb *SimpleMessageBroker) listenSequential(sub *subscription) {
	for message := range sub.ch {
		if message == nil {
			continue
		}
		if message.Ctx == nil {
			message.Ctx = context.Background()
		}
		if message.GroupID != "" {
			message.Ctx = context.WithValue(message.Ctx, consts.TraceIDKey, message.GroupID)
		}
		if err := sub.handler(message); err != nil {
			HandleError(message, err)
			mb.handleRetry(sub, message)
		}
	}
}

func (mb *SimpleMessageBroker) handleRetry(sub *subscription, msg *Message) {
	if msg == nil {
		return
	}
	if sub.maxRetry <= 0 {
		mb.sendToDLQ(sub, msg)
		return
	}
	msg.Retry++
	if msg.Retry > sub.maxRetry {
		mb.sendToDLQ(sub, msg)
		return
	}
	delay := sub.retryDelay(msg.Retry)
	mb.requeue(sub, msg, delay)
}

func (sub *subscription) retryDelay(retry int) time.Duration {
	if len(sub.retryItv) == 0 {
		return 0
	}
	idx := retry - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sub.retryItv) {
		idx = len(sub.retryItv) - 1
	}
	delay := sub.retryItv[idx]
	if delay < 0 {
		delay = 0
	}
	maxDelay := 24 * time.Hour
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

func (mb *SimpleMessageBroker) requeue(sub *subscription, msg *Message, delay time.Duration) {
	if msg == nil {
		return
	}
	if delay < 0 {
		delay = 0
	}
	if mb.store != nil {
		if delay == 0 {
			if err := mb.store.RPush(sub.topic, msg); err != nil {
				logger.Error(msg.Ctx, "message requeue failed", zap.String("topic", sub.topic), zap.Error(err))
			}
		} else {
			score := float64(time.Now().Add(delay).UnixMilli())
			if err := mb.store.AddDelayed(sub.topic+":delay", msg, score); err != nil {
				logger.Error(msg.Ctx, "message add delay failed", zap.String("topic", sub.topic), zap.Error(err))
			}
		}
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error(msg.Ctx, "message requeue panic", zap.String("topic", sub.topic), zap.Any("panic", r))
			}
		}()
		if delay > 0 {
			time.Sleep(delay)
		}
		sub.ch <- msg
	}()
}

func (mb *SimpleMessageBroker) sendToDLQ(sub *subscription, msg *Message) {
	if msg == nil || sub == nil || sub.dlqTopic == "" {
		return
	}
	if mb.store != nil {
		if err := mb.store.RPush(sub.dlqTopic, msg); err != nil {
			logger.Error(msg.Ctx, "message send to dlq failed", zap.String("topic", sub.dlqTopic), zap.Error(err))
		}
		return
	}
	mb.publish(sub.dlqTopic, "", msg)
}

func (mb *SimpleMessageBroker) ReplayDLQ(topic string, limit int) *errors.Error {
	if mb.store == nil {
		return errors.Sys("store not initialized for dlq replay")
	}
	dlq := topic + ".dlq"
	count := 0
	for limit <= 0 || count < limit {
		msg, err := mb.store.LPop(dlq)
		if err != nil {
			if err == cache.ErrCacheMiss {
				break
			}
			return errors.Sys(fmt.Sprintf("dlq pop error: %v", err))
		}
		if msg != nil {
			msg.Retry = 0
		}
		if err := mb.Publish(topic, msg); err != nil {
			return err
		}
		count++
	}
	return nil
}

func (mb *SimpleMessageBroker) promoteDelayed(topic string) {
	if mb.store == nil {
		return
	}
	now := float64(time.Now().UnixMilli())
	msgs, err := mb.store.PopDueDelayed(topic+":delay", now, 100)
	if err != nil {
		logger.Error(nil, "pop due delayed failed", zap.String("topic", topic), zap.Error(err))
		return
	}
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		if err := mb.store.RPush(topic, msg); err != nil {
			logger.Error(msg.Ctx, "push delayed to ready failed", zap.String("topic", topic), zap.Error(err))
		}
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
