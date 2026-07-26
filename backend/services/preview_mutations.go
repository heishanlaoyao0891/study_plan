package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"study_plan_backend/models"
)

type PreviewMutation struct {
	Operation           string          `json:"operation"`
	TaskIdentity        string          `json:"task_identity,omitempty"`
	Task                PlanPreviewTask `json:"task,omitempty"`
	InsertAfterIdentity string          `json:"insert_after_identity,omitempty"`
	OrderedIdentities   []string        `json:"ordered_identities,omitempty"`
	FirstPartMinutes    int             `json:"first_part_minutes,omitempty"`
}

type DerivedPreviewVersion struct {
	Preview PlanPreview
	Version models.PlanningPreviewVersion
}

func CreateDerivedPreviewVersion(ctx context.Context, database *gorm.DB, userID uint, previewID string, baseVersion int, mutation PreviewMutation) (DerivedPreviewVersion, error) {
	var base models.PlanningPreviewVersion
	if err := database.WithContext(ctx).Where("preview_id = ? AND version = ? AND user_id = ?", previewID, baseVersion, userID).First(&base).Error; err != nil {
		return DerivedPreviewVersion{}, err
	}
	if base.CommittedAt != nil || time.Now().UTC().After(base.ExpiresAt) {
		return DerivedPreviewVersion{}, errors.New("preview version is stale, expired, or committed")
	}
	var preview PlanPreview
	var input PlanGenerationInput
	if err := json.Unmarshal([]byte(base.PreviewJSON), &preview); err != nil {
		return DerivedPreviewVersion{}, err
	}
	if err := json.Unmarshal([]byte(base.InputJSON), &input); err != nil {
		return DerivedPreviewVersion{}, err
	}
	if err := applyPreviewMutation(&preview, mutation); err != nil {
		return DerivedPreviewVersion{}, err
	}
	if err := AssignPlanTaskIdentities(&preview); err != nil {
		return DerivedPreviewVersion{}, err
	}
	if err := ValidatePlanPreview(preview, input); err != nil {
		return DerivedPreviewVersion{}, err
	}
	if err := validatePreviewAgainstPersistedOccupancy(ctx, database, userID, preview); err != nil {
		return DerivedPreviewVersion{}, err
	}
	previewJSON, _ := json.Marshal(preview)
	for attempt := 0; attempt < 4; attempt++ {
		var created models.PlanningPreviewVersion
		err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var maxVersion int
			if err := tx.Model(&models.PlanningPreviewVersion{}).Where("preview_id = ? AND user_id = ?", previewID, userID).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion).Error; err != nil {
				return err
			}
			nextVersion := maxVersion + 1
			provenance, err := SignPlanVersionProvenance(userID, base.Source, previewID, nextVersion, base.ContextFingerprint, base.ExpiresAt, preview)
			if err != nil {
				return err
			}
			created = models.PlanningPreviewVersion{PreviewID: previewID, Version: nextVersion, UserID: userID, Source: base.Source, ContextFingerprint: base.ContextFingerprint, ParentVersion: &baseVersion, PreviewJSON: string(previewJSON), InputJSON: base.InputJSON, ProvenanceToken: provenance, ExpiresAt: base.ExpiresAt}
			return tx.Create(&created).Error
		})
		if err == nil {
			return DerivedPreviewVersion{Preview: preview, Version: created}, nil
		}
		if !isPlanningUniqueConflict(err) {
			return DerivedPreviewVersion{}, err
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
	return DerivedPreviewVersion{}, errors.New("preview version changed concurrently")
}

