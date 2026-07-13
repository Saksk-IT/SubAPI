package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIImageJobHandlerRepository struct {
	createOrGet func(context.Context, service.CreateOpenAIImageJobParams) (*service.OpenAIImageJob, bool, error)
	getForUser  func(context.Context, int64, string) (*service.OpenAIImageJob, error)
	cancel      func(context.Context, int64, string) (*service.OpenAIImageJob, error)
}

type recordingOpenAIImageJobCancelNotifier struct {
	requested []string
}

func (n *recordingOpenAIImageJobCancelNotifier) RequestCancel(jobID string) bool {
	n.requested = append(n.requested, jobID)
	return true
}

func (r *openAIImageJobHandlerRepository) CreateOrGet(ctx context.Context, params service.CreateOpenAIImageJobParams) (*service.OpenAIImageJob, bool, error) {
	return r.createOrGet(ctx, params)
}

func (r *openAIImageJobHandlerRepository) GetForUser(ctx context.Context, userID int64, jobID string) (*service.OpenAIImageJob, error) {
	return r.getForUser(ctx, userID, jobID)
}

func (r *openAIImageJobHandlerRepository) CancelForUser(ctx context.Context, userID int64, jobID string) (*service.OpenAIImageJob, error) {
	return r.cancel(ctx, userID, jobID)
}

func (r *openAIImageJobHandlerRepository) ClaimNext(context.Context, string, time.Time) (*service.OpenAIImageJob, error) {
	panic("unexpected ClaimNext")
}

func (r *openAIImageJobHandlerRepository) Heartbeat(context.Context, string, string, time.Time) (bool, error) {
	panic("unexpected Heartbeat")
}

func (r *openAIImageJobHandlerRepository) Complete(context.Context, string, string, service.OpenAIImageJobResponse) error {
	panic("unexpected Complete")
}

func (r *openAIImageJobHandlerRepository) Fail(context.Context, string, string, string, string, bool) error {
	panic("unexpected Fail")
}

func (r *openAIImageJobHandlerRepository) MarkCancelled(context.Context, string, string) error {
	panic("unexpected MarkCancelled")
}

func (r *openAIImageJobHandlerRepository) FailExpiredLeases(context.Context, time.Time) (int64, error) {
	panic("unexpected FailExpiredLeases")
}

func (r *openAIImageJobHandlerRepository) PurgeExpiredPayloads(context.Context, service.OpenAIImageJobCleanupParams) (service.OpenAIImageJobCleanupResult, error) {
	panic("unexpected PurgeExpiredPayloads")
}

func TestOpenAIImageJobHandlerSubmitGenerationReturnsImmediatelyAndReplays(t *testing.T) {
	gin.SetMode(gin.TestMode)
	createdAt := time.Date(2026, 7, 14, 1, 2, 3, 0, time.UTC)
	var captured service.CreateOpenAIImageJobParams
	repo := &openAIImageJobHandlerRepository{
		createOrGet: func(_ context.Context, params service.CreateOpenAIImageJobParams) (*service.OpenAIImageJob, bool, error) {
			captured = params
			return &service.OpenAIImageJob{
				JobID:     "imgjob_123",
				UserID:    params.UserID,
				APIKeyID:  params.APIKeyID,
				Status:    service.OpenAIImageJobStatusQueued,
				CreatedAt: createdAt,
			}, false, nil
		},
	}
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig(), nil)
	body := []byte(`{"model":"gpt-image-2","prompt":"draw a cat","stream":false}`)
	c, recorder := newOpenAIImageJobHandlerContext(http.MethodPost, "/v1/images/generations/jobs", body, "application/json")
	c.Request.Header.Set("Idempotency-Key", "  local-task-123  ")

	h.Submit(c)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Equal(t, "1", recorder.Header().Get("Retry-After"))
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, int64(41), captured.UserID)
	require.Equal(t, int64(73), captured.APIKeyID)
	require.Equal(t, service.OpenAIImageJobEndpointGenerations, captured.Endpoint)
	require.Equal(t, "gpt-image-2", captured.Model)
	require.Equal(t, "local-task-123", captured.IdempotencyKey)
	require.Equal(t, body, captured.RequestBody)
	require.Equal(t, "application/json", captured.ContentType)
	require.Equal(t, "203.0.113.5", captured.ClientIP)
	require.Equal(t, 10, captured.MaxActivePerUser)
	require.Equal(t, 1000, captured.MaxActiveGlobal)
	require.NotContains(t, string(captured.RequestBody), "secret-key")
	require.JSONEq(t, `{"id":"imgjob_123","object":"image_generation.job","status":"queued","created_at":"2026-07-14T01:02:03Z"}`, recorder.Body.String())
}

