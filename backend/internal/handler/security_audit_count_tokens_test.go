package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCountTokensPromptAuditBlocksBeforeBillingSchedulingAndUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		platform string
		handler  func(*securityaudit.Coordinator) gin.HandlerFunc
	}{
		{
			name:     "anthropic gateway",
			platform: service.PlatformAnthropic,
			handler: func(coordinator *securityaudit.Coordinator) gin.HandlerFunc {
				h := &GatewayHandler{securityAuditCoordinator: coordinator, cfg: &config.Config{}}
				return h.CountTokens
			},
		},
		{
			name:     "openai compatibility gateway",
			platform: service.PlatformOpenAI,
			handler: func(coordinator *securityaudit.Coordinator) gin.HandlerFunc {
				// These non-nil shells satisfy the handler's dependency preflight. The
				// billing shell intentionally has no config, so any fallthrough past the
				// audit gate would panic before account scheduling or upstream dispatch.
				h := &OpenAIGatewayHandler{
					gatewayService:           &service.OpenAIGatewayService{},
					billingCacheService:      &service.BillingCacheService{},
					apiKeyService:            &service.APIKeyService{},
					concurrencyHelper:        &ConcurrencyHelper{concurrencyService: &service.ConcurrencyService{}},
					securityAuditCoordinator: coordinator,
					cfg:                      &config.Config{},
				}
				return h.CountTokens
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := blockingHandlerPromptEngine()
			coordinator := securityaudit.NewCoordinator(nil, engine)
			selectedAccount := false
			router := gin.New()
			router.Use(func(c *gin.Context) {
				groupID := int64(3)
				user := &service.User{ID: 7, Username: "count-user", Email: "count@example.test"}
				c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
					ID: 9, UserID: user.ID, User: user, Name: "count-key", GroupID: &groupID,
					Group: &service.Group{ID: groupID, Name: "count-group", Platform: tt.platform, AllowMessagesDispatch: true},
				})
				c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID, Concurrency: 2})
				c.Next()
				_, selectedAccount = c.Get(opsAccountIDKey)
			})
			router.POST("/v1/messages/count_tokens", tt.handler(coordinator))

			body := `{"model":"claude-test","system":"blocked system prompt","messages":[{"role":"user","content":"blocked user prompt"}]}`
			request := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			require.NotPanics(t, func() { router.ServeHTTP(recorder, request) })

			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Contains(t, recorder.Body.String(), securityaudit.ErrorCodeBlocked)
			require.False(t, selectedAccount, "blocking must happen before account scheduling")
			evaluated, _, requests := engine.snapshot()
			require.Equal(t, 1, evaluated)
			require.Len(t, requests, 1)
			require.Equal(t, service.ContentModerationProtocolAnthropicMessages, requests[0].Protocol)
			require.Equal(t, "claude-test", requests[0].Model)
			require.Contains(t, string(requests[0].Body), "blocked system prompt")
			require.Contains(t, string(requests[0].Body), "blocked user prompt")
		})
	}
}
