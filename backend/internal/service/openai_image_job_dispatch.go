package service

import "context"

// openAIImageJobExecutionObserverContextKey is deliberately private: only the
// durable image executor may install a dispatch gate, and callers cannot forge
// one through HTTP headers or request parameters.
type openAIImageJobExecutionObserverContextKey struct{}

type openAIImageJobKnownNonBillableDispatchAcknowledger interface {
	AcknowledgeKnownNonBillableDispatch() bool
}

// WithOpenAIImageJobExecutionObserver carries the worker's one-way dispatch
// gate through the existing Images pipeline. The observer is checked only
// after credentials and the complete upstream request have been prepared.
func WithOpenAIImageJobExecutionObserver(ctx context.Context, observer OpenAIImageJobExecutionObserver) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if observer == nil {
		return ctx
	}
	return context.WithValue(ctx, openAIImageJobExecutionObserverContextKey{}, observer)
}

// OpenAIImageJobExecutionObserverFromContext returns the executor-owned
// dispatch gate, when one is installed.
func OpenAIImageJobExecutionObserverFromContext(ctx context.Context) (OpenAIImageJobExecutionObserver, bool) {
	if ctx == nil {
		return nil, false
	}
	observer, ok := ctx.Value(openAIImageJobExecutionObserverContextKey{}).(OpenAIImageJobExecutionObserver)
	return observer, ok && observer != nil
}

// MarkOpenAIImageJobDispatched opens the job's one-way dispatch gate. Normal
// synchronous Images requests carry no observer and therefore retain their
// existing behavior. A false result is a hard stop: the caller must not invoke
// the upstream transport.
func MarkOpenAIImageJobDispatched(ctx context.Context) bool {
	if ctx == nil {
		return true
	}
	observer, ok := OpenAIImageJobExecutionObserverFromContext(ctx)
	if !ok {
		return true
	}
	return observer.MarkDispatched()
}

// AcknowledgeOpenAIImageJobKnownNonBillableDispatch rearms a durable job's
// one-way dispatch gate exactly once after the upstream has conclusively
// rejected the request before billable work started. Normal HTTP requests do
// not carry an observer and therefore need no rearm. A durable observer that
// does not explicitly support this transition is denied conservatively.
func AcknowledgeOpenAIImageJobKnownNonBillableDispatch(ctx context.Context) bool {
	if ctx == nil || ctx.Err() != nil {
		return false
	}
	observer, ok := OpenAIImageJobExecutionObserverFromContext(ctx)
	if !ok {
		return true
	}
	acknowledger, ok := observer.(openAIImageJobKnownNonBillableDispatchAcknowledger)
	return ok && acknowledger.AcknowledgeKnownNonBillableDispatch()
}
