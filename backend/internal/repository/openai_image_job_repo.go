package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type openAIImageJobRepository struct {
	db *sql.DB
}

const openAIImageJobCreateLockKey int64 = -708202607140001

type openAIImageJobQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

var _ service.OpenAIImageJobRepository = (*openAIImageJobRepository)(nil)

var openAIImageJobColumns = []string{
	"id",
	"job_id",
	"user_id",
	"api_key_id",
	"endpoint",
	"model",
	"content_type",
	"request_body",
	"request_hash",
	"idempotency_key_hash",
	"client_ip",
	"user_agent",
	"status",
	"response_status",
	"response_content_type",
	"response_body",
	"error_code",
	"error_message",
	"failure_unknown",
	"cancel_requested",
	"worker_id",
	"lease_expires_at",
	"attempt_count",
	"version",
	"result_expires_at",
	"created_at",
	"updated_at",
	"started_at",
	"finished_at",
}

func NewOpenAIImageJobRepository(db *sql.DB) service.OpenAIImageJobRepository {
	return &openAIImageJobRepository{db: db}
}

func (r *openAIImageJobRepository) CreateOrGet(ctx context.Context, params service.CreateOpenAIImageJobParams) (*service.OpenAIImageJob, bool, error) {
	if !service.IsSupportedOpenAIImageJobEndpoint(params.Endpoint) {
		return nil, false, fmt.Errorf("unsupported OpenAI image job endpoint %q", params.Endpoint)
	}
	normalizedIdempotencyKey, err := service.NormalizeIdempotencyKey(params.IdempotencyKey)
	if err != nil {
		return nil, false, err
	}
	if normalizedIdempotencyKey == "" {
		return nil, false, service.ErrIdempotencyKeyRequired
	}
	idempotencyKeyHash := service.HashIdempotencyKey(normalizedIdempotencyKey)
	if params.JobID == "" {
		jobID, err := service.NewOpenAIImageJobID()
		if err != nil {
			return nil, false, err
		}
		params.JobID = jobID
	}
	params.RequestHash = service.HashOpenAIImageJobRequest(params.Endpoint, params.ContentType, params.RequestBody)
	if params.MaxActivePerUser > 0 || params.MaxActiveGlobal > 0 {
		return r.createOrGetWithLimits(ctx, params, idempotencyKeyHash)
	}

	job, err := insertOpenAIImageJob(ctx, r.db, params, idempotencyKeyHash)
	if err == nil {
		return job, true, nil
	}
	if !isUniqueConstraintViolation(err) {
		return nil, false, err
	}

	existing, getErr := r.getByIdempotencyKeyHash(ctx, params.UserID, idempotencyKeyHash)
	if getErr != nil {
		// A job-id collision is not an idempotent replay. Preserve the original
		// unique error instead of translating it into a misleading not-found.
		if errors.Is(getErr, service.ErrOpenAIImageJobNotFound) {
			return nil, false, err
		}
		return nil, false, getErr
	}
	if err := service.ValidateOpenAIImageJobIdempotency(existing, params.RequestHash); err != nil {
		return nil, false, err
	}
	return existing, false, nil
}

