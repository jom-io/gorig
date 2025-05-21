package test

import (
	"github.com/gin-gonic/gin"
	"github.com/jom-io/gorig/httpx"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(httpx.Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "error") {
		t.Logf("response: %s", w.Body.String())
		t.Errorf("expected response to contain error message")
	}
}
