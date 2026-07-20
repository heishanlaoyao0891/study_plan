package services

import "testing"

func TestValidatePlanPreviewRejectsSkippedDates(t *testing.T) {
	preview := PlanPreview{
		Title: "Study Go",
		Tasks: []PlanPreviewTask{{Date: "2026-07-21", Title: "Day 1", EstimatedMinutes: 60}},
	}
	input := PlanGenerationInput{SkipDates: []string{"2026-07-21"}}
	if err := ValidatePlanPreview(preview, input); err == nil {
		t.Fatal("expected validation error for skipped date")
	}
}

func TestFallbackPlanPreviewCapsAtThirtyDays(t *testing.T) {
	preview := FallbackPlanPreview(PlanningContext{Input: PlanGenerationInput{Goal: "Study Go", Days: 45, HoursPerDay: 2, StartDate: "2026-07-20"}})
	if len(preview.Tasks) > 30 {
		t.Fatalf("expected at most 30 tasks, got %d", len(preview.Tasks))
	}
}