func (r *openAIImageJobRepository) createOrGetWithLimits(
	ctx context.Context,
	params service.CreateOpenAIImageJobParams,
	idempotencyKeyHash string,
) (*service.OpenAIImageJob, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback() }()

	// This short global transaction lock makes replay-check, active counts and
	// insertion one admission decision across all application instances. A
	// status transition may only reduce the count concurrently, so the bound is
	// exact without locking every active row.
	var ignored any
	if err := tx.QueryRowContext(ctx, `SELECT pg_advisory_xact_lock($1)`, openAIImageJobCreateLockKey).Scan(&ignored); err != nil {
		return nil, false, err
	}

	existing, err := getOpenAIImageJobByIdempotencyKeyHash(ctx, tx, params.UserID, idempotencyKeyHash)
	if err == nil {
		if err := service.ValidateOpenAIImageJobIdempotency(existing, params.RequestHash); err != nil {
			return nil, false, err
		}
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}
	if !errors.Is(err, service.ErrOpenAIImageJobNotFound) {
		return nil, false, err
	}

	var userActive, globalActive int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FILTER (WHERE user_id = $1), COUNT(*)
		FROM openai_image_jobs
		WHERE status IN ($2, $3)`,
		params.UserID,
		service.OpenAIImageJobStatusQueued,
		service.OpenAIImageJobStatusRunning,
	).Scan(&userActive, &globalActive); err != nil {
		return nil, false, err
	}
	if params.MaxActivePerUser > 0 && userActive >= params.MaxActivePerUser {
		return nil, false, service.ErrOpenAIImageJobUserActiveLimit
	}
	if params.MaxActiveGlobal > 0 && globalActive >= params.MaxActiveGlobal {
		return nil, false, service.ErrOpenAIImageJobGlobalActiveLimit
	}

	job, err := insertOpenAIImageJob(ctx, tx, params, idempotencyKeyHash)
	if err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return job, true, nil
}

func insertOpenAIImageJob(
	ctx context.Context,
	queryer openAIImageJobQueryer,
	params service.CreateOpenAIImageJobParams,
	idempotencyKeyHash string,
) (*service.OpenAIImageJob, error) {
	query := `
		INSERT INTO openai_image_jobs (
			job_id, user_id, api_key_id, endpoint, model, content_type,
			request_body, request_hash, idempotency_key_hash, client_ip, user_agent, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING ` + openAIImageJobColumnList("")
	return scanOpenAIImageJob(queryer.QueryRowContext(
		ctx,
		query,
		params.JobID,
		params.UserID,
		params.APIKeyID,
		params.Endpoint,
		params.Model,
		params.ContentType,
		params.RequestBody,
		params.RequestHash,
		idempotencyKeyHash,
		params.ClientIP,
		params.UserAgent,
		service.OpenAIImageJobStatusQueued,
	))
}

func (r *openAIImageJobRepository) GetForUser(ctx context.Context, userID int64, jobID string) (*service.OpenAIImageJob, error) {
	query := `SELECT ` + openAIImageJobStatusColumnList("") + `
		FROM openai_image_jobs
		WHERE user_id = $1 AND job_id = $2`
	job, err := scanOpenAIImageJob(r.db.QueryRowContext(ctx, query, userID, jobID))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrOpenAIImageJobNotFound, nil)
	}
	return job, nil
}

func (r *openAIImageJobRepository) getByIdempotencyKeyHash(ctx context.Context, userID int64, idempotencyKeyHash string) (*service.OpenAIImageJob, error) {
	return getOpenAIImageJobByIdempotencyKeyHash(ctx, r.db, userID, idempotencyKeyHash)
}

func getOpenAIImageJobByIdempotencyKeyHash(ctx context.Context, queryer openAIImageJobQueryer, userID int64, idempotencyKeyHash string) (*service.OpenAIImageJob, error) {
	query := `SELECT ` + openAIImageJobStatusColumnList("") + `
		FROM openai_image_jobs
		WHERE user_id = $1 AND idempotency_key_hash = $2`
	job, err := scanOpenAIImageJob(queryer.QueryRowContext(ctx, query, userID, idempotencyKeyHash))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrOpenAIImageJobNotFound, nil)
	}
	return job, nil
}

func (r *openAIImageJobRepository) ClaimNext(ctx context.Context, workerID string, leaseUntil time.Time) (*service.OpenAIImageJob, error) {
	query := `
		WITH next AS (
			SELECT id
			FROM openai_image_jobs
			WHERE status = $1
			ORDER BY created_at ASC, id ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE openai_image_jobs AS jobs
		SET status = $2,
			worker_id = $3,
			lease_expires_at = $4,
			attempt_count = attempt_count + 1,
			version = version + 1,
			started_at = COALESCE(started_at, CURRENT_TIMESTAMP),
			updated_at = CURRENT_TIMESTAMP
		FROM next
		WHERE jobs.id = next.id
		RETURNING ` + openAIImageJobColumnList("jobs")
	job, err := scanOpenAIImageJob(r.db.QueryRowContext(
		ctx,
		query,
		service.OpenAIImageJobStatusQueued,
		service.OpenAIImageJobStatusRunning,
		workerID,
		leaseUntil,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *openAIImageJobRepository) Heartbeat(ctx context.Context, jobID, workerID string, leaseUntil time.Time) (bool, error) {
	query := `
		UPDATE openai_image_jobs
		SET lease_expires_at = $1,
			updated_at = CURRENT_TIMESTAMP,
			version = version + 1
		WHERE job_id = $2
			AND worker_id = $3
			AND status = $4
		RETURNING cancel_requested`
	var cancelRequested bool
	err := r.db.QueryRowContext(
		ctx,
		query,
		leaseUntil,
		jobID,
		workerID,
		service.OpenAIImageJobStatusRunning,
	).Scan(&cancelRequested)
	if errors.Is(err, sql.ErrNoRows) {
		return false, service.ErrOpenAIImageJobLeaseLost
	}
	if err != nil {
		return false, err
	}
	return cancelRequested, nil
}

func (r *openAIImageJobRepository) Complete(ctx context.Context, jobID, workerID string, response service.OpenAIImageJobResponse) error {
	if response.ResultExpiresAt.IsZero() {
		return service.ErrOpenAIImageJobResultExpiryRequired
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE openai_image_jobs
		SET status = $1,
			response_status = $2,
			response_content_type = $3,
			response_body = $4,
			result_expires_at = $5,
			request_body = NULL,
			error_code = NULL,
			error_message = NULL,
			failure_unknown = FALSE,
			worker_id = NULL,
			lease_expires_at = NULL,
			finished_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP,
			version = version + 1
		WHERE job_id = $6
			AND worker_id = $7
			AND status = $8`,
		service.OpenAIImageJobStatusCompleted,
		response.StatusCode,
		response.ContentType,
		response.Body,
		nullableOpenAIImageJobTime(response.ResultExpiresAt),
		jobID,
		workerID,
		service.OpenAIImageJobStatusRunning,
	)
	return openAIImageJobLeaseGuardResult(result, err)
}

