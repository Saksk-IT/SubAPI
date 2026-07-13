package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	defaultOpenAIImageJobBillingMaxAttempts  = 3
	defaultOpenAIImageJobBillingRetryDelay   = 250 * time.Millisecond
	openAIImageJobBillingAttemptTimeout      = 10 * time.Second
	openAIImageJobMaxBillingAttempts         = 10
	openAIImageJobMaxBillingRetryDelay       = 10 * time.Second
	openAIImageJobExecutionDefaultBodyLimit  = int64(64 << 20)
	openAIImageJobExecutionMaxUserAgentBytes = 512
)

type openAIImageJobAPIKeyProvider interface {
	GetByID(ctx context.Context, id int64) (*service.APIKey, error)
	InvalidateAuthCacheByKey(ctx context.Context, key string)
}

// OpenAIImageJobExecutor reconstructs an isolated HTTP request and runs it
// through the same full authentication and Images handler chain as a live
// request. It never retains the submitting gin.Context or persists a secret.
type OpenAIImageJobExecutor struct {
	apiKeys openAIImageJobAPIKeyProvider
	engine  http.Handler
	cfg     *config.Config
}

var _ service.OpenAIImageJobExecutor = (*OpenAIImageJobExecutor)(nil)

// NewOpenAIImageJobExecutor builds the production adapter used by the durable
// worker. Returning the service interface keeps the worker independent from
// the handler package.
func NewOpenAIImageJobExecutor(
	gatewayHandler *OpenAIGatewayHandler,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	settingService *service.SettingService,
	opsService *service.OpsService,
	cfg *config.Config,
) service.OpenAIImageJobExecutor {
	return &OpenAIImageJobExecutor{
		apiKeys: apiKeyService,
		engine: newOpenAIImageJobExecutionEngine(
			gatewayHandler,
			apiKeyService,
			subscriptionService,
			settingService,
			opsService,
			cfg,
		),
		cfg: cfg,
	}
}

func newOpenAIImageJobExecutorForTest(
	apiKeys openAIImageJobAPIKeyProvider,
	engine http.Handler,
	cfg *config.Config,
) *OpenAIImageJobExecutor {
	return &OpenAIImageJobExecutor{apiKeys: apiKeys, engine: engine, cfg: cfg}
}

func newOpenAIImageJobExecutionEngine(
	gatewayHandler *OpenAIGatewayHandler,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	settingService *service.SettingService,
	opsService *service.OpsService,
	cfg *config.Config,
) http.Handler {
	authCfg := cfg
	if authCfg == nil {
		authCfg = &config.Config{RunMode: config.RunModeSimple}
	}
	bodyLimit := authCfg.Gateway.MaxBodySize
	if bodyLimit <= 0 {
		bodyLimit = openAIImageJobExecutionDefaultBodyLimit
	}

	engine := gin.New()
	// Do not use Gin's default recovery logger here: this private request carries
	// the current API key in memory, and a middleware panic must never dump it.
	engine.Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"type": "api_error", "message": "internal error"},
		})
	}))
	engine.Use(middleware.RequestBodyLimit(bodyLimit))
	engine.Use(middleware.ClientRequestID())
	engine.Use(OpsErrorLoggerMiddleware(opsService))
	engine.Use(InboundEndpointMiddleware())
	engine.Use(gin.HandlerFunc(middleware.NewAPIKeyAuthMiddleware(apiKeyService, subscriptionService, authCfg)))
	engine.Use(middleware.RequireGroupAssignment(settingService, middleware.AnthropicErrorWriter))

	dispatch := func(c *gin.Context) {
		apiKey, ok := middleware.GetAPIKeyFromContext(c)
		if !ok || apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformOpenAI {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Asynchronous Images API is not supported for this platform",
				},
			})
			return
		}
		if gatewayHandler == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "api_error", "message": "internal error"}})
			return
		}
		gatewayHandler.Images(c)
	}
	engine.POST(service.OpenAIImageJobEndpointGenerations, dispatch)
	engine.POST(service.OpenAIImageJobEndpointEdits, dispatch)
	return engine
}