func applyPreviewMutation(preview *PlanPreview, mutation PreviewMutation) error {
	index := -1
	for current := range preview.Tasks {
		if preview.Tasks[current].Identity == mutation.TaskIdentity {
			index = current
			break
		}
	}
	switch mutation.Operation {
	case "add":
		mutation.Task.Identity = ""
		insertAt := len(preview.Tasks)
		if mutation.InsertAfterIdentity != "" {
			insertAt = -1
			for current := range preview.Tasks {
				if preview.Tasks[current].Identity == mutation.InsertAfterIdentity {
					insertAt = current + 1
					break
				}
			}
			if insertAt < 0 {
				return errors.New("insert_after_identity not found")
			}
		}
		preview.Tasks = append(preview.Tasks, PlanPreviewTask{})
		copy(preview.Tasks[insertAt+1:], preview.Tasks[insertAt:])
		preview.Tasks[insertAt] = mutation.Task
	case "remove":
		if index < 0 {
			return errors.New("task_identity not found")
		}
		if len(preview.Tasks) == 1 {
			return errors.New("preview must retain at least one task")
		}
		preview.Tasks = append(preview.Tasks[:index], preview.Tasks[index+1:]...)
	case "split":
		if index < 0 {
			return errors.New("task_identity not found")
		}
		task := preview.Tasks[index]
		if mutation.FirstPartMinutes < 15 || task.EstimatedMinutes-mutation.FirstPartMinutes < 15 {
			return errors.New("split parts must each be at least 15 minutes")
		}
		interval, err := ParseScheduleRange(task.PlannedStart, task.PlannedEnd)
		if err != nil {
			return err
		}
		boundary := interval.Start + mutation.FirstPartMinutes
		first, second := task, task
		first.Title += "（1/2）"
		first.PlannedEnd = formatPlanningMinute(boundary)
		first.EstimatedMinutes = mutation.FirstPartMinutes
		second.Title += "（2/2）"
		second.PlannedStart = formatPlanningMinute(boundary)
		second.EstimatedMinutes = task.EstimatedMinutes - mutation.FirstPartMinutes
		preview.Tasks[index] = first
		preview.Tasks = append(preview.Tasks, PlanPreviewTask{})
		copy(preview.Tasks[index+2:], preview.Tasks[index+1:])
		preview.Tasks[index+1] = second
	case "reorder":
		if len(mutation.OrderedIdentities) != len(preview.Tasks) {
			return errors.New("ordered_identities must contain every task")
		}
		byIdentity := map[string]PlanPreviewTask{}
		for _, task := range preview.Tasks {
			byIdentity[task.Identity] = task
		}
		ordered := make([]PlanPreviewTask, 0, len(preview.Tasks))
		for _, identity := range mutation.OrderedIdentities {
			task, exists := byIdentity[identity]
			if !exists {
				return errors.New("ordered_identities contains an unknown or duplicate task")
			}
			ordered = append(ordered, task)
			delete(byIdentity, identity)
		}
		preview.Tasks = ordered
	default:
		return errors.New("operation must be add, remove, split, or reorder")
	}
	recomputePlanTotals(preview)
	return nil
}

func validatePreviewAgainstPersistedOccupancy(ctx context.Context, database *gorm.DB, userID uint, preview PlanPreview) error {
	dates := make([]string, 0, len(preview.Tasks))
	seen := map[string]bool{}
	for _, task := range preview.Tasks {
		if !seen[task.Date] {
			dates = append(dates, task.Date)
			seen[task.Date] = true
		}
	}
	var persisted []models.DailyTask
	if err := database.WithContext(ctx).Where("user_id = ? AND date IN ? AND status <> ?", userID, dates, models.TaskStatusCompleted).Find(&persisted).Error; err != nil {
		return err
	}
	for _, candidate := range preview.Tasks {
		candidateRange, _ := ParseScheduleRange(candidate.PlannedStart, candidate.PlannedEnd)
		for _, existing := range persisted {
			if existing.Date != candidate.Date {
				continue
			}
			existingRange, err := ParseScheduleRange(existing.PlannedStart, existing.PlannedEnd)
			if err == nil && ScheduleIntervalsOverlap(candidateRange, existingRange) {
				return fmt.Errorf("task conflicts with persisted occupancy on %s", candidate.Date)
			}
		}
	}
	sort.SliceStable(preview.Tasks, func(i, j int) bool { return preview.Tasks[i].Date < preview.Tasks[j].Date })
	return nil
}
