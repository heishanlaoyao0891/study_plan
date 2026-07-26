package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"study_plan_backend/models"
)

const PlanningPreviewRetention = 30 * time.Minute
const PlanningJobRetention = 2 * time.Hour

type PlanningJobOptions struct {
	Enqueue                 bool
	Provider                string
	ModelName               string
	BackgroundBudgetSeconds int
}

type PersistedPlanningBaseline struct {
	Version models.PlanningPreviewVersion
	Job     *models.PlanningJob
	Preview PlanPreview
	Reused  bool
}

func PlanningContextFingerprint(planningContext PlanningContext) (string, error) {
	payload, err := json.Marshal(struct {
		Input           PlanGenerationInput `json:"input"`
		Occupancy       []PlanningOccupancy `json:"occupancy"`
		LearningProfile LearningProfile     `json:"learning_profile"`
		ActivePlanLoad  ActivePlanLoad      `json:"active_plan_load"`
	}{planningContext.Input, planningContext.Occupancy, planningContext.LearningProfile, planningContext.ActivePlanLoad})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func PersistPlanningBaseline(ctx context.Context, database *gorm.DB, userID uint, planningContext PlanningContext, preview PlanPreview, options PlanningJobOptions) (PersistedPlanningBaseline, error) {
	fingerprint, err := PlanningContextFingerprint(planningContext)
	if err != nil {
		return PersistedPlanningBaseline{}, err
	}
	if options.Enqueue {
		if existing, found, findErr := findActivePlanningBaseline(ctx, database, userID, fingerprint); findErr != nil {
			return PersistedPlanningBaseline{}, findErr
		} else if found {
			return existing, nil
		}
	}
	previewJSON, err := json.Marshal(preview)
	if err != nil {
		return PersistedPlanningBaseline{}, err
	}
	requestJSON, err := json.Marshal(planningContext.Input)
	if err != nil {
		return PersistedPlanningBaseline{}, err
	}
	previewID, err := randomPlanningID()
	if err != nil {
		return PersistedPlanningBaseline{}, err
	}
	now := time.Now().UTC()
	expiresAt := now.Add(PlanningPreviewRetention)
	provenanceToken, err := SignPlanVersionProvenance(userID, "local", previewID, 1, fingerprint, expiresAt, preview)
	if err != nil {
		return PersistedPlanningBaseline{}, err
	}
	version := models.PlanningPreviewVersion{
		PreviewID: previewID, Version: 1, UserID: userID, Source: "local",
		ContextFingerprint: fingerprint, PreviewJSON: string(previewJSON), InputJSON: string(requestJSON), ProvenanceToken: provenanceToken,
		ExpiresAt: expiresAt, CreatedAt: now,
	}
	result := PersistedPlanningBaseline{Version: version, Preview: preview}
	err = database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		result.Version = version
		if !options.Enqueue {
			return nil
		}
		jobID, idErr := randomPlanningID()
		if idErr != nil {
			return idErr
		}
		job := models.PlanningJob{
			ID: jobID, UserID: userID, RequestFingerprint: fingerprint,
			Status: models.PlanningJobStatusQueued, Phase: models.PlanningJobStatusQueued,
			Provider: options.Provider, ModelName: options.ModelName,
			BaselinePreviewID: previewID, BaselinePreviewVersion: 1,
			RequestJSON: string(requestJSON), MaxAttempts: 2,
			BackgroundBudgetSeconds: boundedBackgroundBudget(options.BackgroundBudgetSeconds),
			ExpiresAt:               now.Add(PlanningJobRetention), CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
		result.Job = &job
		return nil
	})
	if err == nil {
		return result, nil
	}
	if options.Enqueue && isPlanningUniqueConflict(err) {
		if existing, found, findErr := findActivePlanningBaseline(ctx, database, userID, fingerprint); findErr == nil && found {
			return existing, nil
		}
	}
	return PersistedPlanningBaseline{}, err
}

func findActivePlanningBaseline(ctx context.Context, database *gorm.DB, userID uint, fingerprint string) (PersistedPlanningBaseline, bool, error) {
	var job models.PlanningJob
	err := database.WithContext(ctx).
		Where("user_id = ? AND request_fingerprint = ? AND status IN ?", userID, fingerprint, []string{models.PlanningJobStatusQueued, models.PlanningJobStatusDecomposing, models.PlanningJobStatusScheduling}).
		Order("created_at DESC").First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return PersistedPlanningBaseline{}, false, nil
	}
	if err != nil {
		return PersistedPlanningBaseline{}, false, err
	}
	var version models.PlanningPreviewVersion
	if err := database.WithContext(ctx).Where("preview_id = ? AND version = ? AND user_id = ?", job.BaselinePreviewID, job.BaselinePreviewVersion, userID).First(&version).Error; err != nil {
		return PersistedPlanningBaseline{}, false, err
	}
	var preview PlanPreview
	if err := json.Unmarshal([]byte(version.PreviewJSON), &preview); err != nil {
		return PersistedPlanningBaseline{}, false, fmt.Errorf("decode persisted planning preview: %w", err)
	}
	return PersistedPlanningBaseline{Version: version, Job: &job, Preview: preview, Reused: true}, true, nil
}

func boundedBackgroundBudget(value int) int {
	if value == 0 {
		return 60
	}
	if value < 15 {
		return 15
	}
	if value > 120 {
		return 120
	}
	return value
}

func randomPlanningID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func isPlanningUniqueConflict(err error) bool {
	message := strings.ToLower(err.Error())
	return errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(message, "unique constraint") || strings.Contains(message, "database is locked") || strings.Contains(message, "database table is locked") || strings.Contains(message, "sqlite_busy")
}
