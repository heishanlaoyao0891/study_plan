package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

type generatePlanReq struct {
	Goal              string   `json:"goal" binding:"required"`
	HoursPerDay       int      `json:"hours_per_day"`
	Days              int      `json:"days"`
	StartDate         string   `json:"start_date"`
	AvailableTimeSlot string   `json:"available_time_slot"`
	SkipDates         []string `json:"skip_dates"`
	Refinement        string   `json:"refinement"`
}

type commitAIPlanReq struct {
	Preview         services.PlanPreview `json:"preview" binding:"required"`
	PreviewID       string               `json:"preview_id"`
	PreviewVersion  int                  `json:"preview_version"`
	ProvenanceToken string               `json:"provenance_token" binding:"required"`
	IdempotencyKey  string               `json:"idempotency_key" binding:"required"`
	ConfirmOverload bool                 `json:"confirm_overload"`
}

const maxAIGenerateBodyBytes = 64 << 10
const maxAICommitBodyBytes = 256 << 10
const planningRequestBudget = 5 * time.Second
const planningInteractiveTarget = 2 * time.Second
const planningWorkBudget = 4500 * time.Millisecond
const planningEnrichmentBudget = 8 * time.Second
const commitRaceAttempts = 5

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)

var currentAIProvider = services.CurrentAIProviderContext
var canUseAIGeneration = services.CanUseAIGenerationContext
var validateAIConfigContext = services.ValidateAIConfigContext

