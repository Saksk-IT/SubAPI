# OpenAI Image Async Jobs Implementation Plan

> **For Codex:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task, with superpowers:test-driven-development and superpowers:verification-before-completion.

**Goal:** Replace the managed image playground's long-lived OpenAI Images request with a durable `submit -> job id -> worker -> poll -> result` flow that survives browser closure and server restarts without depending on Cloudflare's request timeout.

**Architecture:** Add a PostgreSQL-backed `openai_image_jobs` state machine and a bounded polling worker. Job submission authenticates and validates the original OpenAI request, persists the complete request plus a user-scoped idempotency key, and returns `202` immediately. The worker claims one queued row with `FOR UPDATE SKIP LOCKED`, reconstructs a fresh internal request context, executes the existing OpenAI Images routing/moderation/concurrency/billing pipeline, persists the final response, and never blindly retries an ambiguously dispatched upstream request. The managed React app uses its existing custom async-provider engine for submission, polling, IndexedDB recovery, and history; the Responses profile remains synchronous.

**Tech Stack:** Go, Gin, PostgreSQL, Wire, React, TypeScript, Zustand, IndexedDB, Vitest.

---

## Contract and invariants

- `POST /v1/images/generations/jobs` and `POST /v1/images/edits/jobs` accept the same body and content type as the corresponding OpenAI endpoint.
- Submission requires `Idempotency-Key`; the managed app uses its local `TaskRecord.id`.
- Idempotency is unique by `(user_id, idempotency_key)`. Repeating the same request returns the same job; reusing the key with another request returns `409`.
- `GET /v1/images/jobs/:id` and `POST /v1/images/jobs/:id/cancel` are authorized by `user_id`, not the selected API-key ID. The creation key ID remains immutable for execution, billing, and audit.
- Public states are `queued`, `running`, `completed`, `failed`, and `cancelled`.
- A completed status response contains the original OpenAI JSON beneath `result`; terminal errors contain `{code,message}` beneath `error`.
- Queued cancellation is final and never calls the upstream. Running cancellation is best effort; if the upstream still returns a billable image, completion wins and the result is retained.
- A worker crash after dispatch is reported as `failed_unknown` after the lease expires and is never automatically resubmitted. This avoids duplicate images and duplicate upstream cost.
- The job ID is injected as the client request ID during execution so the existing usage-billing uniqueness rules remain stable.
- Terminal request payloads are cleared promptly; completed result payloads expire after the configured retention window; metadata is retained longer for diagnosis.
- No raw API key, bearer token, or upstream access token is stored in the job table.
- Direct page refresh still requires reopening from the Sub2API sidebar and choosing a key because the secure popup handoff intentionally destroys the opener capability. The server job itself continues, and reopening under any key owned by the same user resumes it from IndexedDB.

### Task 1: Persisted job model, state machine, and repository

**Files:**

- Create: `backend/migrations/174_openai_image_jobs.sql`
- Create: `backend/internal/service/openai_image_job.go`
- Create: `backend/internal/service/openai_image_job_test.go`
- Create: `backend/internal/repository/openai_image_job_repo.go`
- Create: `backend/internal/repository/openai_image_job_repo_test.go`
- Modify: `backend/internal/repository/wire.go`

**Step 1: Write failing domain tests**

Cover valid/invalid transitions, terminal detection, request hashing, idempotency conflict detection, public serialization, queued/running cancellation, and the rule that completion wins a running cancel race.

Run:

```bash
cd backend && go test ./internal/service -run 'TestOpenAIImageJob' -count=1
```

Expected: FAIL because the job domain does not exist.

**Step 2: Add the migration and domain types**

Create a table with immutable owner/key/endpoint/request fields, request and response `BYTEA` payloads, status/error/cancel fields, worker lease fields, result expiry, version, and timestamps. Add unique indexes for `job_id` and `(user_id,idempotency_key)`, plus a queue-claim index on `(status,created_at)`.

Define repository operations for:

```go
CreateOrGet(ctx, params) (*OpenAIImageJob, bool, error)
GetForUser(ctx, userID, jobID) (*OpenAIImageJob, error)
ClaimNext(ctx, workerID, leaseUntil) (*OpenAIImageJob, error)
Heartbeat(ctx, jobID, workerID, leaseUntil) error
Complete(ctx, jobID, workerID, response) error
Fail(ctx, jobID, workerID, code, message, unknown bool) error
CancelForUser(ctx, userID, jobID) (*OpenAIImageJob, error)
MarkCancelled(ctx, jobID, workerID) error
FailExpiredLeases(ctx, cutoff) (int64, error)
PurgeExpiredPayloads(ctx, now, recordCutoff) (payloads int64, records int64, err error)
```

**Step 3: Write failing repository tests**

Use `sqlmock` to prove the claim query uses `FOR UPDATE SKIP LOCKED`, ownership is user-scoped, creation translates the unique conflict into a read-and-hash comparison, and terminal writes are guarded by job ID, worker ID, and `running` status.