func TestOpenAIImageJobHandlerSubmitMultipartEdit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "edit this"))
	part, err := writer.CreateFormFile("image", "input.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-png"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	var captured service.CreateOpenAIImageJobParams
	repo := &openAIImageJobHandlerRepository{
		createOrGet: func(_ context.Context, params service.CreateOpenAIImageJobParams) (*service.OpenAIImageJob, bool, error) {
			captured = params
			return &service.OpenAIImageJob{JobID: "imgjob_edit", Status: service.OpenAIImageJobStatusQueued, CreatedAt: time.Now()}, true, nil
		},
	}
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig(), nil)
	c, recorder := newOpenAIImageJobHandlerContext(http.MethodPost, "/v1/images/edits/jobs", body.Bytes(), writer.FormDataContentType())
	c.Request.Header.Set("Idempotency-Key", "edit-task")

	h.Submit(c)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Equal(t, service.OpenAIImageJobEndpointEdits, captured.Endpoint)
	require.Equal(t, writer.FormDataContentType(), captured.ContentType)
	require.Equal(t, body.Bytes(), captured.RequestBody)
}

func TestOpenAIImageJobHandlerSubmitValidatesRequestBeforePersistence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name           string
		body           string
		idempotencyKey string
		wantCode       string
	}{
		{name: "missing idempotency key", body: `{"model":"gpt-image-2","prompt":"cat"}`, wantCode: "IDEMPOTENCY_KEY_REQUIRED"},
		{name: "invalid idempotency key", body: `{"model":"gpt-image-2","prompt":"cat"}`, idempotencyKey: "bad\nkey", wantCode: "IDEMPOTENCY_KEY_INVALID"},
		{name: "streaming is not durable", body: `{"model":"gpt-image-2","prompt":"cat","stream":true}`, idempotencyKey: "task-1", wantCode: "IMAGE_JOB_STREAM_UNSUPPORTED"},
		{name: "invalid image request", body: `{"model":"gpt-image-2","n":0}`, idempotencyKey: "task-1", wantCode: "INVALID_IMAGE_REQUEST"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			repo := &openAIImageJobHandlerRepository{
				createOrGet: func(context.Context, service.CreateOpenAIImageJobParams) (*service.OpenAIImageJob, bool, error) {
					called = true
					return nil, false, nil
				},
			}
			h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig(), nil)
			c, recorder := newOpenAIImageJobHandlerContext(http.MethodPost, "/v1/images/generations/jobs", []byte(tt.body), "application/json")
			if tt.idempotencyKey != "" {
				c.Request.Header.Set("Idempotency-Key", tt.idempotencyKey)
			}

			h.Submit(c)

			require.False(t, called)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			var payload struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
			require.Equal(t, tt.wantCode, payload.Error.Code)
		})
	}
}