func GeneratePlan(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAIGenerateBodyBytes)
	var req generatePlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	started := time.Now()
	requestContext, cancelRequest := context.WithTimeout(c.Request.Context(), planningRequestBudget)
	defer cancelRequest()
	workContext, cancelWork := context.WithTimeout(requestContext, planningWorkBudget)
	defer cancelWork()
	ctx, err := services.BuildPlanningContextWithContext(workContext, services.PlanGenerationInput{UserID: uid, Goal: req.Goal, HoursPerDay: req.HoursPerDay, Days: req.Days, StartDate: req.StartDate, AvailableTimeSlot: req.AvailableTimeSlot, SkipDates: req.SkipDates, Refinement: req.Refinement})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			api.Fail(c, http.StatusRequestTimeout, "planning request budget exhausted")
			return
		}
		api.Fail(c, http.StatusBadRequest, "invalid planning request: "+err.Error())
		return
	}
	contextDuration := time.Since(started)
	localStarted := time.Now()
	preview, err := services.BuildLocalPlan(ctx)
	if err != nil {
		api.Fail(c, http.StatusConflict, "build local plan failed: "+err.Error())
		return
	}
	localDuration := time.Since(localStarted)
	if err := services.ValidatePlanPreview(preview, ctx.Input); err != nil {
		api.Fail(c, http.StatusInternalServerError, "local plan validation failed: "+err.Error())
		return
	}
	if err := validateAIPreviewSchedule(db.DB.WithContext(workContext), uid, preview); err != nil {
		if respondScheduleError(c, err) {
			return
		}
		api.Fail(c, http.StatusInternalServerError, "validate AI schedule failed: "+err.Error())
		return
	}
	source := "local"
	enrichmentStatus := "disabled"
	enrichmentReason := "provider_disabled"
	providerName, modelName := "", ""
	usedToday, dailyLimit := int64(0), 0
	backgroundBudgetSeconds := 60
	interactiveTargetSeconds := int(planningInteractiveTarget.Seconds())
	cfg, _, providerErr := currentAIProvider(workContext)
	enqueue := false
	if providerErr != nil {
		enrichmentStatus, enrichmentReason = "configuration_error", "provider_configuration_unavailable"
	} else {
		providerName, modelName, dailyLimit = cfg.Provider, cfg.ModelName, maxPositive(cfg.DailyGenerationLimit, 5)
		backgroundBudgetSeconds = cfg.BackgroundJobTimeoutSeconds
		interactiveTargetSeconds = cfg.InteractiveTargetSeconds
		if interactiveTargetSeconds == 0 {
			interactiveTargetSeconds = 2
		}
		if backgroundBudgetSeconds == 0 {
			backgroundBudgetSeconds = 60
		}
		providerKind := services.NormalizeAIProvider(cfg.Provider)
		if !cfg.Enabled || providerKind == services.AIProviderMock {
			enrichmentStatus, enrichmentReason = "disabled", "provider_disabled"
		} else {
			if canUse, count, quotaErr := canUseAIGeneration(workContext, uid, cfg.DailyGenerationLimit); quotaErr != nil {
				enrichmentStatus, enrichmentReason = "provider_error", "quota_check_failed"
			} else if !canUse {
				usedToday = count
				enrichmentStatus, enrichmentReason = "quota_limited", "daily_enrichment_limit_reached"
			} else {
				usedToday = count
				enqueue = true
				enrichmentStatus, enrichmentReason = "queued", ""
			}
		}
	}
	if err := services.AssignPlanTaskIdentities(&preview); err != nil {
		api.Fail(c, http.StatusInternalServerError, "assign preview identities failed")
		return
	}
	persisted, err := services.PersistPlanningBaseline(workContext, db.DB, uid, ctx, preview, services.PlanningJobOptions{
		Enqueue: enqueue, Provider: providerName, ModelName: modelName, BackgroundBudgetSeconds: backgroundBudgetSeconds,
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "persist planning preview failed: "+err.Error())
		return
	}
	preview = persisted.Preview
	provenanceToken := persisted.Version.ProvenanceToken
	mode := "fallback"
	if persisted.Job != nil {
		mode = "pending"
		enrichmentStatus = persisted.Job.Status
		enrichmentReason = ""
	}
	jobMetadata := any(nil)
	if persisted.Job != nil {
		jobMetadata = planningJobResponse(*persisted.Job)
	}
	data := gin.H{
		"preview": preview, "mode": mode, "source": source, "provenance_token": provenanceToken, "warnings": ctx.Warnings,
		"preview_id": persisted.Version.PreviewID, "preview_version": persisted.Version.Version,
		"expires_at": persisted.Version.ExpiresAt, "context_fingerprint": persisted.Version.ContextFingerprint,
		"job":                   jobMetadata,
		"enrichment":            gin.H{"status": enrichmentStatus, "reason": enrichmentReason, "provider": providerName, "model": modelName},
		"ai_status":             gin.H{"mode": mode, "provider": providerName, "model": modelName, "fallback_reason": enrichmentReason},
		"usage":                 gin.H{"used_today": usedToday, "daily_limit": dailyLimit},
		"request_budget_ms":     planningRequestBudget.Milliseconds(),
		"interactive_target_ms": int64(interactiveTargetSeconds) * 1000,
		"background_budget_ms":  int64(backgroundBudgetSeconds) * 1000,
		"phase_timings_ms":      gin.H{"context": contextDuration.Milliseconds(), "local_planning": localDuration.Milliseconds(), "enrichment": 0, "total": time.Since(started).Milliseconds()},
	}
	api.OK(c, data)
}

func GetPlanningJob(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	jobID := strings.TrimSpace(c.Param("id"))
	if !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(jobID) {
		api.Fail(c, http.StatusNotFound, "planning job not found")
		return
	}
	var job models.PlanningJob
	if err := db.DB.Where("id = ? AND user_id = ?", jobID, uid).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.Fail(c, http.StatusNotFound, "planning job not found")
			return
		}
		api.Fail(c, http.StatusInternalServerError, "query planning job failed")
		return
	}
	if time.Now().UTC().After(job.ExpiresAt) && job.Status != models.PlanningJobStatusExpired {
		now := time.Now().UTC()
		db.DB.Model(&models.PlanningJob{}).Where("id = ? AND user_id = ?", job.ID, uid).Updates(map[string]any{"status": models.PlanningJobStatusExpired, "phase": models.PlanningJobStatusExpired, "finished_at": now})
		job.Status, job.Phase, job.FinishedAt = models.PlanningJobStatusExpired, models.PlanningJobStatusExpired, &now
	}
	if job.Status == models.PlanningJobStatusExpired {
		api.OK(c, gin.H{"job": planningJobResponse(job)})
		return
	}
	versionNumber := job.BaselinePreviewVersion
	if job.ResultPreviewVersion != nil {
		versionNumber = *job.ResultPreviewVersion
	}
	var version models.PlanningPreviewVersion
	if err := db.DB.Where("preview_id = ? AND version = ? AND user_id = ?", job.BaselinePreviewID, versionNumber, uid).First(&version).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query planning preview failed")
		return
	}
	var preview services.PlanPreview
	if err := json.Unmarshal([]byte(version.PreviewJSON), &preview); err != nil {
		api.Fail(c, http.StatusInternalServerError, "decode planning preview failed")
		return
	}
	api.OK(c, gin.H{
		"job": planningJobResponse(job), "preview": preview,
		"preview_id": version.PreviewID, "preview_version": version.Version, "source": version.Source,
		"expires_at": version.ExpiresAt, "context_fingerprint": version.ContextFingerprint,
		"provenance_token": version.ProvenanceToken,
	})
}

