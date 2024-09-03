package messagex

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/global/consts"
	"github.com/jom-io/gorig/utils/errors"
	"github.com/rs/xid"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
)

// MessageType 定义了消息的内容。
type MessageType string

// Message 是消息的结构体。
type Message struct {
	ID      string
	GroupID string
	SubID   uint64
	Topic   string // 主题
	Content map[string]interface{}
}

// BrokerType 定义了代理的类型。
type BrokerType int

const (
	Local    = iota // 本地通道
	RabbitMQ        // RabbitMQ 消息代理
)

// MessageBroker 定义了消息代理的行为。
type MessageBroker interface {
	Subscribe(topic string, handler func(message *Message) *errors.Error) (uint64, *errors.Error)
	SubscribeGroup(topic string, groupID string, handler func(message *Message) *errors.Error) (uint64, *errors.Error) // 指定groupID
	UnSubscribe(topic string, subID uint64) *errors.Error
	Publish(topic string, message *Message) *errors.Error
	PublishGroup(topic string, groupID string, message *Message) *errors.Error // 指定groupID
	TopicList() []string
	//StartListening()
	//StopListening()
	//Startup(topic string) *errors.Error
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

// DeepCopy 创建Message的深拷贝
func (m *Message) DeepCopy() *Message {
	if m == nil {
		return nil
	}

	// 创建一个新的Message实例
	clone := &Message{
		ID:      xid.New().String(),
		GroupID: m.GroupID,
		SubID:   m.SubID,
		Topic:   m.Topic,
		Content: nil, // 初始化为nil，稍后填充
	}

	// 深拷贝Content
	if m.Content != nil {
		clone.Content = make(map[string]interface{}, len(m.Content))
		for key, value := range m.Content {
			// 注意：这里我们假设Content中的值都是基本类型或者类型提供了正确的深拷贝方法
			// 如果map中包含复杂类型或指针，需要更复杂的处理
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

func (m *Message) ToNewGinCtx() *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set(consts.TraceIDKey, m.GroupID)
	ctx.Header(consts.TraceIDKey, m.GroupID)
	ctx.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	return ctx
}
