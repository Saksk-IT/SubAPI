package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"testing"
	"time"
)

func TestOpenAIImageJobTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{name: "queued starts running", from: OpenAIImageJobStatusQueued, to: OpenAIImageJobStatusRunning, want: true},
		{name: "queued can be cancelled", from: OpenAIImageJobStatusQueued, to: OpenAIImageJobStatusCancelled, want: true},
		{name: "queued validation failure", from: OpenAIImageJobStatusQueued, to: OpenAIImageJobStatusFailed, want: true},
		{name: "running completes", from: OpenAIImageJobStatusRunning, to: OpenAIImageJobStatusCompleted, want: true},
		{name: "running fails", from: OpenAIImageJobStatusRunning, to: OpenAIImageJobStatusFailed, want: true},
		{name: "running cancellation finishes", from: OpenAIImageJobStatusRunning, to: OpenAIImageJobStatusCancelled, want: true},
		{name: "queued cannot skip to complete", from: OpenAIImageJobStatusQueued, to: OpenAIImageJobStatusCompleted, want: false},
		{name: "running cannot return to queue", from: OpenAIImageJobStatusRunning, to: OpenAIImageJobStatusQueued, want: false},
		{name: "terminal is immutable", from: OpenAIImageJobStatusCompleted, to: OpenAIImageJobStatusFailed, want: false},
		{name: "unknown source", from: "unknown", to: OpenAIImageJobStatusRunning, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CanTransitionOpenAIImageJob(tt.from, tt.to); got != tt.want {
				t.Fatalf("CanTransitionOpenAIImageJob(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestOpenAIImageJobTerminalDetection(t *testing.T) {
	t.Parallel()

	for _, status := range []string{
		OpenAIImageJobStatusCompleted,
		OpenAIImageJobStatusFailed,
		OpenAIImageJobStatusCancelled,
	} {
		if !IsTerminalOpenAIImageJobStatus(status) {
			t.Fatalf("expected %q to be terminal", status)
		}
	}
	for _, status := range []string{OpenAIImageJobStatusQueued, OpenAIImageJobStatusRunning, ""} {
		if IsTerminalOpenAIImageJobStatus(status) {
			t.Fatalf("expected %q to be non-terminal", status)
		}
	}
}

func TestOpenAIImageJobRequestHash(t *testing.T) {
	t.Parallel()

	body := []byte(`{"model":"gpt-image-1","prompt":"hello"}`)
	want := HashOpenAIImageJobRequest(OpenAIImageJobEndpointGenerations, "application/json", body)
	if want == "" {
		t.Fatal("request hash must not be empty")
	}
	if got := HashOpenAIImageJobRequest(OpenAIImageJobEndpointGenerations, "application/json", append([]byte(nil), body...)); got != want {
		t.Fatalf("same request hash = %q, want %q", got, want)
	}

	changes := []string{
		HashOpenAIImageJobRequest(OpenAIImageJobEndpointEdits, "application/json", body),
		HashOpenAIImageJobRequest(OpenAIImageJobEndpointGenerations, "multipart/form-data; boundary=x", body),
		HashOpenAIImageJobRequest(OpenAIImageJobEndpointGenerations, "application/json", []byte(`{"model":"gpt-image-1","prompt":"changed"}`)),
	}
	for _, got := range changes {
		if got == want {
			t.Fatalf("different request unexpectedly reused hash %q", got)
		}
	}
}

func TestOpenAIImageJobRequestHashCanonicalizesMultipartBoundary(t *testing.T) {
	t.Parallel()

	contentTypeA, bodyA := makeOpenAIImageJobMultipart(t, "boundary-a", []byte("image bytes"))
	contentTypeB, bodyB := makeOpenAIImageJobMultipart(t, "boundary-b", []byte("image bytes"))
	hashA := HashOpenAIImageJobRequest(OpenAIImageJobEndpointEdits, contentTypeA, bodyA)
	hashB := HashOpenAIImageJobRequest(OpenAIImageJobEndpointEdits, contentTypeB, bodyB)
	if hashA != hashB {
		t.Fatalf("equivalent multipart requests with different boundaries differ: %q != %q", hashA, hashB)
	}

	contentTypeChanged, bodyChanged := makeOpenAIImageJobMultipart(t, "boundary-c", []byte("changed image bytes"))
	if changed := HashOpenAIImageJobRequest(OpenAIImageJobEndpointEdits, contentTypeChanged, bodyChanged); changed == hashA {
		t.Fatalf("changed multipart file unexpectedly reused hash %q", changed)
	}
}

func TestOpenAIImageJobRequestHashCanonicalizesJSON(t *testing.T) {
	t.Parallel()

	first := []byte("{\n  \"prompt\": \"hello\", \"model\": \"gpt-image-1\"\n}")
	second := []byte(`{"model":"gpt-image-1","prompt":"hello"}`)
	firstHash := HashOpenAIImageJobRequest(OpenAIImageJobEndpointGenerations, "application/json", first)
	secondHash := HashOpenAIImageJobRequest(OpenAIImageJobEndpointGenerations, "application/json; charset=utf-8", second)
	if firstHash != secondHash {
		t.Fatalf("semantically equal JSON requests differ: %q != %q", firstHash, secondHash)
	}
	changed := HashOpenAIImageJobRequest(
		OpenAIImageJobEndpointGenerations,
		"application/json",
		[]byte(`{"model":"gpt-image-1","prompt":"changed"}`),
	)
	if changed == firstHash {
		t.Fatalf("changed JSON unexpectedly reused hash %q", changed)
	}
}

func TestOpenAIImageJobIdempotencyConflict(t *testing.T) {
	t.Parallel()

	job := &OpenAIImageJob{RequestHash: "same-hash"}
	if err := ValidateOpenAIImageJobIdempotency(job, "same-hash"); err != nil {
		t.Fatalf("same request should be idempotent: %v", err)
	}
	if err := ValidateOpenAIImageJobIdempotency(job, "different-hash"); !errors.Is(err, ErrOpenAIImageJobIdempotencyConflict) {
		t.Fatalf("different request error = %v, want %v", err, ErrOpenAIImageJobIdempotencyConflict)
	}
}

func TestOpenAIImageJobPublicSerialization(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	job := &OpenAIImageJob{
		ID:                 99,
		JobID:              "imgjob_public",
		UserID:             123,
		APIKeyID:           456,
		Endpoint:           OpenAIImageJobEndpointGenerations,
		ContentType:        "application/json",
		RequestBody:        []byte("secret request"),
		RequestHash:        "request-hash",
		IdempotencyKeyHash: HashIdempotencyKey("private-idempotency-key"),
		Status:             OpenAIImageJobStatusCompleted,
		ResponseBody:       []byte(`{"created":1,"data":[{"b64_json":"image"}]}`),
		ResponseStatus:     200,
		WorkerID:           openAIImageJobStringPtr("worker-secret"),
		CreatedAt:          now,
		StartedAt:          openAIImageJobTimePtr(now.Add(time.Second)),
		FinishedAt:         openAIImageJobTimePtr(now.Add(2 * time.Second)),
		CancelRequested:    true,
	}

	encoded, err := json.Marshal(OpenAIImageJobToPublic(job))
	if err != nil {
		t.Fatalf("marshal public job: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("decode public job: %v", err)
	}
	if got["id"] != job.JobID || got["status"] != OpenAIImageJobStatusCompleted {
		t.Fatalf("unexpected public identity: %s", encoded)
	}
	result, ok := got["result"].(map[string]any)
	if !ok || result["created"] != float64(1) {
		t.Fatalf("original OpenAI response missing under result: %s", encoded)
	}
	for _, forbidden := range []string{"user_id", "api_key_id", "request_body", "request_hash", "idempotency_key", "idempotency_key_hash", "worker_id", "response_body"} {
		if _, exists := got[forbidden]; exists {
			t.Fatalf("public job leaks %q: %s", forbidden, encoded)
		}
	}
}

func TestOpenAIImageJobPublicSerializationIncludesTerminalError(t *testing.T) {
	t.Parallel()

	code := "failed_unknown"
	message := "worker lease expired"
	public := OpenAIImageJobToPublic(&OpenAIImageJob{
		JobID:        "imgjob_failed",
		Status:       OpenAIImageJobStatusFailed,
		ErrorCode:    &code,
		ErrorMessage: &message,
		RequestBody:  []byte("must not leak"),
	})
	encoded, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("marshal failed job: %v", err)
	}
	if string(encoded) == "" || public.Error == nil || public.Error.Code != code || public.Error.Message != message {
		t.Fatalf("unexpected public failure: %s", encoded)
	}
	if public.Result != nil {
		t.Fatalf("failed job exposed a result: %s", encoded)
	}
}

func TestOpenAIImageJobPublicSerializationIncludesCancelledError(t *testing.T) {
	t.Parallel()

	public := OpenAIImageJobToPublic(&OpenAIImageJob{
		JobID:  "imgjob_cancelled",
		Status: OpenAIImageJobStatusCancelled,
	})
	if public == nil || public.Error == nil {
		t.Fatalf("cancelled job must expose a public error: %#v", public)
	}
	if public.Error.Code != "cancelled" || public.Error.Message == "" {
		t.Fatalf("unexpected cancelled error: %#v", public.Error)
	}
	if public.Result != nil {
		t.Fatalf("cancelled job exposed a result: %#v", public)
	}
}

func TestOpenAIImageJobQueuedCancellation(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	job := &OpenAIImageJob{Status: OpenAIImageJobStatusQueued, RequestBody: []byte("payload")}
	if err := job.RequestCancellation(now); err != nil {
		t.Fatalf("cancel queued job: %v", err)
	}
	if job.Status != OpenAIImageJobStatusCancelled || !job.CancelRequested {
		t.Fatalf("queued cancellation = status %q requested %v", job.Status, job.CancelRequested)
	}
	if job.FinishedAt == nil || len(job.RequestBody) != 0 {
		t.Fatalf("queued cancellation did not finish and clear payload: %#v", job)
	}
}

func TestOpenAIImageJobRunningCancellation(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	job := &OpenAIImageJob{Status: OpenAIImageJobStatusRunning, RequestBody: []byte("payload")}
	if err := job.RequestCancellation(now); err != nil {
		t.Fatalf("request running cancellation: %v", err)
	}
	if job.Status != OpenAIImageJobStatusRunning || !job.CancelRequested {
		t.Fatalf("running cancellation = status %q requested %v", job.Status, job.CancelRequested)
	}
	if job.FinishedAt != nil || len(job.RequestBody) == 0 {
		t.Fatalf("running request must stay executable until the worker resolves it: %#v", job)
	}
}

func TestOpenAIImageJobCompletionWinsCancelRace(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	job := &OpenAIImageJob{Status: OpenAIImageJobStatusRunning, RequestBody: []byte("payload")}
	if err := job.RequestCancellation(now); err != nil {
		t.Fatal(err)
	}
	response := OpenAIImageJobResponse{
		StatusCode:      200,
		ContentType:     "application/json",
		Body:            []byte(`{"data":[{"url":"https://example.test/image.png"}]}`),
		ResultExpiresAt: now.Add(72 * time.Hour),
	}
	if err := job.MarkCompleted(response, now.Add(time.Second)); err != nil {
		t.Fatalf("complete cancellation race: %v", err)
	}
	if job.Status != OpenAIImageJobStatusCompleted || string(job.ResponseBody) != string(response.Body) {
		t.Fatalf("successful result did not win cancellation race: %#v", job)
	}
	if len(job.RequestBody) != 0 || job.FinishedAt == nil {
		t.Fatalf("completion must clear request and finish: %#v", job)
	}
}

func TestOpenAIImageJobCompletionRequiresResultExpiry(t *testing.T) {
	t.Parallel()

	job := &OpenAIImageJob{Status: OpenAIImageJobStatusRunning, RequestBody: []byte("payload")}
	err := job.MarkCompleted(OpenAIImageJobResponse{
		StatusCode:  200,
		ContentType: "application/json",
		Body:        []byte(`{"data":[]}`),
	}, time.Now().UTC())
	if !errors.Is(err, ErrOpenAIImageJobResultExpiryRequired) {
		t.Fatalf("error = %v, want result expiry required", err)
	}
	if job.Status != OpenAIImageJobStatusRunning || len(job.RequestBody) == 0 {
		t.Fatalf("invalid completion mutated job: %#v", job)
	}
}

func openAIImageJobStringPtr(value string) *string { return &value }

func openAIImageJobTimePtr(value time.Time) *time.Time { return &value }

func makeOpenAIImageJobMultipart(t *testing.T, boundary string, image []byte) (string, []byte) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.SetBoundary(boundary); err != nil {
		t.Fatalf("set multipart boundary: %v", err)
	}
	if err := writer.WriteField("model", "gpt-image-1"); err != nil {
		t.Fatalf("write model: %v", err)
	}
	file, err := writer.CreateFormFile("image", "input.png")
	if err != nil {
		t.Fatalf("create image part: %v", err)
	}
	if _, err := file.Write(image); err != nil {
		t.Fatalf("write image: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}
	return writer.FormDataContentType(), body.Bytes()
}