func (e *OpenAIImageJobExecutor) Execute(
	ctx context.Context,
	job *service.OpenAIImageJob,
	observer service.OpenAIImageJobExecutionObserver,
) (result service.OpenAIImageJobExecutionResult) {
	tracker := &openAIImageJobDispatchTracker{delegate: observer}
	defer func() {
		if recovered := recover(); recovered != nil {
			message := "image generation execution failed unexpectedly"
			if tracker.Dispatched() {
				result = openAIImageJobUnknownFailure("failed_unknown", message)
				return
			}
			result = openAIImageJobKnownFailure("image_job_execution_failed", message)
		}
	}()

	if ctx == nil {
		ctx = context.Background()
	}
	tracker.ctx = ctx
	if ctx.Err() != nil {
		return service.OpenAIImageJobExecutionResult{Outcome: service.OpenAIImageJobExecutionInterrupted}
	}
	if job == nil || strings.TrimSpace(job.JobID) == "" {
		return openAIImageJobKnownFailure("invalid_image_job", "image generation job is invalid")
	}
	if observer == nil {
		return openAIImageJobKnownFailure("invalid_image_job_observer", "image generation dispatch observer is required")
	}
	if !service.IsSupportedOpenAIImageJobEndpoint(job.Endpoint) {
		return openAIImageJobKnownFailure("invalid_image_job_endpoint", "image generation job endpoint is invalid")
	}
	if e == nil || e.apiKeys == nil || e.engine == nil {
		return openAIImageJobKnownFailure("image_job_executor_unavailable", "image generation executor is unavailable")
	}

	clientIP := net.ParseIP(strings.TrimSpace(job.ClientIP))
	if clientIP == nil {
		return openAIImageJobKnownFailure("invalid_client_ip", "image generation job client address is invalid")
	}
	if strings.ContainsAny(job.ContentType, "\r\n") || strings.ContainsAny(job.UserAgent, "\r\n") {
		return openAIImageJobKnownFailure("invalid_image_job_headers", "image generation job metadata is invalid")
	}

	apiKey, err := e.apiKeys.GetByID(ctx, job.APIKeyID)
	if ctx.Err() != nil {
		return service.OpenAIImageJobExecutionResult{Outcome: service.OpenAIImageJobExecutionInterrupted}
	}
	if err != nil || apiKey == nil || strings.TrimSpace(apiKey.Key) == "" {
		return openAIImageJobKnownFailure("api_key_unavailable", "the API key that created this image job is unavailable")
	}
	if apiKey.ID != job.APIKeyID || apiKey.UserID != job.UserID {
		return openAIImageJobKnownFailure("api_key_owner_changed", "the API key that created this image job no longer belongs to its owner")
	}

	// GetByID is an uncached database lookup. Invalidate any prior auth snapshot
	// before replaying the current in-memory secret through the full auth path so
	// disabled keys/users/groups and current quota/balance are re-evaluated.
	e.apiKeys.InvalidateAuthCacheByKey(ctx, apiKey.Key)

	attempt := job.AttemptCount
	if attempt <= 0 {
		attempt = 1
	}
	requestID := fmt.Sprintf("%s/%d", job.JobID, attempt)
	requestCtx := context.WithValue(ctx, ctxkey.ClientRequestID, job.JobID)
	requestCtx = context.WithValue(requestCtx, ctxkey.RequestID, requestID)
	requestCtx = service.WithOpenAIImageJobExecutionObserver(requestCtx, tracker)
	barrier := newOpenAIImageJobBillingBarrierFromConfig(e.cfg)
	requestCtx = withOpenAIImageJobBillingBarrier(requestCtx, barrier)

	request := httptest.NewRequestWithContext(requestCtx, http.MethodPost, job.Endpoint, bytes.NewReader(job.RequestBody))
	request.RemoteAddr = net.JoinHostPort(clientIP.String(), "0")
	request.Header.Set("Authorization", "Bearer "+apiKey.Key)
	if contentType := strings.TrimSpace(job.ContentType); contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if userAgent := truncateValidUTF8(job.UserAgent, openAIImageJobExecutionMaxUserAgentBytes); userAgent != "" {
		request.Header.Set("User-Agent", userAgent)
	}

	recorder := httptest.NewRecorder()
	e.engine.ServeHTTP(recorder, request)
	body := append([]byte(nil), recorder.Body.Bytes()...)
	statusCode := recorder.Code
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	// A verified image response wins a concurrent cancellation. Publishing it is
	// still gated on the synchronous, stable-ID billing barrier.
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices && openAIImageJobHasImageResult(body) {
		resolved, billingErr := barrier.Result()
		if !resolved || billingErr != nil {
			message := "image generation succeeded but billing could not be durably confirmed"
			return openAIImageJobUnknownFailure("billing_failed_unknown", message)
		}
		contentType := strings.TrimSpace(recorder.Header().Get("Content-Type"))
		if contentType == "" {
			contentType = "application/json"
		}
		return service.OpenAIImageJobExecutionResult{
			Outcome: service.OpenAIImageJobExecutionSucceeded,
			Response: service.OpenAIImageJobResponse{
				StatusCode:  statusCode,
				ContentType: contentType,
				Body:        body,
			},
		}
	}

	if tracker.Denied() || (ctx.Err() != nil && !tracker.Dispatched()) {
		return service.OpenAIImageJobExecutionResult{Outcome: service.OpenAIImageJobExecutionInterrupted}
	}
	if tracker.Dispatched() {
		return openAIImageJobUnknownFailure("failed_unknown", "image generation may have reached the upstream, but no verified result was received")
	}
	code, message := openAIImageJobResponseError(body)
	return openAIImageJobKnownFailure(code, message)
}