func CancelPlanningJob(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	jobID := strings.TrimSpace(c.Param("id"))
	if !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(jobID) {
		api.Fail(c, http.StatusNotFound, "planning job not found")
		return
	}
	job, err := services.RequestPlanningJobCancellation(c.Request.Context(), db.DB, uid, jobID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		api.Fail(c, http.StatusNotFound, "planning job not found")
		return
	}
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "cancel planning job failed")
		return
	}
	api.OK(c, gin.H{"job": planningJobResponse(job)})
}

func MutatePlanningPreview(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	previewID := strings.TrimSpace(c.Param("id"))
	baseVersion, err := strconv.Atoi(c.Param("version"))
	if !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(previewID) || err != nil || baseVersion < 1 {
		api.Fail(c, http.StatusNotFound, "planning preview not found")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAICommitBodyBytes)
	var mutation services.PreviewMutation
	if err := c.ShouldBindJSON(&mutation); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid preview mutation: "+err.Error())
		return
	}
	derived, err := services.CreateDerivedPreviewVersion(c.Request.Context(), db.DB, uid, previewID, baseVersion, mutation)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		api.Fail(c, http.StatusNotFound, "planning preview not found")
		return
	}
	if err != nil {
		api.Fail(c, http.StatusConflict, err.Error())
		return
	}
	api.OK(c, gin.H{
		"preview": derived.Preview, "preview_id": derived.Version.PreviewID, "preview_version": derived.Version.Version,
		"source": derived.Version.Source, "expires_at": derived.Version.ExpiresAt,
		"context_fingerprint": derived.Version.ContextFingerprint, "provenance_token": derived.Version.ProvenanceToken,
	})
}

func planningJobResponse(job models.PlanningJob) gin.H {
	return gin.H{
		"id": job.ID, "status": job.Status, "phase": job.Phase,
		"provider": job.Provider, "model": job.ModelName,
		"preview_id": job.BaselinePreviewID, "baseline_version": job.BaselinePreviewVersion,
		"result_version": job.ResultPreviewVersion, "attempt_count": job.AttemptCount,
		"failure_reason": job.FailureReason, "background_budget_seconds": job.BackgroundBudgetSeconds,
		"created_at": job.CreatedAt, "updated_at": job.UpdatedAt, "expires_at": job.ExpiresAt,
	}
}

func generatePlanEnrichment(parent context.Context, provider services.AIProvider, uid uint, providerName string, limit int, used *int64, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, planningEnrichmentBudget)
	defer cancel()
	ctx = services.WithAIInvocationContext(ctx, services.AIInvocationContext{UserID: uid, JobType: "synchronous_enrichment", Phase: "enriching", AgentAttempt: 1})
	return provider.GenerateContext(services.WithAIQuota(ctx, uid, providerName, limit, used), prompt, 1024)
}

func classifyEnrichmentError(err error) (string, string) {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", "enrichment_deadline_exceeded"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled", "request_cancelled"
	}
	if errors.Is(err, services.ErrAIQuotaExceeded) {
		return "quota_limited", "daily_enrichment_limit_reached"
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "http 429") || strings.Contains(message, "quota") {
		return "quota_limited", "provider_quota_limited"
	}
	return "provider_error", "provider_request_failed"
}

