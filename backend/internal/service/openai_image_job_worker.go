package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	defaultOpenAIImageJobWorkerCount       = 2
	defaultOpenAIImageJobPollInterval      = time.Second
	defaultOpenAIImageJobHeartbeatInterval = 10 * time.Second
	defaultOpenAIImageJobLeaseDuration     = 2 * time.Minute
	defaultOpenAIImageJobExecutionTimeout  = 15 * time.Minute
	defaultOpenAIImageJobResultRetention   = 72 * time.Hour
	defaultOpenAIImageJobQueuedRetention   = 24 * time.Hour
	defaultOpenAIImageJobRecordRetention   = 30 * 24 * time.Hour
	defaultOpenAIImageJobRecoveryInterval  = time.Minute
	defaultOpenAIImageJobCleanupInterval   = time.Hour
	defaultOpenAIImageJobShutdownWait      = 10 * time.Second
	maxOpenAIImageJobWorkerCount           = 64
)

type OpenAIImageJobExecutionOutcome string

const (
	// OpenAIImageJobExecutionSucceeded means the executor verified a real 2xx
	// image response and completed the synchronous billing barrier.
	OpenAIImageJobExecutionSucceeded OpenAIImageJobExecutionOutcome = "succeeded"
	// OpenAIImageJobExecutionFailed means the failure itself is known and safe
	// to publish (for example, a verified upstream error response).
	OpenAIImageJobExecutionFailed OpenAIImageJobExecutionOutcome = "failed"
	// OpenAIImageJobExecutionFailedUnknown means an upstream request may have
	// succeeded or incurred cost, but no verified response is available.
	OpenAIImageJobExecutionFailedUnknown OpenAIImageJobExecutionOutcome = "failed_unknown"
	// OpenAIImageJobExecutionInterrupted means execution stopped because its
	// context was cancelled. The worker combines this with the dispatch marker
	// to decide between cancelled and failed_unknown.
	OpenAIImageJobExecutionInterrupted OpenAIImageJobExecutionOutcome = "interrupted"
)

type OpenAIImageJobExecutionResult struct {
	Outcome      OpenAIImageJobExecutionOutcome
	Response     OpenAIImageJobResponse
	ErrorCode    string
	ErrorMessage string
}

// OpenAIImageJobExecutionObserver is the safety boundary between an executor
// and the durable worker. The executor MUST call MarkDispatched immediately
// before any request bytes can reach an upstream. Missing that call can make an
// ambiguous request look safe to cancel.
type OpenAIImageJobExecutionObserver interface {
	// MarkDispatched atomically opens the one-way transition to dispatched.
	// The executor must not contact the upstream when it returns false because
	// the worker has already stopped the attempt before dispatch.
	MarkDispatched() bool
	Dispatched() bool
}

