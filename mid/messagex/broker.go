package messagex

import (
	"context"
	"fmt"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/rs/xid"
	"strconv"
	"strings"
)

type MessageType string

type Message struct {
	Ctx     context.Context `json:"-"`
	ID      string
	GroupID string
	SubID   uint64
	Topic   string
	Content map[string]interface{}
}

type BrokerType int

const (
	Local    = iota
	RabbitMQ // RabbitMQ
	Redis    // Redis
)

// MessageBroker 定义了消息代理的行为。
type MessageBroker interface {
	Subscribe(topic string, handler func(message *Message) *errors.Error) (uint64, *errors.Error)
	SubscribeGroup(topic string, groupID string, handler func(message *Message) *errors.Error) (uint64, *errors.Error) // 指定groupID
	UnSubscribe(topic string, subID uint64) *errors.Error
	Publish(topic string, message *Message) *errors.Error
	PublishGroup(topic string, groupID string, message *Message) *errors.Error // 指定groupID
	TopicList() []string
}

func (m *Message) GetValue(key string) interface{} {
	key = strings.ToLower(key)
	v, ok := m.Content[key]
	if !ok {
		return nil
	}
	return v
}

func (m *Message) GetValueInt64(key string) int64 {
	v := m.GetValue(key)
	if v == nil {
		return 0
	}
	switch v.(type) {
	case int64:
		return v.(int64)
	case int:
		return int64(v.(int))
	case string:
		i, _ := strconv.ParseInt(v.(string), 10, 64)
		return i
	}
	return 0
}

func (m *Message) GetValueFloat64(key string) float64 {
	v := m.GetValue(key)
	if v == nil {
		return 0
	}
	switch v.(type) {
	case float64:
		return v.(float64)
	case string:
		i, _ := strconv.ParseFloat(v.(string), 64)
		return i
	}
	return 0
}

func (m *Message) GetValueStr(key string) string {
	v := m.GetValue(key)
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func (m *Message) SetValue(key string, value interface{}) {
	key = strings.ToLower(key)
	m.Content[key] = value
}

func (m *Message) DeepCopy() *Message {
	if m == nil {
		return nil
	}

	clone := &Message{
		ID:      xid.New().String(),
		GroupID: m.GroupID,
		SubID:   m.SubID,
		Topic:   m.Topic,
		Content: nil,
	}

	// Deep copy Content map
	if m.Content != nil {
		clone.Content = make(map[string]interface{}, len(m.Content))
		for key, value := range m.Content {
			clone.Content[key] = value
		}
	}

	return clone
}

func (m *Message) LowerContentKey() {
	if m.Content != nil {
		for k, v := range m.Content {
			lk := strings.ToLower(k)
			m.Content[lk] = v
			if k != lk {
				delete(m.Content, k)
			}
		}
	}
}