func (r *openAIImageJobRepository) Fail(ctx context.Context, jobID, workerID, code, message string, unknown bool) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE openai_image_jobs
		SET status = $1,
			error_code = $2,
			error_message = $3,
			failure_unknown = $4,
			request_body = NULL,
			response_status = NULL,
			response_content_type = NULL,
			response_body = NULL,
			result_expires_at = NULL,
			worker_id = NULL,
			lease_expires_at = NULL,
			finished_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP,
			version = version + 1
		WHERE job_id = $5
			AND worker_id = $6
			AND status = $7`,
		service.OpenAIImageJobStatusFailed,
		code,
		message,
		unknown,
		jobID,
		workerID,
		service.OpenAIImageJobStatusRunning,
	)
	return openAIImageJobLeaseGuardResult(result, err)
}

func (r *openAIImageJobRepository) CancelForUser(ctx context.Context, userID int64, jobID string) (*service.OpenAIImageJob, error) {
	query := `
		UPDATE openai_image_jobs
		SET cancel_requested = CASE
				WHEN status IN ('queued', 'running') THEN TRUE
				ELSE cancel_requested
			END,
			status = CASE WHEN status = 'queued' THEN 'cancelled' ELSE status END,
			request_body = CASE WHEN status = 'queued' THEN NULL ELSE request_body END,
			finished_at = CASE WHEN status = 'queued' THEN CURRENT_TIMESTAMP ELSE finished_at END,
			updated_at = CASE
				WHEN status IN ('queued', 'running') THEN CURRENT_TIMESTAMP
				ELSE updated_at
			END,
			version = CASE
				WHEN status IN ('queued', 'running') THEN version + 1
				ELSE version
			END
		WHERE user_id = $1 AND job_id = $2
		RETURNING ` + openAIImageJobStatusColumnList("")
	job, err := scanOpenAIImageJob(r.db.QueryRowContext(ctx, query, userID, jobID))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrOpenAIImageJobNotFound, nil)
	}
	return job, nil
}

func (r *openAIImageJobRepository) MarkCancelled(ctx context.Context, jobID, workerID string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE openai_image_jobs
		SET status = $1,
			request_body = NULL,
			response_status = NULL,
			response_content_type = NULL,
			response_body = NULL,
			result_expires_at = NULL,
			worker_id = NULL,
			lease_expires_at = NULL,
			finished_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP,
			version = version + 1
		WHERE job_id = $2
			AND worker_id = $3
			AND status = $4
			AND cancel_requested = TRUE`,
		service.OpenAIImageJobStatusCancelled,
		jobID,
		workerID,
		service.OpenAIImageJobStatusRunning,
	)
	return openAIImageJobLeaseGuardResult(result, err)
}

