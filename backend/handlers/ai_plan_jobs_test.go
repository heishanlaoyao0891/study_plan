package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

func TestSubmitAIPlanJobIsPromptIdempotentAndOwnerIsolated(t *testing.T) {
	setupTestDB(t)
	router := gin.New()
	router.POST("/jobs", func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, uint(11)) }, SubmitAIPlanJob)
	router.GET("/jobs/:id", func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, uint(12)) }, GetAIPlanJob)
	body := `{"goal":"Study Go","days":1,"hours_per_day":1,"start_date":"2026-08-01","available_time_slot":"20:00-21:00","additional_instructions":"focus on practical exercises","idempotency_key":"job_key_1234567890"}`

	started := time.Now()
	first := performAIPlanJobRequest(router, http.MethodPost, "/jobs", body)
	if first.Code != http.StatusAccepted || time.Since(started) > time.Second {
		t.Fatalf("submission did not return promptly with 202: status=%d body=%s", first.Code, first.Body.String())
	}
	var firstResponse struct {
		Data aiPlanJobView `json:"data"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstResponse); err != nil {
		t.Fatal(err)
	}
	second := performAIPlanJobRequest(router, http.MethodPost, "/jobs", body)
	var secondResponse struct {
		Data aiPlanJobView `json:"data"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &secondResponse); err != nil {
		t.Fatal(err)
	}
	if second.Code != http.StatusAccepted || secondResponse.Data.ID != firstResponse.Data.ID {
		t.Fatalf("idempotent submission did not reuse job: first=%s second=%s", first.Body.String(), second.Body.String())
	}

	other := performAIPlanJobRequest(router, http.MethodGet, fmt.Sprintf("/jobs/%d", firstResponse.Data.ID), "")
	if responseCode(t, other.Body.Bytes()) != http.StatusNotFound || strings.Contains(other.Body.String(), "Study Go") {
		t.Fatalf("cross-owner lookup exposed job: %s", other.Body.String())
	}
	var stored models.AIPlanGenerationJob
	if err := db.DB.First(&stored, firstResponse.Data.ID).Error; err != nil {
		t.Fatal(err)
	}
	var request aiPlanJobRequest
	if err := json.Unmarshal([]byte(stored.RequestJSON), &request); err != nil || request.Input.Refinement != "focus on practical exercises" || request.Input.UserID != 0 {
		t.Fatalf("normalized additional instructions were not stored: request=%+v err=%v", request, err)
	}
}

func TestSubmitAIPlanJobReusesOneActiveJobAndRejectsKeyReuse(t *testing.T) {
	setupTestDB(t)
	router := gin.New()
	router.POST("/jobs", func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, uint(21)) }, SubmitAIPlanJob)
	first := performAIPlanJobRequest(router, http.MethodPost, "/jobs", `{"goal":"First","days":1,"start_date":"2026-08-01","idempotency_key":"same_key_12345678"}`)
	activeReuse := performAIPlanJobRequest(router, http.MethodPost, "/jobs", `{"goal":"Different active request","days":1,"start_date":"2026-08-02","idempotency_key":"other_key_1234567"}`)
	var firstEnvelope, activeEnvelope struct {
		Data aiPlanJobView `json:"data"`
	}
	json.Unmarshal(first.Body.Bytes(), &firstEnvelope)
	json.Unmarshal(activeReuse.Body.Bytes(), &activeEnvelope)
	if activeReuse.Code != http.StatusAccepted || firstEnvelope.Data.ID != activeEnvelope.Data.ID {
		t.Fatalf("one-active-job rule was not enforced: first=%s active=%s", first.Body.String(), activeReuse.Body.String())
	}
	if err := db.DB.Model(&models.AIPlanGenerationJob{}).Where("id = ?", firstEnvelope.Data.ID).Updates(map[string]any{"status": models.AIPlanJobStatusFailed, "completed_at": time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	keyConflict := performAIPlanJobRequest(router, http.MethodPost, "/jobs", `{"goal":"Changed payload","days":1,"start_date":"2026-08-03","idempotency_key":"same_key_12345678"}`)
	if keyConflict.Code != http.StatusConflict {
		t.Fatalf("same key with changed request was accepted: %s", keyConflict.Body.String())
	}
}

func TestAIPlanJobClaimsAreAtomicAndRecoverExpiredLease(t *testing.T) {
	setupTestDB(t)
	payload := mustAIPlanJobPayload(t, 31, "Recover Go")
	expired := time.Now().Add(-time.Minute)
	job := models.AIPlanGenerationJob{UserID: 31, RequestJSON: payload, RequestHash: "hash", Status: models.AIPlanJobStatusRunning, AttemptCount: 1, LeaseOwner: "dead-worker", LeaseExpiresAt: &expired}
	if err := db.DB.Create(&job).Error; err != nil {
		t.Fatal(err)
	}

	const contenders = 8
	claimed := 0
	var mu sync.Mutex
	var wait sync.WaitGroup
	for index := 0; index < contenders; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			_, ok, _ := claimAIPlanJob(db.DB, fmt.Sprintf("worker-%d", index), time.Now())
			if ok {
				mu.Lock()
				claimed++
				mu.Unlock()
			}
		}(index)
	}
	wait.Wait()
	if claimed != 1 {
		t.Fatalf("expected exactly one lease claimant, got %d", claimed)
	}
	if err := db.DB.First(&job, job.ID).Error; err != nil || job.AttemptCount != 2 || job.LeaseOwner == "dead-worker" {
		t.Fatalf("expired lease was not recovered: job=%+v err=%v", job, err)
	}
}