type openAIImageJobDispatchTracker struct {
	delegate service.OpenAIImageJobExecutionObserver
	ctx      context.Context
	denied   atomic.Bool
}

func (t *openAIImageJobDispatchTracker) MarkDispatched() bool {
	if t == nil || t.delegate == nil {
		if t != nil {
			t.denied.Store(true)
		}
		return false
	}
	if t.ctx != nil && t.ctx.Err() != nil {
		t.denied.Store(true)
		return false
	}
	allowed := t.delegate.MarkDispatched()
	if !allowed {
		t.denied.Store(true)
	}
	return allowed
}

func (t *openAIImageJobDispatchTracker) Dispatched() bool {
	return t != nil && t.delegate != nil && t.delegate.Dispatched()
}

func (t *openAIImageJobDispatchTracker) Denied() bool {
	return t != nil && t.denied.Load()
}

func openAIImageJobKnownFailure(code, message string) service.OpenAIImageJobExecutionResult {
	if strings.TrimSpace(code) == "" {
		code = "image_generation_failed"
	}
	if strings.TrimSpace(message) == "" {
		message = "image generation failed before upstream dispatch"
	}
	return service.OpenAIImageJobExecutionResult{
		Outcome:      service.OpenAIImageJobExecutionFailed,
		ErrorCode:    code,
		ErrorMessage: message,
	}
}

func openAIImageJobUnknownFailure(code, message string) service.OpenAIImageJobExecutionResult {
	if strings.TrimSpace(code) == "" {
		code = "failed_unknown"
	}
	return service.OpenAIImageJobExecutionResult{
		Outcome:      service.OpenAIImageJobExecutionFailedUnknown,
		ErrorCode:    code,
		ErrorMessage: message,
	}
}

func openAIImageJobHasImageResult(body []byte) bool {
	if len(bytes.TrimSpace(body)) == 0 || !json.Valid(body) {
		return false
	}
	var payload struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	for _, image := range payload.Data {
		if strings.TrimSpace(image.B64JSON) != "" || strings.TrimSpace(image.URL) != "" {
			return true
		}
	}
	return false
}

func openAIImageJobResponseError(body []byte) (string, string) {
	code := "image_generation_failed"
	message := "image generation failed before upstream dispatch"
	if !json.Valid(body) {
		return code, message
	}
	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return code, message
	}
	if value := strings.TrimSpace(payload.Error.Code); value != "" {
		code = value
	} else if value := strings.TrimSpace(payload.Code); value != "" {
		code = value
	}
	if value := strings.TrimSpace(payload.Error.Message); value != "" {
		message = value
	} else if value := strings.TrimSpace(payload.Message); value != "" {
		message = value
	}
	return code, message
}

type openAIImageJobBillingBarrierContextKey struct{}

type openAIImageJobBillingBarrier struct {
	maxAttempts int
	retryDelay  time.Duration
	once        sync.Once
	mu          sync.RWMutex
	resolved    bool
	err         error
}