func TestOpenAIImageJobHandlerStatusIsUserScopedAndNoStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &openAIImageJobHandlerRepository{
		getForUser: func(_ context.Context, userID int64, jobID string) (*service.OpenAIImageJob, error) {
			require.Equal(t, int64(41), userID)
			require.Equal(t, "imgjob_status", jobID)
			code, message := "provider_failed", "upstream rejected request"
			return &service.OpenAIImageJob{
				JobID:        jobID,
				UserID:       userID,
				Status:       service.OpenAIImageJobStatusFailed,
				ErrorCode:    &code,
				ErrorMessage: &message,
				CreatedAt:    time.Date(2026, 7, 14, 1, 2, 3, 0, time.UTC),
			}, nil
		},
	}
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig(), nil)
	c, recorder := newOpenAIImageJobHandlerContext(http.MethodGet, "/v1/images/jobs/imgjob_status", nil, "")
	c.Params = gin.Params{{Key: "id", Value: "imgjob_status"}}

	h.Get(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Empty(t, recorder.Header().Get("Retry-After"))
	require.JSONEq(t, `{
		"id":"imgjob_status",
		"object":"image_generation.job",
		"status":"failed",
		"created_at":"2026-07-14T01:02:03Z",
		"error":{"code":"provider_failed","message":"upstream rejected request"}
	}`, recorder.Body.String())
}