type OpenAIImageJobExecutor interface {
	Execute(ctx context.Context, job *OpenAIImageJob, observer OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult
}

// OpenAIImageJobCancelNotifier accelerates cancellation for work owned by the
// current process. The persisted cancel_requested flag remains authoritative
// across processes.
type OpenAIImageJobCancelNotifier interface {
	RequestCancel(jobID string) bool
}

type OpenAIImageJobWorkerOptions struct {
	WorkerCount       int
	PollInterval      time.Duration
	HeartbeatInterval time.Duration
	LeaseDuration     time.Duration
	ExecutionTimeout  time.Duration
	ResultRetention   time.Duration
	QueuedRetention   time.Duration
	RecordRetention   time.Duration
	RecoveryInterval  time.Duration
	CleanupInterval   time.Duration
	CleanupBatchLimit int
	ShutdownWait      time.Duration
}

type OpenAIImageJobWorkerRuntime struct {
	repo      OpenAIImageJobRepository
	executor  OpenAIImageJobExecutor
	opts      OpenAIImageJobWorkerOptions
	now       func() time.Time
	runtimeID string

	mu             sync.Mutex
	cancel         context.CancelFunc
	done           chan struct{}
	runningMu      sync.Mutex
	runningCancels map[string]context.CancelFunc
}

func NewOpenAIImageJobWorkerRuntime(
	repo OpenAIImageJobRepository,
	executor OpenAIImageJobExecutor,
	opts OpenAIImageJobWorkerOptions,
) *OpenAIImageJobWorkerRuntime {
	return &OpenAIImageJobWorkerRuntime{
		repo:           repo,
		executor:       executor,
		opts:           normalizeOpenAIImageJobWorkerOptions(opts),
		now:            time.Now,
		runtimeID:      uuid.NewString(),
		runningCancels: make(map[string]context.CancelFunc),
	}
}

func normalizeOpenAIImageJobWorkerOptions(opts OpenAIImageJobWorkerOptions) OpenAIImageJobWorkerOptions {
	if opts.WorkerCount <= 0 {
		opts.WorkerCount = defaultOpenAIImageJobWorkerCount
	}
	if opts.WorkerCount > maxOpenAIImageJobWorkerCount {
		opts.WorkerCount = maxOpenAIImageJobWorkerCount
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaultOpenAIImageJobPollInterval
	}
	if opts.HeartbeatInterval <= 0 {
		opts.HeartbeatInterval = defaultOpenAIImageJobHeartbeatInterval
	}
	if opts.LeaseDuration <= 0 {
		opts.LeaseDuration = defaultOpenAIImageJobLeaseDuration
	}
	if opts.LeaseDuration <= opts.HeartbeatInterval {
		opts.LeaseDuration = 3 * opts.HeartbeatInterval
	}
	if opts.ExecutionTimeout <= 0 {
		opts.ExecutionTimeout = defaultOpenAIImageJobExecutionTimeout
	}
	if opts.ResultRetention <= 0 {
		opts.ResultRetention = defaultOpenAIImageJobResultRetention
	}
	if opts.QueuedRetention <= 0 {
		opts.QueuedRetention = defaultOpenAIImageJobQueuedRetention
	}
	if opts.RecordRetention <= 0 {
		opts.RecordRetention = defaultOpenAIImageJobRecordRetention
	}
	if opts.RecoveryInterval <= 0 {
		opts.RecoveryInterval = defaultOpenAIImageJobRecoveryInterval
	}
	if opts.CleanupInterval <= 0 {
		opts.CleanupInterval = defaultOpenAIImageJobCleanupInterval
	}
	if opts.CleanupBatchLimit <= 0 {
		opts.CleanupBatchLimit = DefaultOpenAIImageJobCleanupBatchLimit
	}
	if opts.CleanupBatchLimit > MaxOpenAIImageJobCleanupBatchLimit {
		opts.CleanupBatchLimit = MaxOpenAIImageJobCleanupBatchLimit
	}
	if opts.ShutdownWait <= 0 {
		opts.ShutdownWait = defaultOpenAIImageJobShutdownWait
	}
	return opts
}

func (r *OpenAIImageJobWorkerRuntime) RunOnce(ctx context.Context, workerID string) (bool, error) {
	if r == nil || r.repo == nil || r.executor == nil {
		return false, nil
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if workerID == "" {
		workerID = r.runtimeID + "-manual"
	}

	job, err := r.repo.ClaimNext(ctx, workerID, r.now().Add(r.opts.LeaseDuration))
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}

	cancelRequested, heartbeatErr := r.repo.Heartbeat(ctx, job.JobID, workerID, r.now().Add(r.opts.LeaseDuration))
	if heartbeatErr != nil {
		return true, r.failBeforeExecution(ctx, job, workerID, heartbeatErr)
	}
	if job.CancelRequested || cancelRequested {
		return true, r.repo.MarkCancelled(ctx, job.JobID, workerID)
	}

	return true, r.executeClaimedJob(ctx, job, workerID)
}

func (r *OpenAIImageJobWorkerRuntime) failBeforeExecution(ctx context.Context, job *OpenAIImageJob, workerID string, cause error) error {
	if errors.Is(cause, ErrOpenAIImageJobLeaseLost) {
		return cause
	}
	failErr := r.repo.Fail(
		ctx,
		job.JobID,
		workerID,
		"worker_heartbeat_failed",
		"image generation worker could not confirm its lease before execution",
		false,
	)
	return errors.Join(cause, failErr)
}

func (r *OpenAIImageJobWorkerRuntime) executeClaimedJob(ctx context.Context, job *OpenAIImageJob, workerID string) error {
	observer := &openAIImageJobExecutionObserver{}
	executionCtx, cancelExecution := context.WithCancel(context.Background())
	defer cancelExecution()
	var localCancelRequested atomic.Bool
	requestCancel := func() {
		localCancelRequested.Store(true)
		observer.stopBeforeDispatch()
		cancelExecution()
	}
	if !r.registerRunningCancel(job.JobID, requestCancel) {
		return fmt.Errorf("OpenAI image job %q is already executing in this runtime", job.JobID)
	}
	defer r.unregisterRunningCancel(job.JobID)

	// Close the small race between the heartbeat in RunOnce and registration
	// in the in-process cancellation map. Once this check finishes, either the
	// persisted flag is visible here or RequestCancel can reach requestCancel.
	persistedCancel, err := r.repo.Heartbeat(ctx, job.JobID, workerID, r.now().Add(r.opts.LeaseDuration))
	if err != nil {
		requestCancel()
		return r.failBeforeExecution(ctx, job, workerID, err)
	}
	if persistedCancel || localCancelRequested.Load() {
		requestCancel()
		return r.repo.MarkCancelled(ctx, job.JobID, workerID)
	}

	resultCh := make(chan OpenAIImageJobExecutionResult, 1)
	go func() {
		resultCh <- r.executor.Execute(executionCtx, job, observer)
	}()

	heartbeatTicker := time.NewTicker(r.opts.HeartbeatInterval)
	defer heartbeatTicker.Stop()
	executionTimer := time.NewTimer(r.opts.ExecutionTimeout)
	defer executionTimer.Stop()
	cancelRequested := false

	for {
		select {
		case result := <-resultCh:
			if localCancelRequested.Load() {
				cancelRequested = true
			}
			return r.finalizeReceivedExecutionResult(ctx, job, workerID, observer, cancelRequested, result)

		case <-heartbeatTicker.C:
			persistedCancel, err := r.repo.Heartbeat(ctx, job.JobID, workerID, r.now().Add(r.opts.LeaseDuration))
			if err != nil {
				observer.stopBeforeDispatch()
				cancelExecution()
				if result, ok := tryOpenAIImageJobExecutionResult(resultCh); ok && result.Outcome == OpenAIImageJobExecutionSucceeded {
					return r.finalizeExecutionResult(ctx, job, workerID, observer, cancelRequested, result)
				}
				terminalErr := r.finalizeInterrupted(ctx, job, workerID, observer, cancelRequested, "worker_lease_lost", "image generation worker lost its lease")
				return errors.Join(err, terminalErr)
			}
			if persistedCancel && !cancelRequested {
				cancelRequested = true
				observer.stopBeforeDispatch()
				cancelExecution()
			}

		case <-executionTimer.C:
			observer.stopBeforeDispatch()
			cancelExecution()
			if result, ok := tryOpenAIImageJobExecutionResult(resultCh); ok && result.Outcome == OpenAIImageJobExecutionSucceeded {
				return r.finalizeExecutionResult(ctx, job, workerID, observer, cancelRequested, result)
			}
			return r.finalizeInterrupted(ctx, job, workerID, observer, cancelRequested, "execution_timeout", "image generation execution timed out")

		case <-ctx.Done():
			observer.stopBeforeDispatch()
			cancelExecution()
			joinBudget, finalizationBudget := splitOpenAIImageJobShutdownBudget(r.opts.ShutdownWait)
			joinTimer := time.NewTimer(joinBudget)
			select {
			case result := <-resultCh:
				if !joinTimer.Stop() {
					select {
					case <-joinTimer.C:
					default:
					}
				}
				terminalCtx, terminalCancel := context.WithTimeout(context.Background(), finalizationBudget)
				err := r.finalizeExecutionResult(terminalCtx, job, workerID, observer, cancelRequested, result)
				terminalCancel()
				return errors.Join(ctx.Err(), err)
			case <-joinTimer.C:
			}
			terminalCtx, terminalCancel := context.WithTimeout(context.Background(), finalizationBudget)
			terminalErr := r.finalizeInterrupted(terminalCtx, job, workerID, observer, cancelRequested, "worker_shutdown", "image generation worker stopped during shutdown")
			terminalCancel()
			return errors.Join(ctx.Err(), terminalErr)
		}
	}
}

func (r *OpenAIImageJobWorkerRuntime) finalizeReceivedExecutionResult(
	ctx context.Context,
	job *OpenAIImageJob,
	workerID string,
	observer OpenAIImageJobExecutionObserver,
	cancelRequested bool,
	result OpenAIImageJobExecutionResult,
) error {
	finalizeCtx := ctx
	var finalizeCancel context.CancelFunc
	if ctx == nil || ctx.Err() != nil {
		// The executor can publish a verified result concurrently with runtime
		// shutdown. Do not lose that terminal state merely because the worker
		// loop's parent context has already been cancelled.
		finalizeCtx, finalizeCancel = r.newShutdownFinalizationContext()
	}
	err := r.finalizeExecutionResult(finalizeCtx, job, workerID, observer, cancelRequested, result)
	if finalizeCancel != nil {
		finalizeCancel()
	}
	return err
}

func splitOpenAIImageJobShutdownBudget(total time.Duration) (time.Duration, time.Duration) {
	if total <= 0 {
		total = defaultOpenAIImageJobShutdownWait
	}
	join := total / 2
	if join <= 0 {
		join = total
	}
	finalize := total - join
	if finalize <= 0 {
		finalize = total
	}
	return join, finalize
}

// RequestCancel immediately notifies a job executing in this process. The
// durable cancel_requested flag remains the source of truth and is observed by
// heartbeat on other instances; this method only removes local heartbeat
// latency after the cancel handler has persisted that flag.
func (r *OpenAIImageJobWorkerRuntime) RequestCancel(jobID string) bool {
	if r == nil || jobID == "" {
		return false
	}
	r.runningMu.Lock()
	cancel := r.runningCancels[jobID]
	r.runningMu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (r *OpenAIImageJobWorkerRuntime) registerRunningCancel(jobID string, cancel context.CancelFunc) bool {
	if r == nil || jobID == "" || cancel == nil {
		return false
	}
	r.runningMu.Lock()
	defer r.runningMu.Unlock()
	if r.runningCancels == nil {
		r.runningCancels = make(map[string]context.CancelFunc)
	}
	if _, exists := r.runningCancels[jobID]; exists {
		return false
	}
	r.runningCancels[jobID] = cancel
	return true
}

func (r *OpenAIImageJobWorkerRuntime) unregisterRunningCancel(jobID string) {
	if r == nil || jobID == "" {
		return
	}
	r.runningMu.Lock()
	delete(r.runningCancels, jobID)
	r.runningMu.Unlock()
}

func tryOpenAIImageJobExecutionResult(resultCh <-chan OpenAIImageJobExecutionResult) (OpenAIImageJobExecutionResult, bool) {
	select {
	case result := <-resultCh:
		return result, true
	default:
		return OpenAIImageJobExecutionResult{}, false
	}
}

func (r *OpenAIImageJobWorkerRuntime) finalizeExecutionResult(
	ctx context.Context,
	job *OpenAIImageJob,
	workerID string,
	observer OpenAIImageJobExecutionObserver,
	cancelRequested bool,
	result OpenAIImageJobExecutionResult,
) error {
	switch result.Outcome {
	case OpenAIImageJobExecutionSucceeded:
		if result.Response.StatusCode < 200 || result.Response.StatusCode >= 300 || len(result.Response.Body) == 0 {
			return r.failUnknown(ctx, job, workerID, "", "executor returned an invalid success response")
		}
		result.Response.ResultExpiresAt = r.now().Add(r.opts.ResultRetention)
		return r.repo.Complete(ctx, job.JobID, workerID, result.Response)

	case OpenAIImageJobExecutionFailed:
		code := result.ErrorCode
		if code == "" {
			code = "image_generation_failed"
		}
		message := result.ErrorMessage
		if message == "" {
			message = "image generation failed"
		}
		return r.repo.Fail(ctx, job.JobID, workerID, code, message, false)

	case OpenAIImageJobExecutionFailedUnknown:
		message := result.ErrorMessage
		if message == "" {
			message = "image generation may have reached the upstream, but no verified result was received"
		}
		return r.failUnknown(ctx, job, workerID, result.ErrorCode, message)

	case OpenAIImageJobExecutionInterrupted:
		return r.finalizeInterrupted(
			ctx,
			job,
			workerID,
			observer,
			cancelRequested,
			"execution_interrupted",
			"image generation execution was interrupted",
		)

	default:
		if observer.Dispatched() {
			return r.failUnknown(ctx, job, workerID, "", "image generation executor returned an indeterminate result")
		}
		return r.repo.Fail(
			ctx,
			job.JobID,
			workerID,
			"invalid_execution_result",
			"image generation executor returned an invalid result",
			false,
		)
	}
}

func (r *OpenAIImageJobWorkerRuntime) finalizeInterrupted(
	ctx context.Context,
	job *OpenAIImageJob,
	workerID string,
	observer OpenAIImageJobExecutionObserver,
	cancelRequested bool,
	knownCode string,
	knownMessage string,
) error {
	if observer.Dispatched() {
		return r.failUnknown(ctx, job, workerID, "", "image generation may have reached the upstream, but no verified result was received")
	}
	if cancelRequested || job.CancelRequested {
		return r.repo.MarkCancelled(ctx, job.JobID, workerID)
	}
	return r.repo.Fail(ctx, job.JobID, workerID, knownCode, knownMessage, false)
}

func (r *OpenAIImageJobWorkerRuntime) failUnknown(ctx context.Context, job *OpenAIImageJob, workerID, code, message string) error {
	if code == "" {
		code = "failed_unknown"
	}
	return r.repo.Fail(ctx, job.JobID, workerID, code, message, true)
}

func (r *OpenAIImageJobWorkerRuntime) newShutdownFinalizationContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), r.opts.ShutdownWait)
}

