package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"study_plan_backend/config"
	appdb "study_plan_backend/db"
	"study_plan_backend/models"
)

func validBlueprint() PlanningBlueprint {
	return PlanningBlueprint{
		Title: "Go 并发学习", Summary: "从模型到实践", Rationale: "先理解再练习",
		Stages: []PlanningBlueprintStage{{ID: "foundation", Name: "基础", Objective: "理解并发模型", Order: 1}},
		Tasks: []PlanningBlueprintTask{
			{ID: "concept", StageID: "foundation", Title: "理解 goroutine", Objective: "用示例解释 goroutine 调度", Description: "阅读并运行示例", EffortMinutes: 60, Difficulty: "easy", Order: 1},
			{ID: "practice", StageID: "foundation", Title: "实现 worker pool", Objective: "实现可停止的 worker pool", Description: "编写并验证代码", EffortMinutes: 90, Difficulty: "medium", Order: 2, PrerequisiteIDs: []string{"concept"}},
		},
	}
}

func TestPlanningBlueprintValidationSchedulingAndDynamicTokens(t *testing.T) {
	blueprint := validBlueprint()
	if err := ValidatePlanningBlueprint(blueprint); err != nil {
		t.Fatal(err)
	}
	ctx := PlanningContext{Input: PlanGenerationInput{Goal: "学习 Go 并发", Days: 7, HoursPerDay: 1, StartDate: "2026-08-01", AvailableTimeSlot: "20:00-21:00"}}
	preview, warnings, err := SchedulePlanningBlueprint(ctx, blueprint)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Tasks) != 3 || len(warnings) != 1 || preview.Tasks[1].Title == preview.Tasks[2].Title {
		t.Fatalf("oversized semantic task was not split and ordered: preview=%+v warnings=%+v", preview, warnings)
	}
	if PlanningBlueprintTokenAllowance(PlanGenerationInput{Days: 1}) >= PlanningBlueprintTokenAllowance(PlanGenerationInput{Days: 30}) {
		t.Fatal("blueprint token allowance did not scale with plan scope")
	}
	invalid := blueprint
	invalid.Tasks = append([]PlanningBlueprintTask(nil), blueprint.Tasks...)
	invalid.Tasks[0].PrerequisiteIDs = []string{"missing"}
	if ValidatePlanningBlueprint(invalid) == nil {
		t.Fatal("invalid prerequisite was accepted")
	}
}

func TestPlanningBlueprintPacksShortTasksOnSameDateAndRespectsOccupancy(t *testing.T) {
	blueprint := validBlueprint()
	blueprint.Tasks[0].EffortMinutes = 30
	blueprint.Tasks[1].EffortMinutes = 30
	ctx := PlanningContext{Input: PlanGenerationInput{Goal: "学习 Go 并发", Days: 7, HoursPerDay: 1, StartDate: "2026-08-01", AvailableTimeSlot: "20:00-21:00"}}
	preview, _, err := SchedulePlanningBlueprint(ctx, blueprint)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Tasks) != 2 || preview.Tasks[0].Date != preview.Tasks[1].Date || preview.Tasks[0].PlannedEnd != preview.Tasks[1].PlannedStart {
		t.Fatalf("short tasks were not packed safely on one date: %+v", preview.Tasks)
	}
	ctx.Occupancy = []PlanningOccupancy{{Date: "2026-08-01", PlannedStart: "20:00", PlannedEnd: "20:30"}}
	preview, _, err = SchedulePlanningBlueprint(ctx, blueprint)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Tasks[0].PlannedStart != "20:30" || preview.Tasks[1].Date == preview.Tasks[0].Date {
		t.Fatalf("occupancy was not repaired across eligible dates: %+v", preview.Tasks)
	}
	warnings := PlanningLoadWarnings(3, 55, 2, 3, 56)
	if len(warnings) != 2 {
		t.Fatalf("workload engine did not report both limits: %+v", warnings)
	}
}

type workerTestProvider struct {
	generate func(context.Context, string, int) (string, error)
}

