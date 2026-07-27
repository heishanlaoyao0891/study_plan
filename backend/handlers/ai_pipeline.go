package handlers

import (
	"context"

	"study_plan_backend/db"
	"study_plan_backend/services"
)

type planningPipelineResult struct {
	Preview          services.PlanPreview
	Warnings         []string
	Source           string
	Provider         string
	Model            string
	EnrichmentStatus string
	EnrichmentReason string
	UsedToday        int64
	DailyLimit       int
}

func runPlanningPipeline(workContext context.Context, input services.PlanGenerationInput) (planningPipelineResult, error) {
	planningContext, err := services.BuildPlanningContextWithContext(workContext, input)
	if err != nil {
		return planningPipelineResult{}, err
	}
	preview, err := services.BuildLocalPlan(planningContext)
	if err != nil {
		return planningPipelineResult{}, err
	}
	if err := services.ValidatePlanPreview(preview, planningContext.Input); err != nil {
		return planningPipelineResult{}, err
	}
	if err := validateAIPreviewSchedule(db.DB.WithContext(workContext), input.UserID, preview); err != nil {
		return planningPipelineResult{}, err
	}
	result := planningPipelineResult{Preview: preview, Warnings: planningContext.Warnings, Source: "local", EnrichmentStatus: "disabled", EnrichmentReason: "provider_disabled"}
	cfg, provider, providerErr := currentAIProvider(workContext)
	if providerErr != nil {
		result.EnrichmentStatus, result.EnrichmentReason = "configuration_error", "provider_configuration_unavailable"
		return result, nil
	}
	result.Provider, result.Model, result.DailyLimit = cfg.Provider, cfg.ModelName, maxPositive(cfg.DailyGenerationLimit, 5)
	if !cfg.Enabled || services.NormalizeAIProvider(cfg.Provider) == services.AIProviderMock {
		return result, nil
	}
	if configErr := validateAIConfigContext(workContext, cfg, false); configErr != nil {
		result.EnrichmentStatus, result.EnrichmentReason = "configuration_error", "invalid_provider_configuration"
	} else if canUse, count, quotaErr := canUseAIGeneration(workContext, input.UserID, cfg.DailyGenerationLimit); quotaErr != nil {
		result.EnrichmentStatus, result.EnrichmentReason = "provider_error", "quota_check_failed"
	} else if !canUse {
		result.UsedToday = count
		result.EnrichmentStatus, result.EnrichmentReason = "quota_limited", "daily_enrichment_limit_reached"
	} else if raw, generateErr := generatePlanEnrichment(workContext, provider, input.UserID, cfg.Provider, cfg.DailyGenerationLimit, &result.UsedToday, services.BuildPlanningPrompt(planningContext, preview)); generateErr != nil {
		result.EnrichmentStatus, result.EnrichmentReason = classifyEnrichmentError(generateErr)
	} else if enriched, parseErr := services.ParsePlanPreviewJSON(raw); parseErr != nil {
		result.EnrichmentStatus, result.EnrichmentReason = "invalid_output", "invalid_provider_output"
	} else if merged, mergeErr := services.MergePlanEnrichment(preview, enriched); mergeErr != nil || services.ValidatePlanPreview(merged, planningContext.Input) != nil || validateAIPreviewSchedule(db.DB.WithContext(workContext), input.UserID, merged) != nil {
		result.EnrichmentStatus, result.EnrichmentReason = "invalid_output", "invalid_provider_output"
	} else {
		result.Preview = merged
		result.Source, result.EnrichmentStatus, result.EnrichmentReason = "local_enriched", "success", ""
	}
	return result, nil
}