func (r *openAIImageJobRepository) FailExpiredLeases(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE openai_image_jobs
		SET status = $1,
			error_code = $2,
			error_message = $3,
			failure_unknown = TRUE,
			request_body = NULL,
			worker_id = NULL,
			lease_expires_at = NULL,
			finished_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP,
			version = version + 1
		WHERE status = $4
			AND lease_expires_at IS NOT NULL
			AND lease_expires_at <= $5`,
		service.OpenAIImageJobStatusFailed,
		"failed_unknown",
		"image generation worker stopped after upstream dispatch; the request was not retried",
		service.OpenAIImageJobStatusRunning,
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *openAIImageJobRepository) PurgeExpiredPayloads(ctx context.Context, params service.OpenAIImageJobCleanupParams) (service.OpenAIImageJobCleanupResult, error) {
	limit := normalizeOpenAIImageJobCleanupBatchLimit(params.BatchLimit)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return service.OpenAIImageJobCleanupResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var result service.OpenAIImageJobCleanupResult
	queuedResult, err := tx.ExecContext(ctx, `
		WITH expired_queued AS (
			SELECT id
			FROM openai_image_jobs
			WHERE status = $1
				AND created_at < $2
			ORDER BY created_at ASC, id ASC
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		UPDATE openai_image_jobs AS jobs
		SET status = $4,
			error_code = $5,
			error_message = $6,
			failure_unknown = FALSE,
			request_body = NULL,
			worker_id = NULL,
			lease_expires_at = NULL,
			finished_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP,
			version = version + 1
		FROM expired_queued
		WHERE jobs.id = expired_queued.id`,
		service.OpenAIImageJobStatusQueued,
		params.QueuedCutoff,
		limit,
		service.OpenAIImageJobStatusFailed,
		"queue_expired",
		"image generation job expired before execution",
	)
	if err != nil {
		return service.OpenAIImageJobCleanupResult{}, err
	}
	result.Queued, err = queuedResult.RowsAffected()
	if err != nil {
		return service.OpenAIImageJobCleanupResult{}, err
	}

	payloadResult, err := tx.ExecContext(ctx, `
		WITH expired_payloads AS (
			SELECT id
			FROM openai_image_jobs
			WHERE status IN ('completed', 'failed', 'cancelled')
				AND (
					request_body IS NOT NULL
					OR (
						response_body IS NOT NULL
						AND result_expires_at IS NOT NULL
						AND result_expires_at <= $1
					)
					OR (
						response_body IS NOT NULL
						AND result_expires_at IS NULL
						AND finished_at IS NOT NULL
						AND finished_at < $2
					)
				)
			ORDER BY COALESCE(finished_at, created_at) ASC, id ASC
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		UPDATE openai_image_jobs AS jobs
		SET request_body = NULL,
			response_body = CASE
				WHEN jobs.result_expires_at IS NOT NULL AND jobs.result_expires_at <= $1 THEN NULL
				WHEN jobs.result_expires_at IS NULL AND jobs.finished_at IS NOT NULL AND jobs.finished_at < $2 THEN NULL
				ELSE jobs.response_body
			END
		FROM expired_payloads
		WHERE jobs.id = expired_payloads.id`, params.Now, params.RecordCutoff, limit)
	if err != nil {
		return service.OpenAIImageJobCleanupResult{}, err
	}
	result.Payloads, err = payloadResult.RowsAffected()
	if err != nil {
		return service.OpenAIImageJobCleanupResult{}, err
	}

	recordResult, err := tx.ExecContext(ctx, `
		WITH expired_records AS (
			SELECT id
			FROM openai_image_jobs
			WHERE status IN ('completed', 'failed', 'cancelled')
				AND finished_at IS NOT NULL
				AND finished_at < $1
				AND request_body IS NULL
				AND response_body IS NULL
			ORDER BY finished_at ASC, id ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		DELETE FROM openai_image_jobs AS jobs
		USING expired_records
		WHERE jobs.id = expired_records.id
			AND jobs.request_body IS NULL
			AND jobs.response_body IS NULL`, params.RecordCutoff, limit)
	if err != nil {
		return service.OpenAIImageJobCleanupResult{}, err
	}
	result.Records, err = recordResult.RowsAffected()
	if err != nil {
		return service.OpenAIImageJobCleanupResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return service.OpenAIImageJobCleanupResult{}, err
	}
	return result, nil
}

func normalizeOpenAIImageJobCleanupBatchLimit(limit int) int {
	if limit <= 0 {
		return service.DefaultOpenAIImageJobCleanupBatchLimit
	}
	if limit > service.MaxOpenAIImageJobCleanupBatchLimit {
		return service.MaxOpenAIImageJobCleanupBatchLimit
	}
	return limit
}

func openAIImageJobColumnList(alias string) string {
	if alias == "" {
		return strings.Join(openAIImageJobColumns, ", ")
	}
	qualified := make([]string, 0, len(openAIImageJobColumns))
	for _, column := range openAIImageJobColumns {
		qualified = append(qualified, alias+"."+column)
	}
	return strings.Join(qualified, ", ")
}

// openAIImageJobStatusColumnList is the only projection suitable for public
// status and cancellation lookups. It never moves request bytes out of
// PostgreSQL and only returns result bytes for completed rows.
func openAIImageJobStatusColumnList(alias string) string {
	qualifiedColumn := func(column string) string {
		if alias == "" {
			return column
		}
		return alias + "." + column
	}
	columns := make([]string, 0, len(openAIImageJobColumns))
	for _, column := range openAIImageJobColumns {
		switch column {
		case "request_body":
			columns = append(columns, "NULL::bytea AS request_body")
		case "response_body":
			columns = append(columns, fmt.Sprintf(
				"CASE WHEN %s = 'completed' THEN %s ELSE NULL::bytea END AS response_body",
				qualifiedColumn("status"),
				qualifiedColumn("response_body"),
			))
		default:
			columns = append(columns, qualifiedColumn(column))
		}
	}
	return strings.Join(columns, ", ")
}

type openAIImageJobScanner interface {
	Scan(dest ...any) error
}

func scanOpenAIImageJob(scanner openAIImageJobScanner) (*service.OpenAIImageJob, error) {
	var (
		job             service.OpenAIImageJob
		responseStatus  sql.NullInt64
		responseType    sql.NullString
		errorCode       sql.NullString
		errorMessage    sql.NullString
		workerID        sql.NullString
		leaseExpiresAt  sql.NullTime
		resultExpiresAt sql.NullTime
		startedAt       sql.NullTime
		finishedAt      sql.NullTime
	)
	if err := scanner.Scan(
		&job.ID,
		&job.JobID,
		&job.UserID,
		&job.APIKeyID,
		&job.Endpoint,
		&job.Model,
		&job.ContentType,
		&job.RequestBody,
		&job.RequestHash,
		&job.IdempotencyKeyHash,
		&job.ClientIP,
		&job.UserAgent,
		&job.Status,
		&responseStatus,
		&responseType,
		&job.ResponseBody,
		&errorCode,
		&errorMessage,
		&job.FailureUnknown,
		&job.CancelRequested,
		&workerID,
		&leaseExpiresAt,
		&job.AttemptCount,
		&job.Version,
		&resultExpiresAt,
		&job.CreatedAt,
		&job.UpdatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		return nil, err
	}
	if responseStatus.Valid {
		job.ResponseStatus = int(responseStatus.Int64)
	}
	if responseType.Valid {
		job.ResponseType = responseType.String
	}
	if errorCode.Valid {
		job.ErrorCode = &errorCode.String
	}
	if errorMessage.Valid {
		job.ErrorMessage = &errorMessage.String
	}
	if workerID.Valid {
		job.WorkerID = &workerID.String
	}
	if leaseExpiresAt.Valid {
		job.LeaseExpiresAt = &leaseExpiresAt.Time
	}
	if resultExpiresAt.Valid {
		job.ResultExpiresAt = &resultExpiresAt.Time
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		job.FinishedAt = &finishedAt.Time
	}
	return &job, nil
}

func openAIImageJobLeaseGuardResult(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrOpenAIImageJobLeaseLost
	}
	return nil
}

func nullableOpenAIImageJobTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
