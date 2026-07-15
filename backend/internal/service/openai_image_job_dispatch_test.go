package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

type cancelingOpenAIImageDispatchUpstream struct {
	delegate *httpUpstreamRecorder
	cancel   context.CancelFunc
	cancelAt int32
	calls    atomic.Int32
}

func (s *cancelingOpenAIImageDispatchUpstream) Do(req *http.Request, proxyURL string, accountID int64, concurrency int) (*http.Response, error) {
	if s.calls.Add(1) == s.cancelAt {
		s.cancel()
	}
	return s.delegate.Do(req, proxyURL, accountID, concurrency)
}

func (s *cancelingOpenAIImageDispatchUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
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

func TestOpenAIImageJobDispatchGateAllowsSingleAgentIdentityTaskRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key, privateKey := newTestAgentIdentityKey(t)
	account := &Account{
		ID:          3,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"auth_mode":          OpenAIAuthModeAgentIdentity,
			"agent_runtime_id":   key.runtimeID,
			"agent_private_key":  privateKey,
			"task_id":            "task-old",
			"chatgpt_account_id": "account-agent-image",
		},
	}
	repo := &agentIdentityForwardRepo{account: account}
	registerCalls := 0
	registerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		registerCalls++
		_, _ = io.WriteString(w, `{"task_id":"task-new"}`)
	}))
	defer registerServer.Close()
	oldBase := openAIAgentIdentityAuthAPIBaseURL
	openAIAgentIdentityAuthAPIBaseURL = registerServer.URL
	t.Cleanup(func() { openAIAgentIdentityAuthAPIBaseURL = oldBase })

	successBody := "data: {\"type\":\"response.completed\",\"response\":{\"created_at\":1710000000,\"usage\":{\"input_tokens\":1,\"output_tokens\":2,\"total_tokens\":3,\"output_tokens_details\":{\"image_tokens\":1}},\"output\":[{\"type\":\"image_generation_call\",\"result\":\"aA==\",\"output_format\":\"png\",\"quality\":\"high\",\"size\":\"1024x1024\"}]}}\n\ndata: [DONE]\n\n"
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"invalid_task_id"}}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(successBody)),
		},
	}}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, accountRepo: repo, httpUpstream: upstream}
	body := []byte(`{"model":"gpt-image-2","prompt":"cat","response_format":"b64_json"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	observer := &openAIImageJobExecutionObserver{}
	ctx := WithOpenAIImageJobExecutionObserver(context.Background(), observer)

	result, err := svc.ForwardImages(ctx, c, account, body, parsed, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, result.ImageCount)
	require.Equal(t, 1, registerCalls)
	require.Len(t, upstream.requests, 2)
	require.True(t, observer.Dispatched(), "task recovery must retain the original one-way dispatch claim")
	require.Equal(t, "task-new", account.GetCredential("task_id"))
	require.Equal(t, "task-old", decodeAgentAssertionTask(t, upstream.requests[0].Header.Get("Authorization")))
	require.Equal(t, "task-new", decodeAgentAssertionTask(t, upstream.requests[1].Header.Get("Authorization")))
}

func TestOpenAIImageJobDispatchGateAllowsAllInvalidAgentIdentityBatchRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key, privateKey := newTestAgentIdentityKey(t)
	account := &Account{
		ID:          4,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 2,
		Credentials: map[string]any{
			"auth_mode":          OpenAIAuthModeAgentIdentity,
			"agent_runtime_id":   key.runtimeID,
			"agent_private_key":  privateKey,
			"task_id":            "task-batch-old",
			"chatgpt_account_id": "account-agent-image-batch",
		},
	}
	repo := &agentIdentityForwardRepo{account: account}
	registerCalls := 0
	registerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		registerCalls++
		_, _ = io.WriteString(w, `{"task_id":"task-batch-new"}`)
	}))
	defer registerServer.Close()
	oldBase := openAIAgentIdentityAuthAPIBaseURL
	openAIAgentIdentityAuthAPIBaseURL = registerServer.URL
	t.Cleanup(func() { openAIAgentIdentityAuthAPIBaseURL = oldBase })

	invalidTaskResponse := func() *http.Response {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"invalid_task_id"}}`)),
		}
	}
	successBody := "data: {\"type\":\"response.completed\",\"response\":{\"created_at\":1710000000,\"usage\":{\"input_tokens\":1,\"output_tokens\":2,\"total_tokens\":3,\"output_tokens_details\":{\"image_tokens\":1}},\"output\":[{\"type\":\"image_generation_call\",\"result\":\"aA==\",\"output_format\":\"png\",\"quality\":\"high\",\"size\":\"1024x1024\"}]}}\n\ndata: [DONE]\n\n"
	successResponse := func() *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(successBody)),
		}
	}
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		invalidTaskResponse(), invalidTaskResponse(), successResponse(), successResponse(),
	}}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, accountRepo: repo, httpUpstream: upstream}
	body := []byte(`{"model":"gpt-image-2","prompt":"cat","n":2,"response_format":"b64_json"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	observer := &openAIImageJobExecutionObserver{}
	ctx := WithOpenAIImageJobExecutionObserver(context.Background(), observer)

	result, err := svc.ForwardImages(ctx, c, account, body, parsed, "")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, result.ImageCount)
	require.Equal(t, 1, registerCalls)
	require.Len(t, upstream.requests, 4)
	require.True(t, observer.Dispatched(), "batch task recovery must retain the original one-way dispatch claim")
	require.Equal(t, "task-batch-new", account.GetCredential("task_id"))
	for _, request := range upstream.requests[:2] {
		require.Equal(t, "task-batch-old", decodeAgentAssertionTask(t, request.Header.Get("Authorization")))
	}
	for _, request := range upstream.requests[2:] {
		require.Equal(t, "task-batch-new", decodeAgentAssertionTask(t, request.Header.Get("Authorization")))
	}
}

func TestOpenAIImageJobDispatchGateDoesNotRetryMixedAgentIdentityBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key, privateKey := newTestAgentIdentityKey(t)
	account := &Account{
		ID:          5,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 2,
		Credentials: map[string]any{
			"auth_mode":          OpenAIAuthModeAgentIdentity,
			"agent_runtime_id":   key.runtimeID,
			"agent_private_key":  privateKey,
			"task_id":            "task-mixed-old",
			"chatgpt_account_id": "account-agent-image-mixed",
		},
	}
	repo := &agentIdentityForwardRepo{account: account}
	registerCalls := 0
	registerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		registerCalls++
		_, _ = io.WriteString(w, `{"task_id":"task-mixed-new"}`)
	}))
	defer registerServer.Close()
	oldBase := openAIAgentIdentityAuthAPIBaseURL
	openAIAgentIdentityAuthAPIBaseURL = registerServer.URL
	t.Cleanup(func() { openAIAgentIdentityAuthAPIBaseURL = oldBase })

	successBody := "data: {\"type\":\"response.completed\",\"response\":{\"created_at\":1710000000,\"usage\":{\"input_tokens\":1,\"output_tokens\":2,\"total_tokens\":3,\"output_tokens_details\":{\"image_tokens\":1}},\"output\":[{\"type\":\"image_generation_call\",\"result\":\"aA==\",\"output_format\":\"png\",\"quality\":\"high\",\"size\":\"1024x1024\"}]}}\n\ndata: [DONE]\n\n"
	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"invalid_task_id"}}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(successBody)),
		},
	}}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, accountRepo: repo, httpUpstream: upstream}
	body := []byte(`{"model":"gpt-image-2","prompt":"cat","n":2,"response_format":"b64_json"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	observer := &openAIImageJobExecutionObserver{}
	ctx := WithOpenAIImageJobExecutionObserver(context.Background(), observer)

	result, err := svc.ForwardImages(ctx, c, account, body, parsed, "")

	require.Error(t, err)
	require.Nil(t, result)
	require.Zero(t, registerCalls, "a mixed batch may contain billable work and must never be replayed")
	require.Len(t, upstream.requests, 2)
	require.True(t, observer.Dispatched())
	require.Equal(t, "task-mixed-old", account.GetCredential("task_id"))
}

func TestOpenAIImageJobDispatchGateDoesNotRecoverCanceledAgentIdentityBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key, privateKey := newTestAgentIdentityKey(t)
	account := &Account{
		ID:          6,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 2,
		Credentials: map[string]any{
			"auth_mode":          OpenAIAuthModeAgentIdentity,
			"agent_runtime_id":   key.runtimeID,
			"agent_private_key":  privateKey,
			"task_id":            "task-canceled-old",
			"chatgpt_account_id": "account-agent-image-canceled",
		},
	}
	repo := &agentIdentityForwardRepo{account: account}
	var registerCalls atomic.Int32
	registerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		registerCalls.Add(1)
		_, _ = io.WriteString(w, `{"task_id":"task-canceled-new"}`)
	}))
	defer registerServer.Close()
	oldBase := openAIAgentIdentityAuthAPIBaseURL
	openAIAgentIdentityAuthAPIBaseURL = registerServer.URL
	t.Cleanup(func() { openAIAgentIdentityAuthAPIBaseURL = oldBase })

	invalidTaskResponse := func() *http.Response {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"invalid_task_id"}}`)),
		}
	}
	delegate := &httpUpstreamRecorder{responses: []*http.Response{invalidTaskResponse(), invalidTaskResponse()}}
	requestCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	upstream := &cancelingOpenAIImageDispatchUpstream{delegate: delegate, cancel: cancel, cancelAt: 2}
	svc := &OpenAIGatewayService{cfg: &config.Config{}, accountRepo: repo, httpUpstream: upstream}
	body := []byte(`{"model":"gpt-image-2","prompt":"cat","n":2,"response_format":"b64_json"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)

	result, err := svc.ForwardImages(requestCtx, c, account, body, parsed, "")

	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, result)
	require.Zero(t, registerCalls.Load(), "canceled client request must not register a task")
	require.Len(t, delegate.requests, 2)
	require.Equal(t, "task-canceled-old", account.GetCredential("task_id"))
}
