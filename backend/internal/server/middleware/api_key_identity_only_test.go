//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyIdentityOnlyBypassesGenerationEligibilityChecks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	past := time.Now().Add(-time.Hour)
	tests := []struct {
		name   string
		mutate func(*service.APIKey)
	}{
		{
			name: "declared expired key",
			mutate: func(key *service.APIKey) {
				key.Status = service.StatusAPIKeyExpired
			},
		},
		{
			name: "declared quota exhausted key",
			mutate: func(key *service.APIKey) {
				key.Status = service.StatusAPIKeyQuotaExhausted
			},
		},
		{
			name: "runtime expired key",
			mutate: func(key *service.APIKey) {
				key.ExpiresAt = &past
			},
		},
		{
			name: "runtime quota exhausted key",
			mutate: func(key *service.APIKey) {
				key.Quota = 1
				key.QuotaUsed = 1
			},
		},
		{
			name: "zero balance",
			mutate: func(key *service.APIKey) {
				key.User.Balance = 0
			},
		},
		{
			name: "disabled group",
			mutate: func(key *service.APIKey) {
				key.Group.Status = service.StatusDisabled
			},
		},
		{
			name: "exclusive group no longer allowed",
			mutate: func(key *service.APIKey) {
				key.Group.IsExclusive = true
				key.User.AllowedGroups = nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := newAPIKeyIdentityOnlyTestKey()
			tt.mutate(apiKey)

			var touchedID int64
			repo := &stubApiKeyRepo{
				getByKey: func(_ context.Context, key string) (*service.APIKey, error) {
					if key != apiKey.Key {
						return nil, service.ErrAPIKeyNotFound
					}
					return apiKey, nil
				},
				updateLastUsed: func(_ context.Context, id int64, _ time.Time) error {
					touchedID = id
					return nil
				},
			}
			cfg := &config.Config{RunMode: config.RunModeStandard}
			apiKeyService := service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, cfg)
			router := newAPIKeyIdentityOnlyTestRouter(t, apiKeyService, nil, cfg, apiKey)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/v1/images/jobs/job-1", nil)
			req.Header.Set("x-api-key", apiKey.Key)
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Equal(t, apiKey.ID, touchedID)
		})
	}
}

func TestAPIKeyIdentityOnlyDoesNotLoadSubscription(t *testing.T) {
	gin.SetMode(gin.TestMode)

	apiKey := newAPIKeyIdentityOnlyTestKey()
	apiKey.Group.SubscriptionType = service.SubscriptionTypeSubscription

	repo := &stubApiKeyRepo{
		getByKey: func(_ context.Context, key string) (*service.APIKey, error) {
			if key != apiKey.Key {
				return nil, service.ErrAPIKeyNotFound
			}
			return apiKey, nil
		},
	}
	var subscriptionLookups int
	subRepo := &stubUserSubscriptionRepo{
		getActive: func(context.Context, int64, int64) (*service.UserSubscription, error) {
			subscriptionLookups++
			return nil, service.ErrSubscriptionNotFound
		},
	}
	cfg := &config.Config{RunMode: config.RunModeStandard}
	apiKeyService := service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, cfg)
	subscriptionService := service.NewSubscriptionService(nil, subRepo, nil, nil, cfg)
	t.Cleanup(subscriptionService.Stop)

	router := newAPIKeyIdentityOnlyTestRouter(t, apiKeyService, subscriptionService, cfg, apiKey)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/images/jobs/job-1", nil)
	req.Header.Set("x-api-key", apiKey.Key)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Zero(t, subscriptionLookups, "identity-only authentication must not load billing subscriptions")
}

func TestAPIKeyIdentityOnlyStillEnforcesIdentityChecks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		mutate     func(*service.APIKey)
		requested  string
		remoteAddr string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "unknown key",
			requested:  "unknown-key",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "INVALID_API_KEY",
		},
		{
			name: "disabled key",
			mutate: func(key *service.APIKey) {
				key.Status = service.StatusAPIKeyDisabled
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "API_KEY_DISABLED",
		},
		{
			name: "inactive user",
			mutate: func(key *service.APIKey) {
				key.User.Status = service.StatusDisabled
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "USER_INACTIVE",
		},
		{
			name: "missing user",
			mutate: func(key *service.APIKey) {
				key.User = nil
			},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "INVALID_API_KEY",
		},
		{
			name: "IP mismatch",
			mutate: func(key *service.APIKey) {
				key.IPWhitelist = []string{"1.2.3.4"}
			},
			remoteAddr: "9.9.9.9:12345",
			wantStatus: http.StatusForbidden,
			wantCode:   "ACCESS_DENIED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := newAPIKeyIdentityOnlyTestKey()
			if tt.mutate != nil {
				tt.mutate(apiKey)
			}
			repo := &stubApiKeyRepo{
				getByKey: func(_ context.Context, key string) (*service.APIKey, error) {
					if key != apiKey.Key {
						return nil, service.ErrAPIKeyNotFound
					}
					return apiKey, nil
				},
			}
			cfg := &config.Config{RunMode: config.RunModeStandard}
			apiKeyService := service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, cfg)
			router := gin.New()
			require.NoError(t, router.SetTrustedProxies(nil))
			router.Use(APIKeyIdentityOnly())
			router.Use(gin.HandlerFunc(NewAPIKeyAuthMiddleware(apiKeyService, nil, cfg)))
			router.GET("/v1/images/jobs/:id", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			requestedKey := tt.requested
			if requestedKey == "" {
				requestedKey = apiKey.Key
			}
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/v1/images/jobs/job-1", nil)
			req.Header.Set("x-api-key", requestedKey)
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			}
			router.ServeHTTP(w, req)

			require.Equal(t, tt.wantStatus, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), tt.wantCode)
		})
	}
}