func (r *OpenAIImageJobWorkerRuntime) RunRecoveryOnce(ctx context.Context) (int64, error) {
	if r == nil || r.repo == nil {
		return 0, nil
	}
	return r.repo.FailExpiredLeases(ctx, r.now())
}

func (r *OpenAIImageJobWorkerRuntime) RunCleanupOnce(ctx context.Context) (OpenAIImageJobCleanupResult, error) {
	if r == nil || r.repo == nil {
		return OpenAIImageJobCleanupResult{}, nil
	}
	now := r.now()
	return r.repo.PurgeExpiredPayloads(ctx, OpenAIImageJobCleanupParams{
		Now:          now,
		QueuedCutoff: now.Add(-r.opts.QueuedRetention),
		RecordCutoff: now.Add(-r.opts.RecordRetention),
		BatchLimit:   r.opts.CleanupBatchLimit,
	})
}

func (r *OpenAIImageJobWorkerRuntime) Start() {
	if r == nil || r.repo == nil || r.executor == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	r.cancel = cancel
	r.done = done

	var wg sync.WaitGroup
	for index := 0; index < r.opts.WorkerCount; index++ {
		workerID := fmt.Sprintf("%s-%d", r.runtimeID, index+1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.runWorker(ctx, workerID)
		}()
	}
	wg.Add(2)
	go func() {
		defer wg.Done()
		r.runRecovery(ctx)
	}()
	go func() {
		defer wg.Done()
		r.runCleanup(ctx)
	}()
	go func() {
		wg.Wait()
		close(done)
	}()
}

