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
		{
			name:     "grok local estimation",
			platform: service.PlatformGrok,
			handler: func(coordinator *securityaudit.Coordinator) gin.HandlerFunc {
				h := &OpenAIGatewayHandler{
					securityAuditCoordinator: coordinator,
					cfg: &config.Config{Gateway: config.GatewayConfig{
						MaxBodySize: 1024 * 1024,
					}},
				}
				return h.GrokCountTokens
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
			paths := []string{"/v1/messages/count_tokens"}
			if tt.platform == service.PlatformGrok {
				paths = append(paths, "/messages/count_tokens")
			}
			for _, path := range paths {
				router.POST(path, tt.handler(coordinator))
			}

			body := `{"model":"claude-test","system":"blocked system prompt","messages":[{"role":"user","content":"blocked user prompt"}]}`
			for requestIndex, path := range paths {
				request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				require.NotPanics(t, func() { router.ServeHTTP(recorder, request) })

				require.Equal(t, http.StatusForbidden, recorder.Code, "path=%s", path)
				require.Contains(t, recorder.Body.String(), securityaudit.ErrorCodeBlocked, "path=%s", path)
				require.False(t, selectedAccount, "blocking must happen before account scheduling")
				evaluated, _, requests := engine.snapshot()
				require.Equal(t, requestIndex+1, evaluated)
				require.Len(t, requests, requestIndex+1)
				latest := requests[requestIndex]
				require.Equal(t, service.ContentModerationProtocolAnthropicMessages, latest.Protocol)
				require.Equal(t, "claude-test", latest.Model)
				require.Contains(t, string(latest.Body), "blocked system prompt")
				require.Contains(t, string(latest.Body), "blocked user prompt")
			}
		})
	}
}
