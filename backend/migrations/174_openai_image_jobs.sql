CREATE TABLE IF NOT EXISTS openai_image_jobs (
    id BIGSERIAL PRIMARY KEY,
    job_id VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL,
    api_key_id BIGINT NOT NULL,
    endpoint VARCHAR(64) NOT NULL,
    model VARCHAR(128) NOT NULL DEFAULT '',
    content_type VARCHAR(512) NOT NULL,
    request_body BYTEA,
    request_hash CHAR(64) NOT NULL,
    idempotency_key_hash CHAR(64) NOT NULL,
    client_ip VARCHAR(64) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',

    status VARCHAR(16) NOT NULL DEFAULT 'queued',
    response_status INTEGER,
    response_content_type VARCHAR(255),
    response_body BYTEA,
    error_code VARCHAR(128),
    error_message TEXT,
    failure_unknown BOOLEAN NOT NULL DEFAULT FALSE,
    cancel_requested BOOLEAN NOT NULL DEFAULT FALSE,

    worker_id VARCHAR(128),
    lease_expires_at TIMESTAMPTZ,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 0,

    result_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,

    CONSTRAINT openai_image_jobs_endpoint_check
        CHECK (endpoint IN ('/v1/images/generations', '/v1/images/edits')),
    CONSTRAINT openai_image_jobs_status_check
        CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled')),
    CONSTRAINT openai_image_jobs_completed_expiry_check
        CHECK (status <> 'completed' OR result_expires_at IS NOT NULL),
    CONSTRAINT openai_image_jobs_attempt_count_check CHECK (attempt_count >= 0),
    CONSTRAINT openai_image_jobs_version_check CHECK (version >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS openai_image_jobs_job_id_uq
    ON openai_image_jobs (job_id);
CREATE UNIQUE INDEX IF NOT EXISTS openai_image_jobs_user_idempotency_uq
    ON openai_image_jobs (user_id, idempotency_key_hash);
CREATE INDEX IF NOT EXISTS openai_image_jobs_queue_claim_idx
    ON openai_image_jobs (status, created_at);
CREATE INDEX IF NOT EXISTS openai_image_jobs_running_lease_idx
    ON openai_image_jobs (lease_expires_at)
    WHERE status = 'running';
CREATE INDEX IF NOT EXISTS openai_image_jobs_result_expiry_idx
    ON openai_image_jobs (result_expires_at)
    WHERE response_body IS NOT NULL;
CREATE INDEX IF NOT EXISTS openai_image_jobs_finished_at_idx
    ON openai_image_jobs (finished_at)
    WHERE status IN ('completed', 'failed', 'cancelled');
