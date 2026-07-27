package handlers

import (
	"context"

	"study_plan_backend/db"
	"study_plan_backend/models"
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
	} else if raw, generateErr := planningJobProvider(provider, cfg).GenerateContext(services.WithAIQuota(workContext, input.UserID, cfg.Provider, cfg.DailyGenerationLimit, &result.UsedToday), services.BuildPlanningBlueprintPrompt(planningContext), services.PlanningBlueprintTokenAllowance(input)); generateErr != nil {
		result.EnrichmentStatus, result.EnrichmentReason = classifyEnrichmentError(generateErr)
	} else if blueprint, parseErr := services.ParsePlanningBlueprintJSON(raw); parseErr != nil || services.ValidatePlanningBlueprint(blueprint) != nil {
		result.EnrichmentStatus, result.EnrichmentReason = "invalid_output", "invalid_provider_output"
	} else if decomposed, _, scheduleErr := services.SchedulePlanningBlueprint(planningContext, blueprint); scheduleErr != nil || services.ValidatePlanPreview(decomposed, planningContext.Input) != nil || validateAIPreviewSchedule(db.DB.WithContext(workContext), input.UserID, decomposed) != nil {
		result.EnrichmentStatus, result.EnrichmentReason = "invalid_output", "invalid_provider_output"
	} else {
		result.Preview = decomposed
		result.Source, result.EnrichmentStatus, result.EnrichmentReason = "ai_decomposed", "success", ""
	}
	return result, nil
}

func planningJobProvider(provider services.AIProvider, cfg models.AIConfig) services.AIProvider {
	if _, ok := provider.(*services.OpenAICompatibleProvider); !ok {
		return provider
	}
	cfg.RequestTimeoutSeconds = cfg.BackgroundJobTimeoutSeconds
	return services.NewAIProvider(cfg)
}
