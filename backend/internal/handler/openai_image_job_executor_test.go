package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type openAIImageJobAPIKeyProviderStub struct {
	key         *service.APIKey
	err         error
	invalidated string
	getCalls    atomic.Int32
}

func (s *openAIImageJobAPIKeyProviderStub) GetByID(context.Context, int64) (*service.APIKey, error) {
	s.getCalls.Add(1)
	return s.key, s.err
}

func (s *openAIImageJobAPIKeyProviderStub) InvalidateAuthCacheByKey(_ context.Context, key string) {
	s.invalidated = key
}

type openAIImageJobObserverStub struct {
	allowDispatch bool
	dispatched    atomic.Bool
}

func (s *openAIImageJobObserverStub) MarkDispatched() bool {
	if !s.allowDispatch {
		return false
	}
	s.dispatched.Store(true)
	return true
}

func (s *openAIImageJobObserverStub) Dispatched() bool { return s.dispatched.Load() }

type openAIImageJobAuthRepoStub struct {
	service.APIKeyRepository
	key *service.APIKey
}

func (r *openAIImageJobAuthRepoStub) GetByID(context.Context, int64) (*service.APIKey, error) {
	return cloneOpenAIImageJobAuthKey(r.key), nil
}

func (r *openAIImageJobAuthRepoStub) GetByKeyForAuth(_ context.Context, key string) (*service.APIKey, error) {
	if r.key == nil || key != r.key.Key {
		return nil, service.ErrAPIKeyNotFound
	}
	return cloneOpenAIImageJobAuthKey(r.key), nil
}

func cloneOpenAIImageJobAuthKey(key *service.APIKey) *service.APIKey {
	if key == nil {
		return nil
	}
	clone := *key
	if key.User != nil {
		user := *key.User
		clone.User = &user
	}
	if key.Group != nil {
		group := *key.Group
		clone.Group = &group
	}
	clone.IPWhitelist = append([]string(nil), key.IPWhitelist...)
	clone.IPBlacklist = append([]string(nil), key.IPBlacklist...)
	return &clone
}

func openAIImageExecutorTestJob() *service.OpenAIImageJob {
	return &service.OpenAIImageJob{
		JobID:        "imgjob_executor_123",
		UserID:       42,
		APIKeyID:     9,
		Endpoint:     service.OpenAIImageJobEndpointGenerations,
		ContentType:  "application/json",
		RequestBody:  []byte(`{"model":"gpt-image-2","prompt":"cat"}`),
		ClientIP:     "203.0.113.7",
		UserAgent:    "image-playground-test",
		AttemptCount: 3,
	}
}

func openAIImageExecutorTestKey() *service.APIKey {
	return &service.APIKey{
		ID:     9,
		UserID: 42,
		Key:    "sk-current-secret",
		Status: service.StatusActive,
		User:   &service.User{ID: 42, Status: service.StatusActive},
	}
}

