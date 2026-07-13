package handler

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	openAIImageJobMaxClientIPBytes  = 64
	openAIImageJobMaxUserAgentBytes = 512
)

// OpenAIImageJobHandler exposes the durable Images API job contract. Submit
// only validates and persists work; account selection, moderation, upstream
// dispatch, and billing are deliberately owned by the background executor.
type OpenAIImageJobHandler struct {
	repository     service.OpenAIImageJobRepository
	gatewayService *service.OpenAIGatewayService
}

func NewOpenAIImageJobHandler(
	repository service.OpenAIImageJobRepository,
	gatewayService *service.OpenAIGatewayService,
) *OpenAIImageJobHandler {
	return &OpenAIImageJobHandler{
		repository:     repository,
		gatewayService: gatewayService,
	}
}

// Submit handles POST /v1/images/generations/jobs and
// POST /v1/images/edits/jobs.
func (h *OpenAIImageJobHandler) Submit(c *gin.Context) {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.ID <= 0 {
		openAIImageJobWriteError(c, infraerrors.New(http.StatusUnauthorized, "API_KEY_REQUIRED", "API key is required"))
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 || subject.UserID != apiKey.UserID {
		openAIImageJobWriteError(c, infraerrors.New(http.StatusUnauthorized, "INVALID_API_KEY", "Invalid API key"))
		return
	}
	if !service.GroupAllowsImageGeneration(apiKey.Group) {
		openAIImageJobWriteTypedError(c, http.StatusForbidden, "permission_error", "IMAGE_GENERATION_NOT_ENABLED", service.ImageGenerationPermissionMessage())
		return
	}

	idempotencyKey, err := service.NormalizeIdempotencyKey(c.GetHeader("Idempotency-Key"))
	if err != nil {
		openAIImageJobWriteError(c, err)
		return
	}
	if idempotencyKey == "" {
		openAIImageJobWriteError(c, service.ErrIdempotencyKeyRequired)
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			openAIImageJobWriteTypedError(c, http.StatusRequestEntityTooLarge, "invalid_request_error", "REQUEST_BODY_TOO_LARGE", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		openAIImageJobWriteTypedError(c, http.StatusBadRequest, "invalid_request_error", "INVALID_IMAGE_REQUEST", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		openAIImageJobWriteTypedError(c, http.StatusBadRequest, "invalid_request_error", "INVALID_IMAGE_REQUEST", "Request body is empty")
		return
	}
	if h == nil || h.gatewayService == nil || h.repository == nil {
		openAIImageJobWriteError(c, nil)
		return
	}

	parsed, err := h.gatewayService.ParseOpenAIImagesRequest(c, body)
	if err != nil {
		openAIImageJobWriteTypedError(c, http.StatusBadRequest, "invalid_request_error", "INVALID_IMAGE_REQUEST", err.Error())
		return
	}
	if parsed.Stream {
		openAIImageJobWriteTypedError(c, http.StatusBadRequest, "invalid_request_error", "IMAGE_JOB_STREAM_UNSUPPORTED", "asynchronous image jobs do not support streaming")
		return
	}

	job, _, err := h.repository.CreateOrGet(c.Request.Context(), service.CreateOpenAIImageJobParams{
		UserID:         subject.UserID,
		APIKeyID:       apiKey.ID,
		Endpoint:       parsed.Endpoint,
		Model:          parsed.Model,
		ContentType:    strings.TrimSpace(c.GetHeader("Content-Type")),
		RequestBody:    body,
		IdempotencyKey: idempotencyKey,
		ClientIP:       truncateValidUTF8(ip.GetClientIP(c), openAIImageJobMaxClientIPBytes),
		UserAgent:      truncateValidUTF8(c.GetHeader("User-Agent"), openAIImageJobMaxUserAgentBytes),
	})
	if err != nil {
		openAIImageJobWriteError(c, err)
		return
	}

	setOpenAIImageJobResponseHeaders(c, job)
	c.JSON(http.StatusAccepted, service.OpenAIImageJobToPublic(job))
}

// Get handles GET /v1/images/jobs/:id.
func (h *OpenAIImageJobHandler) Get(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		openAIImageJobWriteError(c, infraerrors.New(http.StatusUnauthorized, "API_KEY_REQUIRED", "API key is required"))
		return
	}
	if h == nil || h.repository == nil {
		openAIImageJobWriteError(c, nil)
		return
	}

	job, err := h.repository.GetForUser(c.Request.Context(), subject.UserID, c.Param("id"))
	if err != nil {
		openAIImageJobWriteError(c, err)
		return
	}
	setOpenAIImageJobResponseHeaders(c, job)
	c.JSON(http.StatusOK, service.OpenAIImageJobToPublic(job))
}

// Cancel handles POST /v1/images/jobs/:id/cancel.
func (h *OpenAIImageJobHandler) Cancel(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		openAIImageJobWriteError(c, infraerrors.New(http.StatusUnauthorized, "API_KEY_REQUIRED", "API key is required"))
		return
	}
	if h == nil || h.repository == nil {
		openAIImageJobWriteError(c, nil)
		return
	}

	job, err := h.repository.CancelForUser(c.Request.Context(), subject.UserID, c.Param("id"))
	if err != nil {
		openAIImageJobWriteError(c, err)
		return
	}
	setOpenAIImageJobResponseHeaders(c, job)
	c.JSON(http.StatusOK, service.OpenAIImageJobToPublic(job))
}

func setOpenAIImageJobResponseHeaders(c *gin.Context, job *service.OpenAIImageJob) {
	c.Header("Cache-Control", "private, no-store")
	if job != nil && !service.IsTerminalOpenAIImageJobStatus(job.Status) {
		c.Header("Retry-After", "1")
	}
}

func openAIImageJobWriteError(c *gin.Context, err error) {
	status := infraerrors.Code(err)
	code := infraerrors.Reason(err)
	message := infraerrors.Message(err)
	if err == nil || status < http.StatusBadRequest || status > http.StatusNetworkAuthenticationRequired {
		status = http.StatusInternalServerError
		code = "INTERNAL_ERROR"
		message = "internal error"
	}
	if errors.Is(err, service.ErrOpenAIImageJobNotFound) {
		status = http.StatusNotFound
		code = "OPENAI_IMAGE_JOB_NOT_FOUND"
		message = "image generation job not found"
	}
	openAIImageJobWriteTypedError(c, status, "invalid_request_error", code, message)
}

func openAIImageJobWriteTypedError(c *gin.Context, status int, errorType, code, message string) {
	c.Header("Cache-Control", "private, no-store")
	c.JSON(status, gin.H{
		"error": gin.H{
			"type":    errorType,
			"code":    code,
			"message": message,
		},
	})
}

func truncateValidUTF8(value string, maxBytes int) string {
	value = strings.ToValidUTF8(strings.TrimSpace(value), "")
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	value = value[:maxBytes]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}
