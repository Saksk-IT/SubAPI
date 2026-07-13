package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
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
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig())
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
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig())
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
			h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig())
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
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig())
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
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig())
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
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig())
	c, recorder := newOpenAIImageJobHandlerContext(http.MethodPost, "/v1/images/jobs/imgjob_cancel/cancel", nil, "")
	c.Params = gin.Params{{Key: "id", Value: "imgjob_cancel"}}

	h.Cancel(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "1", recorder.Header().Get("Retry-After"))
	require.Equal(t, "private, no-store", recorder.Header().Get("Cache-Control"))
	require.JSONEq(t, `{
		"id":"imgjob_cancel","object":"image_generation.job","status":"running",
		"cancel_requested":true,"created_at":"2026-07-14T01:02:03Z"
	}`, recorder.Body.String())
}

func TestOpenAIImageJobHandlerHidesCrossUserJobsAsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &openAIImageJobHandlerRepository{
		getForUser: func(context.Context, int64, string) (*service.OpenAIImageJob, error) {
			return nil, service.ErrOpenAIImageJobNotFound
		},
	}
	h := NewOpenAIImageJobHandler(repo, &service.OpenAIGatewayService{}, openAIImageJobHandlerTestConfig())
	c, recorder := newOpenAIImageJobHandlerContext(http.MethodGet, "/v1/images/jobs/other-user-job", nil, "")
	c.Params = gin.Params{{Key: "id", Value: "other-user-job"}}

	h.Get(c)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.JSONEq(t, `{"error":{"type":"invalid_request_error","code":"OPENAI_IMAGE_JOB_NOT_FOUND","message":"image generation job not found"}}`, recorder.Body.String())
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