func TestOpenAIImageJobExecutorBuildsFreshSafeRequestAndSucceedsAfterBilling(t *testing.T) {
	provider := &openAIImageJobAPIKeyProviderStub{key: openAIImageExecutorTestKey()}
	observer := &openAIImageJobObserverStub{allowDispatch: true}
	var gotRequest *http.Request
	engine := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest = r.Clone(r.Context())
		require.True(t, service.MarkOpenAIImageJobDispatched(r.Context()))
		barrier, ok := openAIImageJobBillingBarrierFromContext(r.Context())
		require.True(t, ok)
		barrier.Record(r.Context(), func(ctx context.Context) error {
			require.Equal(t, "imgjob_executor_123", ctx.Value(ctxkey.ClientRequestID))
			require.Equal(t, "imgjob_executor_123/3", ctx.Value(ctxkey.RequestID))
			return nil
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"created":1,"data":[{"b64_json":"aGVsbG8="}]}`))
	})

	executor := newOpenAIImageJobExecutorForTest(provider, engine, &config.Config{})
	result := executor.Execute(context.Background(), openAIImageExecutorTestJob(), observer)

	require.Equal(t, service.OpenAIImageJobExecutionSucceeded, result.Outcome)
	require.Equal(t, http.StatusOK, result.Response.StatusCode)
	require.JSONEq(t, `{"created":1,"data":[{"b64_json":"aGVsbG8="}]}`, string(result.Response.Body))
	require.NotNil(t, gotRequest)
	require.Equal(t, http.MethodPost, gotRequest.Method)
	require.Equal(t, "/v1/images/generations", gotRequest.URL.Path)
	require.Empty(t, gotRequest.URL.RawQuery)
	require.Empty(t, gotRequest.URL.Fragment)
	require.Equal(t, "Bearer sk-current-secret", gotRequest.Header.Get("Authorization"))
	require.Equal(t, "application/json", gotRequest.Header.Get("Content-Type"))
	require.Equal(t, "image-playground-test", gotRequest.Header.Get("User-Agent"))
	require.Empty(t, gotRequest.Header.Get("Cookie"))
	require.Empty(t, gotRequest.Header.Get("Idempotency-Key"))
	require.Equal(t, "203.0.113.7:0", gotRequest.RemoteAddr)
	require.Equal(t, "imgjob_executor_123", gotRequest.Context().Value(ctxkey.ClientRequestID))
	require.Equal(t, "imgjob_executor_123/3", gotRequest.Context().Value(ctxkey.RequestID))
	require.Equal(t, "sk-current-secret", provider.invalidated)
}

func TestOpenAIImageJobExecutorRejectsMissingOrMismatchedKeyBeforeEngine(t *testing.T) {
	tests := []struct {
		name     string
		provider *openAIImageJobAPIKeyProviderStub
		wantCode string
	}{
		{name: "missing", provider: &openAIImageJobAPIKeyProviderStub{err: service.ErrAPIKeyNotFound}, wantCode: "api_key_unavailable"},
		{name: "owner changed", provider: &openAIImageJobAPIKeyProviderStub{key: &service.APIKey{ID: 9, UserID: 99, Key: "secret"}}, wantCode: "api_key_owner_changed"},
		{name: "secret empty", provider: &openAIImageJobAPIKeyProviderStub{key: &service.APIKey{ID: 9, UserID: 42}}, wantCode: "api_key_unavailable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var engineCalls atomic.Int32
			executor := newOpenAIImageJobExecutorForTest(tt.provider, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				engineCalls.Add(1)
			}), &config.Config{})
			result := executor.Execute(context.Background(), openAIImageExecutorTestJob(), &openAIImageJobObserverStub{allowDispatch: true})
			require.Equal(t, service.OpenAIImageJobExecutionFailed, result.Outcome)
			require.Equal(t, tt.wantCode, result.ErrorCode)
			require.Zero(t, engineCalls.Load())
		})
	}
}

func TestOpenAIImageJobExecutorReplaysFullCurrentAuthentication(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	tests := []struct {
		name     string
		mutate   func(*service.APIKey)
		wantCode string
	}{
		{name: "key disabled", mutate: func(key *service.APIKey) { key.Status = service.StatusDisabled }, wantCode: "API_KEY_DISABLED"},
		{name: "user inactive", mutate: func(key *service.APIKey) { key.User.Status = service.StatusDisabled }, wantCode: "USER_INACTIVE"},
		{name: "group disabled", mutate: func(key *service.APIKey) { key.Group.Status = service.StatusDisabled }, wantCode: "GROUP_DISABLED"},
		{name: "runtime expired", mutate: func(key *service.APIKey) { key.ExpiresAt = &past }, wantCode: "API_KEY_EXPIRED"},
		{name: "runtime quota exhausted", mutate: func(key *service.APIKey) { key.Quota, key.QuotaUsed = 1, 1 }, wantCode: "API_KEY_QUOTA_EXHAUSTED"},
		{name: "balance exhausted", mutate: func(key *service.APIKey) { key.User.Balance = 0 }, wantCode: "INSUFFICIENT_BALANCE"},
		{name: "current IP ACL", mutate: func(key *service.APIKey) { key.IPWhitelist = []string{"198.51.100.4"} }, wantCode: "ACCESS_DENIED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groupID := int64(77)
			key := &service.APIKey{
				ID:      9,
				UserID:  42,
				Key:     "sk-current-secret",
				GroupID: &groupID,
				Status:  service.StatusActive,
				User: &service.User{
					ID:      42,
					Status:  service.StatusActive,
					Balance: 10,
				},
				Group: &service.Group{
					ID:                   groupID,
					Status:               service.StatusActive,
					Hydrated:             true,
					Platform:             service.PlatformOpenAI,
					AllowImageGeneration: true,
				},
			}
			tt.mutate(key)
			cfg := &config.Config{RunMode: config.RunModeStandard}
			cfg.Gateway.MaxBodySize = 1 << 20
			apiKeyService := service.NewAPIKeyService(
				&openAIImageJobAuthRepoStub{key: key}, nil, nil, nil, nil, nil, cfg,
			)
			executor := NewOpenAIImageJobExecutor(&OpenAIGatewayHandler{}, apiKeyService, nil, nil, nil, cfg)
			observer := &openAIImageJobObserverStub{allowDispatch: true}

			result := executor.Execute(context.Background(), openAIImageExecutorTestJob(), observer)

			require.Equal(t, service.OpenAIImageJobExecutionFailed, result.Outcome)
			require.Equal(t, tt.wantCode, result.ErrorCode)
			require.False(t, observer.Dispatched())
		})
	}
}

func TestOpenAIImageJobExecutorClassifiesPreAndPostDispatchFailures(t *testing.T) {
	provider := &openAIImageJobAPIKeyProviderStub{key: openAIImageExecutorTestKey()}

	t.Run("pre-dispatch rejection is known", func(t *testing.T) {
		executor := newOpenAIImageJobExecutorForTest(provider, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":{"code":"API_KEY_EXPIRED","message":"expired"}}`))
		}), &config.Config{})
		result := executor.Execute(context.Background(), openAIImageExecutorTestJob(), &openAIImageJobObserverStub{allowDispatch: true})
		require.Equal(t, service.OpenAIImageJobExecutionFailed, result.Outcome)
		require.Equal(t, "API_KEY_EXPIRED", result.ErrorCode)
		require.Equal(t, "expired", result.ErrorMessage)
	})

	t.Run("post-dispatch error is unknown", func(t *testing.T) {
		observer := &openAIImageJobObserverStub{allowDispatch: true}
		executor := newOpenAIImageJobExecutorForTest(provider, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.True(t, service.MarkOpenAIImageJobDispatched(r.Context()))
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":{"message":"transport failed"}}`))
		}), &config.Config{})
		result := executor.Execute(context.Background(), openAIImageExecutorTestJob(), observer)
		require.Equal(t, service.OpenAIImageJobExecutionFailedUnknown, result.Outcome)
		require.True(t, observer.Dispatched())
	})

	t.Run("dispatch gate denial is interrupted", func(t *testing.T) {
		observer := &openAIImageJobObserverStub{allowDispatch: false}
		executor := newOpenAIImageJobExecutorForTest(provider, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.False(t, service.MarkOpenAIImageJobDispatched(r.Context()))
			w.WriteHeader(http.StatusServiceUnavailable)
		}), &config.Config{})
		result := executor.Execute(context.Background(), openAIImageExecutorTestJob(), observer)
		require.Equal(t, service.OpenAIImageJobExecutionInterrupted, result.Outcome)
		require.False(t, observer.Dispatched())
	})
}

