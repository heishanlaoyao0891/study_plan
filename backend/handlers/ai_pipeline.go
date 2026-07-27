package handlers

import (
	"context"
	"fmt"

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
	return runPlanningPipelineWithCheckpoint(workContext, input, nil, nil)
}

func runPlanningPipelineWithCheckpoint(workContext context.Context, input services.PlanGenerationInput, checkpoint *services.PlanningBlueprintCheckpoint, save func(services.PlanningBlueprintCheckpoint) error) (planningPipelineResult, error) {
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
	result := planningPipelineResult{Preview: preview, Warnings: planningContext.Warnings, Source: "local", EnrichmentStatus: "pending"}
	cfg, provider, providerErr := currentAIProvider(workContext)
	if providerErr != nil {
		return result, fmt.Errorf("provider configuration unavailable: %w", providerErr)
	}
	result.Provider, result.Model, result.DailyLimit = cfg.Provider, cfg.ModelName, maxPositive(cfg.DailyGenerationLimit, 5)
	if !cfg.Enabled || services.NormalizeAIProvider(cfg.Provider) == services.AIProviderMock {
		return result, fmt.Errorf("provider is disabled")
	}
	if configErr := validateAIConfigContext(workContext, cfg, false); configErr != nil {
		return result, fmt.Errorf("invalid provider configuration: %w", configErr)
	} else if canUse, count, quotaErr := canUseAIGeneration(workContext, input.UserID, cfg.DailyGenerationLimit); quotaErr != nil {
		return result, fmt.Errorf("quota check failed: %w", quotaErr)
	} else if !canUse {
		result.UsedToday = count
		return result, services.ErrAIQuotaExceeded
	}
	providerContext := services.WithAIQuota(workContext, input.UserID, cfg.Provider, cfg.DailyGenerationLimit, &result.UsedToday)
	blueprint, generateErr := services.GeneratePlanningBlueprintWithCheckpoint(providerContext, planningJobProvider(provider, cfg), planningContext, 4, checkpoint, save)
	if generateErr != nil {
		return result, generateErr
	}
	decomposed, repairWarnings, scheduleErr := services.SchedulePlanningBlueprint(planningContext, blueprint)
	if scheduleErr != nil {
		return result, fmt.Errorf("schedule ai blueprint: %w", scheduleErr)
	}
	if err := services.ValidatePlanPreview(decomposed, planningContext.Input); err != nil {
		return result, fmt.Errorf("validate ai preview: %w", err)
	}
	if err := validateAIPreviewSchedule(db.DB.WithContext(workContext), input.UserID, decomposed); err != nil {
		return result, fmt.Errorf("validate ai schedule: %w", err)
	}
	result.Preview = decomposed
	result.Warnings = append(result.Warnings, repairWarnings...)
	result.Source, result.EnrichmentStatus, result.EnrichmentReason = "ai_decomposed", "success", ""
	return result, nil
}

func planningJobProvider(provider services.AIProvider, cfg models.AIConfig) services.AIProvider {
	if _, ok := provider.(*services.OpenAICompatibleProvider); !ok {
		return provider
	}
	cfg.RequestTimeoutSeconds = cfg.BackgroundJobTimeoutSeconds
	return services.NewAIProvider(cfg)
}
