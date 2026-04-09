package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/HiroLiang/tentserv-chat-server/internal/interface/http/middleware"
	"github.com/gin-gonic/gin"
)

type registerRateLimiterStub struct {
	err error
}

func (s *registerRateLimiterStub) CheckRegisterAttempt(_ context.Context, _ string) error {
	return s.err
}

func newRegisterRateLimitRouter(limiter *registerRateLimiterStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/register", middleware.RegisterRateLimitMiddleware(limiter), func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})
	return router
}

func performRegisterLimitRequest(router *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/register", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func TestRegisterRateLimitMiddleware_PassthroughWhenUnderLimitHasStructuredLog(t *testing.T) {
	start := time.Now()
	limiter := &registerRateLimiterStub{err: nil}
	router := newRegisterRateLimitRouter(limiter)

	t.Log("Given: register rate limiter reports no limit exceeded")
	t.Log("Input: IP within allowed registration attempt window")
	t.Log("Action: POST /register through RegisterRateLimitMiddleware")

	resp := performRegisterLimitRequest(router)

	t.Logf("Output: status=%d", resp.Code)
	t.Log("Mutation: none")
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 when under rate limit, got %d", resp.Code)
	}
}

func TestRegisterRateLimitMiddleware_RejectsWhenLimitExceededHasStructuredLog(t *testing.T) {
	start := time.Now()
	limiter := &registerRateLimiterStub{err: errors.New("rate limit exceeded")}
	router := newRegisterRateLimitRouter(limiter)

	t.Log("Given: register rate limiter signals limit exceeded for this IP")
	t.Log("Input: IP has exceeded the per-IP registration attempt cap")
	t.Log("Action: POST /register through RegisterRateLimitMiddleware")

	resp := performRegisterLimitRequest(router)

	t.Logf("Output: status=%d body=%s", resp.Code, resp.Body.String())
	t.Log("Mutation: request aborted before handler")
	t.Logf("Duration: %s", time.Since(start))

	if resp.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 when rate limit exceeded, got %d body=%s", resp.Code, resp.Body.String())
	}
}