func TestAIPlanJobWorkerPersistsFallbackPlanAndSurvivesRestart(t *testing.T) {
	setupTestDB(t)
	job := models.AIPlanGenerationJob{UserID: 41, RequestJSON: mustAIPlanJobPayload(t, 41, "Study Go"), RequestHash: "hash", Status: models.AIPlanJobStatusPending}
	if err := db.DB.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	worker := StartAIPlanJobWorker(db.DB)
	waitForAIPlanJobStatus(t, job.ID, models.AIPlanJobStatusSucceeded)
	shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := worker.Shutdown(shutdownContext); err != nil {
		t.Fatal(err)
	}
	if err := db.DB.First(&job, job.ID).Error; err != nil || job.ResultPlanID == nil || job.GenerationSource != "local" || job.EnrichmentStatus != "disabled" {
		t.Fatalf("worker did not persist truthful fallback result: job=%+v err=%v", job, err)
	}
	var plan models.Plan
	var taskCount int64
	if err := db.DB.First(&plan, *job.ResultPlanID).Error; err != nil {
		t.Fatal(err)
	}
	db.DB.Model(&models.DailyTask{}).Where("plan_id = ?", plan.ID).Count(&taskCount)
	if !plan.AIGenerated || plan.UserID != job.UserID || taskCount != 1 {
		t.Fatalf("generated plan was incomplete: plan=%+v task_count=%d", plan, taskCount)
	}
}

func TestAIPlanJobFailureIsAtomicAndRetryExhaustionIsTerminal(t *testing.T) {
	setupTestDB(t)
	if err := db.DB.Exec(`CREATE TRIGGER fail_ai_job_tasks BEFORE INSERT ON daily_tasks BEGIN SELECT RAISE(ABORT, 'forced task failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	job := models.AIPlanGenerationJob{UserID: 51, RequestJSON: mustAIPlanJobPayload(t, 51, "Atomic Go"), RequestHash: "hash", Status: models.AIPlanJobStatusPending}
	if err := db.DB.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	claimed, ok, err := claimAIPlanJob(db.DB, "test-worker", time.Now())
	if err != nil || !ok {
		t.Fatalf("claim failed: ok=%v err=%v", ok, err)
	}
	worker := &AIPlanJobWorker{db: db.DB, owner: "test-worker"}
	worker.process(context.Background(), claimed)
	var planCount int64
	db.DB.Model(&models.Plan{}).Where("user_id = ?", job.UserID).Count(&planCount)
	if planCount != 0 {
		t.Fatalf("failed task persistence left %d partial plans", planCount)
	}

	expired := time.Now().Add(-time.Minute)
	if err := db.DB.Model(&models.AIPlanGenerationJob{}).Where("id = ?", job.ID).Updates(map[string]any{"status": models.AIPlanJobStatusRunning, "attempt_count": aiPlanJobMaxAttempts, "lease_owner": "dead", "lease_expires_at": expired}).Error; err != nil {
		t.Fatal(err)
	}
	if _, ok, err := claimAIPlanJob(db.DB, "next", time.Now()); err != nil || ok {
		t.Fatalf("exhausted job was reclaimed: ok=%v err=%v", ok, err)
	}
	if err := db.DB.First(&job, job.ID).Error; err != nil || job.Status != models.AIPlanJobStatusFailed || job.ErrorCode != "attempts_exhausted" || strings.Contains(job.ErrorMessage, "forced") {
		t.Fatalf("retry exhaustion did not expose a safe terminal failure: job=%+v err=%v", job, err)
	}
}

func performAIPlanJobRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func mustAIPlanJobPayload(t *testing.T, userID uint, goal string) string {
	t.Helper()
	payload, err := json.Marshal(aiPlanJobRequest{Input: services.PlanGenerationInput{UserID: userID, Goal: goal, Days: 1, HoursPerDay: 1, StartDate: "2026-08-01", AvailableTimeSlot: "20:00-21:00"}})
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}

func waitForAIPlanJobStatus(t *testing.T, id uint, status string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var job models.AIPlanGenerationJob
		if err := db.DB.First(&job, id).Error; err == nil && job.Status == status {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	var job models.AIPlanGenerationJob
	db.DB.First(&job, id)
	t.Fatalf("job did not reach %s: %+v", status, job)
}
