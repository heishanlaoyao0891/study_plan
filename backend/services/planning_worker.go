package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"study_plan_backend/models"
)

var planningWorkerOnce sync.Once
var currentPlanningProvider = CurrentAIProviderContext
var newPlanningProvider = NewAIProvider

func StartPlanningJobWorker(database *gorm.DB) {
	if database == nil {
		return
	}
	planningWorkerOnce.Do(func() {
		owner, err := randomPlanningID()
		if err != nil {
			owner = "planning-worker"
		}
		go runPlanningWorker(database, owner)
	})
}

func runPlanningWorker(database *gorm.DB, owner string) {
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = ExpirePlanningJobs(ctx, database, time.Now().UTC())
		jobs, err := ClaimPlanningJobs(ctx, database, owner, 2, defaultPlanningLease)
		cancel()
		if err == nil {
			for index := range jobs {
				job := jobs[index]
				go processPlanningJob(database, owner, job)
			}
		}
		<-ticker.C
	}
}

func processPlanningJob(database *gorm.DB, owner string, job models.PlanningJob) {
	budget := boundedBackgroundBudget(job.BackgroundBudgetSeconds)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(budget)*time.Second)
	defer cancel()
	ctx = WithAIInvocationContext(ctx, AIInvocationContext{UserID: job.UserID, JobType: "planning_preview", JobID: job.ID, Phase: "decomposing"})
	stopHeartbeat := make(chan struct{})
	defer close(stopHeartbeat)
	go planningJobHeartbeat(ctx, cancel, database, owner, job.ID, stopHeartbeat)

	phase := models.PlanningJobStatusDecomposing
	fallback := func(reason string) {
		if reason == "cancelled" {
			_, _ = TransitionPlanningJob(context.Background(), database, job.ID, owner, phase, models.PlanningJobStatusCancelled, "")
			return
		}
		_, _ = TransitionPlanningJob(context.Background(), database, job.ID, owner, phase, models.PlanningJobStatusFallback, reason)
	}
	var input PlanGenerationInput
	if err := json.Unmarshal([]byte(job.RequestJSON), &input); err != nil {
		fallback("invalid_persisted_request")
		return
	}
	input.UserID = job.UserID
	planningContext, err := BuildPlanningContextWithContext(ctx, input)
	if err != nil {
		fallback(planningWorkerFailureReason(ctx, err, "context_unavailable"))
		return
	}
	cfg, _, err := currentPlanningProvider(ctx)
	if err != nil || !cfg.Enabled || NormalizeAIProvider(cfg.Provider) == AIProviderMock {
		fallback("provider_configuration_unavailable")
		return
	}
	cfg.RequestTimeoutSeconds = budget
	provider := newPlanningProvider(cfg)
	used := int64(0)
	telemetry := &AIProviderTelemetry{}
	providerStarted := time.Now()
	providerContext := WithAIProviderTelemetry(WithAIQuota(ctx, job.UserID, cfg.Provider, cfg.DailyGenerationLimit, &used), telemetry)
	raw, err := provider.GenerateContext(providerContext, BuildPlanningBlueprintPrompt(planningContext), PlanningBlueprintTokenAllowance(input))
	providerLatency := time.Since(providerStarted)
	_ = database.WithContext(context.Background()).Model(&models.PlanningJob{}).Where("id = ? AND lease_owner = ?", job.ID, owner).Updates(map[string]any{
		"provider_latency_ms": providerLatency.Milliseconds(), "prompt_tokens": telemetry.PromptTokens,
		"completion_tokens": telemetry.CompletionTokens, "total_tokens": telemetry.TotalTokens,
		"phase_timings_json": fmt.Sprintf(`{"decomposition_ms":%d}`, providerLatency.Milliseconds()),
	}).Error
	if err != nil {
		fallback(planningWorkerFailureReason(ctx, err, "provider_request_failed"))
		return
	}
	blueprint, err := ParsePlanningBlueprintJSON(raw)
	if err != nil || ValidatePlanningBlueprint(blueprint) != nil {
		fallback("invalid_blueprint")
		return
	}
	transitioned, err := TransitionPlanningJob(ctx, database, job.ID, owner, phase, models.PlanningJobStatusScheduling, "")
	if err != nil || !transitioned {
		fallback("lease_lost")
		return
	}
	phase = models.PlanningJobStatusScheduling
	preview, _, err := SchedulePlanningBlueprint(planningContext, blueprint)
	if err != nil {
		fallback("blueprint_unschedulable")
		return
	}
	if err := AssignPlanTaskIdentities(&preview); err != nil {
		fallback("preview_identity_failed")
		return
	}
	provenance, err := SignPlanVersionProvenance(job.UserID, "ai_decomposed", job.BaselinePreviewID, 2, job.RequestFingerprint, job.ExpiresAt, preview)
	if err != nil {
		fallback("preview_provenance_failed")
		return
	}
	if err := publishPlanningJobResult(ctx, database, owner, job, preview, provenance); err != nil {
		fallback("preview_publication_failed")
	}
}

func planningJobHeartbeat(ctx context.Context, cancel context.CancelFunc, database *gorm.DB, owner, jobID string, stop <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			cancelled, err := PlanningJobCancellationRequested(context.Background(), database, jobID, owner)
			if err != nil || cancelled {
				cancel()
				return
			}
			ok, err := RenewPlanningJobLease(context.Background(), database, jobID, owner, defaultPlanningLease)
			if err != nil || !ok {
				cancel()
				return
			}
		}
	}
}

func publishPlanningJobResult(ctx context.Context, database *gorm.DB, owner string, job models.PlanningJob, preview PlanPreview, provenance string) error {
	previewJSON, err := json.Marshal(preview)
	if err != nil {
		return err
	}
	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current models.PlanningJob
		if err := tx.Where("id = ? AND lease_owner = ? AND status = ? AND cancel_requested_at IS NULL", job.ID, owner, models.PlanningJobStatusScheduling).First(&current).Error; err != nil {
			return err
		}
		var existing models.PlanningPreviewVersion
		err := tx.Where("preview_id = ? AND version = ? AND user_id = ?", job.BaselinePreviewID, 2, job.UserID).First(&existing).Error
		if err == nil {
			return tx.Model(&current).Updates(map[string]any{"status": models.PlanningJobStatusReady, "phase": models.PlanningJobStatusReady, "result_preview_version": 2, "finished_at": time.Now().UTC(), "lease_owner": "", "lease_expires_at": nil}).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		version := models.PlanningPreviewVersion{PreviewID: job.BaselinePreviewID, Version: 2, UserID: job.UserID, Source: "ai_decomposed", ContextFingerprint: job.RequestFingerprint, ParentVersion: &job.BaselinePreviewVersion, PreviewJSON: string(previewJSON), InputJSON: job.RequestJSON, ProvenanceToken: provenance, ExpiresAt: current.ExpiresAt}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		return tx.Model(&current).Updates(map[string]any{"status": models.PlanningJobStatusReady, "phase": models.PlanningJobStatusReady, "result_preview_version": 2, "finished_at": now, "lease_owner": "", "lease_expires_at": nil}).Error
	})
}

func planningWorkerFailureReason(ctx context.Context, err error, fallback string) string {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return "background_deadline_exceeded"
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	if errors.Is(err, ErrAIQuotaExceeded) {
		return "daily_generation_limit_reached"
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "http 429") || strings.Contains(message, "quota") {
		return "provider_quota_limited"
	}
	return fallback
}