func CommitAIPlan(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAICommitBodyBytes)
	var req commitAIPlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if !idempotencyKeyPattern.MatchString(req.IdempotencyKey) {
		api.Fail(c, http.StatusBadRequest, "invalid idempotency_key")
		return
	}
	provenance, err := services.ParsePlanProvenance(req.ProvenanceToken, uid)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := services.ValidateCommittedPlanProvenance(req.Preview, provenance); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	preview, err := services.NormalizeCommittedPlanPreview(req.Preview)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid preview: "+err.Error())
		return
	}
	committedHash := services.HashPlanPreview(preview)
	if handled := respondExistingAIPlanCommit(c, uid, req.IdempotencyKey, committedHash); handled {
		return
	}
	if err := validatePersistedPreviewVersion(uid, req.PreviewID, req.PreviewVersion, provenance); err != nil {
		api.Fail(c, http.StatusConflict, err.Error())
		return
	}
	weeklyTarget := recomputeWeeklyTarget(preview)
	if _, err := checkOverload(uid, weeklyTarget, req.ConfirmOverload); err != nil {
		if respondExistingAIPlanCommit(c, uid, req.IdempotencyKey, committedHash) {
			return
		}
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateAIPreviewSchedule(db.DB, uid, preview); err != nil {
		if respondExistingAIPlanCommit(c, uid, req.IdempotencyKey, committedHash) {
			return
		}
		if respondScheduleError(c, err) {
			return
		}
		api.Fail(c, http.StatusInternalServerError, "validate AI schedule failed: "+err.Error())
		return
	}
	var plan models.Plan
	var tasks []models.DailyTask
	for attempt := 0; attempt < commitRaceAttempts; attempt++ {
		plan, tasks, err = createAIPlanCommit(uid, req.IdempotencyKey, committedHash, provenance.Source, req.PreviewID, req.PreviewVersion, preview, weeklyTarget, req.ConfirmOverload)
		if err == nil {
			break
		}
		if respondExistingAIPlanCommit(c, uid, req.IdempotencyKey, committedHash) {
			return
		}
		if !isSQLiteCommitRace(err) {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
	if err != nil && respondExistingAIPlanCommit(c, uid, req.IdempotencyKey, committedHash) {
		return
	}
	if err != nil {
		if respondScheduleError(c, err) {
			return
		}
		api.Fail(c, http.StatusInternalServerError, "commit ai plan failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"plan": plan, "tasks": tasks})
}

func createAIPlanCommit(uid uint, idempotencyKey, committedHash, source, previewID string, previewVersion int, preview services.PlanPreview, weeklyTarget int, confirmOverload bool) (models.Plan, []models.DailyTask, error) {
	first, last := preview.Tasks[0], preview.Tasks[len(preview.Tasks)-1]
	plan := models.Plan{UserID: uid, Title: preview.Title, Description: preview.Summary, Status: models.PlanStatusActive, WeeklyTargetHours: weeklyTarget, AIGenerated: true, GenerationSource: source, StartDate: first.Date, EndDate: last.Date, DefaultPlannedStart: first.PlannedStart, DefaultPlannedEnd: first.PlannedEnd}
	seenStudyDates := map[string]bool{}
	for _, previewTask := range preview.Tasks {
		if !seenStudyDates[previewTask.Date] {
			plan.StudyDates = append(plan.StudyDates, previewTask.Date)
			seenStudyDates[previewTask.Date] = true
		}
	}
	tasks := make([]models.DailyTask, 0, len(preview.Tasks))
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.AIPlanCommit
		if err := tx.Where("user_id = ? AND idempotency_key = ?", uid, idempotencyKey).First(&existing).Error; err == nil {
			return gorm.ErrDuplicatedKey
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if _, err := checkOverloadWithDB(tx, uid, weeklyTarget, confirmOverload); err != nil {
			return err
		}
		if err := validateAIPreviewSchedule(tx, uid, preview); err != nil {
			return err
		}
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		for i, previewTask := range preview.Tasks {
			task := models.DailyTask{UserID: uid, PlanID: plan.ID, Date: previewTask.Date, Title: previewTask.Title, Objective: previewTask.Objective, Description: previewTask.Description, SortOrder: i, PlannedStart: previewTask.PlannedStart, PlannedEnd: previewTask.PlannedEnd, EstimatedMinutes: previewTask.EstimatedMinutes, Difficulty: previewTask.Difficulty, Status: models.TaskStatusPending}
			if err := tx.Create(&task).Error; err != nil {
				return err
			}
			tasks = append(tasks, task)
		}
		if previewID != "" && previewVersion > 0 {
			now := time.Now().UTC()
			result := tx.Model(&models.PlanningPreviewVersion{}).Where("preview_id = ? AND version = ? AND user_id = ? AND committed_at IS NULL AND expires_at > ?", previewID, previewVersion, uid, now).Update("committed_at", now)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("preview version is stale, expired, or already committed")
			}
		}
		return tx.Create(&models.AIPlanCommit{UserID: uid, IdempotencyKey: idempotencyKey, PlanID: plan.ID, PreviewHash: committedHash}).Error
	})
	return plan, tasks, err
}

func validatePersistedPreviewVersion(uid uint, previewID string, previewVersion int, claims *services.PlanProvenanceClaims) error {
	if claims.PreviewID == "" && claims.PreviewVersion == 0 {
		return nil
	}
	if previewID == "" || previewVersion < 1 || previewID != claims.PreviewID || previewVersion != claims.PreviewVersion {
		return errors.New("preview version does not match provenance")
	}
	var version models.PlanningPreviewVersion
	if err := db.DB.Where("preview_id = ? AND version = ? AND user_id = ?", previewID, previewVersion, uid).First(&version).Error; err != nil {
		return errors.New("preview version not found")
	}
	if time.Now().UTC().After(version.ExpiresAt) || version.CommittedAt != nil {
		return errors.New("preview version is stale, expired, or already committed")
	}
	if version.ContextFingerprint != claims.ContextFingerprint || version.Source != claims.Source {
		return errors.New("preview version context does not match provenance")
	}
	return nil
}

func respondExistingAIPlanCommit(c *gin.Context, uid uint, key, previewHash string) bool {
	var commit models.AIPlanCommit
	err := db.DB.Where("user_id = ? AND idempotency_key = ?", uid, key).First(&commit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false
	}
	if err != nil {
		return false
	}
	if commit.PreviewHash != previewHash {
		api.Fail(c, http.StatusConflict, "idempotency_key was already used for a different preview")
		return true
	}
	var plan models.Plan
	var tasks []models.DailyTask
	if err := db.DB.Where("id = ? AND user_id = ?", commit.PlanID, uid).First(&plan).Error; err != nil {
		return false
	}
	if err := db.DB.Where("plan_id = ? AND user_id = ?", commit.PlanID, uid).Order("sort_order ASC").Find(&tasks).Error; err != nil {
		return false
	}
	api.OK(c, gin.H{"plan": plan, "tasks": tasks, "idempotent_replay": true})
	return true
}

func isSQLiteCommitRace(err error) bool {
	message := strings.ToLower(err.Error())
	return errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(message, "unique constraint") || strings.Contains(message, "database is locked") || strings.Contains(message, "database table is locked") || strings.Contains(message, "sqlite_busy")
}

func recomputeWeeklyTarget(preview services.PlanPreview) int {
	first, _ := time.Parse("2006-01-02", preview.Tasks[0].Date)
	last, _ := time.Parse("2006-01-02", preview.Tasks[len(preview.Tasks)-1].Date)
	calendarDays := int(last.Sub(first).Hours()/24) + 1
	weeks := int(math.Ceil(float64(calendarDays) / 7))
	return int(math.Ceil(preview.EstimatedTotalHours / float64(weeks)))
}

func validateAIPreviewSchedule(tx *gorm.DB, uid uint, preview services.PlanPreview) error {
	tasks := make([]models.DailyTask, 0, len(preview.Tasks))
	for _, row := range preview.Tasks {
		tasks = append(tasks, models.DailyTask{UserID: uid, Date: row.Date, Title: row.Title, PlannedStart: row.PlannedStart, PlannedEnd: row.PlannedEnd})
	}
	return validateScheduleMutation(tx, uid, tasks)
}

func RegeneratePlan(c *gin.Context) { GeneratePlan(c) }

func EditAIPlan(c *gin.Context) {
	api.OK(c, gin.H{"message": "AI plan edit is covered by normal plan/task APIs in MVP"})
}

func maxPositive(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}
