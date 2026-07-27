package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
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

const (
	aiPlanJobMaxAttempts       = 3
	aiPlanJobLease             = 30 * time.Second
	aiPlanJobScanPeriod        = 2 * time.Second
	aiPlanJobDefaultWorkBudget = 5 * time.Minute
)

type submitAIPlanJobReq struct {
	Goal                   string   `json:"goal" binding:"required"`
	HoursPerDay            int      `json:"hours_per_day"`
	Days                   int      `json:"days"`
	StartDate              string   `json:"start_date"`
	AvailableTimeSlot      string   `json:"available_time_slot"`
	SkipDates              []string `json:"skip_dates"`
	AdditionalInstructions string   `json:"additional_instructions"`
	Refinement             string   `json:"refinement"`
	IdempotencyKey         string   `json:"idempotency_key"`
	ConfirmOverload        bool     `json:"confirm_overload"`
}

type aiPlanJobRequest struct {
	Input           services.PlanGenerationInput `json:"input"`
	ConfirmOverload bool                         `json:"confirm_overload"`
}

type aiPlanJobView struct {
	ID               uint       `json:"id"`
	Status           string     `json:"status"`
	AttemptCount     int        `json:"attempt_count"`
	ResultPlanID     *uint      `json:"result_plan_id,omitempty"`
	ErrorCode        string     `json:"error_code,omitempty"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	GenerationSource string     `json:"generation_source,omitempty"`
	Provider         string     `json:"provider,omitempty"`
	ModelName        string     `json:"model,omitempty"`
	EnrichmentStatus string     `json:"enrichment_status,omitempty"`
	EnrichmentReason string     `json:"enrichment_reason,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func SubmitAIPlanJob(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAIGenerateBodyBytes)
	var req submitAIPlanJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" {
		key = strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	}
	if key != "" && !idempotencyKeyPattern.MatchString(key) {
		api.Fail(c, http.StatusBadRequest, "invalid idempotency_key")
		return
	}
	instructions := req.AdditionalInstructions
	if strings.TrimSpace(instructions) == "" {
		instructions = req.Refinement
	}
	input, _, err := services.NormalizePlanGenerationInput(services.PlanGenerationInput{UserID: uid, Goal: req.Goal, HoursPerDay: req.HoursPerDay, Days: req.Days, StartDate: req.StartDate, AvailableTimeSlot: req.AvailableTimeSlot, SkipDates: req.SkipDates, Refinement: instructions})
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid planning request: "+err.Error())
		return
	}
	payload, err := json.Marshal(aiPlanJobRequest{Input: input, ConfirmOverload: req.ConfirmOverload})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "store planning request failed")
		return
	}
	digest := sha256.Sum256(payload)
	job := models.AIPlanGenerationJob{UserID: uid, RequestJSON: string(payload), RequestHash: hex.EncodeToString(digest[:]), IdempotencyKey: key, Status: models.AIPlanJobStatusPending}
	job, conflict, err := createOrReuseAIPlanJob(db.DB, job)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "submit plan generation failed")
		return
	}
	if conflict {
		api.Conflict(c, "idempotency_key was already used for a different request", aiPlanJobViewFromModel(job))
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"code": 0, "data": aiPlanJobViewFromModel(job)})
}

