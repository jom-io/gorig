package messagex

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	"go.uber.org/zap"
	"gorig/httpx"
	"gorig/utils/errors"
	"gorig/utils/logger"
	"gorig/utils/sys"
	"reflect"
	"strings"
)

type MessageService struct {
	BrokerType BrokerType
	Broker     MessageBroker
}

var GService MessageService

func GetDef() *MessageService {
	if GService.Broker == nil {
		GService = *Get(Local)
	}
	return &GService
}

func GetInstance(brokerType BrokerType) *MessageService {
	if GService.Broker == nil {
		GService = *Get(brokerType)
	}
	return &GService
}

func Get(brokerType BrokerType) *MessageService {
	var broker MessageBroker
	switch brokerType {
	case Local:
		broker = NewSimple()
	case RabbitMQ:
		// broker = NewRabbitMQMessageBroker()
	default:
		panic("Unsupported broker type")
	}
	return &MessageService{
		BrokerType: brokerType,
		Broker:     broker,
	}
}

//func (s *MessageService) StartListening() {
//	s.Broker.StartListening()
//}

func getTopicStr(topic any) string {
	if _, ok := topic.(string); !ok {
		topicValue := reflect.ValueOf(topic)
		topicType := topicValue.Type()
		if topicType.ConvertibleTo(reflect.TypeOf("")) {
			return topicValue.Convert(reflect.TypeOf("")).Interface().(string)
		}
		panic("topic must be string or convertible to string")
	} else {
		return topic.(string)
	}
}

func RegisterTopic(topic any, handler func(message *Message) *errors.Error) (uint64, *errors.Error) {
	topicStr := getTopicStr(topic)
	subId, e := GetDef().Broker.Subscribe(topicStr, handler)
	sys.Info(" # Reg Topic: ", topic, " # SubID: ", subId)
	if e != nil {
		logger.Error(nil, "Registering topic failed", zap.String("topic", topicStr), zap.Error(e))
	}
	return subId, e
}

func UnSubscribe(topic any, subID uint64) *errors.Error {
	sys.Info(" # UnReg Topic: ", topic, " # SubID: ", subID)
	topicStr := getTopicStr(topic)
	return GetDef().Broker.UnSubscribe(topicStr, subID)
}

func Publish(topic any, message *Message) (error *errors.Error) {
	if message == nil {
		return errors.Sys("message is nil")
	}
	if topic != MsgStartup && topic != "" {
		sys.Info(" # Publish Topic: ", topic)
		logger.Info(message.ToNewGinCtx(), "Publishing message", zap.String("group_id", message.GroupID), zap.String("topic", topic.(string)), zap.Any("content", message.Content))
	}
	error = GetDef().Broker.Publish(getTopicStr(topic), message)
	if error != nil {
		logger.Error(message.ToNewGinCtx(), "Publishing message failed", zap.String("topic", topic.(string)), zap.Error(error))
	}
	return
}

func PublishWithCtx(ctx *gin.Context, topic any, message *Message) *errors.Error {
	topicStr := getTopicStr(topic)
	return Publish(topicStr, message)
}

func PublishNewMsg(ctx *gin.Context, topic any, content interface{}, groupId ...string) {
	topicStr := getTopicStr(topic)
	if topicStr == "" {
		logger.Error(ctx, "PublishNewMsg: topic is empty")
		return
	}
	gid := xid.New().String()
	if len(groupId) > 0 {
		gid = groupId[0]
	} else {
		if ctx != nil {
			gid = httpx.GetTraceID(ctx)
		}
	}
	msg := &Message{
		ID:      xid.New().String(),
		GroupID: gid,
		Topic:   topicStr,
		Content: ToMap(content),
	}
	msg.LowerContentKey()
	Publish(msg.Topic, msg)
}

const (
	MsgStartup = "messagex.startup"
)

//func Startup(port string) error {
//	for _, topic := range GService.Broker.TopicList() {
//		sys.Info(" # Listening Topic: ", topic)
//	}
//	GetDef().Broker.StartListening()
//	return nil
//}
//
//func Shutdown(context context.Context) error {
//	GetDef().Broker.StopListening()
//	return nil
//}
//
//func StartupTopic(topic string) *errors.Error {
//	sys.Info(" # Listening Topic: ", topic)
//	return GetDef().Broker.Startup(topic)
//}

// ToMap converts a struct to a map[string]interface{} where the keys are the struct's field names
// and the values are the respective field values.
// Note: This function only works with structs and will return nil for non-struct parameters.
func ToMap(param interface{}) map[string]interface{} {
	// Return nil if the parameter is nil
	if param == nil {
		return nil
	}
	// Return the parameter if it is already a map
	if reflect.TypeOf(param).Kind() == reflect.Map {
		return param.(map[string]interface{})
	}
	// Initialize a nil map for non-struct parameters
	result := make(map[string]interface{})

	// Get the type and value of the parameter
	val := reflect.ValueOf(param)
	typ := reflect.TypeOf(param)

	// Check if the passed interface is a pointer, and if so, get the element it points to
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	// Return nil if the parameter is not a struct
	if val.Kind() != reflect.Struct {
		return nil
	}

	// Loop through the struct's fields
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		// Use the type to get the field name and convert it to lowercase
		name := strings.ToLower(typ.Field(i).Name)
		// Add the field name and value to the map
		result[name] = field.Interface()
	}

	return result
}