func (p workerTestProvider) Test() error { return nil }
func (p workerTestProvider) Generate(prompt string, tokens int) (string, error) {
	return p.GenerateContext(context.Background(), prompt, tokens)
}
func (p workerTestProvider) GenerateContext(ctx context.Context, prompt string, tokens int) (string, error) {
	return p.generate(ctx, prompt, tokens)
}

func TestPlanningWorkerUsesBackgroundDeadlineAndPublishesVersion(t *testing.T) {
	database := planningJobsTestDB(t)
	if err := database.AutoMigrate(&models.Plan{}, &models.DailyTask{}, &models.DailyCheckin{}, &models.StudySession{}, &models.PostponeRecord{}, &models.AIGenerationUsage{}, &models.AIConfig{}); err != nil {
		t.Fatal(err)
	}
	appdb.DB = database
	config.App = &config.Config{JWTSecret: "planning-worker-test-secret"}
	originalCurrent, originalNew := currentPlanningProvider, newPlanningProvider
	t.Cleanup(func() { currentPlanningProvider, newPlanningProvider = originalCurrent, originalNew })
	currentPlanningProvider = func(context.Context) (models.AIConfig, AIProvider, error) {
		return models.AIConfig{Provider: AIProviderSiliconFlow, ModelName: "test", Enabled: true, DailyGenerationLimit: 5, BackgroundJobTimeoutSeconds: 60}, nil, nil
	}
	blueprintJSON, _ := json.Marshal(validBlueprint())
	newPlanningProvider = func(models.AIConfig) AIProvider {
		return workerTestProvider{generate: func(ctx context.Context, prompt string, tokens int) (string, error) {
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) < 55*time.Second {
				t.Fatalf("background deadline was truncated: ok=%v remaining=%v", ok, time.Until(deadline))
			}
			return string(blueprintJSON), nil
		}}
	}
	now := time.Now().UTC()
	baseline := PlanPreview{Title: "local", Tasks: []PlanPreviewTask{{Identity: "base", Date: "2026-08-01", PlannedStart: "20:00", PlannedEnd: "21:00", Title: "local", Objective: "complete local task", EstimatedMinutes: 60}}}
	baselineJSON, _ := json.Marshal(baseline)
	version := models.PlanningPreviewVersion{PreviewID: "44444444444444444444444444444444", Version: 1, UserID: 9, Source: "local", ContextFingerprint: "fingerprint", PreviewJSON: string(baselineJSON), ProvenanceToken: "local-token", ExpiresAt: now.Add(time.Hour)}
	if err := database.Create(&version).Error; err != nil {
		t.Fatal(err)
	}
	requestJSON, _ := json.Marshal(PlanGenerationInput{Goal: "学习 Go 并发", Days: 7, HoursPerDay: 1, StartDate: "2026-08-01", AvailableTimeSlot: "20:00-21:00"})
	job := models.PlanningJob{ID: "55555555555555555555555555555555", UserID: 9, RequestFingerprint: "fingerprint", Status: models.PlanningJobStatusQueued, Phase: models.PlanningJobStatusQueued, BaselinePreviewID: version.PreviewID, BaselinePreviewVersion: 1, RequestJSON: string(requestJSON), MaxAttempts: 2, BackgroundBudgetSeconds: 60, ExpiresAt: now.Add(time.Hour)}
	if err := database.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	claimed, err := ClaimPlanningJobs(context.Background(), database, "worker-test", 1, time.Minute)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim failed: jobs=%+v err=%v", claimed, err)
	}
	processPlanningJob(database, "worker-test", claimed[0])
	var finished models.PlanningJob
	database.First(&finished, "id = ?", job.ID)
	if finished.Status != models.PlanningJobStatusReady || finished.ResultPreviewVersion == nil || *finished.ResultPreviewVersion != 2 {
		t.Fatalf("worker did not publish ready version: %+v", finished)
	}
	var published models.PlanningPreviewVersion
	if err := database.Where("preview_id = ? AND version = 2", version.PreviewID).First(&published).Error; err != nil || published.Source != "ai_decomposed" {
		t.Fatalf("AI preview version missing: version=%+v err=%v", published, err)
	}
}
