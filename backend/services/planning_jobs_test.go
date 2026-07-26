package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"study_plan_backend/models"
)

func planningJobsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&models.PlanningJob{}, &models.PlanningPreviewVersion{}); err != nil {
		t.Fatal(err)
	}
	return database
}

func queuedPlanningJob(id string, now time.Time) models.PlanningJob {
	return models.PlanningJob{ID: id, UserID: 1, RequestFingerprint: id, Status: models.PlanningJobStatusQueued, Phase: models.PlanningJobStatusQueued, BaselinePreviewID: id, BaselinePreviewVersion: 1, RequestJSON: `{}`, MaxAttempts: 2, BackgroundBudgetSeconds: 60, ExpiresAt: now.Add(time.Hour)}
}

func TestPlanningJobClaimLeaseRecoveryAndAttemptBound(t *testing.T) {
	database := planningJobsTestDB(t)
	now := time.Now().UTC()
	job := queuedPlanningJob("11111111111111111111111111111111", now)
	if err := database.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	claimed, err := ClaimPlanningJobs(context.Background(), database, "worker-a", 1, 20*time.Millisecond)
	if err != nil || len(claimed) != 1 || claimed[0].AttemptCount != 1 || claimed[0].Status != models.PlanningJobStatusDecomposing {
		t.Fatalf("unexpected first claim: jobs=%+v err=%v", claimed, err)
	}
	if duplicate, err := ClaimPlanningJobs(context.Background(), database, "worker-b", 1, time.Minute); err != nil || len(duplicate) != 0 {
		t.Fatalf("active lease was claimed twice: jobs=%+v err=%v", duplicate, err)
	}
	time.Sleep(25 * time.Millisecond)
	claimed, err = ClaimPlanningJobs(context.Background(), database, "worker-b", 1, time.Minute)
	if err != nil || len(claimed) != 1 || claimed[0].AttemptCount != 2 {
		t.Fatalf("expired lease was not recovered once: jobs=%+v err=%v", claimed, err)
	}
	leaseExpired := time.Now().UTC().Add(-time.Second)
	if err := database.Model(&models.PlanningJob{}).Where("id = ?", job.ID).Update("lease_expires_at", leaseExpired).Error; err != nil {
		t.Fatal(err)
	}
	if err := RecoverAbandonedPlanningJobs(context.Background(), database, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	var exhausted models.PlanningJob
	database.First(&exhausted, "id = ?", job.ID)
	if exhausted.Status != models.PlanningJobStatusFallback || exhausted.FailureReason != "worker_attempts_exhausted" {
		t.Fatalf("attempt exhaustion did not terminate safely: %+v", exhausted)
	}
}

func TestPlanningJobCancellationAndExpiry(t *testing.T) {
	database := planningJobsTestDB(t)
	now := time.Now().UTC()
	cancelled := queuedPlanningJob("22222222222222222222222222222222", now)
	expired := queuedPlanningJob("33333333333333333333333333333333", now)
	expired.ExpiresAt = now.Add(-time.Second)
	if err := database.Create(&cancelled).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&expired).Error; err != nil {
		t.Fatal(err)
	}
	result, err := RequestPlanningJobCancellation(context.Background(), database, cancelled.UserID, cancelled.ID)
	if err != nil || result.Status != models.PlanningJobStatusCancelled || result.FinishedAt == nil {
		t.Fatalf("queued cancellation failed: job=%+v err=%v", result, err)
	}
	if err := ExpirePlanningJobs(context.Background(), database, now); err != nil {
		t.Fatal(err)
	}
	var expiredResult models.PlanningJob
	database.First(&expiredResult, "id = ?", expired.ID)
	if expiredResult.Status != models.PlanningJobStatusExpired {
		t.Fatalf("job did not expire: %+v", expiredResult)
	}
}
