package test

import (
	"context"
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