func createOrReuseAIPlanJob(database *gorm.DB, requested models.AIPlanGenerationJob) (models.AIPlanGenerationJob, bool, error) {
	var job models.AIPlanGenerationJob
	err := database.Transaction(func(tx *gorm.DB) error {
		if requested.IdempotencyKey != "" {
			err := tx.Where("user_id = ? AND idempotency_key = ?", requested.UserID, requested.IdempotencyKey).First(&job).Error
			if err == nil {
				return nil
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		err := tx.Where("user_id = ? AND status IN ?", requested.UserID, []string{models.AIPlanJobStatusPending, models.AIPlanJobStatusRunning}).Order("created_at DESC, id DESC").First(&job).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		job = requested
		return tx.Create(&job).Error
	})
	if err != nil && isSQLiteCommitRace(err) {
		query := database.Where("user_id = ?", requested.UserID)
		if requested.IdempotencyKey != "" {
			query = query.Where("idempotency_key = ? OR status IN ?", requested.IdempotencyKey, []string{models.AIPlanJobStatusPending, models.AIPlanJobStatusRunning})
		} else {
			query = query.Where("status IN ?", []string{models.AIPlanJobStatusPending, models.AIPlanJobStatusRunning})
		}
		err = query.Order("created_at DESC, id DESC").First(&job).Error
	}
	return job, job.IdempotencyKey == requested.IdempotencyKey && job.IdempotencyKey != "" && job.RequestHash != requested.RequestHash, err
}

func GetCurrentAIPlanJob(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var job models.AIPlanGenerationJob
	err := db.DB.Where("user_id = ?", uid).Order("CASE WHEN status IN ('pending','running') THEN 0 ELSE 1 END, created_at DESC, id DESC").First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		api.OK(c, nil)
		return
	}
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "load plan generation failed")
		return
	}
	api.OK(c, aiPlanJobViewFromModel(job))
}

func GetAIPlanJob(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		api.Fail(c, http.StatusNotFound, "plan generation job not found")
		return
	}
	var job models.AIPlanGenerationJob
	if err := db.DB.Where("id = ? AND user_id = ?", uint(id), uid).First(&job).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "plan generation job not found")
		return
	}
	api.OK(c, aiPlanJobViewFromModel(job))
}

func aiPlanJobViewFromModel(job models.AIPlanGenerationJob) aiPlanJobView {
	return aiPlanJobView{ID: job.ID, Status: job.Status, AttemptCount: job.AttemptCount, ResultPlanID: job.ResultPlanID, ErrorCode: job.ErrorCode, ErrorMessage: job.ErrorMessage, GenerationSource: job.GenerationSource, Provider: job.Provider, ModelName: job.ModelName, EnrichmentStatus: job.EnrichmentStatus, EnrichmentReason: job.EnrichmentReason, StartedAt: job.StartedAt, CompletedAt: job.CompletedAt, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt}
}

type AIPlanJobWorker struct {
	db     *gorm.DB
	owner  string
	cancel context.CancelFunc
	done   chan struct{}
}

func StartAIPlanJobWorker(database *gorm.DB) *AIPlanJobWorker {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &AIPlanJobWorker{db: database, owner: newAIPlanWorkerID(), cancel: cancel, done: make(chan struct{})}
	go worker.run(ctx)
	return worker
}

