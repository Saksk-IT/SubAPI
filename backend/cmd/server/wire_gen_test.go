package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestProvideServiceBuildInfo(t *testing.T) {
	in := handler.BuildInfo{
		Version:   "v-test",
		BuildType: "release",
	}
	out := provideServiceBuildInfo(in)
	require.Equal(t, in.Version, out.Version)
	require.Equal(t, in.BuildType, out.BuildType)
}

type recordingPromptAuditRuntime struct {
	trace *[]string
	err   error
	mode  securityaudit.Mode
}

func (r recordingPromptAuditRuntime) Start(context.Context) error {
	*r.trace = append(*r.trace, "prompt_audit")
	return r.err
}

func (r recordingPromptAuditRuntime) EffectiveMode() securityaudit.Mode { return r.mode }

type recordingImageJobWorkerRuntime struct{ trace *[]string }

func (r recordingImageJobWorkerRuntime) Start() {
	*r.trace = append(*r.trace, "image_job_worker")
}

func TestStartPromptAuditBeforeImageJobsOrdersAndGatesWorkerStartup(t *testing.T) {
	t.Run("ready prompt audit starts image worker afterwards", func(t *testing.T) {
		trace := []string{}
		err := startPromptAuditBeforeImageJobs(
			context.Background(),
			true,
			recordingPromptAuditRuntime{trace: &trace},
			recordingImageJobWorkerRuntime{trace: &trace},
		)
		require.NoError(t, err)
		require.Equal(t, []string{"prompt_audit", "image_job_worker"}, trace)
	})

	t.Run("non blocking startup failure keeps image worker stopped", func(t *testing.T) {
		trace := []string{}
		startErr := errors.New("prompt audit unavailable")
		err := startPromptAuditBeforeImageJobs(
			context.Background(),
			true,
			recordingPromptAuditRuntime{trace: &trace, err: startErr, mode: securityaudit.ModeOff},
			recordingImageJobWorkerRuntime{trace: &trace},
		)
		require.ErrorIs(t, err, startErr)
		require.Equal(t, []string{"prompt_audit"}, trace)
	})

	t.Run("fail closed startup failure permits guarded replay", func(t *testing.T) {
		trace := []string{}
		startErr := errors.New("prompt audit config unavailable")
		err := startPromptAuditBeforeImageJobs(
			context.Background(),
			true,
			recordingPromptAuditRuntime{trace: &trace, err: startErr, mode: securityaudit.ModeBlocking},
			recordingImageJobWorkerRuntime{trace: &trace},
		)
		require.ErrorIs(t, err, startErr)
		require.Equal(t, []string{"prompt_audit", "image_job_worker"}, trace)
	})

	t.Run("disabled image jobs only starts prompt audit", func(t *testing.T) {
		trace := []string{}
		require.NoError(t, startPromptAuditBeforeImageJobs(
			context.Background(),
			false,
			recordingPromptAuditRuntime{trace: &trace},
			recordingImageJobWorkerRuntime{trace: &trace},
		))
		require.Equal(t, []string{"prompt_audit"}, trace)
	})
}

func TestProvideCleanup_WithMinimalDependencies_NoPanic(t *testing.T) {
	cfg := &config.Config{}

	oauthSvc := service.NewOAuthService(nil, nil)
	openAIOAuthSvc := service.NewOpenAIOAuthService(nil, nil)
	geminiOAuthSvc := service.NewGeminiOAuthService(nil, nil, nil, nil, cfg)
	antigravityOAuthSvc := service.NewAntigravityOAuthService(nil)

	tokenRefreshSvc := service.NewTokenRefreshService(
		nil,
		oauthSvc,
		openAIOAuthSvc,
		geminiOAuthSvc,
		antigravityOAuthSvc,
		nil,
		nil,
		cfg,
		nil,
	)
	accountExpirySvc := service.NewAccountExpiryService(nil, time.Second)
	proxyExpirySvc := service.NewProxyExpiryService(nil, time.Second)
	subscriptionExpirySvc := service.NewSubscriptionExpiryService(nil, time.Second)
	pricingSvc := service.NewPricingService(cfg, nil)
	emailQueueSvc := service.NewEmailQueueService(nil, 1)
	billingCacheSvc := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	idempotencyCleanupSvc := service.NewIdempotencyCleanupService(nil, cfg)
	schedulerSnapshotSvc := service.NewSchedulerSnapshotService(nil, nil, nil, nil, cfg)
	opsSystemLogSinkSvc := service.NewOpsSystemLogSink(nil)

	cleanup := provideCleanup(
		nil, // entClient
		nil, // redis
		&service.OpsMetricsCollector{},
		&service.OpsAggregationService{},
		&service.OpsAlertEvaluatorService{},
		&service.OpsCleanupService{},
		&service.OpsScheduledReportService{},
		opsSystemLogSinkSvc,
		nil, // groupRateSchedule
		schedulerSnapshotSvc,
		tokenRefreshSvc,
		accountExpirySvc,
		proxyExpirySvc,
		subscriptionExpirySvc,
		&service.UsageCleanupService{},
		idempotencyCleanupSvc,
		&service.BatchImageCleanupService{},
		nil, // batchImageWorker
		nil, // openAIImageJobWorker
		pricingSvc,
		emailQueueSvc,
		billingCacheSvc,
		&service.UsageRecordWorkerPool{},
		&service.SubscriptionService{},
		oauthSvc,
		openAIOAuthSvc,
		geminiOAuthSvc,
		antigravityOAuthSvc,
		nil, // grokOAuth
		nil, // openAIGateway
		nil, // scheduledTestRunner
		nil, // backupSvc
		nil, // paymentOrderExpiry
		nil, // channelMonitorRunner
		nil, // quotaFlusher
		nil, // upstreamBillingProbe
		nil, // auditLog
		nil, // promptAudit
	)

	require.NotPanics(t, func() {
		cleanup()
	})
}
