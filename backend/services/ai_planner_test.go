package services

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"study_plan_backend/db"
	"study_plan_backend/models"
)

func setupPlannerTestDB(t *testing.T) {
	t.Helper()
	connection, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	db.DB = connection
	if err := connection.AutoMigrate(&models.Plan{}, &models.DailyTask{}, &models.DailyCheckin{}, &models.StudySession{}, &models.PostponeRecord{}); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizePlanGenerationInputValidationAndCapacityWarning(t *testing.T) {
	tests := []PlanGenerationInput{
		{Goal: " ", Days: 1, HoursPerDay: 1},
		{Goal: "x", Days: 31, HoursPerDay: 1},
		{Goal: "x", Days: 1, HoursPerDay: 13},
		{Goal: "x", Days: 1, HoursPerDay: 1, StartDate: "2026/08/01"},
		{Goal: "x", Days: 1, HoursPerDay: 1, AvailableTimeSlot: "21:00-20:00"},
		{Goal: "x", Days: 1, HoursPerDay: 1, SkipDates: []string{"tomorrow"}},
	}
	for _, input := range tests {
		if _, _, err := NormalizePlanGenerationInput(input); err == nil {
			t.Fatalf("expected invalid input to fail: %+v", input)
		}
	}
	tooManySkipDates := make([]string, maxPlanningSkipDates+1)
	for index := range tooManySkipDates {
		tooManySkipDates[index] = "2026-08-01"
	}
	if _, _, err := NormalizePlanGenerationInput(PlanGenerationInput{Goal: "x", Days: 1, HoursPerDay: 1, SkipDates: tooManySkipDates}); err == nil {
		t.Fatal("oversized skip_dates must be rejected before deduplication")
	}
	normalized, warnings, err := NormalizePlanGenerationInput(PlanGenerationInput{Goal: " Study Go ", Days: 2, HoursPerDay: 3, StartDate: "2026-08-01", AvailableTimeSlot: "20:00-21:00"})
	if err != nil || normalized.Goal != "Study Go" || len(warnings) != 1 {
		t.Fatalf("expected normalized input and capacity warning: input=%+v warnings=%v err=%v", normalized, warnings, err)
	}
}

func TestLocalPlanningStageTemplatesAndProfilePacing(t *testing.T) {
	cases := map[string]string{
		"学习 Go":   "夯实基础",
		"阅读一本书":   "明确范围",
		"准备资格考试":  "诊断摸底",
		"开发项目并上线": "澄清需求",
		"改善时间管理":  "理解目标",
	}
	for goal, firstStage := range cases {
		ctx := PlanningContext{Input: PlanGenerationInput{Goal: goal, Days: 5, HoursPerDay: 2, StartDate: "2026-08-01", AvailableTimeSlot: "19:00-21:00"}, LearningProfile: LearningProfile{CompletionRate: 0.9}}
		preview, err := BuildLocalPlan(ctx)
		if err != nil {
			t.Fatalf("%s: %v", goal, err)
		}
		if !strings.Contains(preview.Tasks[0].Title, firstStage) || !strings.Contains(preview.Tasks[len(preview.Tasks)-1].Title, planningStageTemplates[classifyPlanningGoal(goal)][len(planningStageTemplates[classifyPlanningGoal(goal)])-1].Name) {
			t.Fatalf("%s did not progress through template: %+v", goal, preview.Tasks)
		}
	}
	high, _ := BuildLocalPlan(PlanningContext{Input: PlanGenerationInput{Goal: "学习 Go", Days: 2, HoursPerDay: 2, StartDate: "2026-08-01", AvailableTimeSlot: "19:00-21:00"}, LearningProfile: LearningProfile{CompletionRate: 0.9}})
	low, _ := BuildLocalPlan(PlanningContext{Input: PlanGenerationInput{Goal: "学习 Go", Days: 2, HoursPerDay: 2, StartDate: "2026-08-01", AvailableTimeSlot: "19:00-21:00"}, LearningProfile: LearningProfile{CompletionRate: 0.4}})
	if low.Tasks[0].EstimatedMinutes >= high.Tasks[0].EstimatedMinutes || low.EstimatedTotalHours >= high.EstimatedTotalHours {
		t.Fatalf("low completion must reduce pacing: high=%+v low=%+v", high, low)
	}
	cadence, _ := BuildLocalPlan(PlanningContext{Input: PlanGenerationInput{Goal: "学习 Go", Days: 7, HoursPerDay: 1, StartDate: "2026-08-01", AvailableTimeSlot: "20:00-21:00"}})
	if !strings.Contains(cadence.Tasks[4].Title, "复盘缓冲") || !strings.Contains(cadence.Tasks[6].Title, "复盘巩固") {
		t.Fatalf("expected periodic and final review cadence: %+v", cadence.Tasks)
	}
}

func TestOneHourRequestIsNotReducedByProfile(t *testing.T) {
	for _, profile := range []LearningProfile{{CompletionRate: 0.4}, {CompletionRate: 0.7}, {AverageStudyMins: 40}} {
		ctx := PlanningContext{Input: PlanGenerationInput{HoursPerDay: 1, AvailableTimeSlot: "20:00-21:00"}, LearningProfile: profile}
		if got := planningTaskMinutes(ctx); got != 60 {
			t.Fatalf("one-hour request became %d minutes for profile %+v", got, profile)
		}
	}
	ctx := PlanningContext{Input: PlanGenerationInput{HoursPerDay: 2, AvailableTimeSlot: "19:00-21:00"}, LearningProfile: LearningProfile{CompletionRate: 0.4}}
	if got := planningTaskMinutes(ctx); got != 80 {
		t.Fatalf("larger workload should retain adaptive pacing, got %d", got)
	}
}

func TestLocalPlanIntroductionStaysChineseDuringEnrichment(t *testing.T) {
	ctx := PlanningContext{Input: PlanGenerationInput{Goal: "学习 Go", Days: 1, HoursPerDay: 1, StartDate: "2026-08-01", AvailableTimeSlot: "20:00-21:00"}}
	local, err := BuildLocalPlan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(local.Summary, "day") || strings.Contains(local.Rationale, "Agent") {
		t.Fatalf("local introduction contains English copy: %+v", local)
	}
	enriched := local
	enriched.Summary = "English summary"
	enriched.Rationale = "English rationale"
	merged, err := MergePlanEnrichment(local, enriched)
	if err != nil {
		t.Fatal(err)
	}
	if merged.Summary != local.Summary || merged.Rationale != local.Rationale {
		t.Fatalf("enrichment replaced canonical Chinese introduction: %+v", merged)
	}
}

func TestLocalPlanningRepairsOccupancyAndHonorsSkipDates(t *testing.T) {
	ctx := PlanningContext{
		Input:     PlanGenerationInput{Goal: "学习 Go", Days: 2, HoursPerDay: 1, StartDate: "2026-08-01", AvailableTimeSlot: "20:00-21:00", SkipDates: []string{"2026-08-02"}},
		Occupancy: []PlanningOccupancy{{Date: "2026-08-01", PlannedStart: "20:00", PlannedEnd: "21:00"}},
	}
	preview, err := BuildLocalPlan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Tasks[0].Date != "2026-08-03" || preview.Tasks[1].Date != "2026-08-04" {
		t.Fatalf("expected occupied and skipped dates repaired: %+v", preview.Tasks)
	}
}

func TestLocalPlanningUsesAlternateSlotBeforeLaterDate(t *testing.T) {
	ctx := PlanningContext{
		Input:     PlanGenerationInput{Goal: "学习 Go", Days: 1, HoursPerDay: 1, StartDate: "2026-08-01", AvailableTimeSlot: "19:00-21:00"},
		Occupancy: []PlanningOccupancy{{Date: "2026-08-01", PlannedStart: "19:00", PlannedEnd: "20:00"}},
	}
	preview, err := BuildLocalPlan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Tasks[0].Date != "2026-08-01" || preview.Tasks[0].PlannedStart != "20:00" || preview.Tasks[0].PlannedEnd != "21:00" {
		t.Fatalf("expected alternate slot on preferred date: %+v", preview.Tasks[0])
	}
}

func TestLoadPlanningOccupancyExcludesCompletedTasks(t *testing.T) {
	setupPlannerTestDB(t)
	rows := []models.DailyTask{
		{UserID: 7, Date: "2026-08-01", PlannedStart: "09:00", PlannedEnd: "10:00", Status: models.TaskStatusPending},
		{UserID: 7, Date: "2026-08-02", PlannedStart: "09:00", PlannedEnd: "10:00", Status: models.TaskStatusCompleted},
	}
	if err := db.DB.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	occupancy, err := LoadPlanningOccupancy(7, "2026-08-01", 3)
	if err != nil || len(occupancy) != 1 || occupancy[0].Date != "2026-08-01" {
		t.Fatalf("unexpected occupancy: %+v err=%v", occupancy, err)
	}
}

func TestMergePlanEnrichmentCannotAlterSchedule(t *testing.T) {
	local := PlanPreview{Title: "Local", Summary: "Local summary", EstimatedTotalHours: 1, Rationale: "Local rationale", Tasks: []PlanPreviewTask{{Date: "2026-08-01", PlannedStart: "20:00", PlannedEnd: "21:00", EstimatedMinutes: 60, Title: "Local task", Objective: "Complete local exercise", Description: "Local description", Difficulty: "easy"}}}
	enriched := PlanPreview{Title: "Better", EstimatedTotalHours: 99, Tasks: []PlanPreviewTask{{Date: "2030-01-01", PlannedStart: "01:00", PlannedEnd: "05:00", EstimatedMinutes: 240, Title: "Better task", Objective: "Complete enriched exercise", Description: "Better description", Difficulty: "hard"}}}
	if _, err := MergePlanEnrichment(local, enriched); err == nil {
		t.Fatal("enrichment schedule mutation must be rejected")
	}
}

func TestMergePlanEnrichmentRejectsTaskReorder(t *testing.T) {
	local := PlanPreview{Tasks: []PlanPreviewTask{
		{Date: "2026-08-01", PlannedStart: "20:00", PlannedEnd: "21:00"},
		{Date: "2026-08-02", PlannedStart: "20:00", PlannedEnd: "21:00"},
	}}
	enriched := PlanPreview{Tasks: []PlanPreviewTask{local.Tasks[1], local.Tasks[0]}}
	if _, err := MergePlanEnrichment(local, enriched); err == nil {
		t.Fatal("enrichment task reorder must be rejected")
	}
}

func TestNormalizeCommittedPlanPreviewRecomputesAndSorts(t *testing.T) {
	preview := PlanPreview{Title: "Plan", EstimatedTotalHours: 99, Tasks: []PlanPreviewTask{
		{Date: "2026-08-02", PlannedStart: "20:00", PlannedEnd: "20:30", Title: "Second", Objective: "complete the second exercise", EstimatedMinutes: 999},
		{Date: "2026-08-01", PlannedStart: "19:00", PlannedEnd: "20:15", Title: "First", Objective: "complete the first exercise", EstimatedMinutes: 1},
	}}
	normalized, err := NormalizeCommittedPlanPreview(preview)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.Tasks[0].Date != "2026-08-01" || normalized.Tasks[0].EstimatedMinutes != 75 || normalized.Tasks[1].EstimatedMinutes != 30 || normalized.EstimatedTotalHours != 1.75 {
		t.Fatalf("derived commit values were not authoritative: %+v", normalized)
	}
}

func TestPlanningPromptIncludesSkeletonAggregateContextAndRefinement(t *testing.T) {
	ctx := PlanningContext{Input: PlanGenerationInput{Goal: "Study Go", Days: 1, HoursPerDay: 1, Refinement: "focus on concurrency"}, ActivePlanLoad: ActivePlanLoad{ActivePlanCount: 2}, Occupancy: []PlanningOccupancy{{Date: "2026-08-01"}}}
	local := PlanPreview{Title: "Study Go", Tasks: []PlanPreviewTask{{Date: "2026-08-02", PlannedStart: "20:00", PlannedEnd: "21:00", EstimatedMinutes: 60}}}
	prompt := BuildPlanningPrompt(ctx, local)
	for _, expected := range []string{"CANONICAL_LOCAL_SKELETON", "unfinished_occupancy_count", "focus on concurrency", "task count", "Simplified Chinese"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt omitted %q: %s", expected, prompt)
		}
	}
}