Run:

```bash
cd backend && go test ./internal/repository -run 'TestOpenAIImageJobRepository' -count=1
```

Expected: FAIL until repository behavior is implemented.

**Step 4: Implement the repository and wire provider**

Keep SQL transitions atomic. Never select arbitrary payload paths. Ensure a duplicate idempotency key with a different request hash returns the domain conflict error.

**Step 5: Verify Task 1**

```bash
cd backend && go test ./internal/service ./internal/repository -run 'OpenAIImageJob' -count=1
```

Expected: PASS.

### Task 2: Submission API, worker runtime, cancellation, and cleanup

**Files:**

- Create: `backend/internal/handler/openai_image_job_handler.go`
- Create: `backend/internal/handler/openai_image_job_handler_test.go`
- Create: `backend/internal/handler/openai_image_job_executor.go`
- Create: `backend/internal/handler/openai_image_job_executor_test.go`
- Create: `backend/internal/service/openai_image_job_worker.go`
- Create: `backend/internal/service/openai_image_job_worker_test.go`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/internal/handler/handler.go`
- Modify: `backend/internal/handler/wire.go`
- Modify: `backend/internal/server/routes/gateway.go`
- Modify: `backend/cmd/server/wire.go`
- Regenerate: `backend/cmd/server/wire_gen.go`

**Step 1: Write failing handler contract tests**

Test both JSON generation and multipart edit submission, required/validated `Idempotency-Key`, `stream=false`, immediate `202`, replay, hash conflict, user-scoped status, completed result shape, failed result shape, queued cancellation, running cancellation request, and cross-user denial.

Run:

```bash
cd backend && go test ./internal/handler -run 'TestOpenAIImageJobHandler' -count=1
```

Expected: FAIL because the routes and handler do not exist.

**Step 2: Implement fast submission and polling handlers**

Read the already body-limited request, parse it with the existing OpenAI Images parser for early validation, reject streaming jobs, hash endpoint/content type/body, and persist without calling account selection or the upstream. Return `Retry-After: 1` while non-terminal.

**Step 3: Write failing worker tests**

Use fake repository/executor implementations to prove:

- returning `202`/cancelling the submission context does not stop execution;
- only one worker owns a claimed job;
- heartbeats continue while execution runs;
- queued cancellation never invokes the executor;
- a running cancellation invokes best-effort cancellation;
- success wins a cancellation race;
- shutdown/lease loss becomes `failed_unknown`, not a requeue;
- stale running rows become `failed_unknown`;
- cleanup purges payloads and old metadata.

Run:

```bash
cd backend && go test ./internal/service -run 'TestOpenAIImageJobWorker' -count=1
```

Expected: FAIL until the runtime exists.

**Step 4: Implement the bounded worker runtime**

Start the configured number of polling workers from one runtime object. Give each process/worker a unique owner ID, a lease heartbeat, an execution timeout, a map of running cancellation functions, and a cleanup/recovery loop. On shutdown, cancel active executions and wait with the existing server cleanup mechanism.

**Step 5: Implement the fresh-context executor**

Load the creation API key by ID, re-authenticate it through the existing auth path, and build a brand-new Gin request/context from the persisted endpoint, content type, body, client address, and safe headers. Inject `ClientRequestID=job_id` and `RequestID=job_id/attempt`, then invoke the existing Images pipeline and capture its non-stream response. Never retain or use the original request's `*gin.Context` across goroutines.

Tests must prove key/user/group disablement is rechecked and the stable job ID reaches usage recording. The adapter is deliberately internal so the synchronous API remains unchanged.

**Step 6: Add routes, config defaults, Wire providers, and cleanup**

Defaults: enabled, 2 workers, 1-second poll, 10-second heartbeat, 2-minute stale lease, 15-minute execution timeout, 72-hour result retention, 30-day metadata retention, hourly cleanup. Register the five job routes under the existing `/v1` authenticated group.

Regenerate Wire:

```bash
cd backend && go generate ./cmd/server
```

**Step 7: Verify Task 2**

```bash
cd backend && go test ./internal/handler ./internal/service ./internal/repository -run 'OpenAIImageJob' -count=1
```

Expected: PASS.

### Task 3: Managed React submission, polling, idempotency, and recovery

**Files:**

- Modify: `frontend/image-playground/src/types.ts`
- Modify: `frontend/image-playground/src/lib/imageApiShared.ts`
- Modify: `frontend/image-playground/src/lib/openaiCompatibleImageApi.ts`
- Modify: `frontend/image-playground/src/lib/managedMode.ts`
- Modify: `frontend/image-playground/src/lib/managedMode.test.ts`
- Modify: `frontend/image-playground/src/lib/api.test.ts`
- Modify: `frontend/image-playground/src/store.ts`
- Modify: `frontend/image-playground/src/store.test.ts`

**Step 1: Write failing managed-profile tests**

Assert that managed Images uses the internal `sub2api-async` custom provider with generation/edit job paths, one-second polling, completed/failed/cancelled mapping, and OpenAI result extraction; managed Responses remains the built-in OpenAI provider. Assert the fetch guard permits only exact same-origin job routes/methods and rejects query strings, fragments, traversal, malformed IDs, and cross-origin URLs.

Run:

```bash
npm --prefix frontend/image-playground test -- --run src/lib/managedMode.test.ts src/lib/api.test.ts
```

Expected: FAIL until managed async mapping is installed.

**Step 2: Implement the managed async provider**

Map the existing task parameters to the original OpenAI generation/edit body. On submit, send `Idempotency-Key` from the local task ID. Poll `/v1/images/jobs/{task_id}` and extract `result.data.*.b64_json` or `result.data.*.url`.

**Step 3: Write failing persistence/recovery tests**

Cover:

- a task is persisted before submission;
- the local task ID is passed as the idempotency key;
- the server job ID write is awaited before polling;
- a managed async task without a job ID remains submission-pending after restart and safely resubmits with the same key;
- a task with a job ID resumes polling after reopening;
- another selected key under the same user can recover the task;
- completion uses the existing image/history pipeline;
- failed/cancelled status maps to a terminal local error;
- Agent image tasks use the same async path and continue after recovery.

Run:

```bash
npm --prefix frontend/image-playground test -- --run src/store.test.ts
```

Expected: FAIL until pending submission and awaited job-ID persistence are implemented.

**Step 4: Implement pending submission and recovery**

Add a persisted `customSubmissionPending` marker. Pass `TaskRecord.id` into every image API call. Allow `onCustomTaskEnqueued` to return a promise and await it. On startup, resume tasks with a job ID via polling; safely rerun submission-pending tasks without a job ID. Never persist the API key.

**Step 5: Verify Task 3**

```bash
npm --prefix frontend/image-playground test -- --run src/lib/managedMode.test.ts src/lib/api.test.ts src/store.test.ts
```

Expected: PASS.

### Task 4: User-visible cancellation and task lifecycle polish

**Files:**

- Create: `frontend/image-playground/src/lib/sub2apiImageJobApi.ts`
- Create: `frontend/image-playground/src/lib/sub2apiImageJobApi.test.ts`
- Modify: `frontend/image-playground/src/store.ts`
- Modify: `frontend/image-playground/src/store.test.ts`
- Modify: `frontend/image-playground/src/components/TaskCard.tsx`
- Modify: `frontend/image-playground/src/components/DetailModal.tsx`
- Modify: `frontend/image-playground/src/components/TaskGrid.tsx`

**Step 1: Write failing cancellation-client and store tests**

Prove legal same-origin cancel calls, idempotent terminal responses, path validation, cancel/completion races, Agent stop propagation, and delete-running-task confirmation semantics.

Run:

```bash
npm --prefix frontend/image-playground test -- --run src/lib/sub2apiImageJobApi.test.ts src/store.test.ts
```

Expected: FAIL until cancellation is connected.

**Step 2: Implement cancellation UX**

Expose Cancel for queued/running managed jobs in the card and detail views. Keep the task running while cancellation is being requested. If the server reports `cancelled`, persist `status=error` with `已取消生成。`; if it reports `completed`, fetch/store the result. Deleting a running task asks whether to cancel the server job first. Closing the tab never cancels automatically.

**Step 3: Verify Task 4**

```bash
npm --prefix frontend/image-playground test -- --run src/lib/sub2apiImageJobApi.test.ts src/store.test.ts
```

Expected: PASS.

### Task 5: End-to-end timeout proof, full verification, and review

**Files:**

- Create or modify focused integration tests under `backend/internal/handler/` and `frontend/image-playground/src/` only as needed.
- Modify: `docs/superpowers/plans/2026-07-14-openai-image-async-jobs.md` only if the implemented contract intentionally changes.

**Step 1: Add the slow-upstream proof**

Use a controllable fake executor/upstream: submission must return `202` before the fake work is released; polling must show queued/running; releasing it must produce completed plus the original image response. Add a restart/lease test showing an uncertain running job is not re-executed.

**Step 2: Run focused race and contract tests**

```bash
cd backend && go test -race ./internal/service ./internal/handler ./internal/repository -run 'OpenAIImageJob' -count=1
npm --prefix frontend/image-playground test -- --run src/lib/managedMode.test.ts src/lib/api.test.ts src/lib/sub2apiImageJobApi.test.ts src/store.test.ts
```

Expected: PASS.

**Step 3: Run full repository verification**

```bash
cd backend && go test ./...
pnpm --dir frontend run typecheck
make test-frontend-critical
npm --prefix frontend/image-playground test
```

Expected: all commands PASS with no new warnings attributable to this change.

**Step 4: Perform two-stage code review**

First review implementation against this plan and API invariants; then review code quality, security boundaries, cancellation races, SQL ownership guards, worker shutdown, and secret handling. Fix findings and rerun the affected tests before claiming completion.