func TestOpenAIImageJobHandlerStatusCompletedEmbedsOriginalResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	expiresAt := time.Now().Add(time.Hour)
	repo := &openAIImageJobHandlerRepository{
		getForUser: func(context.Context, int64, string) (*service.OpenAIImageJob, error) {
			return &service.OpenAIImageJob{
				JobID:           "imgjob_done",
				Status:          service.OpenAIImageJobStatusCompleted,
				ResponseBody:    []byte(`{"created":123,"data":[{"b64_json":"abc"}]}`),
				ResultExpiresAt: &expiresAt,
				CreatedAt:       time.Date(2026, 7, 14, 1, 2, 3, 0, time.UTC),
			}, nil
		},
	}
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig(), nil)
	c, recorder := newOpenAIImageJobHandlerContext(http.MethodGet, "/v1/images/jobs/imgjob_done", nil, "")
	c.Params = gin.Params{{Key: "id", Value: "imgjob_done"}}

	h.Get(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{
		"id":"imgjob_done","object":"image_generation.job","status":"completed",
		"created_at":"2026-07-14T01:02:03Z",
		"result":{"created":123,"data":[{"b64_json":"abc"}]}
	}`, recorder.Body.String())
}

func TestOpenAIImageJobHandlerCancelUsesAuthenticatedUserAndIsIdempotent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &openAIImageJobHandlerRepository{
		cancel: func(_ context.Context, userID int64, jobID string) (*service.OpenAIImageJob, error) {
			require.Equal(t, int64(41), userID)
			require.Equal(t, "imgjob_cancel", jobID)
			return &service.OpenAIImageJob{
				JobID:           jobID,
				Status:          service.OpenAIImageJobStatusRunning,
				CancelRequested: true,
				CreatedAt:       time.Date(2026, 7, 14, 1, 2, 3, 0, time.UTC),
			}, nil
		},
	}
	notifier := &recordingOpenAIImageJobCancelNotifier{}
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig(), notifier)
	c, recorder := newOpenAIImageJobHandlerContext(http.MethodPost, "/v1/images/jobs/imgjob_cancel/cancel", nil, "")
	c.Params = gin.Params{{Key: "id", Value: "imgjob_cancel"}}

	h.Cancel(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "1", recorder.Header().Get("Retry-After"))
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.Equal(t, []string{"imgjob_cancel"}, notifier.requested)
	require.JSONEq(t, `{
		"id":"imgjob_cancel","object":"image_generation.job","status":"running",
		"cancel_requested":true,"created_at":"2026-07-14T01:02:03Z"
	}`, recorder.Body.String())
}

func TestOpenAIImageJobHandlerCancelDoesNotNotifyBeforePersistenceOrForTerminalJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Run("persistence error", func(t *testing.T) {
		notifier := &recordingOpenAIImageJobCancelNotifier{}
		repo := &openAIImageJobHandlerRepository{
			cancel: func(context.Context, int64, string) (*service.OpenAIImageJob, error) {
				return nil, service.ErrOpenAIImageJobNotFound
			},
		}
		h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig(), notifier)
		c, recorder := newOpenAIImageJobHandlerContext(http.MethodPost, "/v1/images/jobs/missing/cancel", nil, "")
		c.Params = gin.Params{{Key: "id", Value: "missing"}}

		h.Cancel(c)

		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.Empty(t, notifier.requested)
	})

	t.Run("already completed", func(t *testing.T) {
		notifier := &recordingOpenAIImageJobCancelNotifier{}
		repo := &openAIImageJobHandlerRepository{
			cancel: func(context.Context, int64, string) (*service.OpenAIImageJob, error) {
				return &service.OpenAIImageJob{JobID: "imgjob_done", Status: service.OpenAIImageJobStatusCompleted}, nil
			},
		}
		h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig(), notifier)
		c, recorder := newOpenAIImageJobHandlerContext(http.MethodPost, "/v1/images/jobs/imgjob_done/cancel", nil, "")
		c.Params = gin.Params{{Key: "id", Value: "imgjob_done"}}

		h.Cancel(c)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.Empty(t, notifier.requested)
	})
}

func TestOpenAIImageJobHandlerHidesCrossUserJobsAsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &openAIImageJobHandlerRepository{
		getForUser: func(context.Context, int64, string) (*service.OpenAIImageJob, error) {
			return nil, service.ErrOpenAIImageJobNotFound
		},
	}
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig(), nil)
	c, recorder := newOpenAIImageJobHandlerContext(http.MethodGet, "/v1/images/jobs/other-user-job", nil, "")
	c.Params = gin.Params{{Key: "id", Value: "other-user-job"}}

	h.Get(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.JSONEq(t, `{"error":{"type":"invalid_request_error","code":"OPENAI_IMAGE_JOB_NOT_FOUND","message":"image generation job not found"}}`, recorder.Body.String())
}

func TestOpenAIImageJobAsyncHTTPFlowReturnsBeforeSlowExecutionAndPollsToCompletion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newOpenAIImageJobHTTPFlowRepository()
	executorStarted := make(chan struct{})
	releaseExecutor := make(chan struct{})
	releaseOnce := sync.Once{}
	defer releaseOnce.Do(func() { close(releaseExecutor) })

	executor := openAIImageJobHTTPFlowExecutor(func(ctx context.Context, _ *service.OpenAIImageJob, observer service.OpenAIImageJobExecutionObserver) service.OpenAIImageJobExecutionResult {
		if !observer.MarkDispatched() {
			return service.OpenAIImageJobExecutionResult{Outcome: service.OpenAIImageJobExecutionInterrupted}
		}
		close(executorStarted)
		select {
		case <-releaseExecutor:
			return service.OpenAIImageJobExecutionResult{
				Outcome: service.OpenAIImageJobExecutionSucceeded,
				Response: service.OpenAIImageJobResponse{
					StatusCode:  http.StatusOK,
					ContentType: "application/json",
					Body:        []byte(`{"created":123,"data":[{"b64_json":"abc"}]}`),
				},
			}
		case <-ctx.Done():
			return service.OpenAIImageJobExecutionResult{Outcome: service.OpenAIImageJobExecutionInterrupted}
		}
	})
	worker := service.NewOpenAIImageJobWorkerRuntime(repo, executor, service.OpenAIImageJobWorkerOptions{
		HeartbeatInterval: time.Hour,
		ExecutionTimeout:  time.Minute,
	})
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig(), worker)

	workerCtx, cancelWorker := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelWorker()
	workerDone := make(chan error, 1)
	go func() {
		processed, err := worker.RunOnce(workerCtx, "http-flow-worker")
		if err == nil && !processed {
			err = service.ErrOpenAIImageJobNotFound
		}
		workerDone <- err
	}()

	submitDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		body := []byte(`{"model":"gpt-image-2","prompt":"draw a durable cat","stream":false}`)
		c, recorder := newOpenAIImageJobHandlerContext(http.MethodPost, "/v1/images/generations/jobs", body, "application/json")
		c.Request.Header.Set("Idempotency-Key", "http-flow-task")
		h.Submit(c)
		submitDone <- recorder
	}()

	// The worker is already consuming the newly persisted row and the fake
	// upstream remains blocked. The submit response must not wait for that
	// long-running execution to finish.
	var submitRecorder *httptest.ResponseRecorder
	for submitRecorder == nil {
		select {
		case submitRecorder = <-submitDone:
		case <-time.After(time.Second):
			t.Fatal("submit HTTP request waited for the blocked image executor")
		}
	}
	require.Equal(t, http.StatusAccepted, submitRecorder.Code)
	require.JSONEq(t, `{
		"id":"imgjob_0123456789abcdef0123456789abcdef",
		"object":"image_generation.job",
		"status":"queued",
		"created_at":"2026-07-14T01:02:03Z"
	}`, submitRecorder.Body.String())

	select {
	case <-executorStarted:
	case <-time.After(time.Second):
		t.Fatal("worker did not claim the submitted job")
	}

	runningContext, runningRecorder := newOpenAIImageJobHandlerContext(http.MethodGet, "/v1/images/jobs/imgjob_0123456789abcdef0123456789abcdef", nil, "")
	runningContext.Params = gin.Params{{Key: "id", Value: "imgjob_0123456789abcdef0123456789abcdef"}}
	h.Get(runningContext)
	require.Equal(t, http.StatusOK, runningRecorder.Code)
	require.Equal(t, "1", runningRecorder.Header().Get("Retry-After"))
	require.Contains(t, runningRecorder.Body.String(), `"status":"running"`)

	releaseOnce.Do(func() { close(releaseExecutor) })
	select {
	case err := <-workerDone:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("worker did not persist the released executor result")
	}

	completedContext, completedRecorder := newOpenAIImageJobHandlerContext(http.MethodGet, "/v1/images/jobs/imgjob_0123456789abcdef0123456789abcdef", nil, "")
	completedContext.Params = gin.Params{{Key: "id", Value: "imgjob_0123456789abcdef0123456789abcdef"}}
	h.Get(completedContext)
	require.Equal(t, http.StatusOK, completedRecorder.Code)
	require.Empty(t, completedRecorder.Header().Get("Retry-After"))
	require.JSONEq(t, `{
		"id":"imgjob_0123456789abcdef0123456789abcdef",
		"object":"image_generation.job",
		"status":"completed",
		"created_at":"2026-07-14T01:02:03Z",
		"started_at":"2026-07-14T01:02:04Z",
		"finished_at":"2026-07-14T01:02:05Z",
		"result":{"created":123,"data":[{"b64_json":"abc"}]}
	}`, completedRecorder.Body.String())
}

type openAIImageJobHTTPFlowExecutor func(context.Context, *service.OpenAIImageJob, service.OpenAIImageJobExecutionObserver) service.OpenAIImageJobExecutionResult

func (f openAIImageJobHTTPFlowExecutor) Execute(ctx context.Context, job *service.OpenAIImageJob, observer service.OpenAIImageJobExecutionObserver) service.OpenAIImageJobExecutionResult {
	return f(ctx, job, observer)
}

// openAIImageJobHTTPFlowRepository is a one-row, mutex-protected durable-store
// harness shared by the real HTTP handler and the real worker runtime. It keeps
// returned snapshots separate so the submit JSON cannot race the worker's
// queued -> running transition.
type openAIImageJobHTTPFlowRepository struct {
	mu          sync.Mutex
	created     chan struct{}
	createdOnce sync.Once
	job         *service.OpenAIImageJob
	claimed     bool
}

func newOpenAIImageJobHTTPFlowRepository() *openAIImageJobHTTPFlowRepository {
	return &openAIImageJobHTTPFlowRepository{created: make(chan struct{})}
}

func (r *openAIImageJobHTTPFlowRepository) CreateOrGet(_ context.Context, params service.CreateOpenAIImageJobParams) (*service.OpenAIImageJob, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.job != nil {
		return cloneOpenAIImageJobHTTPFlow(r.job), false, nil
	}
	r.job = &service.OpenAIImageJob{
		JobID:        "imgjob_0123456789abcdef0123456789abcdef",
		UserID:       params.UserID,
		APIKeyID:     params.APIKeyID,
		Endpoint:     params.Endpoint,
		Model:        params.Model,
		ContentType:  params.ContentType,
		RequestBody:  append([]byte(nil), params.RequestBody...),
		RequestHash:  service.HashOpenAIImageJobRequest(params.Endpoint, params.ContentType, params.RequestBody),
		Status:       service.OpenAIImageJobStatusQueued,
		CreatedAt:    time.Date(2026, 7, 14, 1, 2, 3, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 7, 14, 1, 2, 3, 0, time.UTC),
		AttemptCount: 0,
	}
	created := cloneOpenAIImageJobHTTPFlow(r.job)
	r.createdOnce.Do(func() { close(r.created) })
	return created, true, nil
}

func (r *openAIImageJobHTTPFlowRepository) GetForUser(_ context.Context, userID int64, jobID string) (*service.OpenAIImageJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.job == nil || r.job.UserID != userID || r.job.JobID != jobID {
		return nil, service.ErrOpenAIImageJobNotFound
	}
	return cloneOpenAIImageJobHTTPFlow(r.job), nil
}

func (r *openAIImageJobHTTPFlowRepository) ClaimNext(ctx context.Context, workerID string, leaseUntil time.Time) (*service.OpenAIImageJob, error) {
	select {
	case <-r.created:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.job == nil || r.claimed || r.job.Status != service.OpenAIImageJobStatusQueued {
		return nil, nil
	}
	r.claimed = true
	r.job.Status = service.OpenAIImageJobStatusRunning
	r.job.AttemptCount++
	workerIDCopy := workerID
	r.job.WorkerID = &workerIDCopy
	r.job.LeaseExpiresAt = cloneOpenAIImageJobHTTPFlowTime(&leaseUntil)
	startedAt := time.Date(2026, 7, 14, 1, 2, 4, 0, time.UTC)
	r.job.StartedAt = &startedAt
	r.job.UpdatedAt = startedAt
	return cloneOpenAIImageJobHTTPFlow(r.job), nil
}

func (r *openAIImageJobHTTPFlowRepository) Heartbeat(_ context.Context, jobID, workerID string, leaseUntil time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.ownsRunningJob(jobID, workerID) {
		return false, service.ErrOpenAIImageJobLeaseLost
	}
	r.job.LeaseExpiresAt = cloneOpenAIImageJobHTTPFlowTime(&leaseUntil)
	return r.job.CancelRequested, nil
}

func (r *openAIImageJobHTTPFlowRepository) Complete(_ context.Context, jobID, workerID string, response service.OpenAIImageJobResponse) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.ownsRunningJob(jobID, workerID) {
		return service.ErrOpenAIImageJobLeaseLost
	}
	return r.job.MarkCompleted(response, time.Date(2026, 7, 14, 1, 2, 5, 0, time.UTC))
}

func (r *openAIImageJobHTTPFlowRepository) Fail(_ context.Context, jobID, workerID, code, message string, unknown bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.ownsRunningJob(jobID, workerID) {
		return service.ErrOpenAIImageJobLeaseLost
	}
	r.job.Status = service.OpenAIImageJobStatusFailed
	r.job.ErrorCode = &code
	r.job.ErrorMessage = &message
	r.job.FailureUnknown = unknown
	r.job.WorkerID = nil
	r.job.LeaseExpiresAt = nil
	return nil
}

func (r *openAIImageJobHTTPFlowRepository) CancelForUser(_ context.Context, userID int64, jobID string) (*service.OpenAIImageJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.job == nil || r.job.UserID != userID || r.job.JobID != jobID {
		return nil, service.ErrOpenAIImageJobNotFound
	}
	if err := r.job.RequestCancellation(time.Now()); err != nil {
		return nil, err
	}
	return cloneOpenAIImageJobHTTPFlow(r.job), nil
}

func (r *openAIImageJobHTTPFlowRepository) MarkCancelled(_ context.Context, jobID, workerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.ownsRunningJob(jobID, workerID) {
		return service.ErrOpenAIImageJobLeaseLost
	}
	r.job.Status = service.OpenAIImageJobStatusCancelled
	r.job.WorkerID = nil
	r.job.LeaseExpiresAt = nil
	return nil
}

func (r *openAIImageJobHTTPFlowRepository) FailExpiredLeases(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (r *openAIImageJobHTTPFlowRepository) PurgeExpiredPayloads(context.Context, service.OpenAIImageJobCleanupParams) (service.OpenAIImageJobCleanupResult, error) {
	return service.OpenAIImageJobCleanupResult{}, nil
}

func (r *openAIImageJobHTTPFlowRepository) ownsRunningJob(jobID, workerID string) bool {
	return r.job != nil && r.job.JobID == jobID && r.job.Status == service.OpenAIImageJobStatusRunning && r.job.WorkerID != nil && *r.job.WorkerID == workerID
}

func cloneOpenAIImageJobHTTPFlow(job *service.OpenAIImageJob) *service.OpenAIImageJob {
	if job == nil {
		return nil
	}
	clone := *job
	clone.RequestBody = append([]byte(nil), job.RequestBody...)
	clone.ResponseBody = append([]byte(nil), job.ResponseBody...)
	clone.ErrorCode = cloneOpenAIImageJobHTTPFlowString(job.ErrorCode)
	clone.ErrorMessage = cloneOpenAIImageJobHTTPFlowString(job.ErrorMessage)
	clone.WorkerID = cloneOpenAIImageJobHTTPFlowString(job.WorkerID)
	clone.LeaseExpiresAt = cloneOpenAIImageJobHTTPFlowTime(job.LeaseExpiresAt)
	clone.ResultExpiresAt = cloneOpenAIImageJobHTTPFlowTime(job.ResultExpiresAt)
	clone.StartedAt = cloneOpenAIImageJobHTTPFlowTime(job.StartedAt)
	clone.FinishedAt = cloneOpenAIImageJobHTTPFlowTime(job.FinishedAt)
	return &clone
}

func cloneOpenAIImageJobHTTPFlowString(value *string) *string {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneOpenAIImageJobHTTPFlowTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func newOpenAIImageJobHandlerContext(method, path string, body []byte, contentType string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.RemoteAddr = "203.0.113.5:4567"
	request.Header.Set("Authorization", "Bearer secret-key")
	request.Header.Set("User-Agent", "image-playground-test")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	c.Request = request
	user := &service.User{ID: 41, Status: service.StatusActive}
	group := &service.Group{ID: 9, Status: service.StatusActive, AllowImageGeneration: true}
	apiKey := &service.APIKey{ID: 73, UserID: user.ID, User: user, GroupID: &group.ID, Group: group, Status: service.StatusAPIKeyActive}
	c.Set(string(middleware.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: user.ID, Concurrency: 2})
	return c, recorder
}

func openAIImageJobHandlerTestConfig() *config.Config {
	return &config.Config{OpenAIImageJobs: config.OpenAIImageJobsConfig{
		Enabled:          true,
		MaxActivePerUser: 10,
		MaxActiveGlobal:  1000,
	}}
}