func TestOpenAIImageJobExecutorRequiresVerifiedImageAndResolvedBilling(t *testing.T) {
	provider := &openAIImageJobAPIKeyProviderStub{key: openAIImageExecutorTestKey()}
	tests := []struct {
		name       string
		body       string
		dispatch   bool
		bill       bool
		billingErr error
		want       service.OpenAIImageJobExecutionOutcome
		wantCode   string
	}{
		{name: "empty recorder 200", want: service.OpenAIImageJobExecutionFailed, wantCode: "image_generation_failed"},
		{name: "non json after dispatch", body: "not-json", dispatch: true, bill: true, want: service.OpenAIImageJobExecutionFailedUnknown, wantCode: "failed_unknown"},
		{name: "json without image", body: `{"data":[{}]}`, dispatch: true, bill: true, want: service.OpenAIImageJobExecutionFailedUnknown, wantCode: "failed_unknown"},
		{name: "billing missing", body: `{"data":[{"url":"https://example.test/image.png"}]}`, dispatch: true, want: service.OpenAIImageJobExecutionFailedUnknown, wantCode: "billing_failed_unknown"},
		{name: "billing failed", body: `{"data":[{"b64_json":"aA=="}]}`, dispatch: true, bill: true, billingErr: errors.New("db unavailable"), want: service.OpenAIImageJobExecutionFailedUnknown, wantCode: "billing_failed_unknown"},
		{name: "url image", body: `{"data":[{"url":"https://example.test/image.png"}]}`, dispatch: true, bill: true, want: service.OpenAIImageJobExecutionSucceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observer := &openAIImageJobObserverStub{allowDispatch: true}
			executor := newOpenAIImageJobExecutorForTest(provider, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.dispatch {
					require.True(t, service.MarkOpenAIImageJobDispatched(r.Context()))
				}
				if tt.bill {
					barrier, ok := openAIImageJobBillingBarrierFromContext(r.Context())
					require.True(t, ok)
					barrier.Record(r.Context(), func(context.Context) error { return tt.billingErr })
				}
				if tt.body != "" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(tt.body))
				}
			}), &config.Config{})
			result := executor.Execute(context.Background(), openAIImageExecutorTestJob(), observer)
			require.Equal(t, tt.want, result.Outcome)
			require.Equal(t, tt.wantCode, result.ErrorCode)
		})
	}
}

