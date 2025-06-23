package test

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/httpx/ssex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mock handler for testing SSE
func sseHandler(c *gin.Context) {
	err := ssex.SendOK(c, "test", map[string]interface{}{
		"foo": "bar",
	})
	if err != nil {
		return
	}
}

func TestSSEMiddleware_AllowsGET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/sse", ssex.Mid(), sseHandler)

	req, _ := http.NewRequest(http.MethodGet, "/sse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	respBody := w.Body.String()

	fmt.Println("Response Body:", respBody) // Log the response body for debugging
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !strings.Contains(respBody, "event: test") {
		t.Errorf("expected 'event: test' in response, got %s", respBody)
	}

	if !strings.Contains(respBody, `"status":"ok"`) {
		t.Errorf("expected 'status: ok' in JSON, got %s", respBody)
	}

	if !strings.Contains(w.Header().Get("Content-Type"), "text/event-stream") {
		t.Errorf("expected Content-Type text/event-stream, got %s", w.Header().Get("Content-Type"))
	}
}

func TestSSEMiddleware_RejectsNonGET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/sse", ssex.Mid(), sseHandler)

	body := bytes.NewBuffer(nil)
	req, _ := http.NewRequest(http.MethodPost, "/sse", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}

	expected := `{"status":"error","message":"Only GET method is allowed"}`
	if strings.TrimSpace(w.Body.String()) != expected {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}