func (r *OpenAIImageJobWorkerRuntime) runWorker(ctx context.Context, workerID string) {
	for ctx.Err() == nil {
		processed, err := r.RunOnce(ctx, workerID)
		if err != nil && ctx.Err() == nil {
			logger.L().Warn("openai_image_job.worker_run_failed", zap.String("worker_id", workerID), zap.Error(err))
		}
		if processed && err == nil {
			continue
		}
		waitOpenAIImageJobWorkerInterval(ctx, r.opts.PollInterval)
	}
}

func (r *OpenAIImageJobWorkerRuntime) runRecovery(ctx context.Context) {
	r.runPeriodicMaintenance(ctx, r.opts.RecoveryInterval, "openai_image_job.recovery_failed", func(ctx context.Context) error {
		_, err := r.RunRecoveryOnce(ctx)
		return err
	})
}

func (r *OpenAIImageJobWorkerRuntime) runCleanup(ctx context.Context) {
	r.runPeriodicMaintenance(ctx, r.opts.CleanupInterval, "openai_image_job.cleanup_failed", func(ctx context.Context) error {
		_, err := r.RunCleanupOnce(ctx)
		return err
	})
}

func (r *OpenAIImageJobWorkerRuntime) runPeriodicMaintenance(ctx context.Context, interval time.Duration, event string, run func(context.Context) error) {
	if err := run(ctx); err != nil && ctx.Err() == nil {
		logger.L().Warn(event, zap.Error(err))
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := run(ctx); err != nil && ctx.Err() == nil {
				logger.L().Warn(event, zap.Error(err))
			}
		}
	}
}

