package services

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"study_plan_backend/models"
)

const defaultPlanningLease = 30 * time.Second

func ClaimPlanningJobs(ctx context.Context, database *gorm.DB, owner string, limit int, lease time.Duration) ([]models.PlanningJob, error) {
	if limit < 1 {
		return nil, nil
	}
	if limit > 8 {
		limit = 8
	}
	if lease <= 0 {
		lease = defaultPlanningLease
	}
	now := time.Now().UTC()
	if err := RecoverAbandonedPlanningJobs(ctx, database, now); err != nil {
		return nil, err
	}
	var candidates []models.PlanningJob
	if err := database.WithContext(ctx).
		Where("status = ? AND expires_at > ? AND cancel_requested_at IS NULL AND attempt_count < max_attempts", models.PlanningJobStatusQueued, now).
		Order("created_at ASC").Limit(limit * 2).Find(&candidates).Error; err != nil {
		return nil, err
	}
	claimed := make([]models.PlanningJob, 0, limit)
	for _, candidate := range candidates {
		if len(claimed) >= limit {
			break
		}
		leaseExpiry := now.Add(lease)
		updates := map[string]any{
			"status": models.PlanningJobStatusDecomposing, "phase": models.PlanningJobStatusDecomposing,
			"lease_owner": owner, "lease_expires_at": leaseExpiry,
			"attempt_count": gorm.Expr("attempt_count + 1"), "started_at": gorm.Expr("COALESCE(started_at, ?)", now),
		}
		result := database.WithContext(ctx).Model(&models.PlanningJob{}).
			Where("id = ? AND status = ? AND attempt_count < max_attempts AND cancel_requested_at IS NULL", candidate.ID, models.PlanningJobStatusQueued).
			Updates(updates)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected != 1 {
			continue
		}
		var job models.PlanningJob
		if err := database.WithContext(ctx).First(&job, "id = ?", candidate.ID).Error; err != nil {
			return nil, err
		}
		claimed = append(claimed, job)
	}
	return claimed, nil
}

func RecoverAbandonedPlanningJobs(ctx context.Context, database *gorm.DB, now time.Time) error {
	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.PlanningJob{}).
			Where("status IN ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ? AND attempt_count >= max_attempts", []string{models.PlanningJobStatusDecomposing, models.PlanningJobStatusScheduling}, now).
			Updates(map[string]any{"status": models.PlanningJobStatusFallback, "phase": models.PlanningJobStatusFallback, "failure_reason": "worker_attempts_exhausted", "lease_owner": "", "lease_expires_at": nil, "finished_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&models.PlanningJob{}).
			Where("status IN ? AND lease_expires_at IS NOT NULL AND lease_expires_at <= ? AND attempt_count < max_attempts", []string{models.PlanningJobStatusDecomposing, models.PlanningJobStatusScheduling}, now).
			Updates(map[string]any{"status": models.PlanningJobStatusQueued, "phase": models.PlanningJobStatusQueued, "lease_owner": "", "lease_expires_at": nil}).Error
	})
}

func RenewPlanningJobLease(ctx context.Context, database *gorm.DB, jobID, owner string, lease time.Duration) (bool, error) {
	if lease <= 0 {
		lease = defaultPlanningLease
	}
	result := database.WithContext(ctx).Model(&models.PlanningJob{}).
		Where("id = ? AND lease_owner = ? AND status IN ? AND cancel_requested_at IS NULL", jobID, owner, []string{models.PlanningJobStatusDecomposing, models.PlanningJobStatusScheduling}).
		Update("lease_expires_at", time.Now().UTC().Add(lease))
	return result.RowsAffected == 1, result.Error
}

func TransitionPlanningJob(ctx context.Context, database *gorm.DB, jobID, owner, from, to, failureReason string) (bool, error) {
	updates := map[string]any{"status": to, "phase": to}
	if failureReason != "" {
		updates["failure_reason"] = failureReason
	}
	if to == models.PlanningJobStatusReady || to == models.PlanningJobStatusFallback || to == models.PlanningJobStatusCancelled || to == models.PlanningJobStatusExpired {
		updates["finished_at"] = time.Now().UTC()
		updates["lease_owner"] = ""
		updates["lease_expires_at"] = nil
	}
	query := database.WithContext(ctx).Model(&models.PlanningJob{}).Where("id = ? AND status = ?", jobID, from)
	if owner != "" {
		query = query.Where("lease_owner = ?", owner)
	}
	result := query.Updates(updates)
	return result.RowsAffected == 1, result.Error
}

func RequestPlanningJobCancellation(ctx context.Context, database *gorm.DB, userID uint, jobID string) (models.PlanningJob, error) {
	var job models.PlanningJob
	err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
			return err
		}
		if job.Status == models.PlanningJobStatusQueued {
			now := time.Now().UTC()
			if err := tx.Model(&job).Updates(map[string]any{"status": models.PlanningJobStatusCancelled, "phase": models.PlanningJobStatusCancelled, "cancel_requested_at": now, "finished_at": now}).Error; err != nil {
				return err
			}
			job.Status, job.Phase, job.CancelRequestedAt, job.FinishedAt = models.PlanningJobStatusCancelled, models.PlanningJobStatusCancelled, &now, &now
		} else if job.Status == models.PlanningJobStatusDecomposing || job.Status == models.PlanningJobStatusScheduling {
			now := time.Now().UTC()
			if err := tx.Model(&job).Update("cancel_requested_at", now).Error; err != nil {
				return err
			}
			job.CancelRequestedAt = &now
		}
		return nil
	})
	return job, err
}

func ExpirePlanningJobs(ctx context.Context, database *gorm.DB, now time.Time) error {
	return database.WithContext(ctx).Model(&models.PlanningJob{}).
		Where("expires_at <= ? AND status NOT IN ?", now, []string{models.PlanningJobStatusReady, models.PlanningJobStatusFallback, models.PlanningJobStatusCancelled, models.PlanningJobStatusExpired}).
		Updates(map[string]any{"status": models.PlanningJobStatusExpired, "phase": models.PlanningJobStatusExpired, "lease_owner": "", "lease_expires_at": nil, "finished_at": now}).Error
}

func PlanningJobCancellationRequested(ctx context.Context, database *gorm.DB, jobID, owner string) (bool, error) {
	var job models.PlanningJob
	if err := database.WithContext(ctx).Select("cancel_requested_at", "lease_owner").First(&job, "id = ?", jobID).Error; err != nil {
		return false, err
	}
	if job.LeaseOwner != owner {
		return false, errors.New("planning job lease lost")
	}
	return job.CancelRequestedAt != nil, nil
}
