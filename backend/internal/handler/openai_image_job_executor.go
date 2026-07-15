package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
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
	// resultBodyLimit is intentionally independent from the inbound request
	// limit: base64 image responses can be much larger than their prompt.
	resultBodyLimit int64
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
		cfg:             cfg,
		resultBodyLimit: resolveOpenAIImageJobExecutionResultBodyLimit(cfg),
	}
}

func newOpenAIImageJobExecutorForTest(
	apiKeys openAIImageJobAPIKeyProvider,
	engine http.Handler,
	cfg *config.Config,
) *OpenAIImageJobExecutor {
	return &OpenAIImageJobExecutor{
		apiKeys:         apiKeys,
		engine:          engine,
		cfg:             cfg,
		resultBodyLimit: resolveOpenAIImageJobExecutionResultBodyLimit(cfg),
	}
}

func resolveOpenAIImageJobExecutionResultBodyLimit(cfg *config.Config) int64 {
	if cfg != nil && cfg.Gateway.UpstreamResponseReadMaxBytes > 0 {
		return cfg.Gateway.UpstreamResponseReadMaxBytes
	}
	return config.DefaultUpstreamResponseReadMaxBytes
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
	// The replay request carries the already authenticated client address in
	// RemoteAddr and deliberately has no forwarding headers. Never let Gin's
	// default proxy trust policy reinterpret that identity.
	if err := engine.SetTrustedProxies(nil); err != nil {
		panic("failed to disable trusted proxies for image job executor")
	}
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
	requestCtx = service.WithAPIKeyAuthCacheBypass(requestCtx)
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

	resultBodyLimit := e.resultBodyLimit
	if resultBodyLimit <= 0 {
		resultBodyLimit = resolveOpenAIImageJobExecutionResultBodyLimit(e.cfg)
	}
	recorder := newOpenAIImageJobResponseRecorder(resultBodyLimit)
	e.engine.ServeHTTP(recorder, request)
	body := recorder.BodyBytes()
	statusCode := recorder.StatusCode()
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	contentType := strings.TrimSpace(recorder.Header().Get("Content-Type"))

	if recorder.Exceeded() {
		if tracker.Dispatched() {
			return openAIImageJobUnknownFailure("response_too_large", "image generation response exceeded the durable result size limit")
		}
		return openAIImageJobKnownFailure("response_too_large", "image generation response exceeded the durable result size limit")
	}

	// A verified image response wins a concurrent cancellation. Publishing it is
	// still gated on the synchronous, stable-ID billing barrier.
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices &&
		tracker.Dispatched() && openAIImageJobJSONContentType(contentType) && openAIImageJobHasImageResult(body) {
		resolved, billingErr := barrier.Result()
		if !resolved || billingErr != nil {
			message := "image generation succeeded but billing could not be durably confirmed"
			return openAIImageJobUnknownFailure("billing_failed_unknown", message)
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

	if tracker.Dispatched() {
		if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError && openAIImageJobJSONContentType(contentType) {
			if code, message, structured := openAIImageJobStructuredResponseError(body); structured {
				return openAIImageJobKnownFailure(code, message)
			}
		}
		return openAIImageJobUnknownFailure("failed_unknown", "image generation may have reached the upstream, but no verified result was received")
	}
	if tracker.Denied() || ctx.Err() != nil {
		return service.OpenAIImageJobExecutionResult{Outcome: service.OpenAIImageJobExecutionInterrupted}
	}
	code, message := openAIImageJobResponseError(body)
	return openAIImageJobKnownFailure(code, message)
}

type openAIImageJobResponseRecorder struct {
	header   http.Header
	body     bytes.Buffer
	limit    int64
	status   int
	exceeded bool
	closed   chan bool
}

func newOpenAIImageJobResponseRecorder(limit int64) *openAIImageJobResponseRecorder {
	if limit <= 0 {
		limit = config.DefaultUpstreamResponseReadMaxBytes
	}
	return &openAIImageJobResponseRecorder{header: make(http.Header), limit: limit, closed: make(chan bool)}
}

func (r *openAIImageJobResponseRecorder) Header() http.Header {
	return r.header
}

func (r *openAIImageJobResponseRecorder) WriteHeader(statusCode int) {
	if r.status != 0 {
		return
	}
	r.status = statusCode
}

func (r *openAIImageJobResponseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	originalLen := len(data)
	remaining := r.limit - int64(r.body.Len())
	if remaining <= 0 {
		if originalLen > 0 {
			r.exceeded = true
		}
		return originalLen, nil
	}
	if int64(len(data)) > remaining {
		data = data[:int(remaining)]
		r.exceeded = true
	}
	_, err := r.body.Write(data)
	if err != nil {
		return 0, err
	}
	// Report the original length so the application does not retry or replace a
	// successfully capped write. The recorder retains at most limit bytes.
	return originalLen, nil
}

func (r *openAIImageJobResponseRecorder) StatusCode() int {
	if r == nil {
		return 0
	}
	return r.status
}

func (r *openAIImageJobResponseRecorder) BodyBytes() []byte {
	if r == nil {
		return nil
	}
	return r.body.Bytes()
}

func (r *openAIImageJobResponseRecorder) Exceeded() bool {
	return r != nil && r.exceeded
}

func (r *openAIImageJobResponseRecorder) Flush() {
	if r != nil && r.status == 0 {
		r.status = http.StatusOK
	}
}

func (r *openAIImageJobResponseRecorder) CloseNotify() <-chan bool {
	if r == nil || r.closed == nil {
		closed := make(chan bool)
		return closed
	}
	return r.closed
}

func (r *openAIImageJobResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.New("image job response recorder does not support hijacking")
}

type openAIImageJobDispatchTracker struct {
	delegate                      service.OpenAIImageJobExecutionObserver
	ctx                           context.Context
	state                         atomic.Uint32
	denied                        atomic.Bool
	knownNonBillableRearmConsumed atomic.Bool
}

const (
	openAIImageJobExecutorDispatchPending uint32 = iota
	openAIImageJobExecutorDispatchClaimed
	openAIImageJobExecutorDispatchStarted
	openAIImageJobExecutorDispatchRearming
	openAIImageJobExecutorDispatchDenied
)

func (t *openAIImageJobDispatchTracker) MarkDispatched() bool {
	if t == nil || t.delegate == nil {
		if t != nil {
			t.denied.Store(true)
		}
		return false
	}
	// This is deliberately a one-shot gate. Account failover reuses the same
	// execution context, so any attempt after the first must be denied even when
	// the worker's delegate remains in its irreversible dispatched state.
	if !t.state.CompareAndSwap(openAIImageJobExecutorDispatchPending, openAIImageJobExecutorDispatchClaimed) {
		t.denied.Store(true)
		return false
	}
	if t.ctx != nil && t.ctx.Err() != nil {
		t.denied.Store(true)
		t.state.Store(openAIImageJobExecutorDispatchDenied)
		return false
	}
	allowed := t.delegate.MarkDispatched()
	if !allowed {
		t.denied.Store(true)
		t.state.Store(openAIImageJobExecutorDispatchDenied)
		return false
	}
	t.state.Store(openAIImageJobExecutorDispatchStarted)
	return true
}

func (t *openAIImageJobDispatchTracker) Dispatched() bool {
	if t == nil {
		return false
	}
	state := t.state.Load()
	return state == openAIImageJobExecutorDispatchStarted || state == openAIImageJobExecutorDispatchRearming
}

func (t *openAIImageJobDispatchTracker) AcknowledgeKnownNonBillableDispatch() bool {
	if t == nil || t.delegate == nil || (t.ctx != nil && t.ctx.Err() != nil) {
		return false
	}
	acknowledger, ok := t.delegate.(interface {
		AcknowledgeKnownNonBillableDispatch() bool
	})
	if !ok || !t.knownNonBillableRearmConsumed.CompareAndSwap(false, true) {
		return false
	}
	if !t.state.CompareAndSwap(openAIImageJobExecutorDispatchStarted, openAIImageJobExecutorDispatchRearming) {
		return false
	}
	if t.ctx != nil && t.ctx.Err() != nil {
		t.state.Store(openAIImageJobExecutorDispatchStarted)
		return false
	}
	if !acknowledger.AcknowledgeKnownNonBillableDispatch() {
		t.state.Store(openAIImageJobExecutorDispatchStarted)
		return false
	}
	t.state.Store(openAIImageJobExecutorDispatchPending)
	return true
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

func openAIImageJobJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		return false
	}
	mediaType = strings.ToLower(mediaType)
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func openAIImageJobStructuredResponseError(body []byte) (string, string, bool) {
	if !json.Valid(body) {
		return "", "", false
	}
	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Error   *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", false
	}
	code := strings.TrimSpace(payload.Code)
	message := strings.TrimSpace(payload.Message)
	if payload.Error != nil {
		if value := strings.TrimSpace(payload.Error.Code); value != "" {
			code = value
		}
		if value := strings.TrimSpace(payload.Error.Message); value != "" {
			message = value
		}
	}
	if code == "" && message == "" {
		return "", "", false
	}
	if code == "" {
		code = "image_generation_failed"
	}
	if message == "" {
		message = "image generation request was rejected by the upstream"
	}
	return code, message, true
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
	budget      time.Duration
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
	return newOpenAIImageJobBillingBarrierWithBudget(maxAttempts, retryDelay, 0)
}

func newOpenAIImageJobBillingBarrierWithBudget(maxAttempts int, retryDelay, budget time.Duration) *openAIImageJobBillingBarrier {
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
	if budget <= 0 {
		budget = time.Duration(maxAttempts)*openAIImageJobBillingAttemptTimeout + time.Duration(maxAttempts-1)*retryDelay
	}
	return &openAIImageJobBillingBarrier{maxAttempts: maxAttempts, retryDelay: retryDelay, budget: budget}
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
		billingCtx, cancelBilling := context.WithTimeout(base, b.budget)
		defer cancelBilling()
		for attempt := 1; attempt <= b.maxAttempts; attempt++ {
			attemptCtx, cancel := context.WithTimeout(billingCtx, openAIImageJobBillingAttemptTimeout)
			resultErr = task(attemptCtx)
			cancel()
			if resultErr == nil || errors.Is(resultErr, service.ErrUsageBillingRequestConflict) || attempt == b.maxAttempts {
				return
			}
			if b.retryDelay > 0 {
				timer := time.NewTimer(b.retryDelay)
				select {
				case <-timer.C:
				case <-billingCtx.Done():
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					resultErr = errors.Join(resultErr, billingCtx.Err())
					return
				}
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