func waitOpenAIImageJobWorkerInterval(ctx context.Context, interval time.Duration) {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func (r *OpenAIImageJobWorkerRuntime) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel := r.cancel
	done := r.done
	r.cancel = nil
	r.done = nil
	r.mu.Unlock()

	if cancel == nil {
		return
	}
	cancel()
	if done == nil {
		return
	}
	timer := time.NewTimer(r.opts.ShutdownWait)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		logger.L().Warn("openai_image_job.worker_shutdown_timeout", zap.Duration("timeout", r.opts.ShutdownWait))
	}
}

func (r *OpenAIImageJobWorkerRuntime) Running() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cancel != nil
}

type openAIImageJobExecutionObserver struct {
	state                         atomic.Uint32
	knownNonBillableRearmConsumed atomic.Bool
}

const (
	openAIImageJobDispatchPending uint32 = iota
	openAIImageJobDispatchStarted
	openAIImageJobDispatchStopped
)

func (o *openAIImageJobExecutionObserver) MarkDispatched() bool {
	if o == nil {
		return false
	}
	for {
		switch state := o.state.Load(); state {
		case openAIImageJobDispatchStarted:
			// Dispatch is one-shot. A later account failover must not issue a
			// second billable upstream request for the same durable job.
			return false
		case openAIImageJobDispatchStopped:
			return false
		default:
			if o.state.CompareAndSwap(openAIImageJobDispatchPending, openAIImageJobDispatchStarted) {
				return true
			}
		}
	}
}

func (o *openAIImageJobExecutionObserver) Dispatched() bool {
	return o != nil && o.state.Load() == openAIImageJobDispatchStarted
}

func (o *openAIImageJobExecutionObserver) AcknowledgeKnownNonBillableDispatch() bool {
	if o == nil || !o.knownNonBillableRearmConsumed.CompareAndSwap(false, true) {
		return false
	}
	return o.state.CompareAndSwap(openAIImageJobDispatchStarted, openAIImageJobDispatchPending)
}

// stopBeforeDispatch closes the dispatch gate. Its return value reports
// whether dispatch had already won the race and therefore remains ambiguous.
func (o *openAIImageJobExecutionObserver) stopBeforeDispatch() bool {
	if o == nil {
		return false
	}
	for {
		switch state := o.state.Load(); state {
		case openAIImageJobDispatchStarted:
			return true
		case openAIImageJobDispatchStopped:
			return false
		default:
			if o.state.CompareAndSwap(openAIImageJobDispatchPending, openAIImageJobDispatchStopped) {
				return false
			}
		}
	}
}
