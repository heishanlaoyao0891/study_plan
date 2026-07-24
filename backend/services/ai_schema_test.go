package services

import (
	"encoding/json"
	"testing"
)

func TestValidatePlanPreviewRejectsSkippedDates(t *testing.T) {
	preview := PlanPreview{
		Title: "Study Go",
		Tasks: []PlanPreviewTask{{Date: "2026-07-21", Title: "Day 1", Objective: "Finish lesson one", PlannedStart: "20:00", PlannedEnd: "21:00", EstimatedMinutes: 60}},
	}
	input := PlanGenerationInput{SkipDates: []string{"2026-07-21"}}
	if err := ValidatePlanPreview(preview, input); err == nil {
		t.Fatal("expected validation error for skipped date")
	}
}

func TestValidatePlanPreviewRequiresConcreteObjective(t *testing.T) {
	base := PlanPreview{Title: "Study Go", Tasks: []PlanPreviewTask{{Date: "2026-07-21", Title: "Day 1", PlannedStart: "20:00", PlannedEnd: "21:00", EstimatedMinutes: 60}}}
	if err := ValidatePlanPreview(base, PlanGenerationInput{}); err == nil {
		t.Fatal("missing objective must fail")
	}
	base.Tasks[0].Objective = base.Tasks[0].Title
	if err := ValidatePlanPreview(base, PlanGenerationInput{}); err == nil {
		t.Fatal("repeated title objective must fail")
	}
	base.Tasks[0].Objective = "Complete examples from lesson one"
	if err := ValidatePlanPreview(base, PlanGenerationInput{}); err != nil {
		t.Fatalf("concrete objective should pass: %v", err)
	}
}

func TestFallbackPlanPreviewCapsAtThirtyDays(t *testing.T) {
	preview := FallbackPlanPreview(PlanningContext{Input: PlanGenerationInput{Goal: "Study Go", Days: 45, HoursPerDay: 2, StartDate: "2026-07-20"}})
	if len(preview.Tasks) > 30 {
		t.Fatalf("expected at most 30 tasks, got %d", len(preview.Tasks))
	}
}

func TestFallbackPlanPreviewUsesAvailableTimeSlot(t *testing.T) {
	preview := FallbackPlanPreview(PlanningContext{Input: PlanGenerationInput{Goal: "Study Go", Days: 1, HoursPerDay: 2, StartDate: "2026-07-20", AvailableTimeSlot: "09:00-11:00"}})
	if len(preview.Tasks) != 1 || preview.Tasks[0].PlannedStart != "09:00" || preview.Tasks[0].PlannedEnd != "11:00" || preview.Tasks[0].EstimatedMinutes != 120 {
		t.Fatalf("expected available slot to define task timing: %+v", preview.Tasks)
	}
}

func TestDecodeCompletionContentPreservesStructuredJSON(t *testing.T) {
	raw := json.RawMessage(`{"title":"Study Go","metadata":{"source":"model"}}`)
	content, err := decodeCompletionContent(raw)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(content), &decoded); err != nil || decoded["title"] != "Study Go" {
		t.Fatalf("structured content was not preserved: %q, %v", content, err)
	}
}
