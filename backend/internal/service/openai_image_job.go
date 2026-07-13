package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	OpenAIImageJobStatusQueued    = "queued"
	OpenAIImageJobStatusRunning   = "running"
	OpenAIImageJobStatusCompleted = "completed"
	OpenAIImageJobStatusFailed    = "failed"
	OpenAIImageJobStatusCancelled = "cancelled"
)

const (
	OpenAIImageJobEndpointGenerations = "/v1/images/generations"
	OpenAIImageJobEndpointEdits       = "/v1/images/edits"
)

var (
	ErrOpenAIImageJobNotFound = infraerrors.New(
		http.StatusNotFound,
		"OPENAI_IMAGE_JOB_NOT_FOUND",
		"image generation job not found",
	)
	ErrOpenAIImageJobIdempotencyConflict = infraerrors.New(
		http.StatusConflict,
		"OPENAI_IMAGE_JOB_IDEMPOTENCY_CONFLICT",
		"idempotency key reused with a different image request",
	)
	ErrOpenAIImageJobInvalidTransition = infraerrors.New(
		http.StatusConflict,
		"OPENAI_IMAGE_JOB_INVALID_TRANSITION",
		"image generation job status transition is not allowed",
	)
	ErrOpenAIImageJobLeaseLost = infraerrors.New(
		http.StatusConflict,
		"OPENAI_IMAGE_JOB_LEASE_LOST",
		"image generation job worker lease was lost",
	)
	ErrOpenAIImageJobResultExpiryRequired = infraerrors.New(
		http.StatusInternalServerError,
		"OPENAI_IMAGE_JOB_RESULT_EXPIRY_REQUIRED",
		"completed image generation job requires a result expiry",
	)
	ErrOpenAIImageJobUserActiveLimit = infraerrors.New(
		http.StatusTooManyRequests,
		"OPENAI_IMAGE_JOB_USER_ACTIVE_LIMIT",
		"too many active image generation jobs for this user",
	)
	ErrOpenAIImageJobGlobalActiveLimit = infraerrors.New(
		http.StatusServiceUnavailable,
		"OPENAI_IMAGE_JOB_GLOBAL_ACTIVE_LIMIT",
		"image generation queue is currently full",
	)
)

// OpenAIImageJob contains the complete durable execution state for one Images
// API request. Authentication secrets are deliberately absent: the immutable
// APIKeyID is resolved and re-authenticated immediately before execution.
type OpenAIImageJob struct {
	ID                 int64
	JobID              string
	UserID             int64
	APIKeyID           int64
	Endpoint           string
	Model              string
	ContentType        string
	RequestBody        []byte
	RequestHash        string
	IdempotencyKeyHash string
	ClientIP           string
	UserAgent          string

	Status          string
	ResponseStatus  int
	ResponseType    string
	ResponseBody    []byte
	ErrorCode       *string
	ErrorMessage    *string
	FailureUnknown  bool
	CancelRequested bool

	WorkerID       *string
	LeaseExpiresAt *time.Time
	AttemptCount   int
	Version        int

	ResultExpiresAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
}

type CreateOpenAIImageJobParams struct {
	JobID       string
	UserID      int64
	APIKeyID    int64
	Endpoint    string
	Model       string
	ContentType string
	RequestBody []byte
	// RequestHash is informational input only. The repository recomputes it
	// from Endpoint, ContentType and RequestBody before every insert/replay.
	RequestHash string
	// IdempotencyKey is normalized and hashed by the repository. The raw value
	// is never persisted.
	IdempotencyKey string
	ClientIP       string
	UserAgent      string
	// Positive limits enable an atomic, cross-instance admission check. The
	// repository serializes the short count-and-insert transaction so bursts
	// cannot race past the configured queue bounds.
	MaxActivePerUser int
	MaxActiveGlobal  int
}

type OpenAIImageJobResponse struct {
	StatusCode      int
	ContentType     string
	Body            []byte
	ResultExpiresAt time.Time
}

const (
	DefaultOpenAIImageJobCleanupBatchLimit = 100
	MaxOpenAIImageJobCleanupBatchLimit     = 1000
)

type OpenAIImageJobCleanupParams struct {
	Now          time.Time
	QueuedCutoff time.Time
	RecordCutoff time.Time
	BatchLimit   int
}

type OpenAIImageJobCleanupResult struct {
	Queued   int64
	Payloads int64
	Records  int64
}

