//go:build unit

package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type batchRefreshOpenAIClientStub struct{}

func (batchRefreshOpenAIClientStub) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL, clientID string) (*openai.TokenResponse, error) {
	panic("unexpected call")
}

func (batchRefreshOpenAIClientStub) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*openai.TokenResponse, error) {
	return &openai.TokenResponse{
		AccessToken:  "refreshed-access",
		RefreshToken: "refreshed-refresh",
		ExpiresIn:    3600,
	}, nil
}

func (batchRefreshOpenAIClientStub) RefreshTokenWithClientID(ctx context.Context, refreshToken, proxyURL string, clientID string) (*openai.TokenResponse, error) {
	return &openai.TokenResponse{
		AccessToken:  "refreshed-access",
		RefreshToken: "refreshed-refresh",
		ExpiresIn:    3600,
	}, nil
}

type batchRefreshFailableService struct {
	*stubAdminService
	failOnAccountID int64
}

func (s *batchRefreshFailableService) GetAccountsByIDs(ctx context.Context, ids []int64) ([]*service.Account, error) {
	accounts := make([]*service.Account, 0, len(ids))
	for _, id := range ids {
		switch id {
		case 1:
			accounts = append(accounts, &service.Account{
				ID:       1,
				Name:     "alpha",
				Platform: service.PlatformOpenAI,
				Type:     service.AccountTypeOAuth,
				Status:   service.StatusActive,
				Credentials: map[string]any{
					"refresh_token": "rt-1",
				},
			})
		case 2:
			accounts = append(accounts, &service.Account{
				ID:       2,
				Name:     "beta",
				Platform: service.PlatformOpenAI,
				Type:     service.AccountTypeOAuth,
				Status:   service.StatusActive,
				Credentials: map[string]any{
					"refresh_token": "rt-2",
				},
			})
		default:
			accounts = append(accounts, nil)
		}
	}
	return accounts, nil
}

func (s *batchRefreshFailableService) UpdateAccount(ctx context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error) {
	if id == s.failOnAccountID {
		return nil, errors.New("database error")
	}
	return s.stubAdminService.UpdateAccount(ctx, id, input)
}

func setupBatchRefreshRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	openaiSvc := service.NewOpenAIOAuthService(nil, batchRefreshOpenAIClientStub{})
	handler := NewAccountHandler(adminSvc, nil, openaiSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/batch-refresh", handler.BatchRefresh)
	return router
}

func TestBatchRefreshReturnsPerAccountResults(t *testing.T) {
	svc := &batchRefreshFailableService{
		stubAdminService: newStubAdminService(),
		failOnAccountID:  2,
	}
	router := setupBatchRefreshRouter(svc)

	body, err := json.Marshal(map[string]any{"account_ids": []int64{1, 2, 999}})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)

	require.Equal(t, float64(3), data["total"])
	require.Equal(t, float64(1), data["success"])
	require.Equal(t, float64(2), data["failed"])
	require.ElementsMatch(t, []any{float64(1)}, data["success_ids"])
	require.ElementsMatch(t, []any{float64(2), float64(999)}, data["failed_ids"])

	results := data["results"].([]any)
	require.Len(t, results, 3)

	resultByID := make(map[float64]map[string]any, len(results))
	for _, item := range results {
		result := item.(map[string]any)
		resultByID[result["account_id"].(float64)] = result
	}

	require.Equal(t, "alpha", resultByID[1]["name"])
	require.Equal(t, true, resultByID[1]["success"])
	require.Equal(t, "beta", resultByID[2]["name"])
	require.Equal(t, false, resultByID[2]["success"])
	require.Equal(t, "database error", resultByID[2]["error"])
	require.Equal(t, false, resultByID[999]["success"])
	require.Equal(t, "account not found", resultByID[999]["error"])
}
