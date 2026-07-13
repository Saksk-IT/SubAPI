package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIImageDispatchObserverStub struct {
	allow bool
	seen  atomic.Int32
}

func (s *openAIImageDispatchObserverStub) MarkDispatched() bool {
	s.seen.Add(1)
	return s.allow
}

func (s *openAIImageDispatchObserverStub) Dispatched() bool { return s.allow && s.seen.Load() > 0 }

type openAIImageDispatchUpstreamStub struct{ calls atomic.Int32 }

func (s *openAIImageDispatchUpstreamStub) Do(*http.Request, string, int64, int) (*http.Response, error) {
	s.calls.Add(1)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(`{"data":[{"b64_json":"aA=="}]}`)),
	}, nil
}

func (s *openAIImageDispatchUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, concurrency)
}

func TestOpenAIImageJobDispatchGateStopsBothImageTransportsBeforeDo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name    string
		account *Account
	}{
		{
			name: "api key",
			account: &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
				"api_key": "upstream-secret", "base_url": "https://images.example/v1",
			}},
		},
		{
			name: "oauth",
			account: &Account{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{
				"access_token": "oauth-secret", "chatgpt_account_id": "acct-1",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"model":"gpt-image-2","prompt":"cat","response_format":"b64_json"}`)
			req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = req

			upstream := &openAIImageDispatchUpstreamStub{}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
			parsed, err := svc.ParseOpenAIImagesRequest(c, body)
			require.NoError(t, err)
			observer := &openAIImageDispatchObserverStub{allow: false}
			ctx := WithOpenAIImageJobExecutionObserver(context.Background(), observer)

			result, err := svc.ForwardImages(ctx, c, tt.account, body, parsed, "")

			require.Nil(t, result)
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, int32(1), observer.seen.Load())
			require.Zero(t, upstream.calls.Load(), "dispatch denial must prevent upstream Do")
		})
	}
}

func TestOpenAIImageJobDispatchGateIsAbsentForNormalHTTPRequests(t *testing.T) {
	body := []byte(`{"model":"gpt-image-2","prompt":"cat","response_format":"b64_json"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	upstream := &openAIImageDispatchUpstreamStub{}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	account := &Account{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
		"api_key": "upstream-secret", "base_url": "https://images.example/v1",
	}}

	result, err := svc.ForwardImages(context.Background(), c, account, body, parsed, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int32(1), upstream.calls.Load())
}

func TestOpenAIImageJobDispatchGateDeniesCrossTransportFailoverAfterFirstDo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	accounts := []*Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{
			"api_key": "upstream-secret", "base_url": "https://images.example/v1",
		}},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{
			"access_token": "oauth-secret", "chatgpt_account_id": "acct-1",
		}},
	}
	for firstIndex, name := range []string{"api-key-then-oauth", "oauth-then-api-key"} {
		t.Run(name, func(t *testing.T) {
			body := []byte(`{"model":"gpt-image-2","prompt":"cat","response_format":"b64_json"}`)
			req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = req
			upstream := &openAIImageDispatchUpstreamStub{}
			svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
			parsed, err := svc.ParseOpenAIImagesRequest(c, body)
			require.NoError(t, err)
			observer := &openAIImageJobExecutionObserver{}
			ctx := WithOpenAIImageJobExecutionObserver(context.Background(), observer)

			_, _ = svc.ForwardImages(ctx, c, accounts[firstIndex], body, parsed, "")
			require.Equal(t, int32(1), upstream.calls.Load(), "first transport must be the only upstream Do")

			secondIndex := 1 - firstIndex
			result, err := svc.ForwardImages(ctx, c, accounts[secondIndex], body, parsed, "")
			require.Nil(t, result)
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, int32(1), upstream.calls.Load(), "failover transport must be denied before Do")
			require.True(t, observer.Dispatched())
		})
	}
}