func (worker *AIPlanJobWorker) Shutdown(ctx context.Context) error {
	worker.cancel()
	select {
	case <-worker.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (worker *AIPlanJobWorker) run(ctx context.Context) {
	defer close(worker.done)
	ticker := time.NewTicker(aiPlanJobScanPeriod)
	defer ticker.Stop()
	for {
		worker.runAvailable(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (worker *AIPlanJobWorker) runAvailable(ctx context.Context) {
	for ctx.Err() == nil {
		job, claimed, err := claimAIPlanJob(worker.db, worker.owner, time.Now())
		if err != nil || !claimed {
			return
		}
		worker.process(ctx, job)
	}
}

func claimAIPlanJob(database *gorm.DB, owner string, now time.Time) (models.AIPlanGenerationJob, bool, error) {
	if err := database.Model(&models.AIPlanGenerationJob{}).
		Where("status = ? AND attempt_count >= ? AND (lease_expires_at IS NULL OR lease_expires_at <= ?)", models.AIPlanJobStatusRunning, aiPlanJobMaxAttempts, now).
		Updates(map[string]any{"status": models.AIPlanJobStatusFailed, "error_code": "attempts_exhausted", "error_message": "计划生成未能完成，请重新提交。", "lease_owner": "", "lease_expires_at": nil, "completed_at": now}).Error; err != nil {
		return models.AIPlanGenerationJob{}, false, err
	}
	var candidate models.AIPlanGenerationJob
	err := database.Where("(status = ? OR (status = ? AND lease_expires_at <= ?)) AND attempt_count < ?", models.AIPlanJobStatusPending, models.AIPlanJobStatusRunning, now, aiPlanJobMaxAttempts).Order("created_at ASC, id ASC").First(&candidate).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.AIPlanGenerationJob{}, false, nil
	}
	if err != nil {
		return models.AIPlanGenerationJob{}, false, err
	}
	expires := now.Add(aiPlanJobLease)
	claim := database.Model(&models.AIPlanGenerationJob{}).Where("id = ? AND (status = ? OR (status = ? AND lease_expires_at <= ?)) AND attempt_count < ?", candidate.ID, models.AIPlanJobStatusPending, models.AIPlanJobStatusRunning, now, aiPlanJobMaxAttempts).Updates(map[string]any{"status": models.AIPlanJobStatusRunning, "attempt_count": gorm.Expr("attempt_count + 1"), "lease_owner": owner, "lease_expires_at": expires, "started_at": now, "error_code": "", "error_message": ""})
	if claim.Error != nil || claim.RowsAffected != 1 {
		return models.AIPlanGenerationJob{}, false, claim.Error
	}
	if err := database.First(&candidate, candidate.ID).Error; err != nil {
		return models.AIPlanGenerationJob{}, false, err
	}
	return candidate, true, nil
}

func (worker *AIPlanJobWorker) process(parent context.Context, job models.AIPlanGenerationJob) {
	ctx, cancel := context.WithTimeout(parent, currentAIPlanJobWorkBudget(parent))
	defer cancel()
	leaseDone := make(chan struct{})
	go worker.renewLease(ctx, cancel, job.ID, leaseDone)

	var request aiPlanJobRequest
	err := json.Unmarshal([]byte(job.RequestJSON), &request)
	if err == nil {
		request.Input.UserID = job.UserID
		var result planningPipelineResult
		result, err = runPlanningPipeline(ctx, request.Input)
		if err == nil {
			err = persistAIPlanJobResult(worker.db.WithContext(ctx), job, request, result, time.Now())
		}
	}
	cancel()
	<-leaseDone
	if err != nil && !errors.Is(err, errAIPlanJobLeaseLost) {
		worker.recordFailure(job, err)
	}
}

func currentAIPlanJobWorkBudget(ctx context.Context) time.Duration {
	cfg, _, err := currentAIProvider(ctx)
	if err != nil || cfg.BackgroundJobTimeoutSeconds <= 0 {
		return aiPlanJobDefaultWorkBudget
	}
	seconds := cfg.BackgroundJobTimeoutSeconds
	if seconds != 300 && seconds != 600 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second
}

var errAIPlanJobLeaseLost = errors.New("ai plan job lease lost")

func (worker *AIPlanJobWorker) renewLease(ctx context.Context, cancel context.CancelFunc, jobID uint, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(aiPlanJobLease / 3)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			result := worker.db.Model(&models.AIPlanGenerationJob{}).Where("id = ? AND status = ? AND lease_owner = ?", jobID, models.AIPlanJobStatusRunning, worker.owner).Update("lease_expires_at", now.Add(aiPlanJobLease))
			if result.Error != nil || result.RowsAffected != 1 {
				cancel()
				return
			}
		}
	}
}

func persistAIPlanJobResult(database *gorm.DB, job models.AIPlanGenerationJob, request aiPlanJobRequest, result planningPipelineResult, now time.Time) error {
	preview := result.Preview
	weeklyTarget := recomputeWeeklyTarget(preview)
	first, last := preview.Tasks[0], preview.Tasks[len(preview.Tasks)-1]
	plan := models.Plan{UserID: job.UserID, Title: preview.Title, Description: preview.Summary, Status: models.PlanStatusActive, WeeklyTargetHours: weeklyTarget, AIGenerated: true, GenerationSource: result.Source, StartDate: first.Date, EndDate: last.Date, DefaultPlannedStart: first.PlannedStart, DefaultPlannedEnd: first.PlannedEnd}
	for _, row := range preview.Tasks {
		plan.StudyDates = append(plan.StudyDates, row.Date)
	}
	return database.Transaction(func(tx *gorm.DB) error {
		var current models.AIPlanGenerationJob
		if err := tx.Where("id = ? AND status = ? AND lease_owner = ?", job.ID, models.AIPlanJobStatusRunning, job.LeaseOwner).First(&current).Error; err != nil {
			return errAIPlanJobLeaseLost
		}
		if _, err := checkOverloadWithDB(tx, job.UserID, weeklyTarget, request.ConfirmOverload); err != nil {
			return err
		}
		tasks := make([]models.DailyTask, 0, len(preview.Tasks))
		for index, row := range preview.Tasks {
			tasks = append(tasks, models.DailyTask{UserID: job.UserID, Date: row.Date, Title: row.Title, Objective: row.Objective, Description: row.Description, SortOrder: index, PlannedStart: row.PlannedStart, PlannedEnd: row.PlannedEnd, EstimatedMinutes: row.EstimatedMinutes, Difficulty: row.Difficulty, Status: models.TaskStatusPending})
		}
		if err := validateScheduleMutation(tx, job.UserID, tasks); err != nil {
			return err
		}
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		for index := range tasks {
			tasks[index].PlanID = plan.ID
		}
		if err := tx.Create(&tasks).Error; err != nil {
			return err
		}
		update := tx.Model(&models.AIPlanGenerationJob{}).Where("id = ? AND status = ? AND lease_owner = ?", job.ID, models.AIPlanJobStatusRunning, job.LeaseOwner).Updates(map[string]any{"status": models.AIPlanJobStatusSucceeded, "result_plan_id": plan.ID, "generation_source": result.Source, "provider": result.Provider, "model_name": result.Model, "enrichment_status": result.EnrichmentStatus, "enrichment_reason": result.EnrichmentReason, "lease_owner": "", "lease_expires_at": nil, "completed_at": now})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return errAIPlanJobLeaseLost
		}
		return nil
	})
}

func (worker *AIPlanJobWorker) recordFailure(job models.AIPlanGenerationJob, processingErr error) {
	now := time.Now()
	status := models.AIPlanJobStatusPending
	updates := map[string]any{"status": status, "lease_owner": "", "lease_expires_at": nil, "error_code": "", "error_message": ""}
	errorCode, errorMessage, terminal := classifyAIPlanJobFailure(processingErr)
	if terminal || job.AttemptCount >= aiPlanJobMaxAttempts {
		updates["status"] = models.AIPlanJobStatusFailed
		if errorCode == "" {
			errorCode, errorMessage = "generation_failed", "计划生成服务暂时未能完成处理，请稍后重试。"
		}
		updates["error_code"] = errorCode
		updates["error_message"] = errorMessage
		updates["completed_at"] = now
	}
	worker.db.Model(&models.AIPlanGenerationJob{}).Where("id = ? AND status = ? AND lease_owner = ?", job.ID, models.AIPlanJobStatusRunning, worker.owner).Updates(updates)
}

func classifyAIPlanJobFailure(err error) (string, string, bool) {
	if err == nil {
		return "", "", false
	}
	if strings.Contains(err.Error(), "confirm_overload required") {
		return "overload_confirmation_required", "当前活动计划较多或每周学习负荷较高，请确认后继续生成。", true
	}
	var scheduleErr *scheduleConflictError
	var dateErr *taskDateConflictError
	if errors.As(err, &scheduleErr) || errors.As(err, &dateErr) {
		return "schedule_conflict", "当前任务日程与生成计划冲突，请调整可用时段后重试。", true
	}
	if strings.Contains(err.Error(), "could not allocate") {
		return "no_available_schedule", "未来可安排日期内没有足够的空闲时段，请扩大或更换可用时间后重试。", true
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "generation_timeout", "计划生成处理超时，请稍后重试。", false
	}
	return "", "", false
}

func newAIPlanWorkerID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(value)
}
