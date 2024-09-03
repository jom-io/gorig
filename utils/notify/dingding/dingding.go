package dingding

import (
	"bytes"
	"encoding/json"
	"fmt"
	configure "gorig/utils/cofigure"
	"net/http"
	"os"
)

// 配置需要通知的联系人
const (
	AtAll = "all"
)

func Notify(atUser string, title, msg string) {
	webhookURL := "https://oapi.dingtalk.com/robot/send?access_token="
	webhookURL = webhookURL + configure.GetString("notify.dingding.token")
	hostname, _ := os.Hostname()
	title = fmt.Sprintf("[%s系统][%s环境]%s \nHostname:%s", configure.GetString("sys.name"), configure.GetString("sys.mode"), title, hostname)
	message := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": fmt.Sprintf("%s\n%s", title, msg),
		},
	}

	if atUser != AtAll {
		message["at"] = map[string]interface{}{
			"atMobiles": []string{atUser},
			"isAtAll":   false,
		}
	} else {
		message["at"] = map[string]interface{}{
			"isAtAll": true,
		}
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(messageBytes))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		//logger.Logger.Error("Failed to send message to DingTalk")
		//sys.Error("Failed to send message to DingTalk")
		fmt.Println("Failed to send message to DingTalk")
		return
	}
	fmt.Println("Message sent to DingTalk successfully")
	//sys.Info("Message sent to DingTalk successfully")
}

func PanicNotifyDefault(msg string) {
	PanicNotify(AtAll, msg)
}

func PanicNotify(atUser string, msg string) {
	Notify(atUser, "发生【错误！！！】", msg)
}

func ErrNotifyDefault(msg string) {
	ErrNotify(AtAll, msg)
}

func ErrNotify(atUser string, msg string) {
	Notify(atUser, "发生【异常】", msg)
}
