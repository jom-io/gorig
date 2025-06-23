package ssex

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

type sseEvent string

func (event sseEvent) String() string {
	return string(event)
}

type ssePayload struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func Mid() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.AbortWithStatusJSON(http.StatusMethodNotAllowed, gin.H{
				"status":  "error",
				"message": "Only GET method is allowed",
			})
			return
		}

		header := c.Writer.Header()
		header.Set("Content-Type", "text/event-stream")
		header.Set("Cache-Control", "no-cache")
		header.Set("Connection", "keep-alive")
		header.Set("Access-Control-Allow-Origin", "*")

		c.Writer.Flush()
	}
}

func SendOK(c *gin.Context, event sseEvent, data interface{}) error {
	return send(c, event, ssePayload{
		Status: "ok",
		Data:   data,
	})
}

func SendError(c *gin.Context, event sseEvent, message string) error {
	return send(c, event, ssePayload{
		Status:  "error",
		Message: message,
	})
}

func send(c *gin.Context, event sseEvent, payload ssePayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, string(jsonData))
	if err != nil {
		return err
	}

	c.Writer.Flush()
	return nil
}
