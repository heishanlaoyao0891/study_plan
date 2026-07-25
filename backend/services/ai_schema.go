package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type PlanValidationError struct {
	Message string
}

func (e PlanValidationError) Error() string { return e.Message }

func ParsePlanPreviewJSON(raw string) (PlanPreview, error) {
	var preview PlanPreview
	cleaned := strings.TrimSpace(raw)
	if cleaned == "" {
		return preview, errors.New("empty plan preview")
	}
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)
	if err := json.Unmarshal([]byte(cleaned), &preview); err != nil {
		return preview, err
	}
	return preview, nil
}

func RepairPlanPreviewJSON(raw string) (string, bool) {
	cleaned := strings.TrimSpace(raw)
	repaired := false
	if strings.HasPrefix(cleaned, "{") && strings.HasSuffix(cleaned, "}") {
		return cleaned, false
	}
	cleaned = strings.ReplaceAll(cleaned, "'", "\"")
	repaired = true
	return cleaned, repaired
}

func ValidatePlanPreview(preview PlanPreview, input PlanGenerationInput) error {
	if strings.TrimSpace(preview.Title) == "" {
		return PlanValidationError{Message: "plan title is required"}
	}
	if len(preview.Tasks) == 0 {
		return PlanValidationError{Message: "at least one task is required"}
	}
	if len(preview.Tasks) > defaultMaxPreviewDays {
		return PlanValidationError{Message: fmt.Sprintf("plan preview exceeds %d days", defaultMaxPreviewDays)}
	}
	skip := map[string]bool{}
	for _, date := range input.SkipDates {
		skip[date] = true
	}
	seen := map[string]bool{}
	availableStart, availableEnd, slotErr := parsePlanningSlot(input.AvailableTimeSlot)
	for _, task := range preview.Tasks {
		if strings.TrimSpace(task.Date) == "" {
			return PlanValidationError{Message: "task date is required"}
		}
		if _, err := time.Parse(aiPlanDateLayout, task.Date); err != nil {
			return PlanValidationError{Message: fmt.Sprintf("task date %s must use YYYY-MM-DD", task.Date)}
		}
		if skip[task.Date] {
			return PlanValidationError{Message: fmt.Sprintf("task falls on skipped date %s", task.Date)}
		}
		if seen[task.Date] {
			return PlanValidationError{Message: fmt.Sprintf("duplicate task date %s", task.Date)}
		}
		seen[task.Date] = true
		if task.EstimatedMinutes <= 0 {
			return PlanValidationError{Message: fmt.Sprintf("task %s must have positive estimated minutes", task.Date)}
		}
		if strings.TrimSpace(task.Title) == "" {
			return PlanValidationError{Message: fmt.Sprintf("task %s missing title", task.Date)}
		}
		objective := strings.TrimSpace(task.Objective)
		if objective == "" {
			return PlanValidationError{Message: fmt.Sprintf("task %s missing objective", task.Date)}
		}
		if strings.EqualFold(objective, strings.TrimSpace(task.Title)) {
			return PlanValidationError{Message: fmt.Sprintf("task %s objective must be more specific than title", task.Date)}
		}
		if len([]rune(objective)) > 500 {
			return PlanValidationError{Message: fmt.Sprintf("task %s objective is too long", task.Date)}
		}
		start, startErr := time.Parse("15:04", task.PlannedStart)
		end, endErr := time.Parse("15:04", task.PlannedEnd)
		if startErr != nil || endErr != nil || !end.After(start) {
			return PlanValidationError{Message: fmt.Sprintf("task %s has invalid planned time range", task.Date)}
		}
		plannedMinutes := int(end.Sub(start).Minutes())
		if task.EstimatedMinutes != plannedMinutes {
			return PlanValidationError{Message: fmt.Sprintf("task %s estimated minutes must match planned time range", task.Date)}
		}
		if slotErr == nil {
			taskStart, taskEnd := start.Hour()*60+start.Minute(), end.Hour()*60+end.Minute()
			if taskStart < availableStart || taskEnd > availableEnd {
				return PlanValidationError{Message: fmt.Sprintf("task %s falls outside available time slot", task.Date)}
			}
		}
	}
	return nil
}
