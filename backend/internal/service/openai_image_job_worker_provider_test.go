package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestProvideOpenAIImageJobWorkerRuntimeMapsConfigAndStartsWhenEnabled(t *testing.T) {
	repo := newOpenAIImageJobWorkerRepositoryFake()
	executor := openAIImageJobExecutorFunc(func(context.Context, *OpenAIImageJob, OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
		t.Fatal("executor must not run without a queued job")
		return OpenAIImageJobExecutionResult{}
	})
	cfg := &config.Config{OpenAIImageJobs: config.OpenAIImageJobsConfig{
		Enabled:                  true,
		WorkerCount:              3,
		PollIntervalSeconds:      2,
		HeartbeatIntervalSeconds: 4,
		LeaseDurationSeconds:     30,
		ExecutionTimeoutSeconds:  90,
		ResultRetentionHours:     12,
		QueuedRetentionHours:     6,
		MetadataRetentionDays:    8,
		CleanupIntervalSeconds:   45,
		CleanupBatchSize:         77,
		ShutdownWaitSeconds:      5,
	}}

	runtime := ProvideOpenAIImageJobWorkerRuntime(repo, executor, cfg)
	defer runtime.Stop()

	if !runtime.Running() {
		t.Fatal("runtime was not started")
	}
	if runtime.opts.WorkerCount != 3 || runtime.opts.PollInterval != 2*time.Second {
		t.Fatalf("worker/poll options = %d/%s", runtime.opts.WorkerCount, runtime.opts.PollInterval)
	}
	if runtime.opts.HeartbeatInterval != 4*time.Second || runtime.opts.LeaseDuration != 30*time.Second {
		t.Fatalf("heartbeat/lease options = %s/%s", runtime.opts.HeartbeatInterval, runtime.opts.LeaseDuration)
	}
	if runtime.opts.ExecutionTimeout != 90*time.Second || runtime.opts.ShutdownWait != 5*time.Second {
		t.Fatalf("execution/shutdown options = %s/%s", runtime.opts.ExecutionTimeout, runtime.opts.ShutdownWait)
	}
	if runtime.opts.ResultRetention != 12*time.Hour || runtime.opts.QueuedRetention != 6*time.Hour || runtime.opts.RecordRetention != 8*24*time.Hour {
		t.Fatalf("retention options = %s/%s/%s", runtime.opts.ResultRetention, runtime.opts.QueuedRetention, runtime.opts.RecordRetention)
	}
	if runtime.opts.CleanupInterval != 45*time.Second || runtime.opts.CleanupBatchLimit != 77 {
		t.Fatalf("cleanup options = %s/%d", runtime.opts.CleanupInterval, runtime.opts.CleanupBatchLimit)
	}
}

func TestProvideOpenAIImageJobWorkerRuntimeDoesNotStartWhenDisabled(t *testing.T) {
	runtime := ProvideOpenAIImageJobWorkerRuntime(
		newOpenAIImageJobWorkerRepositoryFake(),
		openAIImageJobExecutorFunc(func(context.Context, *OpenAIImageJob, OpenAIImageJobExecutionObserver) OpenAIImageJobExecutionResult {
			return OpenAIImageJobExecutionResult{}
		}),
		&config.Config{OpenAIImageJobs: config.OpenAIImageJobsConfig{Enabled: false}},
	)
	defer runtime.Stop()

	if runtime.Running() {
		t.Fatal("disabled runtime unexpectedly started")
	}
}