func TestOpenAIImageJobExecutorVerifiedSuccessWinsCancelledContext(t *testing.T) {
	provider := &openAIImageJobAPIKeyProviderStub{key: openAIImageExecutorTestKey()}
	observer := &openAIImageJobObserverStub{allowDispatch: true}
	ctx, cancel := context.WithCancel(context.Background())
	executor := newOpenAIImageJobExecutorForTest(provider, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, service.MarkOpenAIImageJobDispatched(r.Context()))
		barrier, ok := openAIImageJobBillingBarrierFromContext(r.Context())
		require.True(t, ok)
		barrier.Record(r.Context(), func(context.Context) error { return nil })
		cancel()
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"aA=="}]}`))
	}), &config.Config{})

	result := executor.Execute(ctx, openAIImageExecutorTestJob(), observer)
	require.Equal(t, service.OpenAIImageJobExecutionSucceeded, result.Outcome)
}

func TestOpenAIImageJobExecutorCancelledBeforeDispatchIsInterrupted(t *testing.T) {
	provider := &openAIImageJobAPIKeyProviderStub{key: openAIImageExecutorTestKey()}
	var engineCalls atomic.Int32
	executor := newOpenAIImageJobExecutorForTest(provider, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		engineCalls.Add(1)
	}), &config.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := executor.Execute(ctx, openAIImageExecutorTestJob(), &openAIImageJobObserverStub{allowDispatch: true})

	require.Equal(t, service.OpenAIImageJobExecutionInterrupted, result.Outcome)
	require.Zero(t, provider.getCalls.Load())
	require.Zero(t, engineCalls.Load())
}

func TestOpenAIImageJobDispatchTrackerStopsLaterFailoverAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	observer := &openAIImageJobObserverStub{allowDispatch: true}
	tracker := &openAIImageJobDispatchTracker{delegate: observer, ctx: ctx}

	require.True(t, tracker.MarkDispatched())
	cancel()
	require.False(t, tracker.MarkDispatched(), "a later failover must not dispatch after cancellation")
	require.True(t, tracker.Dispatched(), "the first dispatch remains ambiguous")
	require.True(t, tracker.Denied())
}

func TestOpenAIImageJobBillingBarrierRetriesAndPreservesStableIDs(t *testing.T) {
	parent := context.WithValue(context.Background(), ctxkey.ClientRequestID, "imgjob_stable")
	parent = context.WithValue(parent, ctxkey.RequestID, "imgjob_stable/1")
	parent, cancel := context.WithCancel(parent)
	cancel()

	barrier := newOpenAIImageJobBillingBarrier(3, time.Millisecond)
	var attempts atomic.Int32
	barrier.Record(parent, func(ctx context.Context) error {
		require.NoError(t, ctx.Err())
		require.Equal(t, "imgjob_stable", ctx.Value(ctxkey.ClientRequestID))
		require.Equal(t, "imgjob_stable/1", ctx.Value(ctxkey.RequestID))
		if attempts.Add(1) < 3 {
			return errors.New("temporary billing error")
		}
		return nil
	})

	resolved, err := barrier.Result()
	require.True(t, resolved)
	require.NoError(t, err)
	require.Equal(t, int32(3), attempts.Load())
}

func TestOpenAIImageJobBillingBarrierConflictDoesNotRetryAndPanicResolves(t *testing.T) {
	t.Run("conflict", func(t *testing.T) {
		barrier := newOpenAIImageJobBillingBarrier(5, time.Millisecond)
		var attempts atomic.Int32
		barrier.Record(context.Background(), func(context.Context) error {
			attempts.Add(1)
			return service.ErrUsageBillingRequestConflict
		})
		resolved, err := barrier.Result()
		require.True(t, resolved)
		require.ErrorIs(t, err, service.ErrUsageBillingRequestConflict)
		require.Equal(t, int32(1), attempts.Load())
	})

	t.Run("panic", func(t *testing.T) {
		barrier := newOpenAIImageJobBillingBarrier(3, 0)
		require.NotPanics(t, func() {
			barrier.Record(context.Background(), func(context.Context) error { panic("billing panic") })
		})
		resolved, err := barrier.Result()
		require.True(t, resolved)
		require.Error(t, err)
		require.Contains(t, err.Error(), "panic")
	})
}

func TestOpenAIImageUsageSubmissionUsesPoolNormallyAndBarrierSynchronously(t *testing.T) {
	t.Run("normal request keeps mandatory pool path", func(t *testing.T) {
		pool := newUsageRecordTestPool(t)
		h := &OpenAIGatewayHandler{usageRecordWorkerPool: pool}
		started := make(chan struct{})
		release := make(chan struct{})
		returned := make(chan struct{})
		go func() {
			h.submitOpenAIImageUsageRecordTask(context.Background(), func(context.Context) error {
				close(started)
				<-release
				return nil
			})
			close(returned)
		}()
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("normal image billing task did not start")
		}
		select {
		case <-returned:
			// The normal HTTP path remains asynchronous.
		case <-time.After(time.Second):
			t.Fatal("normal image billing unexpectedly blocked the HTTP path")
		}
		close(release)
	})

	t.Run("job request blocks on barrier", func(t *testing.T) {
		barrier := newOpenAIImageJobBillingBarrier(1, 0)
		ctx := withOpenAIImageJobBillingBarrier(context.Background(), barrier)
		h := &OpenAIGatewayHandler{}
		started := make(chan struct{})
		release := make(chan struct{})
		returned := make(chan struct{})
		go func() {
			h.submitOpenAIImageUsageRecordTask(ctx, func(context.Context) error {
				close(started)
				<-release
				return nil
			})
			close(returned)
		}()
		<-started
		select {
		case <-returned:
			t.Fatal("job billing barrier returned before billing completed")
		default:
		}
		close(release)
		select {
		case <-returned:
		case <-time.After(time.Second):
			t.Fatal("job billing barrier did not resolve")
		}
		resolved, err := barrier.Result()
		require.True(t, resolved)
		require.NoError(t, err)
	})
}

func TestOpenAIImageJobExecutorDoesNotReplayCallerHeadersOrBodyAliases(t *testing.T) {
	provider := &openAIImageJobAPIKeyProviderStub{key: openAIImageExecutorTestKey()}
	job := openAIImageExecutorTestJob()
	originalBody := append([]byte(nil), job.RequestBody...)
	engine := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get("X-Forwarded-For"))
		require.Empty(t, r.Header.Get("X-Original-Header"))
		readBody := make([]byte, len(originalBody))
		_, _ = r.Body.Read(readBody)
		require.True(t, bytes.Equal(originalBody, readBody))
		_, _ = w.Write([]byte(`{"error":{"message":"stop before dispatch"}}`))
	})
	executor := newOpenAIImageJobExecutorForTest(provider, engine, &config.Config{})
	_ = executor.Execute(context.Background(), job, &openAIImageJobObserverStub{allowDispatch: true})
	require.Equal(t, originalBody, job.RequestBody)
}
