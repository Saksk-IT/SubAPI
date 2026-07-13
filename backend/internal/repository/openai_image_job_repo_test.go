package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func TestOpenAIImageJobRepositoryClaimUsesSkipLocked(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	now := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	leaseUntil := now.Add(2 * time.Minute)
	mock.ExpectQuery(`(?s)FOR UPDATE SKIP LOCKED.*UPDATE openai_image_jobs.*RETURNING`).
		WithArgs(service.OpenAIImageJobStatusQueued, service.OpenAIImageJobStatusRunning, "worker-1", leaseUntil).
		WillReturnRows(openAIImageJobRows().AddRow(openAIImageJobRowValues(
			"imgjob_claim", 12, 34, service.OpenAIImageJobStatusRunning, "request-hash", "idem", now,
		)...))

	job, err := repo.ClaimNext(context.Background(), "worker-1", leaseUntil)
	if err != nil {
		t.Fatalf("claim next: %v", err)
	}
	if job == nil || job.JobID != "imgjob_claim" || job.Status != service.OpenAIImageJobStatusRunning {
		t.Fatalf("unexpected claimed job: %#v", job)
	}
	if len(job.RequestBody) == 0 {
		t.Fatal("worker claim projection omitted the durable request payload")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIImageJobRepositoryGetIsUserScoped(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	now := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	row := openAIImageJobRowValues(
		"imgjob_owned", 12, 34, service.OpenAIImageJobStatusQueued, "request-hash", "idem", now,
	)
	row[7] = nil
	mock.ExpectQuery(`(?s)SELECT.*NULL::bytea AS request_body.*CASE WHEN status = 'completed' THEN response_body ELSE NULL::bytea END AS response_body.*FROM openai_image_jobs\s+WHERE user_id = \$1 AND job_id = \$2`).
		WithArgs(int64(12), "imgjob_owned").
		WillReturnRows(openAIImageJobRows().AddRow(row...))

	job, err := repo.GetForUser(context.Background(), 12, "imgjob_owned")
	if err != nil {
		t.Fatalf("get for user: %v", err)
	}
	if job.UserID != 12 || job.JobID != "imgjob_owned" {
		t.Fatalf("unexpected owned job: %#v", job)
	}
	if len(job.RequestBody) != 0 {
		t.Fatalf("status lookup leaked request payload: %#v", job.RequestBody)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIImageJobRepositoryHeartbeatReturnsCrossInstanceCancellation(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	leaseUntil := time.Date(2026, time.July, 14, 10, 2, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)UPDATE openai_image_jobs.*WHERE job_id = \$2\s+AND worker_id = \$3\s+AND status = \$4.*RETURNING cancel_requested`).
		WithArgs(leaseUntil, "imgjob_running", "worker-owner", service.OpenAIImageJobStatusRunning).
		WillReturnRows(sqlmock.NewRows([]string{"cancel_requested"}).AddRow(true))

	cancelRequested, err := repo.Heartbeat(context.Background(), "imgjob_running", "worker-owner", leaseUntil)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if !cancelRequested {
		t.Fatal("heartbeat did not expose cancellation requested by another instance")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIImageJobRepositoryCreateReturnsExistingForSameRequest(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	now := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	params := openAIImageJobCreateParams("imgjob_new", "same-hash")
	actualHash := service.HashOpenAIImageJobRequest(params.Endpoint, params.ContentType, params.RequestBody)
	mock.ExpectQuery(`(?s)INSERT INTO openai_image_jobs`).
		WithArgs(
			params.JobID, params.UserID, params.APIKeyID, params.Endpoint, params.Model,
			params.ContentType, params.RequestBody, actualHash, service.HashIdempotencyKey(params.IdempotencyKey),
			params.ClientIP, params.UserAgent, service.OpenAIImageJobStatusQueued,
		).
		WillReturnError(&pq.Error{Code: "23505", Constraint: "openai_image_jobs_user_id_idempotency_key_key"})
	mock.ExpectQuery(`(?s)FROM openai_image_jobs\s+WHERE user_id = \$1 AND idempotency_key_hash = \$2`).
		WithArgs(params.UserID, service.HashIdempotencyKey(params.IdempotencyKey)).
		WillReturnRows(openAIImageJobRows().AddRow(openAIImageJobRowValues(
			"imgjob_existing", params.UserID, params.APIKeyID, service.OpenAIImageJobStatusRunning,
			actualHash, service.HashIdempotencyKey(params.IdempotencyKey), now,
		)...))

	job, created, err := repo.CreateOrGet(context.Background(), params)
	if err != nil {
		t.Fatalf("create or get: %v", err)
	}
	if created || job.JobID != "imgjob_existing" {
		t.Fatalf("expected existing job, created=%v job=%#v", created, job)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIImageJobRepositoryCreateDetectsHashConflict(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	now := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	params := openAIImageJobCreateParams("imgjob_new", "new-hash")
	actualHash := service.HashOpenAIImageJobRequest(params.Endpoint, params.ContentType, params.RequestBody)
	mock.ExpectQuery(`(?s)INSERT INTO openai_image_jobs`).
		WithArgs(
			params.JobID, params.UserID, params.APIKeyID, params.Endpoint, params.Model,
			params.ContentType, params.RequestBody, actualHash, service.HashIdempotencyKey(params.IdempotencyKey),
			params.ClientIP, params.UserAgent, service.OpenAIImageJobStatusQueued,
		).
		WillReturnError(&pq.Error{Code: "23505"})
	mock.ExpectQuery(`(?s)FROM openai_image_jobs\s+WHERE user_id = \$1 AND idempotency_key_hash = \$2`).
		WithArgs(params.UserID, service.HashIdempotencyKey(params.IdempotencyKey)).
		WillReturnRows(openAIImageJobRows().AddRow(openAIImageJobRowValues(
			"imgjob_existing", params.UserID, params.APIKeyID, service.OpenAIImageJobStatusQueued,
			"different-hash", service.HashIdempotencyKey(params.IdempotencyKey), now,
		)...))

	_, created, err := repo.CreateOrGet(context.Background(), params)
	if created || !errors.Is(err, service.ErrOpenAIImageJobIdempotencyConflict) {
		t.Fatalf("created=%v error=%v, want idempotency conflict", created, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIImageJobRepositoryCreateRecomputesCallerRequestHash(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	now := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	params := openAIImageJobCreateParams("imgjob_rehash", "caller-forged-hash")
	actualHash := service.HashOpenAIImageJobRequest(params.Endpoint, params.ContentType, params.RequestBody)
	if actualHash == params.RequestHash {
		t.Fatal("test setup did not create a mismatched caller hash")
	}
	mock.ExpectQuery(`(?s)INSERT INTO openai_image_jobs`).
		WithArgs(
			params.JobID, params.UserID, params.APIKeyID, params.Endpoint, params.Model,
			params.ContentType, params.RequestBody, actualHash, service.HashIdempotencyKey(params.IdempotencyKey),
			params.ClientIP, params.UserAgent, service.OpenAIImageJobStatusQueued,
		).
		WillReturnRows(openAIImageJobRows().AddRow(openAIImageJobRowValues(
			params.JobID, params.UserID, params.APIKeyID, service.OpenAIImageJobStatusQueued,
			actualHash, service.HashIdempotencyKey(params.IdempotencyKey), now,
		)...))

	job, created, err := repo.CreateOrGet(context.Background(), params)
	if err != nil {
		t.Fatalf("create or get: %v", err)
	}
	if !created || job.RequestHash != actualHash {
		t.Fatalf("created=%v job hash=%q, want recomputed %q", created, job.RequestHash, actualHash)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIImageJobRepositoryCreateWithLimitsReturnsReplayBeforeCounting(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	now := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	params := openAIImageJobCreateParams("imgjob_replay", "ignored")
	params.MaxActivePerUser = 3
	params.MaxActiveGlobal = 20
	actualHash := service.HashOpenAIImageJobRequest(params.Endpoint, params.ContentType, params.RequestBody)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT pg_advisory_xact_lock\(\$1\)`).
		WithArgs(openAIImageJobCreateLockKey).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_xact_lock"}).AddRow(nil))
	mock.ExpectQuery(`(?s)FROM openai_image_jobs\s+WHERE user_id = \$1 AND idempotency_key_hash = \$2`).
		WithArgs(params.UserID, service.HashIdempotencyKey(params.IdempotencyKey)).
		WillReturnRows(openAIImageJobRows().AddRow(openAIImageJobRowValues(
			"imgjob_existing", params.UserID, params.APIKeyID, service.OpenAIImageJobStatusRunning,
			actualHash, service.HashIdempotencyKey(params.IdempotencyKey), now,
		)...))
	mock.ExpectCommit()

	job, created, err := repo.CreateOrGet(context.Background(), params)
	if err != nil {
		t.Fatalf("create replay: %v", err)
	}
	if created || job.JobID != "imgjob_existing" {
		t.Fatalf("created=%v job=%#v, want existing replay", created, job)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIImageJobRepositoryCreateWithLimitsRejectsUserCapacityAtomically(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	params := openAIImageJobCreateParams("imgjob_limited", "ignored")
	params.MaxActivePerUser = 3
	params.MaxActiveGlobal = 20

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT pg_advisory_xact_lock\(\$1\)`).
		WithArgs(openAIImageJobCreateLockKey).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_xact_lock"}).AddRow(nil))
	mock.ExpectQuery(`(?s)FROM openai_image_jobs\s+WHERE user_id = \$1 AND idempotency_key_hash = \$2`).
		WithArgs(params.UserID, service.HashIdempotencyKey(params.IdempotencyKey)).
		WillReturnRows(openAIImageJobRows())
	mock.ExpectQuery(`(?s)COUNT\(\*\) FILTER \(WHERE user_id = \$1\).*COUNT\(\*\).*status IN \(\$2, \$3\)`).
		WithArgs(params.UserID, service.OpenAIImageJobStatusQueued, service.OpenAIImageJobStatusRunning).
		WillReturnRows(sqlmock.NewRows([]string{"user_active", "global_active"}).AddRow(3, 9))
	mock.ExpectRollback()

	job, created, err := repo.CreateOrGet(context.Background(), params)
	if job != nil || created || !errors.Is(err, service.ErrOpenAIImageJobUserActiveLimit) {
		t.Fatalf("job=%#v created=%v error=%v, want user active limit", job, created, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIImageJobRepositoryCreateWithLimitsInsertsInsideSerializedTransaction(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	now := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	params := openAIImageJobCreateParams("imgjob_limited_ok", "ignored")
	params.MaxActivePerUser = 3
	params.MaxActiveGlobal = 20
	actualHash := service.HashOpenAIImageJobRequest(params.Endpoint, params.ContentType, params.RequestBody)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT pg_advisory_xact_lock\(\$1\)`).
		WithArgs(openAIImageJobCreateLockKey).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_xact_lock"}).AddRow(nil))
	mock.ExpectQuery(`(?s)FROM openai_image_jobs\s+WHERE user_id = \$1 AND idempotency_key_hash = \$2`).
		WithArgs(params.UserID, service.HashIdempotencyKey(params.IdempotencyKey)).
		WillReturnRows(openAIImageJobRows())
	mock.ExpectQuery(`(?s)COUNT\(\*\) FILTER \(WHERE user_id = \$1\).*COUNT\(\*\).*status IN \(\$2, \$3\)`).
		WithArgs(params.UserID, service.OpenAIImageJobStatusQueued, service.OpenAIImageJobStatusRunning).
		WillReturnRows(sqlmock.NewRows([]string{"user_active", "global_active"}).AddRow(2, 19))
	mock.ExpectQuery(`(?s)INSERT INTO openai_image_jobs`).
		WithArgs(
			params.JobID, params.UserID, params.APIKeyID, params.Endpoint, params.Model,
			params.ContentType, params.RequestBody, actualHash, service.HashIdempotencyKey(params.IdempotencyKey),
			params.ClientIP, params.UserAgent, service.OpenAIImageJobStatusQueued,
		).
		WillReturnRows(openAIImageJobRows().AddRow(openAIImageJobRowValues(
			params.JobID, params.UserID, params.APIKeyID, service.OpenAIImageJobStatusQueued,
			actualHash, service.HashIdempotencyKey(params.IdempotencyKey), now,
		)...))
	mock.ExpectCommit()

	job, created, err := repo.CreateOrGet(context.Background(), params)
	if err != nil {
		t.Fatalf("create within limits: %v", err)
	}
	if !created || job.JobID != params.JobID {
		t.Fatalf("created=%v job=%#v, want new job", created, job)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIImageJobRepositoryTerminalWritesRequireWorkerOwnership(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(service.OpenAIImageJobRepository) error
		args []driver.Value
	}{
		{
			name: "complete",
			run: func(repo service.OpenAIImageJobRepository) error {
				return repo.Complete(context.Background(), "imgjob_terminal", "worker-owner", service.OpenAIImageJobResponse{
					StatusCode:      200,
					ContentType:     "application/json",
					Body:            []byte(`{"data":[]}`),
					ResultExpiresAt: time.Date(2026, time.July, 17, 10, 0, 0, 0, time.UTC),
				})
			},
			args: []driver.Value{
				service.OpenAIImageJobStatusCompleted, 200, "application/json", []byte(`{"data":[]}`),
				time.Date(2026, time.July, 17, 10, 0, 0, 0, time.UTC), "imgjob_terminal", "worker-owner",
				service.OpenAIImageJobStatusRunning,
			},
		},
		{
			name: "fail",
			run: func(repo service.OpenAIImageJobRepository) error {
				return repo.Fail(context.Background(), "imgjob_terminal", "worker-owner", "upstream_error", "failed", false)
			},
			args: []driver.Value{
				service.OpenAIImageJobStatusFailed, "upstream_error", "failed", false,
				"imgjob_terminal", "worker-owner", service.OpenAIImageJobStatusRunning,
			},
		},
		{
			name: "mark cancelled",
			run: func(repo service.OpenAIImageJobRepository) error {
				return repo.MarkCancelled(context.Background(), "imgjob_terminal", "worker-owner")
			},
			args: []driver.Value{
				service.OpenAIImageJobStatusCancelled, "imgjob_terminal", "worker-owner",
				service.OpenAIImageJobStatusRunning,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, mock, repo := newOpenAIImageJobRepositoryMock(t)
			mock.ExpectExec(`(?s)UPDATE openai_image_jobs.*WHERE job_id = \$[0-9]+\s+AND worker_id = \$[0-9]+\s+AND status = \$[0-9]+`).
				WithArgs(tt.args...).
				WillReturnResult(sqlmock.NewResult(0, 1))
			if err := tt.run(repo); err != nil {
				t.Fatalf("terminal write: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOpenAIImageJobRepositoryTerminalWriteReportsLeaseLoss(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	mock.ExpectExec(`(?s)UPDATE openai_image_jobs.*WHERE job_id`).
		WithArgs(
			service.OpenAIImageJobStatusFailed, "upstream_error", "failed", false,
			"imgjob_lost", "worker-old", service.OpenAIImageJobStatusRunning,
		).
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.Fail(context.Background(), "imgjob_lost", "worker-old", "upstream_error", "failed", false); !errors.Is(err, service.ErrOpenAIImageJobLeaseLost) {
		t.Fatalf("error = %v, want lease lost", err)
	}
}

func TestOpenAIImageJobRepositoryCompleteRequiresResultExpiry(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	err := repo.Complete(context.Background(), "imgjob_no_expiry", "worker-owner", service.OpenAIImageJobResponse{
		StatusCode:  200,
		ContentType: "application/json",
		Body:        []byte(`{"data":[]}`),
	})
	if !errors.Is(err, service.ErrOpenAIImageJobResultExpiryRequired) {
		t.Fatalf("error = %v, want result expiry required", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("zero expiry should not reach SQL: %v", err)
	}
}

func TestOpenAIImageJobRepositoryCancelForUserStateAndOwnership(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name             string
		jobID            string
		row              []driver.Value
		wantStatus       string
		wantRequest      bool
		wantResponse     bool
		wantCancellation bool
	}{
		{
			name:             "queued becomes cancelled and clears request",
			jobID:            "imgjob_queued",
			row:              openAIImageJobCancelRow("imgjob_queued", service.OpenAIImageJobStatusCancelled, true, now),
			wantStatus:       service.OpenAIImageJobStatusCancelled,
			wantRequest:      false,
			wantCancellation: true,
		},
		{
			name:             "running only records cancellation intent",
			jobID:            "imgjob_running",
			row:              openAIImageJobCancelRow("imgjob_running", service.OpenAIImageJobStatusRunning, true, now),
			wantStatus:       service.OpenAIImageJobStatusRunning,
			wantRequest:      false,
			wantCancellation: true,
		},
		{
			name:             "terminal completion remains authoritative",
			jobID:            "imgjob_completed",
			row:              openAIImageJobCancelRow("imgjob_completed", service.OpenAIImageJobStatusCompleted, false, now),
			wantStatus:       service.OpenAIImageJobStatusCompleted,
			wantRequest:      false,
			wantResponse:     true,
			wantCancellation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, mock, repo := newOpenAIImageJobRepositoryMock(t)
			mock.ExpectQuery(`(?s)UPDATE openai_image_jobs.*status IN \('queued', 'running'\).*status = CASE WHEN status = 'queued' THEN 'cancelled' ELSE status END.*WHERE user_id = \$1 AND job_id = \$2.*RETURNING.*NULL::bytea AS request_body.*CASE WHEN status = 'completed' THEN response_body ELSE NULL::bytea END AS response_body`).
				WithArgs(int64(12), tt.jobID).
				WillReturnRows(openAIImageJobRows().AddRow(tt.row...))

			job, err := repo.CancelForUser(context.Background(), 12, tt.jobID)
			if err != nil {
				t.Fatalf("cancel for user: %v", err)
			}
			if job.Status != tt.wantStatus ||
				(len(job.RequestBody) > 0) != tt.wantRequest ||
				(len(job.ResponseBody) > 0) != tt.wantResponse ||
				job.CancelRequested != tt.wantCancellation {
				t.Fatalf("unexpected cancellation result: %#v", job)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOpenAIImageJobRepositoryCleanupBatchLimitIsBounded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input int
		want  int
	}{
		{input: 0, want: service.DefaultOpenAIImageJobCleanupBatchLimit},
		{input: -1, want: service.DefaultOpenAIImageJobCleanupBatchLimit},
		{input: 25, want: 25},
		{input: service.MaxOpenAIImageJobCleanupBatchLimit + 1, want: service.MaxOpenAIImageJobCleanupBatchLimit},
	}
	for _, tt := range tests {
		if got := normalizeOpenAIImageJobCleanupBatchLimit(tt.input); got != tt.want {
			t.Fatalf("normalize cleanup limit %d = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestOpenAIImageJobRepositoryCancelForUserHidesOtherUsersJob(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	mock.ExpectQuery(`(?s)UPDATE openai_image_jobs.*WHERE user_id = \$1 AND job_id = \$2`).
		WithArgs(int64(99), "imgjob_owned_by_12").
		WillReturnRows(openAIImageJobRows())

	job, err := repo.CancelForUser(context.Background(), 99, "imgjob_owned_by_12")
	if job != nil || !errors.Is(err, service.ErrOpenAIImageJobNotFound) {
		t.Fatalf("job=%#v error=%v, want not found", job, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIImageJobRepositoryFailExpiredLeasesOnlyTouchesExpiredRows(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	cutoff := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	mock.ExpectExec(`(?s)UPDATE openai_image_jobs.*WHERE status = \$4\s+AND lease_expires_at IS NOT NULL\s+AND lease_expires_at <= \$5`).
		WithArgs(
			service.OpenAIImageJobStatusFailed,
			"failed_unknown",
			"image generation worker stopped after upstream dispatch; the request was not retried",
			service.OpenAIImageJobStatusRunning,
			cutoff,
		).
		WillReturnResult(sqlmock.NewResult(0, 2))

	affected, err := repo.FailExpiredLeases(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("fail expired leases: %v", err)
	}
	if affected != 2 {
		t.Fatalf("affected = %d, want 2", affected)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIImageJobRepositoryPurgeExpiredPayloadsUsesBoundedThreeStageTransaction(t *testing.T) {
	t.Parallel()

	_, mock, repo := newOpenAIImageJobRepositoryMock(t)
	now := time.Date(2026, time.July, 17, 10, 0, 0, 0, time.UTC)
	queuedCutoff := now.Add(-24 * time.Hour)
	recordCutoff := now.Add(-30 * 24 * time.Hour)
	const batchLimit = 25
	mock.ExpectBegin()
	mock.ExpectExec(`(?s)WITH expired_queued AS.*status = \$1.*created_at < \$2.*LIMIT \$3.*FOR UPDATE SKIP LOCKED.*UPDATE openai_image_jobs.*request_body = NULL`).
		WithArgs(
			service.OpenAIImageJobStatusQueued,
			queuedCutoff,
			batchLimit,
			service.OpenAIImageJobStatusFailed,
			"queue_expired",
			"image generation job expired before execution",
		).
		WillReturnResult(sqlmock.NewResult(0, 4))
	mock.ExpectExec(`(?s)WITH expired_payloads AS.*LIMIT \$3.*FOR UPDATE SKIP LOCKED.*UPDATE openai_image_jobs.*response_body = CASE.*result_expires_at <= \$1.*result_expires_at IS NULL.*finished_at < \$2`).
		WithArgs(now, recordCutoff, batchLimit).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(`(?s)WITH expired_records AS.*finished_at < \$1.*LIMIT \$2.*FOR UPDATE SKIP LOCKED.*DELETE FROM openai_image_jobs.*request_body IS NULL.*response_body IS NULL`).
		WithArgs(recordCutoff, batchLimit).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	result, err := repo.PurgeExpiredPayloads(context.Background(), service.OpenAIImageJobCleanupParams{
		Now:          now,
		QueuedCutoff: queuedCutoff,
		RecordCutoff: recordCutoff,
		BatchLimit:   batchLimit,
	})
	if err != nil {
		t.Fatalf("purge expired payloads: %v", err)
	}
	if result.Queued != 4 || result.Payloads != 3 || result.Records != 2 {
		t.Fatalf("cleanup result = %#v, want queued=4 payloads=3 records=2", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func newOpenAIImageJobRepositoryMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, service.OpenAIImageJobRepository) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock, NewOpenAIImageJobRepository(db)
}

func openAIImageJobCreateParams(jobID, requestHash string) service.CreateOpenAIImageJobParams {
	return service.CreateOpenAIImageJobParams{
		JobID:          jobID,
		UserID:         12,
		APIKeyID:       34,
		Endpoint:       service.OpenAIImageJobEndpointGenerations,
		Model:          "gpt-image-1",
		ContentType:    "application/json",
		RequestBody:    []byte(`{"model":"gpt-image-1","prompt":"hello"}`),
		RequestHash:    requestHash,
		IdempotencyKey: "local-task-id",
		ClientIP:       "203.0.113.4",
		UserAgent:      "image-playground-test",
	}
}

func openAIImageJobRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "job_id", "user_id", "api_key_id", "endpoint", "model", "content_type",
		"request_body", "request_hash", "idempotency_key_hash", "client_ip", "user_agent",
		"status", "response_status", "response_content_type", "response_body",
		"error_code", "error_message", "failure_unknown", "cancel_requested",
		"worker_id", "lease_expires_at", "attempt_count", "version", "result_expires_at",
		"created_at", "updated_at", "started_at", "finished_at",
	})
}

func openAIImageJobRowValues(jobID string, userID, apiKeyID int64, status, requestHash, idempotencyKey string, now time.Time) []driver.Value {
	return []driver.Value{
		int64(1), jobID, userID, apiKeyID, service.OpenAIImageJobEndpointGenerations, "gpt-image-1", "application/json",
		[]byte(`{"model":"gpt-image-1","prompt":"hello"}`), requestHash, idempotencyKey, "203.0.113.4", "image-playground-test",
		status, nil, nil, nil, nil, nil, false, false, nil, nil, 0, 0, nil,
		now, now, nil, nil,
	}
}

func openAIImageJobCancelRow(jobID, status string, cancelRequested bool, now time.Time) []driver.Value {
	row := openAIImageJobRowValues(jobID, 12, 34, status, "request-hash", service.HashIdempotencyKey(jobID), now)
	// CancelForUser uses the status projection and never returns request bytes,
	// including while the durable running row still retains them for execution.
	row[7] = nil
	row[19] = cancelRequested
	if status == service.OpenAIImageJobStatusCompleted {
		row[13] = int64(200)
		row[14] = "application/json"
		row[15] = []byte(`{"data":[]}`)
		row[24] = now.Add(72 * time.Hour)
	}
	if service.IsTerminalOpenAIImageJobStatus(status) {
		row[28] = now
	}
	return row
}