func TestAPIKeyIdentityOnlyMarkerDoesNotChangeOrdinaryAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		mutate     func(*service.APIKey)
		wantStatus int
		wantCode   string
	}{
		{
			name: "expired key remains rejected without marker",
			mutate: func(key *service.APIKey) {
				key.Status = service.StatusAPIKeyExpired
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "API_KEY_EXPIRED",
		},
		{
			name: "quota exhausted key remains rejected without marker",
			mutate: func(key *service.APIKey) {
				key.Status = service.StatusAPIKeyQuotaExhausted
			},
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "API_KEY_QUOTA_EXHAUSTED",
		},
		{
			name: "zero balance remains rejected without marker",
			mutate: func(key *service.APIKey) {
				key.User.Balance = 0
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "INSUFFICIENT_BALANCE",
		},
		{
			name: "disabled group remains rejected without marker",
			mutate: func(key *service.APIKey) {
				key.Group.Status = service.StatusDisabled
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "GROUP_DISABLED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := newAPIKeyIdentityOnlyTestKey()
			tt.mutate(apiKey)
			repo := &stubApiKeyRepo{
				getByKey: func(_ context.Context, key string) (*service.APIKey, error) {
					if key != apiKey.Key {
						return nil, service.ErrAPIKeyNotFound
					}
					return apiKey, nil
				},
			}
			cfg := &config.Config{RunMode: config.RunModeStandard}
			apiKeyService := service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, cfg)
			router := newAuthTestRouter(apiKeyService, nil, cfg)

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/t", nil)
			req.Header.Set("x-api-key", apiKey.Key)
			router.ServeHTTP(w, req)

			require.Equal(t, tt.wantStatus, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), tt.wantCode)
		})
	}
}

func newAPIKeyIdentityOnlyTestKey() *service.APIKey {
	groupID := int64(300)
	user := &service.User{
		ID:            70,
		Role:          service.RoleUser,
		Status:        service.StatusActive,
		Balance:       10,
		Concurrency:   4,
		AllowedGroups: []int64{groupID},
	}
	return &service.APIKey{
		ID:      100,
		UserID:  user.ID,
		GroupID: &groupID,
		Key:     "identity-only-key",
		Status:  service.StatusActive,
		User:    user,
		Group: &service.Group{
			ID:       groupID,
			Name:     "openai",
			Platform: service.PlatformOpenAI,
			Status:   service.StatusActive,
			Hydrated: true,
		},
	}
}

func newAPIKeyIdentityOnlyTestRouter(
	t *testing.T,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	cfg *config.Config,
	wantKey *service.APIKey,
) *gin.Engine {
	t.Helper()
	router := gin.New()
	router.Use(APIKeyIdentityOnly())
	router.Use(gin.HandlerFunc(NewAPIKeyAuthMiddleware(apiKeyService, subscriptionService, cfg)))
	router.GET("/v1/images/jobs/:id", func(c *gin.Context) {
		apiKey, ok := GetAPIKeyFromContext(c)
		require.True(t, ok)
		require.Equal(t, wantKey.ID, apiKey.ID)

		subject, ok := GetAuthSubjectFromContext(c)
		require.True(t, ok)
		require.Equal(t, wantKey.User.ID, subject.UserID)
		require.Equal(t, wantKey.User.Concurrency, subject.Concurrency)

		role, ok := GetUserRoleFromContext(c)
		require.True(t, ok)
		require.Equal(t, wantKey.User.Role, role)

		userID, ok := c.Request.Context().Value(ctxkey.UserID).(int64)
		require.True(t, ok)
		require.Equal(t, wantKey.User.ID, userID)
		require.Nil(t, c.Request.Context().Value(ctxkey.Group))
		_, hasSubscription := GetSubscriptionFromContext(c)
		require.False(t, hasSubscription)

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return router
}
