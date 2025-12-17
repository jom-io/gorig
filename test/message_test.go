package test

import (
	"context"
	"github.com/jom-io/gorig/cache"
	"github.com/jom-io/gorig/mid/messagex"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/jom-io/gorig/utils/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type testPayload struct {
	Name string
	Age  int
}

func TestMessageBroker_LocalMemory(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	topic := "test.topic.memory"
	received := false

	subID, err := messagex.RegisterTopic(topic, func(msg *messagex.Message) *errors.Error {
		defer wg.Done()
		received = true
		logger.Info(context.Background(), "Received message", zap.String("topic", msg.Topic), zap.Any("content", msg.Content))
		assert.Equal(t, "test", msg.Content["name"])
		assert.Equal(t, int(18), msg.Content["age"])
		return nil
	})
	assert.Nil(t, err)
	defer messagex.UnSubscribe(topic, subID)

	msg := &testPayload{Name: "test", Age: 18}
	messagex.PublishNewMsg(context.Background(), topic, msg)

	waitDone(t, &wg)
	assert.True(t, received, "Message should have been received")
}

func TestMessageBroker_WithStore(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	topic := "test.topic.redis"
	received := false

	svc := messagex.Ins(messagex.Redis)

	subID, err := svc.RegisterTopic(topic, func(msg *messagex.Message) *errors.Error {
		defer wg.Done()
		received = true
		assert.Equal(t, "stored", msg.Content["name"])
		assert.Equal(t, float64(99), msg.Content["age"])
		return nil
	})
	assert.Nil(t, err)
	defer svc.UnRegisterTopic(topic, subID)

	svc.PublishNewMsg(logger.NewCtx(), topic, testPayload{Name: "stored", Age: 99})
	waitDone(t, &wg)
	assert.True(t, received, "Stored message should have been received")
}

func TestMessageBroker_ConcurrentMessages_Local(t *testing.T) {
	topic := "test.topic.concurrent.local"
	totalMessages := 100

	var wg sync.WaitGroup
	wg.Add(totalMessages)

	received := make([]bool, totalMessages)

	subID, err := messagex.RegisterTopic(topic, func(msg *messagex.Message) *errors.Error {
		index := msg.GetValueInt64("index")
		received[index] = true
		wg.Done()
		return nil
	})
	assert.Nil(t, err)
	defer messagex.UnSubscribe(topic, subID)

	for i := 0; i < totalMessages; i++ {
		go func(i int) {
			messagex.PublishNewMsg(context.Background(), topic, map[string]any{
				"name":  "user",
				"age":   i,
				"index": i,
			})
		}(i)
	}

	waitDone(t, &wg)

	for i := 0; i < totalMessages; i++ {
		assert.True(t, received[i], "Message #%d should have been received", i)
	}
}

func TestMessageBroker_ConcurrentMessages_WithStore(t *testing.T) {
	topic := "test.topic.concurrent.store"
	totalMessages := 100

	var wg sync.WaitGroup
	wg.Add(totalMessages)

	received := make([]int32, totalMessages)

	svc := messagex.Ins(messagex.Redis)

	subID, err := svc.RegisterTopic(topic, func(msg *messagex.Message) *errors.Error {
		logger.Info(context.Background(), "Received message", zap.String("topic", msg.Topic), zap.Any("content", msg.Content))
		index := int(msg.Content["index"].(float64))
		if index < 0 || index >= totalMessages {
			t.Logf("⚠️ Skip invalid index: %v", index)
			return nil
		}
		atomic.StoreInt32(&received[index], 1)
		wg.Done()
		return nil
	})
	assert.Nil(t, err)
	defer svc.UnRegisterTopic(topic, subID)

	for i := 0; i < totalMessages; i++ {
		i := i
		go func() {
			svc.PublishNewMsg(context.Background(), topic, map[string]any{
				"name":  "stored",
				"age":   i,
				"index": i,
			})
		}()
	}

	waitDone(t, &wg)

	for i := 0; i < totalMessages; i++ {
		assert.Equal(t, int32(1), atomic.LoadInt32(&received[i]), "Stored message #%d should have been received", i)
	}
}

func TestMessageBroker_SequentialOrder_Local(t *testing.T) {
	topic := "test.topic.seq.local"
	totalMessages := 10

	var wg sync.WaitGroup
	wg.Add(totalMessages)

	orders := make([]int, 0, totalMessages)

	subID, err := messagex.RegisterTopicSeq(topic, func(msg *messagex.Message) *errors.Error {
		defer wg.Done()
		index := msg.GetValueInt64("index")
		orders = append(orders, int(index))
		return nil
	})
	assert.Nil(t, err)
	defer messagex.UnSubscribe(topic, subID)

	for i := 0; i < totalMessages; i++ {
		messagex.PublishNewMsg(context.Background(), topic, map[string]any{
			"index": i,
		})
	}

	waitDone(t, &wg)
	assert.Equal(t, totalMessages, len(orders))
	for i := 0; i < totalMessages; i++ {
		assert.Equal(t, i, orders[i], "Sequential handler should preserve publish order")
	}
}

func TestMessageBroker_SequentialRequeue_Local(t *testing.T) {
	topic := "test.topic.seq.requeue"
	totalMessages := 3

	var wg sync.WaitGroup
	// message index 1 会重试一次，因此总处理次数 +1
	wg.Add(totalMessages + 1)

	orders := make([]int, 0, totalMessages+1)
	var retryCount int32

	subID, err := messagex.RegisterTopicSeq(topic, func(msg *messagex.Message) *errors.Error {
		defer wg.Done()
		index := msg.GetValueInt64("index")
		orders = append(orders, int(index))
		if index == 1 && atomic.CompareAndSwapInt32(&retryCount, 0, 1) {
			return errors.Sys("force retry")
		}
		return nil
	}, messagex.WithMaxRetry(1), messagex.WithRetryIntervals(0))
	assert.Nil(t, err)
	defer messagex.UnSubscribe(topic, subID)

	for i := 0; i < totalMessages; i++ {
		messagex.PublishNewMsg(context.Background(), topic, map[string]any{
			"index": i,
		})
	}

	waitDone(t, &wg)
	assert.Equal(t, int32(1), retryCount, "should trigger exactly one retry")
	// 本地顺序 + 回队尾，顺序为 0,1,2,1（队列里 2 先到达，重试追加在尾部）
	assert.Equal(t, []int{0, 1, 2, 1}, orders)
}

func TestMessageBroker_SequentialOrder_Redis(t *testing.T) {
	topic := "test.topic.seq.redis"
	totalMessages := 10

	if cache.GetRedisInstance[*messagex.Message](context.Background()) == nil {
		t.Skip("redis not available")
	}

	var wg sync.WaitGroup
	wg.Add(totalMessages)

	orders := make([]int, 0, totalMessages)

	svc := messagex.Ins(messagex.Redis)

	subID, err := svc.RegisterTopicSeq(topic, func(msg *messagex.Message) *errors.Error {
		defer wg.Done()
		index := msg.GetValueInt64("index")
		orders = append(orders, int(index))
		return nil
	})
	assert.Nil(t, err)
	defer svc.UnRegisterTopic(topic, subID)

	for i := 0; i < totalMessages; i++ {
		svc.PublishNewMsg(context.Background(), topic, map[string]any{
			"index": i,
		})
	}

	waitDone(t, &wg)
	assert.Equal(t, totalMessages, len(orders))
	for i := 0; i < totalMessages; i++ {
		assert.Equal(t, i, orders[i], "Sequential handler should preserve publish order (redis)")
	}
}

func TestMessageBroker_SequentialRequeue_Redis(t *testing.T) {
	topic := "test.topic.seq.requeue.redis"
	totalMessages := 3

	if cache.GetRedisInstance[*messagex.Message](context.Background()) == nil {
		t.Skip("redis not available")
	}

	var wg sync.WaitGroup
	wg.Add(totalMessages + 1)

	orders := make([]int, 0, totalMessages+1)
	var retryCount int32

	svc := messagex.Ins(messagex.Redis)

	subID, err := svc.RegisterTopicSeq(topic, func(msg *messagex.Message) *errors.Error {
		defer wg.Done()
		index := msg.GetValueInt64("index")
		orders = append(orders, int(index))
		if index == 1 && atomic.CompareAndSwapInt32(&retryCount, 0, 1) {
			return errors.Sys("force retry")
		}
		return nil
	}, messagex.WithMaxRetry(1), messagex.WithRetryIntervals(50*time.Millisecond))
	assert.Nil(t, err)
	defer svc.UnRegisterTopic(topic, subID)

	for i := 0; i < totalMessages; i++ {
		svc.PublishNewMsg(context.Background(), topic, map[string]any{
			"index": i,
		})
	}

	waitDone(t, &wg)
	assert.Equal(t, int32(1), retryCount, "should trigger exactly one retry (redis)")
	// Redis 顺序 + 回队尾，顺序为 0,1,2,1（队列里 2 先到达，重试追加在尾部）
	assert.Equal(t, []int{0, 1, 2, 1}, orders)
}

func TestMessageBroker_ReplayDLQ_Redis(t *testing.T) {
	topic := "test.topic.dlq.redis"

	if cache.GetRedisInstance[*messagex.Message](context.Background()) == nil {
		t.Skip("redis not available")
	}

	var wg sync.WaitGroup
	wg.Add(1)

	svc := messagex.Ins(messagex.Redis)

	// 初次消费失败，走 DLQ
	subID, err := svc.RegisterTopicSeq(topic, func(msg *messagex.Message) *errors.Error {
		return errors.Sys("force dlq")
	}, messagex.WithMaxRetry(0))
	assert.Nil(t, err)

	svc.PublishNewMsg(context.Background(), topic, map[string]any{"index": 1})
	time.Sleep(100 * time.Millisecond)
	svc.UnRegisterTopic(topic, subID)

	// 再次注册成功消费，从 DLQ 归队
	subID2, err := svc.RegisterTopicSeq(topic, func(msg *messagex.Message) *errors.Error {
		defer wg.Done()
		assert.Equal(t, int64(1), msg.GetValueInt64("index"))
		return nil
	})
	assert.Nil(t, err)
	defer svc.UnRegisterTopic(topic, subID2)

	err = svc.ReplayDLQ(topic, 1)
	assert.Nil(t, err)

	waitDone(t, &wg)
}

func TestMessageBroker_MultiSubscribers(t *testing.T) {
	topic := "test.topic.multi.sub"
	totalMessages := 2
	subCount := 2

	var wg sync.WaitGroup
	wg.Add(totalMessages * subCount)

	received := make([][]int32, subCount)
	for i := range received {
		received[i] = make([]int32, totalMessages)
	}

	ins := messagex.Ins(messagex.Redis)

	for sub := 0; sub < subCount; sub++ {
		s := sub
		subID, err := ins.RegisterTopic(topic, func(msg *messagex.Message) *errors.Error {
			logger.Info(context.Background(), "Received message", zap.String("topic", msg.Topic), zap.Any("content", msg.Content))
			index := msg.GetValueInt64("index")
			if index < 0 || index >= int64(totalMessages) {
				t.Logf("⚠️ Skip invalid index: %v", index)
				return nil
			}
			atomic.StoreInt32(&received[s][index], 1)
			wg.Done()
			return nil
		})
		assert.Nil(t, err)
		defer messagex.UnSubscribe(topic, subID)
	}

	for i := 0; i < totalMessages; i++ {
		ins.PublishNewMsg(context.Background(), topic, map[string]any{
			"name":  "multi",
			"age":   i,
			"index": i,
		})
	}

	waitDone(t, &wg)

	//for sub := 0; sub < subCount; sub++ {
	//	for i := 0; i < totalMessages; i++ {
	//		assert.Equal(t, int32(1), atomic.LoadInt32(&received[sub][i]), "Subscriber %d should have received message #%d", sub, i)
	//	}
	//}
}

func waitDone(t *testing.T, wg *sync.WaitGroup) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("test timeout")
	}
}
