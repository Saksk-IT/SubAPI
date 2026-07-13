package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenAIImageJobWorkerRunOnceClaimsAndCompletesOneJob(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_one", Status: OpenAIImageJobStatusRunning})
	executor := openAIImageJobExecutorFunc(func(_ context.Context, _ *OpenAIImageJob, _ OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		return successfulOpenAIImageJobExecution()
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, now)

	processed, err := worker.RunOnce(context.Background(), "worker-1")
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if !processed {
		t.Fatal("RunOnce() processed = false, want true")
	}

	snapshot := repo.snapshot()
	if snapshot.claimCalls != 1 || snapshot.completeCalls != 1 {
		t.Fatalf("claim/complete calls = %d/%d, want 1/1", snapshot.claimCalls, snapshot.completeCalls)
	}
	if snapshot.failCalls != 0 || snapshot.cancelCalls != 0 {
		t.Fatalf("unexpected fail/cancel calls = %d/%d", snapshot.failCalls, snapshot.cancelCalls)
	}
	wantExpiry := now.Add(worker.opts.ResultRetention)
	if !snapshot.completedResponse.ResultExpiresAt.Equal(wantExpiry) {
		t.Fatalf("result expiry = %v, want %v", snapshot.completedResponse.ResultExpiresAt, wantExpiry)
	}
	if got := snapshot.lastLeaseUntil; !got.Equal(now.Add(worker.opts.LeaseDuration)) {
		t.Fatalf("claim lease = %v, want %v", got, now.Add(worker.opts.LeaseDuration))
	}
}

func TestOpenAIImageJobWorkerInitialHeartbeatCancellationSkipsExecutor(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_cancelled", Status: OpenAIImageJobStatusRunning})
	repo.heartbeatResults = []openAIImageJobHeartbeatResult{{cancelRequested: true}}
	var executions atomic.Int32
	executor := openAIImageJobExecutorFunc(func(_ context.Context, _ *OpenAIImageJob, _ OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		executions.Add(1)
		return successfulOpenAIImageJobExecution()
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())

	processed, err := worker.RunOnce(context.Background(), "worker-1")
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if !processed {
		t.Fatal("RunOnce() processed = false, want true")
	}
	if executions.Load() != 0 {
		t.Fatalf("executor calls = %d, want 0", executions.Load())
	}
	if got := repo.snapshot().cancelCalls; got != 1 {
		t.Fatalf("MarkCancelled calls = %d, want 1", got)
	}
}

func TestOpenAIImageJobWorkerHeartbeatsWhileExecutionRuns(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_heartbeat", Status: OpenAIImageJobStatusRunning})
	release := make(chan struct{})
	executor := openAIImageJobExecutorFunc(func(_ context.Context, _ *OpenAIImageJob, _ OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		<-release
		return successfulOpenAIImageJobExecution()
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())
	worker.opts.HeartbeatInterval = 2 * time.Millisecond

	done := make(chan error, 1)
	go func() {
		_, err := worker.RunOnce(context.Background(), "worker-1")
		done <- err
	}()
	waitForOpenAIImageJobWorkerCondition(t, 250*time.Millisecond, func() bool {
		return repo.snapshot().heartbeatCalls >= 3
	})
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if got := repo.snapshot().completeCalls; got != 1 {
		t.Fatalf("Complete calls = %d, want 1", got)
	}
}

func TestOpenAIImageJobWorkerCrossInstanceCancellationCancelsExecutionBeforeDispatch(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_cross_cancel", Status: OpenAIImageJobStatusRunning})
	repo.heartbeatResults = []openAIImageJobHeartbeatResult{{}, {}, {cancelRequested: true}}
	executorStarted := make(chan struct{})
	executor := openAIImageJobExecutorFunc(func(ctx context.Context, _ *OpenAIImageJob, _ OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		close(executorStarted)
		<-ctx.Done()
		return OpenAIImageJobExecutionResult{Outcome: OpenAIImageJobExecutionInterrupted}
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())
	worker.opts.HeartbeatInterval = 2 * time.Millisecond

	processed, err := worker.RunOnce(context.Background(), "worker-1")
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if !processed {
		t.Fatal("RunOnce() processed = false, want true")
	}
	select {
	case <-executorStarted:
	default:
		t.Fatal("executor did not start")
	}
	snapshot := repo.snapshot()
	if snapshot.cancelCalls != 1 || snapshot.failCalls != 0 || snapshot.completeCalls != 0 {
		t.Fatalf("cancel/fail/complete calls = %d/%d/%d, want 1/0/0", snapshot.cancelCalls, snapshot.failCalls, snapshot.completeCalls)
	}
}

func TestOpenAIImageJobWorkerRequestCancelImmediatelyCancelsLocalExecutionAndUnregisters(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_local_cancel", Status: OpenAIImageJobStatusRunning})
	executorStarted := make(chan struct{})
	executor := openAIImageJobExecutorFunc(func(ctx context.Context, _ *OpenAIImageJob, _ OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		close(executorStarted)
		<-ctx.Done()
		return OpenAIImageJobExecutionResult{Outcome: OpenAIImageJobExecutionInterrupted}
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())
	worker.opts.HeartbeatInterval = time.Hour // Prove this does not wait for a database heartbeat.

	done := make(chan error, 1)
	go func() {
		_, err := worker.RunOnce(context.Background(), "worker-1")
		done <- err
	}()
	select {
	case <-executorStarted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("executor did not start")
	}
	if !worker.RequestCancel("imgjob_local_cancel") {
		t.Fatal("RequestCancel() = false for a running local job")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunOnce() error = %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("local cancellation did not promptly stop execution")
	}

	snapshot := repo.snapshot()
	if snapshot.cancelCalls != 1 || snapshot.failCalls != 0 || snapshot.completeCalls != 0 {
		t.Fatalf("cancel/fail/complete calls = %d/%d/%d, want 1/0/0", snapshot.cancelCalls, snapshot.failCalls, snapshot.completeCalls)
	}
	if worker.RequestCancel("imgjob_local_cancel") {
		t.Fatal("RequestCancel() = true after execution was unregistered")
	}
	if worker.RequestCancel("imgjob_missing") {
		t.Fatal("RequestCancel() = true for an unknown job")
	}
}

func TestOpenAIImageJobWorkerRequestCancelDuringPreflightNeverStartsExecutor(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_preflight_cancel", Status: OpenAIImageJobStatusRunning})
	preflightEntered := make(chan struct{})
	releasePreflight := make(chan struct{})
	repo.heartbeatHook = func(call int) {
		if call == 2 {
			close(preflightEntered)
			<-releasePreflight
		}
	}
	var executions atomic.Int32
	executor := openAIImageJobExecutorFunc(func(_ context.Context, _ *OpenAIImageJob, _ OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		executions.Add(1)
		return OpenAIImageJobExecutionResult{Outcome: OpenAIImageJobExecutionInterrupted}
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())

	done := make(chan error, 1)
	go func() {
		_, err := worker.RunOnce(context.Background(), "worker-1")
		done <- err
	}()
	select {
	case <-preflightEntered:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("worker did not enter registered preflight heartbeat")
	}
	if !worker.RequestCancel("imgjob_preflight_cancel") {
		t.Fatal("RequestCancel() = false during registered preflight")
	}
	close(releasePreflight)
	if err := <-done; err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if executions.Load() != 0 {
		t.Fatalf("executor calls = %d, want 0", executions.Load())
	}
	if got := repo.snapshot().cancelCalls; got != 1 {
		t.Fatalf("MarkCancelled calls = %d, want 1", got)
	}
}

func TestOpenAIImageJobWorkerLocalCancellationStillAllowsVerifiedSuccessToWin(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_local_success", Status: OpenAIImageJobStatusRunning})
	executorStarted := make(chan struct{})
	executor := openAIImageJobExecutorFunc(func(ctx context.Context, _ *OpenAIImageJob, observer OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		observer.MarkDispatched()
		close(executorStarted)
		<-ctx.Done()
		return successfulOpenAIImageJobExecution()
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())
	worker.opts.HeartbeatInterval = time.Hour

	done := make(chan error, 1)
	go func() {
		_, err := worker.RunOnce(context.Background(), "worker-1")
		done <- err
	}()
	select {
	case <-executorStarted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("executor did not start")
	}
	if !worker.RequestCancel("imgjob_local_success") {
		t.Fatal("RequestCancel() = false for a running local job")
	}
	if err := <-done; err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	snapshot := repo.snapshot()
	if snapshot.completeCalls != 1 || snapshot.failCalls != 0 || snapshot.cancelCalls != 0 {
		t.Fatalf("complete/fail/cancel calls = %d/%d/%d, want 1/0/0", snapshot.completeCalls, snapshot.failCalls, snapshot.cancelCalls)
	}
}

func TestOpenAIImageJobWorkerConcurrentWorkersExecuteClaimedJobOnlyOnce(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_compete", Status: OpenAIImageJobStatusRunning})
	var executions atomic.Int32
	executor := openAIImageJobExecutorFunc(func(_ context.Context, _ *OpenAIImageJob, _ OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		executions.Add(1)
		return successfulOpenAIImageJobExecution()
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())

	start := make(chan struct{})
	results := make(chan bool, 2)
	errorsCh := make(chan error, 2)
	for _, workerID := range []string{"worker-1", "worker-2"} {
		go func(workerID string) {
			<-start
			processed, err := worker.RunOnce(context.Background(), workerID)
			results <- processed
			errorsCh <- err
		}(workerID)
	}
	close(start)
	processedCount := 0
	for range 2 {
		if <-results {
			processedCount++
		}
		if err := <-errorsCh; err != nil {
			t.Fatalf("RunOnce() error = %v", err)
		}
	}

	if processedCount != 1 {
		t.Fatalf("processed workers = %d, want 1", processedCount)
	}
	if executions.Load() != 1 {
		t.Fatalf("executor calls = %d, want 1", executions.Load())
	}
	snapshot := repo.snapshot()
	if snapshot.claimCalls != 2 || snapshot.completeCalls != 1 {
		t.Fatalf("claim/complete calls = %d/%d, want 2/1", snapshot.claimCalls, snapshot.completeCalls)
	}
}

func TestOpenAIImageJobWorkerSuccessWinsCancellationRace(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_success_wins", Status: OpenAIImageJobStatusRunning})
	repo.heartbeatResults = []openAIImageJobHeartbeatResult{{}, {}, {cancelRequested: true}}
	executor := openAIImageJobExecutorFunc(func(ctx context.Context, _ *OpenAIImageJob, observer OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		observer.MarkDispatched()
		<-ctx.Done()
		return successfulOpenAIImageJobExecution()
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())
	worker.opts.HeartbeatInterval = 2 * time.Millisecond

	if _, err := worker.RunOnce(context.Background(), "worker-1"); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	snapshot := repo.snapshot()
	if snapshot.completeCalls != 1 || snapshot.failCalls != 0 || snapshot.cancelCalls != 0 {
		t.Fatalf("complete/fail/cancel calls = %d/%d/%d, want 1/0/0", snapshot.completeCalls, snapshot.failCalls, snapshot.cancelCalls)
	}
}

func TestOpenAIImageJobWorkerPostDispatchCancellationWithoutResultFailsUnknown(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_cancel_unknown", Status: OpenAIImageJobStatusRunning})
	repo.heartbeatResults = []openAIImageJobHeartbeatResult{{}, {}, {cancelRequested: true}}
	release := make(chan struct{})
	executor := openAIImageJobExecutorFunc(func(_ context.Context, _ *OpenAIImageJob, observer OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		observer.MarkDispatched()
		<-release // Deliberately model an OAuth transport that ignores cancellation.
		return successfulOpenAIImageJobExecution()
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())
	worker.opts.HeartbeatInterval = time.Millisecond
	worker.opts.ExecutionTimeout = 12 * time.Millisecond

	if _, err := worker.RunOnce(context.Background(), "worker-1"); err != nil {
		close(release)
		t.Fatalf("RunOnce() error = %v", err)
	}
	close(release)
	snapshot := repo.snapshot()
	if snapshot.failCalls != 1 || !snapshot.failedUnknown || snapshot.failedCode != "failed_unknown" {
		t.Fatalf("failure = calls:%d unknown:%v code:%q, want 1/true/failed_unknown", snapshot.failCalls, snapshot.failedUnknown, snapshot.failedCode)
	}
	if snapshot.completeCalls != 0 || snapshot.cancelCalls != 0 {
		t.Fatalf("unexpected complete/cancel calls = %d/%d", snapshot.completeCalls, snapshot.cancelCalls)
	}
}

func TestOpenAIImageJobWorkerPostDispatchTimeoutFailsUnknownAndDoesNotRetry(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_timeout", Status: OpenAIImageJobStatusRunning})
	release := make(chan struct{})
	var executions atomic.Int32
	executor := openAIImageJobExecutorFunc(func(_ context.Context, _ *OpenAIImageJob, observer OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		executions.Add(1)
		observer.MarkDispatched()
		<-release
		return successfulOpenAIImageJobExecution()
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())
	worker.opts.ExecutionTimeout = 8 * time.Millisecond

	if _, err := worker.RunOnce(context.Background(), "worker-1"); err != nil {
		close(release)
		t.Fatalf("RunOnce() error = %v", err)
	}
	close(release)
	snapshot := repo.snapshot()
	if executions.Load() != 1 || snapshot.claimCalls != 1 || snapshot.failCalls != 1 {
		t.Fatalf("executions/claims/fails = %d/%d/%d, want 1/1/1", executions.Load(), snapshot.claimCalls, snapshot.failCalls)
	}
	if !snapshot.failedUnknown || snapshot.failedCode != "failed_unknown" {
		t.Fatalf("failure unknown/code = %v/%q, want true/failed_unknown", snapshot.failedUnknown, snapshot.failedCode)
	}
}

func TestOpenAIImageJobWorkerKnownFailureIsTerminalWithoutRetry(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_known_failure", Status: OpenAIImageJobStatusRunning})
	var executions atomic.Int32
	executor := openAIImageJobExecutorFunc(func(_ context.Context, _ *OpenAIImageJob, observer OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		executions.Add(1)
		observer.MarkDispatched()
		return OpenAIImageJobExecutionResult{
			Outcome:      OpenAIImageJobExecutionFailed,
			ErrorCode:    "upstream_rejected",
			ErrorMessage: "upstream rejected the image request",
		}
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())

	if _, err := worker.RunOnce(context.Background(), "worker-1"); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	snapshot := repo.snapshot()
	if executions.Load() != 1 || snapshot.failCalls != 1 || snapshot.failedUnknown {
		t.Fatalf("executions/fails/unknown = %d/%d/%v, want 1/1/false", executions.Load(), snapshot.failCalls, snapshot.failedUnknown)
	}
	if snapshot.failedCode != "upstream_rejected" {
		t.Fatalf("failure code = %q, want upstream_rejected", snapshot.failedCode)
	}
}

func TestOpenAIImageJobWorkerUnknownFailurePreservesExecutorCode(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_billing_unknown", Status: OpenAIImageJobStatusRunning})
	executor := openAIImageJobExecutorFunc(func(_ context.Context, _ *OpenAIImageJob, observer OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		observer.MarkDispatched()
		return OpenAIImageJobExecutionResult{
			Outcome:      OpenAIImageJobExecutionFailedUnknown,
			ErrorCode:    "billing_failed_unknown",
			ErrorMessage: "image generation completed but billing could not be confirmed",
		}
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())

	if _, err := worker.RunOnce(context.Background(), "worker-1"); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	snapshot := repo.snapshot()
	if snapshot.failCalls != 1 || !snapshot.failedUnknown {
		t.Fatalf("fails/unknown = %d/%v, want 1/true", snapshot.failCalls, snapshot.failedUnknown)
	}
	if snapshot.failedCode != "billing_failed_unknown" {
		t.Fatalf("failure code = %q, want billing_failed_unknown", snapshot.failedCode)
	}
}

func TestOpenAIImageJobWorkerHeartbeatLeaseLossAfterDispatchFailsUnknown(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_lease_lost", Status: OpenAIImageJobStatusRunning})
	repo.heartbeatResults = []openAIImageJobHeartbeatResult{{}, {}, {err: errors.New("heartbeat unavailable")}}
	release := make(chan struct{})
	executor := openAIImageJobExecutorFunc(func(_ context.Context, _ *OpenAIImageJob, observer OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		observer.MarkDispatched()
		<-release
		return successfulOpenAIImageJobExecution()
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())
	worker.opts.HeartbeatInterval = time.Millisecond

	if _, err := worker.RunOnce(context.Background(), "worker-1"); err == nil {
		close(release)
		t.Fatal("RunOnce() error = nil, want heartbeat error")
	}
	close(release)
	snapshot := repo.snapshot()
	if snapshot.failCalls != 1 || !snapshot.failedUnknown {
		t.Fatalf("fails/unknown = %d/%v, want 1/true", snapshot.failCalls, snapshot.failedUnknown)
	}
}

func TestOpenAIImageJobWorkerRecoveryAndCleanupUseConfiguredCutoffs(t *testing.T) {
	now := time.Date(2026, 7, 14, 15, 0, 0, 0, time.UTC)
	repo := newOpenAIImageJobWorkerRepositoryFake(nil)
	worker := newTestOpenAIImageJobWorker(repo, nil, now)
	worker.opts.QueuedRetention = 18 * time.Hour
	worker.opts.RecordRetention = 21 * 24 * time.Hour
	worker.opts.CleanupBatchLimit = 37

	if _, err := worker.RunRecoveryOnce(context.Background()); err != nil {
		t.Fatalf("RunRecoveryOnce() error = %v", err)
	}
	if _, err := worker.RunCleanupOnce(context.Background()); err != nil {
		t.Fatalf("RunCleanupOnce() error = %v", err)
	}
	snapshot := repo.snapshot()
	if snapshot.recoveryCalls != 1 || !snapshot.recoveryCutoff.Equal(now) {
		t.Fatalf("recovery calls/cutoff = %d/%v, want 1/%v", snapshot.recoveryCalls, snapshot.recoveryCutoff, now)
	}
	wantCleanup := OpenAIImageJobCleanupParams{
		Now:          now,
		QueuedCutoff: now.Add(-18 * time.Hour),
		RecordCutoff: now.Add(-21 * 24 * time.Hour),
		BatchLimit:   37,
	}
	if snapshot.cleanupCalls != 1 || snapshot.cleanupParams != wantCleanup {
		t.Fatalf("cleanup calls/params = %d/%+v, want 1/%+v", snapshot.cleanupCalls, snapshot.cleanupParams, wantCleanup)
	}
}

func TestOpenAIImageJobWorkerStopIsBoundedWhenExecutorIgnoresContext(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake(&OpenAIImageJob{JobID: "imgjob_shutdown", Status: OpenAIImageJobStatusRunning})
	release := make(chan struct{})
	executorStarted := make(chan struct{})
	executor := openAIImageJobExecutorFunc(func(_ context.Context, _ *OpenAIImageJob, observer OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		observer.MarkDispatched()
		close(executorStarted)
		<-release
		return successfulOpenAIImageJobExecution()
	})
	worker := newTestOpenAIImageJobWorker(repo, executor, time.Now())
	worker.opts.PollInterval = time.Millisecond
	worker.opts.ShutdownWait = 25 * time.Millisecond
	worker.Start()

	select {
	case <-executorStarted:
	case <-time.After(250 * time.Millisecond):
		worker.Stop()
		close(release)
		t.Fatal("executor did not start")
	}
	started := time.Now()
	worker.Stop()
	elapsed := time.Since(started)
	close(release)

	if elapsed > 100*time.Millisecond {
		t.Fatalf("Stop() took %v, want bounded well below 100ms", elapsed)
	}
	snapshot := repo.snapshot()
	if snapshot.failCalls != 1 || !snapshot.failedUnknown {
		t.Fatalf("shutdown fails/unknown = %d/%v, want 1/true", snapshot.failCalls, snapshot.failedUnknown)
	}
}

func TestOpenAIImageJobWorkerOptionsDefaults(t *testing.T) {
	opts := normalizeOpenAIImageJobWorkerOptions(OpenAIImageJobWorkerOptions{})
	if opts.WorkerCount != 2 || opts.PollInterval != time.Second || opts.HeartbeatInterval != 10*time.Second {
		t.Fatalf("worker defaults = count:%d poll:%v heartbeat:%v", opts.WorkerCount, opts.PollInterval, opts.HeartbeatInterval)
	}
	if opts.LeaseDuration != 2*time.Minute || opts.ExecutionTimeout != 15*time.Minute {
		t.Fatalf("execution defaults = lease:%v timeout:%v", opts.LeaseDuration, opts.ExecutionTimeout)
	}
	if opts.ResultRetention != 72*time.Hour || opts.QueuedRetention != 24*time.Hour || opts.RecordRetention != 30*24*time.Hour {
		t.Fatalf("retention defaults = result:%v queued:%v record:%v", opts.ResultRetention, opts.QueuedRetention, opts.RecordRetention)
	}
	if opts.CleanupInterval != time.Hour || opts.CleanupBatchLimit != DefaultOpenAIImageJobCleanupBatchLimit || opts.ShutdownWait != 10*time.Second {
		t.Fatalf("maintenance defaults = cleanup:%v batch:%d shutdown:%v", opts.CleanupInterval, opts.CleanupBatchLimit, opts.ShutdownWait)
	}
}

func TestOpenAIImageJobExecutionObserverRejectsDispatchAfterWorkerStopsAttempt(t *testing.T) {
	observer := &openAIImageJobExecutionObserver{}
	if dispatched := observer.stopBeforeDispatch(); dispatched {
		t.Fatal("stopBeforeDispatch() reported dispatched before any dispatch")
	}
	if accepted := observer.MarkDispatched(); accepted {
		t.Fatal("MarkDispatched() = true after the worker stopped the attempt")
	}
	if observer.Dispatched() {
		t.Fatal("Dispatched() = true after rejected late dispatch")
	}
}

func TestOpenAIImageJobWorkerOptionsBoundConcurrencyAndCleanupBatch(t *testing.T) {
	opts := normalizeOpenAIImageJobWorkerOptions(OpenAIImageJobWorkerOptions{
		WorkerCount:       1_000_000,
		CleanupBatchLimit: 1_000_000,
	})
	if opts.WorkerCount != maxOpenAIImageJobWorkerCount {
		t.Fatalf("WorkerCount = %d, want cap %d", opts.WorkerCount, maxOpenAIImageJobWorkerCount)
	}
	if opts.CleanupBatchLimit != MaxOpenAIImageJobCleanupBatchLimit {
		t.Fatalf("CleanupBatchLimit = %d, want cap %d", opts.CleanupBatchLimit, MaxOpenAIImageJobCleanupBatchLimit)
	}
}

type openAIImageJobExecutorFunc func(context.Context, *OpenAIImageJob, OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult

func (f openAIImageJobExecutorFunc) Execute(ctx context.Context, job *OpenAIImageJob, observer OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
	return f(ctx, job, observer)
}

func successfulOpenAIImageJobExecution() OpenAIImageJobExecutionResult {
	return OpenAIImageJobExecutionResult{
		Outcome: OpenAIImageJobExecutionSucceeded,
		Response: OpenAIImageJobResponse{
			StatusCode:  httpStatusOK,
			ContentType: "application/json",
			Body:        []byte(`{"data":[{"b64_json":"result"}]}`),
		},
	}
}

const httpStatusOK = 200

func newTestOpenAIImageJobWorker(repo OpenAIImageJobRepository, executor OpenAIImageJobExecutor, now time.Time) *OpenAIImageJobWorkerRuntime {
	worker := NewOpenAIImageJobWorkerRuntime(repo, executor, OpenAIImageJobWorkerOptions{
		WorkerCount:       1,
		PollInterval:      5 * time.Millisecond,
		HeartbeatInterval: 20 * time.Millisecond,
		LeaseDuration:     time.Minute,
		ExecutionTimeout:  time.Second,
		ResultRetention:   time.Hour,
		QueuedRetention:   time.Hour,
		RecordRetention:   24 * time.Hour,
		RecoveryInterval:  time.Hour,
		CleanupInterval:   time.Hour,
		CleanupBatchLimit: 10,
		ShutdownWait:      50 * time.Millisecond,
	})
	worker.now = func() time.Time { return now }
	return worker
}

func waitForOpenAIImageJobWorkerCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition was not satisfied before timeout")
}

type openAIImageJobHeartbeatResult struct {
	cancelRequested bool
	err             error
}

type openAIImageJobWorkerRepositoryFake struct {
	mu sync.Mutex

	jobs             []*OpenAIImageJob
	heartbeatResults []openAIImageJobHeartbeatResult
	heartbeatIndex   int
	heartbeatHook    func(call int)

	claimCalls        int
	heartbeatCalls    int
	completeCalls     int
	failCalls         int
	cancelCalls       int
	recoveryCalls     int
	cleanupCalls      int
	lastLeaseUntil    time.Time
	completedResponse OpenAIImageJobResponse
	failedCode        string
	failedMessage     string
	failedUnknown     bool
	recoveryCutoff    time.Time
	cleanupParams     OpenAIImageJobCleanupParams
}

func newOpenAIImageJobWorkerRepositoryFake(jobs ...*OpenAIImageJob) *openAIImageJobWorkerRepositoryFake {
	filtered := make([]*OpenAIImageJob, 0, len(jobs))
	for _, job := range jobs {
		if job != nil {
			filtered = append(filtered, job)
		}
	}
	return &openAIImageJobWorkerRepositoryFake{jobs: filtered}
}

func (r *openAIImageJobWorkerRepositoryFake) snapshot() openAIImageJobWorkerRepositoryFake {
	r.mu.Lock()
	defer r.mu.Unlock()
	return openAIImageJobWorkerRepositoryFake{
		claimCalls:        r.claimCalls,
		heartbeatCalls:    r.heartbeatCalls,
		completeCalls:     r.completeCalls,
		failCalls:         r.failCalls,
		cancelCalls:       r.cancelCalls,
		recoveryCalls:     r.recoveryCalls,
		cleanupCalls:      r.cleanupCalls,
		lastLeaseUntil:    r.lastLeaseUntil,
		completedResponse: r.completedResponse,
		failedCode:        r.failedCode,
		failedMessage:     r.failedMessage,
		failedUnknown:     r.failedUnknown,
		recoveryCutoff:    r.recoveryCutoff,
		cleanupParams:     r.cleanupParams,
	}
}

func (r *openAIImageJobWorkerRepositoryFake) CreateOrGet(context.Context, CreateOpenAIImageJobParams) (*OpenAIImageJob, bool, error) {
	return nil, false, errors.New("not implemented by worker fake")
}

func (r *openAIImageJobWorkerRepositoryFake) GetForUser(context.Context, int64, string) (*OpenAIImageJob, error) {
	return nil, errors.New("not implemented by worker fake")
}

func (r *openAIImageJobWorkerRepositoryFake) ClaimNext(_ context.Context, _ string, leaseUntil time.Time) (*OpenAIImageJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.claimCalls++
	r.lastLeaseUntil = leaseUntil
	if len(r.jobs) == 0 {
		return nil, nil
	}
	job := r.jobs[0]
	r.jobs = r.jobs[1:]
	return job, nil
}

func (r *openAIImageJobWorkerRepositoryFake) Heartbeat(_ context.Context, _, _ string, leaseUntil time.Time) (bool, error) {
	r.mu.Lock()
	r.heartbeatCalls++
	call := r.heartbeatCalls
	r.lastLeaseUntil = leaseUntil
	if len(r.heartbeatResults) == 0 {
		hook := r.heartbeatHook
		r.mu.Unlock()
		if hook != nil {
			hook(call)
		}
		return false, nil
	}
	index := r.heartbeatIndex
	if index >= len(r.heartbeatResults) {
		index = len(r.heartbeatResults) - 1
	} else {
		r.heartbeatIndex++
	}
	result := r.heartbeatResults[index]
	hook := r.heartbeatHook
	r.mu.Unlock()
	if hook != nil {
		hook(call)
	}
	return result.cancelRequested, result.err
}

func (r *openAIImageJobWorkerRepositoryFake) Complete(_ context.Context, _, _ string, response OpenAIImageJobResponse) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completeCalls++
	r.completedResponse = response
	return nil
}

func (r *openAIImageJobWorkerRepositoryFake) Fail(_ context.Context, _, _, code, message string, unknown bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failCalls++
	r.failedCode = code
	r.failedMessage = message
	r.failedUnknown = unknown
	return nil
}

func (r *openAIImageJobWorkerRepositoryFake) CancelForUser(context.Context, int64, string) (*OpenAIImageJob, error) {
	return nil, errors.New("not implemented by worker fake")
}

func (r *openAIImageJobWorkerRepositoryFake) MarkCancelled(context.Context, string, string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cancelCalls++
	return nil
}

func (r *openAIImageJobWorkerRepositoryFake) FailExpiredLeases(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recoveryCalls++
	r.recoveryCutoff = cutoff
	return 0, nil
}

func (r *openAIImageJobWorkerRepositoryFake) PurgeExpiredPayloads(_ context.Context, params OpenAIImageJobCleanupParams) (OpenAIImageJobCleanupResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleanupCalls++
	r.cleanupParams = params
	return OpenAIImageJobCleanupResult{}, nil
}