func newOpenAIImageJobBillingBarrierFromConfig(cfg *config.Config) *openAIImageJobBillingBarrier {
	attempts := defaultOpenAIImageJobBillingMaxAttempts
	delay := defaultOpenAIImageJobBillingRetryDelay
	if cfg != nil {
		if cfg.OpenAIImageJobs.BillingMaxAttempts > 0 {
			attempts = cfg.OpenAIImageJobs.BillingMaxAttempts
		}
		if cfg.OpenAIImageJobs.BillingRetryDelayMS >= 0 {
			delay = time.Duration(cfg.OpenAIImageJobs.BillingRetryDelayMS) * time.Millisecond
		}
	}
	return newOpenAIImageJobBillingBarrier(attempts, delay)
}

func newOpenAIImageJobBillingBarrier(maxAttempts int, retryDelay time.Duration) *openAIImageJobBillingBarrier {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	if maxAttempts > openAIImageJobMaxBillingAttempts {
		maxAttempts = openAIImageJobMaxBillingAttempts
	}
	if retryDelay < 0 {
		retryDelay = 0
	}
	if retryDelay > openAIImageJobMaxBillingRetryDelay {
		retryDelay = openAIImageJobMaxBillingRetryDelay
	}
	return &openAIImageJobBillingBarrier{maxAttempts: maxAttempts, retryDelay: retryDelay}
}

func withOpenAIImageJobBillingBarrier(ctx context.Context, barrier *openAIImageJobBillingBarrier) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if barrier == nil {
		return ctx
	}
	return context.WithValue(ctx, openAIImageJobBillingBarrierContextKey{}, barrier)
}

func openAIImageJobBillingBarrierFromContext(ctx context.Context) (*openAIImageJobBillingBarrier, bool) {
	if ctx == nil {
		return nil, false
	}
	barrier, ok := ctx.Value(openAIImageJobBillingBarrierContextKey{}).(*openAIImageJobBillingBarrier)
	return barrier, ok && barrier != nil
}

// Record executes billing inline exactly once. It deliberately detaches from
// request cancellation while preserving the stable request IDs, then applies a
// bounded retry policy. A panic is converted into a resolved barrier error so
// the worker can never publish an unbilled completion accidentally.
func (b *openAIImageJobBillingBarrier) Record(parent context.Context, task func(context.Context) error) {
	if b == nil {
		return
	}
	b.once.Do(func() {
		var resultErr error
		defer func() {
			if recovered := recover(); recovered != nil {
				resultErr = errors.New("image job billing panic recovered")
			}
			b.mu.Lock()
			b.err = resultErr
			b.resolved = true
			b.mu.Unlock()
		}()
		if task == nil {
			resultErr = errors.New("image job billing task is missing")
			return
		}
		base := context.Background()
		if parent != nil {
			base = context.WithoutCancel(parent)
		}
		for attempt := 1; attempt <= b.maxAttempts; attempt++ {
			attemptCtx, cancel := context.WithTimeout(base, openAIImageJobBillingAttemptTimeout)
			resultErr = task(attemptCtx)
			cancel()
			if resultErr == nil || errors.Is(resultErr, service.ErrUsageBillingRequestConflict) || attempt == b.maxAttempts {
				return
			}
			if b.retryDelay > 0 {
				timer := time.NewTimer(b.retryDelay)
				<-timer.C
			}
		}
	})
}

func (b *openAIImageJobBillingBarrier) Result() (bool, error) {
	if b == nil {
		return false, nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.resolved, b.err
}

// submitOpenAIImageUsageRecordTask is the only behavioral fork in Images
// billing. Normal requests keep the existing mandatory worker-pool path;
// durable jobs run synchronously through their billing barrier.
func (h *OpenAIGatewayHandler) submitOpenAIImageUsageRecordTask(parent context.Context, task func(context.Context) error) {
	if task == nil {
		return
	}
	if barrier, ok := openAIImageJobBillingBarrierFromContext(parent); ok {
		barrier.Record(parent, task)
		return
	}
	h.submitMandatoryUsageRecordTask(parent, func(ctx context.Context) {
		_ = task(ctx)
	})
}