type OpenAIImageJobPublicError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type OpenAIImageJobPublic struct {
	ID              string                     `json:"id"`
	Object          string                     `json:"object"`
	Status          string                     `json:"status"`
	CancelRequested bool                       `json:"cancel_requested,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	StartedAt       *time.Time                 `json:"started_at,omitempty"`
	FinishedAt      *time.Time                 `json:"finished_at,omitempty"`
	Result          json.RawMessage            `json:"result,omitempty"`
	Error           *OpenAIImageJobPublicError `json:"error,omitempty"`
}

// OpenAIImageJobRepository is the persistence boundary used by both the HTTP
// surface and the background worker. Every worker terminal write is guarded by
// both worker ownership and the running state.
type OpenAIImageJobRepository interface {
	CreateOrGet(ctx context.Context, params CreateOpenAIImageJobParams) (*OpenAIImageJob, bool, error)
	GetForUser(ctx context.Context, userID int64, jobID string) (*OpenAIImageJob, error)
	ClaimNext(ctx context.Context, workerID string, leaseUntil time.Time) (*OpenAIImageJob, error)
	Heartbeat(ctx context.Context, jobID, workerID string, leaseUntil time.Time) (cancelRequested bool, err error)
	Complete(ctx context.Context, jobID, workerID string, response OpenAIImageJobResponse) error
	Fail(ctx context.Context, jobID, workerID, code, message string, unknown bool) error
	CancelForUser(ctx context.Context, userID int64, jobID string) (*OpenAIImageJob, error)
	MarkCancelled(ctx context.Context, jobID, workerID string) error
	FailExpiredLeases(ctx context.Context, cutoff time.Time) (int64, error)
	PurgeExpiredPayloads(ctx context.Context, params OpenAIImageJobCleanupParams) (OpenAIImageJobCleanupResult, error)
}

func NewOpenAIImageJobID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "imgjob_" + hex.EncodeToString(value[:]), nil
}

func IsSupportedOpenAIImageJobEndpoint(endpoint string) bool {
	switch endpoint {
	case OpenAIImageJobEndpointGenerations, OpenAIImageJobEndpointEdits:
		return true
	default:
		return false
	}
}

func IsTerminalOpenAIImageJobStatus(status string) bool {
	switch status {
	case OpenAIImageJobStatusCompleted, OpenAIImageJobStatusFailed, OpenAIImageJobStatusCancelled:
		return true
	default:
		return false
	}
}

func CanTransitionOpenAIImageJob(from, to string) bool {
	if from == "" || to == "" || IsTerminalOpenAIImageJobStatus(from) {
		return false
	}
	switch from {
	case OpenAIImageJobStatusQueued:
		return to == OpenAIImageJobStatusRunning ||
			to == OpenAIImageJobStatusFailed ||
			to == OpenAIImageJobStatusCancelled
	case OpenAIImageJobStatusRunning:
		return to == OpenAIImageJobStatusCompleted ||
			to == OpenAIImageJobStatusFailed ||
			to == OpenAIImageJobStatusCancelled
	default:
		return false
	}
}

// HashOpenAIImageJobRequest binds an idempotency key to one semantic request.
// JSON and multipart transport noise is canonicalized so a browser can safely
// rebuild a retry; malformed payloads fall back to hashing the exact bytes.
func HashOpenAIImageJobRequest(endpoint, contentType string, body []byte) string {
	hash := sha256.New()
	endpoint = strings.TrimSpace(endpoint)
	contentType = strings.TrimSpace(contentType)
	payload := body
	if mediaType, params, err := mime.ParseMediaType(contentType); err == nil {
		switch {
		case strings.EqualFold(mediaType, "multipart/form-data") && params["boundary"] != "":
			if canonical, ok := canonicalOpenAIImageJobMultipart(body, params["boundary"]); ok {
				contentType = strings.ToLower(mediaType)
				payload = canonical
			}
		case strings.EqualFold(mediaType, "application/json") || strings.HasSuffix(strings.ToLower(mediaType), "+json"):
			if canonical, ok := canonicalOpenAIImageJobJSON(body); ok {
				contentType = strings.ToLower(mediaType)
				payload = canonical
			}
		}
	}
	hashOpenAIImageJobComponent(hash, []byte(endpoint))
	hashOpenAIImageJobComponent(hash, []byte(contentType))
	hashOpenAIImageJobComponent(hash, payload)
	return hex.EncodeToString(hash.Sum(nil))
}

func canonicalOpenAIImageJobJSON(body []byte) ([]byte, bool) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, false
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	return canonical, true
}

type openAIImageJobHashWriter interface {
	Write([]byte) (int, error)
}

func hashOpenAIImageJobComponent(writer openAIImageJobHashWriter, value []byte) {
	_, _ = fmt.Fprintf(writer, "%d:", len(value))
	_, _ = writer.Write(value)
}

// canonicalOpenAIImageJobMultipart deliberately excludes the transport-only
// boundary. It preserves part order and binds the semantic part metadata and
// exact content bytes, allowing a browser to rebuild equivalent FormData on a
// safe idempotent retry.
func canonicalOpenAIImageJobMultipart(body []byte, boundary string) ([]byte, bool) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var canonical bytes.Buffer
	partCount := 0
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, false
		}

		contentHash := sha256.New()
		if _, err := io.Copy(contentHash, part); err != nil {
			_ = part.Close()
			return nil, false
		}
		if err := part.Close(); err != nil {
			return nil, false
		}

		hashOpenAIImageJobComponent(&canonical, []byte(part.FormName()))
		hashOpenAIImageJobComponent(&canonical, []byte(part.FileName()))
		hashOpenAIImageJobComponent(&canonical, []byte(strings.TrimSpace(part.Header.Get("Content-Type"))))
		hashOpenAIImageJobComponent(&canonical, contentHash.Sum(nil))
		partCount++
	}
	if partCount == 0 {
		return nil, false
	}
	return canonical.Bytes(), true
}

func ValidateOpenAIImageJobIdempotency(job *OpenAIImageJob, requestHash string) error {
	if job == nil {
		return ErrOpenAIImageJobNotFound
	}
	if len(job.RequestHash) != len(requestHash) ||
		subtle.ConstantTimeCompare([]byte(job.RequestHash), []byte(requestHash)) != 1 {
		return ErrOpenAIImageJobIdempotencyConflict
	}
	return nil
}

func OpenAIImageJobToPublic(job *OpenAIImageJob) *OpenAIImageJobPublic {
	return OpenAIImageJobToPublicAt(job, time.Now())
}

// OpenAIImageJobToPublicAt makes retention behavior deterministic for callers
// and tests. A completed row is never exposed without a usable result.
func OpenAIImageJobToPublicAt(job *OpenAIImageJob, now time.Time) *OpenAIImageJobPublic {
	if job == nil {
		return nil
	}
	public := &OpenAIImageJobPublic{
		ID:              job.JobID,
		Object:          "image_generation.job",
		Status:          job.Status,
		CancelRequested: job.CancelRequested,
		CreatedAt:       job.CreatedAt,
		StartedAt:       cloneOpenAIImageJobTime(job.StartedAt),
		FinishedAt:      cloneOpenAIImageJobTime(job.FinishedAt),
	}
	if job.Status == OpenAIImageJobStatusCompleted {
		if job.ResultExpiresAt == nil || !job.ResultExpiresAt.After(now) || len(job.ResponseBody) == 0 {
			public.Status = OpenAIImageJobStatusFailed
			public.Error = &OpenAIImageJobPublicError{
				Code:    "result_expired",
				Message: "image generation result has expired",
			}
			return public
		}
		public.Result = append(json.RawMessage(nil), job.ResponseBody...)
	}
	if job.Status == OpenAIImageJobStatusFailed {
		public.Error = &OpenAIImageJobPublicError{
			Code:    derefOpenAIImageJobString(job.ErrorCode, "image_generation_failed"),
			Message: derefOpenAIImageJobString(job.ErrorMessage, "image generation failed"),
		}
	}
	if job.Status == OpenAIImageJobStatusCancelled {
		public.Error = &OpenAIImageJobPublicError{
			Code:    "cancelled",
			Message: "image generation was cancelled",
		}
	}
	return public
}

// RequestCancellation applies the public cancellation semantics in memory.
// Queued work is cancelled immediately. Running work records intent only so a
// successful (and potentially billable) upstream response can still win.
func (job *OpenAIImageJob) RequestCancellation(now time.Time) error {
	if job == nil {
		return ErrOpenAIImageJobNotFound
	}
	switch job.Status {
	case OpenAIImageJobStatusQueued:
		job.Status = OpenAIImageJobStatusCancelled
		job.CancelRequested = true
		job.RequestBody = nil
		job.FinishedAt = cloneOpenAIImageJobTime(&now)
		job.UpdatedAt = now
		job.Version++
		return nil
	case OpenAIImageJobStatusRunning:
		job.CancelRequested = true
		job.UpdatedAt = now
		job.Version++
		return nil
	case OpenAIImageJobStatusCompleted, OpenAIImageJobStatusFailed, OpenAIImageJobStatusCancelled:
		return nil
	default:
		return ErrOpenAIImageJobInvalidTransition
	}
}

// MarkCompleted intentionally does not reject CancelRequested. Completion wins
// a running cancellation race because an upstream result may already be billed.
func (job *OpenAIImageJob) MarkCompleted(response OpenAIImageJobResponse, now time.Time) error {
	if job == nil {
		return ErrOpenAIImageJobNotFound
	}
	if !CanTransitionOpenAIImageJob(job.Status, OpenAIImageJobStatusCompleted) {
		return ErrOpenAIImageJobInvalidTransition
	}
	if response.ResultExpiresAt.IsZero() {
		return ErrOpenAIImageJobResultExpiryRequired
	}
	job.Status = OpenAIImageJobStatusCompleted
	job.ResponseStatus = response.StatusCode
	job.ResponseType = response.ContentType
	job.ResponseBody = append(job.ResponseBody[:0], response.Body...)
	job.ResultExpiresAt = cloneOpenAIImageJobTime(&response.ResultExpiresAt)
	job.RequestBody = nil
	job.ErrorCode = nil
	job.ErrorMessage = nil
	job.FailureUnknown = false
	job.WorkerID = nil
	job.LeaseExpiresAt = nil
	job.FinishedAt = cloneOpenAIImageJobTime(&now)
	job.UpdatedAt = now
	job.Version++
	return nil
}

func cloneOpenAIImageJobTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func derefOpenAIImageJobString(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return *value
}
