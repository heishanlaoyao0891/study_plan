package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

type makeupTaskReq struct {
	ActualDate  string `json:"actual_date"`
	ActualStart string `json:"actual_start"`
	ActualEnd   string `json:"actual_end" binding:"required"`
	Reason      string `json:"reason"`
}

type createTaskReq struct {
	Date             string `json:"date" binding:"required"`
	Title            string `json:"title" binding:"required"`
	Description      string `json:"description"`
	Objective        string `json:"objective" binding:"required"`
	SortOrder        int    `json:"sort_order"`
	PlannedStart     string `json:"planned_start"`
	PlannedEnd       string `json:"planned_end"`
	EstimatedMinutes int    `json:"estimated_minutes"`
	Difficulty       string `json:"difficulty"`
	PublicToGroup    bool   `json:"public_to_group"`
}

type updateTaskReq struct {
	Date             *string `json:"date"`
	Title            *string `json:"title"`
	Description      *string `json:"description"`
	Objective        *string `json:"objective"`
	Status           *string `json:"status"`
	SortOrder        *int    `json:"sort_order"`
	PlannedStart     *string `json:"planned_start"`
	PlannedEnd       *string `json:"planned_end"`
	EstimatedMinutes *int    `json:"estimated_minutes"`
	Difficulty       *string `json:"difficulty"`
	PublicToGroup    *bool   `json:"public_to_group"`
}

type completeTaskReq struct {
	Reflection *string `json:"reflection"`
}
type reflectionReq struct {
	Reflection string `json:"reflection"`
}

type taskTimerView struct {
	models.DailyTask
	TargetMinutes      int                  `json:"target_minutes"`
	AccumulatedSeconds int                  `json:"accumulated_seconds"`
	ActiveSession      *models.StudySession `json:"active_session"`
	TimerState         string               `json:"timer_state"`
	RemainingSeconds   int                  `json:"remaining_seconds"`
	OvertimeSeconds    int                  `json:"overtime_seconds"`
}

type reorderTaskReq struct {
	TaskIDs []uint `json:"task_ids" binding:"required"`
}

type postponeTaskReq struct {
	Date            string `json:"date" binding:"required"`
	PlannedStart    string `json:"planned_start"`
	PlannedEnd      string `json:"planned_end"`
	ConfirmConflict bool   `json:"confirm_conflict"`
	Reason          string `json:"reason"`
}

const (
	suspiciousSessionMinutes = 180
	suspiciousDailyMinutes   = 480
)

func ListPlanTasks(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	plan, err := mustGetOwnedPlan(c, uid)
	if err != nil {
		return
	}
	var tasks []models.DailyTask
	if err := db.DB.Where("user_id = ? AND plan_id = ?", uid, plan.ID).Order("date ASC, sort_order ASC, id ASC").Find(&tasks).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query tasks failed: "+err.Error())
		return
	}
	views, err := buildTaskTimerViews(tasks, time.Now())
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query sessions failed: "+err.Error())
		return
	}
	api.OK(c, views)
}

func CreatePlanTask(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	plan, err := mustGetOwnedPlan(c, uid)
	if err != nil {
		return
	}
	var req createTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if _, err := time.Parse(dateLayout, req.Date); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid date, expect YYYY-MM-DD")
		return
	}
	if err := validateObjective(req.Title, req.Objective); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	task := models.DailyTask{UserID: uid, PlanID: plan.ID, Date: req.Date, Title: req.Title, Description: req.Description, Objective: strings.TrimSpace(req.Objective), SortOrder: req.SortOrder, PlannedStart: defaultPlannedStart(), PlannedEnd: defaultPlannedEnd(), EstimatedMinutes: defaultPlannedMinutes(), Difficulty: defaultPlannedDifficulty(), PublicToGroup: req.PublicToGroup, Status: models.TaskStatusPending}
	if req.PlannedStart != "" {
		task.PlannedStart = req.PlannedStart
	}
	if req.PlannedEnd != "" {
		task.PlannedEnd = req.PlannedEnd
	}
	if req.EstimatedMinutes > 0 {
		task.EstimatedMinutes = req.EstimatedMinutes
	}
	if req.Difficulty != "" {
		task.Difficulty = req.Difficulty
	}
	if err := validatePlannedRange(task.PlannedStart, task.PlannedEnd); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := db.DB.Create(&task).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "create task failed: "+err.Error())
		return
	}
	view, _ := buildTaskTimerView(task, time.Now())
	api.OK(c, view)
}

func GetTask(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}
	var plan models.Plan
	if err := db.DB.First(&plan, task.PlanID).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query plan failed: "+err.Error())
		return
	}
	var history []models.PostponeRecord
	_ = db.DB.Where("task_id = ?", task.ID).Order("id DESC").Find(&history).Error
	view, err := buildTaskTimerView(*task, time.Now())
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query timer failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"task": view, "plan": plan, "history": history})
}

func StartTask(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}
	now := time.Now()
	var session models.StudySession
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", task.ID, uid).First(task).Error; err != nil {
			return err
		}
		if task.Status == models.TaskStatusCompleted {
			return errTaskCompleted
		}
		err := tx.Where("task_id = ? AND user_id = ? AND end_time IS NULL", task.ID, uid).First(&session).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		session = models.StudySession{TaskID: task.ID, UserID: uid, StartTime: now}
		if task.ActualStart == nil {
			task.ActualStart = &now
		}
		task.Status = models.TaskStatusInProgress
		if e := tx.Save(task).Error; e != nil {
			return e
		}
		return tx.Create(&session).Error
	})
	if errors.Is(err, errTaskCompleted) {
		api.Fail(c, http.StatusConflict, "completed task cannot be restarted")
		return
	}
	if err != nil && isUniqueConstraintError(err) {
		if queryErr := db.DB.Where("task_id = ? AND user_id = ? AND end_time IS NULL", task.ID, uid).First(&session).Error; queryErr == nil {
			db.DB.First(task, task.ID)
			api.OK(c, gin.H{"task": task, "session": session})
			return
		}
	}
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "start task failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"task": task, "session": session})
}

func PauseTask(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}
	now := time.Now()
	var closed *models.StudySession
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", task.ID, uid).First(task).Error; err != nil {
			return err
		}
		if task.Status == models.TaskStatusCompleted {
			return errTaskCompleted
		}
		session, err := closeActiveSession(tx, task, uid, now)
		if err != nil {
			return err
		}
		closed = session
		task.Status = models.TaskStatusPending
		task.NeedsDecision = false
		return tx.Save(task).Error
	})
	if errors.Is(err, errTaskCompleted) {
		api.Fail(c, http.StatusConflict, "completed task cannot be paused")
		return
	}
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "pause task failed: "+err.Error())
		return
	}
	view, _ := buildTaskTimerView(*task, now)
	api.OK(c, gin.H{"task": view, "session": closed})
}

func StopTask(c *gin.Context)     { completeTask(c, false) }
func CompleteTask(c *gin.Context) { completeTask(c, true) }

func completeTask(c *gin.Context, requireAchieved bool) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}
	var req completeTaskReq
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
			return
		}
	}
	if req.Reflection != nil && utf8.RuneCountInString(*req.Reflection) > 500 {
		api.Fail(c, http.StatusBadRequest, "reflection cannot exceed 500 characters")
		return
	}
	now := time.Now()
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", task.ID, uid).First(task).Error; err != nil {
			return err
		}
		if task.Status == models.TaskStatusCompleted {
			return nil
		}
		if requireAchieved {
			accumulated, err := taskAccumulatedSeconds(tx, task, uid, now)
			if err != nil {
				return err
			}
			target := plannedRangeMinutes(task.PlannedStart, task.PlannedEnd)
			if target <= 0 {
				target = task.EstimatedMinutes
			}
			if target <= 0 {
				target = defaultPlannedMinutes()
			}
			if accumulated < target*60 {
				return errTargetNotReached
			}
		}
		if _, err := closeActiveSession(tx, task, uid, now); err != nil {
			return err
		}
		task.ActualEnd = &now
		task.Status = models.TaskStatusCompleted
		task.NeedsDecision = false
		if req.Reflection != nil {
			task.Reflection = strings.TrimSpace(*req.Reflection)
		}
		return tx.Save(task).Error
	}); errors.Is(err, errTargetNotReached) {
		api.Fail(c, http.StatusBadRequest, "target duration has not been reached; use stop for early completion")
		return
	} else if err != nil {
		api.Fail(c, http.StatusInternalServerError, "complete task failed: "+err.Error())
		return
	}
	view, _ := buildTaskTimerView(*task, now)
	api.OK(c, view)
}

func UpdateTaskReflection(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}
	if task.Status != models.TaskStatusCompleted {
		api.Fail(c, http.StatusConflict, "reflection can only be edited after completion")
		return
	}
	var req reflectionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if utf8.RuneCountInString(req.Reflection) > 500 {
		api.Fail(c, http.StatusBadRequest, "reflection cannot exceed 500 characters")
		return
	}
	task.Reflection = strings.TrimSpace(req.Reflection)
	if err := db.DB.Model(task).Update("reflection", task.Reflection).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "update reflection failed: "+err.Error())
		return
	}
	api.OK(c, task)
}

func UpdateTask(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}
	var req updateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	updates := map[string]interface{}{}
	if req.Date != nil {
		if _, err := time.Parse(dateLayout, *req.Date); err != nil {
			api.Fail(c, http.StatusBadRequest, "invalid date, expect YYYY-MM-DD")
			return
		}
		updates["date"] = *req.Date
	}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	title, objective := task.Title, task.Objective
	if req.Title != nil {
		title = *req.Title
	}
	if req.Objective != nil {
		objective = *req.Objective
	}
	if err := validateObjective(title, objective); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Objective != nil {
		updates["objective"] = strings.TrimSpace(*req.Objective)
	}
	if req.Status != nil {
		api.Fail(c, http.StatusBadRequest, "status must be changed through task state actions")
		return
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.PlannedStart != nil {
		updates["planned_start"] = *req.PlannedStart
	}
	if req.PlannedEnd != nil {
		updates["planned_end"] = *req.PlannedEnd
	}
	if req.EstimatedMinutes != nil {
		updates["estimated_minutes"] = *req.EstimatedMinutes
	}
	if req.Difficulty != nil {
		updates["difficulty"] = *req.Difficulty
	}
	if req.PublicToGroup != nil {
		updates["public_to_group"] = *req.PublicToGroup
	}
	plannedStart, plannedEnd := task.PlannedStart, task.PlannedEnd
	if req.PlannedStart != nil {
		plannedStart = *req.PlannedStart
	}
	if req.PlannedEnd != nil {
		plannedEnd = *req.PlannedEnd
	}
	if err := validatePlannedRange(plannedStart, plannedEnd); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(updates) > 0 {
		if err := db.DB.Model(task).Updates(updates).Error; err != nil {
			api.Fail(c, http.StatusInternalServerError, "update task failed: "+err.Error())
			return
		}
		db.DB.First(task, task.ID)
	}
	api.OK(c, task)
}

func DeleteTask(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", task.ID).Delete(&models.StudySession{}).Error; err != nil {
			return err
		}
		return tx.Delete(task).Error
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "delete task failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"deleted": task.ID})
}

func ReorderPlanTasks(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	plan, err := mustGetOwnedPlan(c, uid)
	if err != nil {
		return
	}
	var req reorderTaskReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.TaskIDs) == 0 {
		api.Fail(c, http.StatusBadRequest, "task_ids required")
		return
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		for i, id := range req.TaskIDs {
			if err := tx.Model(&models.DailyTask{}).Where("id = ? AND user_id = ? AND plan_id = ?", id, uid, plan.ID).Update("sort_order", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "reorder tasks failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"reordered": len(req.TaskIDs)})
}

func PostponeTask(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}
	var req postponeTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if _, err := time.Parse(dateLayout, req.Date); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid date, expect YYYY-MM-DD")
		return
	}
	if task.Status == models.TaskStatusInProgress {
		api.Fail(c, http.StatusConflict, "running task cannot be postponed")
		return
	}
	plannedStart := task.PlannedStart
	plannedEnd := task.PlannedEnd
	if req.PlannedStart != "" {
		plannedStart = req.PlannedStart
	}
	if req.PlannedEnd != "" {
		plannedEnd = req.PlannedEnd
	}
	if err := validatePlannedRange(plannedStart, plannedEnd); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	conflicts, err := findTaskSlotConflicts(uid, task.ID, req.Date, plannedStart, plannedEnd)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query task conflicts failed: "+err.Error())
		return
	}
	if len(conflicts) > 0 && !req.ConfirmConflict {
		api.Fail(c, http.StatusConflict, "schedule conflict, confirm_conflict required")
		return
	}
	record := models.PostponeRecord{TaskID: task.ID, UserID: uid, PlanID: task.PlanID, OldDate: task.Date, NewDate: req.Date, Reason: req.Reason}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		task.Date = req.Date
		task.PlannedStart = plannedStart
		task.PlannedEnd = plannedEnd
		task.Status = models.TaskStatusPending
		task.NeedsDecision = false
		return tx.Save(task).Error
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "postpone task failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"task": task, "record": record})
}

func findTaskSlotConflicts(uid, taskID uint, date, plannedStart, plannedEnd string) ([]models.DailyTask, error) {
	if plannedStart == "" || plannedEnd == "" {
		return nil, nil
	}
	var conflicts []models.DailyTask
	err := db.DB.Where(
		"user_id = ? AND id <> ? AND date = ? AND status <> ? AND planned_start < ? AND planned_end > ?",
		uid, taskID, date, models.TaskStatusCompleted, plannedEnd, plannedStart,
	).Find(&conflicts).Error
	return conflicts, err
}

func MakeupTask(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}
	var req makeupTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	end, err := parseMakeupDateTime(req.ActualEnd, req.ActualDate, loc)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid actual_end, use RFC3339 or yyyy-mm-dd hh:mm")
		return
	}
	start := end.Add(-time.Duration(defaultPlannedMinutes()) * time.Minute)
	if task.ActualStart != nil {
		start = *task.ActualStart
	}
	if req.ActualStart != "" {
		start, err = parseMakeupDateTime(req.ActualStart, req.ActualDate, loc)
		if err != nil {
			api.Fail(c, http.StatusBadRequest, "invalid actual_start, use RFC3339 or yyyy-mm-dd hh:mm")
			return
		}
	}
	if !end.After(start) {
		api.Fail(c, http.StatusBadRequest, "actual_end must be later than actual_start")
		return
	}
	if end.After(time.Now().In(loc)) {
		api.Fail(c, http.StatusBadRequest, "actual_end cannot be in the future")
		return
	}
	seconds := int(end.Sub(start).Seconds())
	minutes := seconds / 60
	if minutes > 8*60 {
		api.Fail(c, http.StatusBadRequest, "corrected session cannot exceed 8 hours")
		return
	}
	task.ActualStart = &start
	task.ActualEnd = &end
	task.StudyMinutes = minutes
	task.StudySeconds = seconds
	task.Status = models.TaskStatusPending
	task.NeedsDecision = false
	cost := makeupSlackCost(uid, minutes)
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query user failed: "+err.Error())
		return
	}
	if cost > user.SlackBalance {
		api.Fail(c, http.StatusBadRequest, "not enough slack minutes for makeup")
		return
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		markStudyReviewFlags(task, nil)
		if err := tx.Model(&models.User{}).Where("id = ?", uid).Update("slack_balance", gorm.Expr("slack_balance - ?", cost)).Error; err != nil {
			return err
		}
		if cost > 0 {
			if err := recordSlackDelta(tx, uid, "补录消耗: "+task.Title, -cost); err != nil {
				return err
			}
		}
		return tx.Save(task).Error
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "makeup task failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"task": task, "makeup_cost_minutes": cost})
}

func PendingDecisionTasks(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	date := c.DefaultQuery("date", time.Now().Format(dateLayout))
	var tasks []models.DailyTask
	if err := db.DB.Where("user_id = ? AND date = ? AND (status = ? OR needs_decision = ?)", uid, date, models.TaskStatusInProgress, true).Find(&tasks).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query tasks failed: "+err.Error())
		return
	}
	api.OK(c, tasks)
}

func CompensateMidnightTasks(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	closed, err := closeOvernightTasks(uid, time.Now())
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "midnight compensation failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"closed": closed})
}

func closeOvernightTasks(uid uint, now time.Time) (int, error) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return 0, err
	}
	localNow := now.In(loc)
	today := localNow.Format(dateLayout)
	var tasks []models.DailyTask
	if err := db.DB.Where("user_id = ? AND date < ? AND status = ?", uid, today, models.TaskStatusInProgress).Find(&tasks).Error; err != nil {
		return 0, err
	}
	closed := 0
	for _, task := range tasks {
		dateStart, err := time.ParseInLocation(dateLayout, task.Date, loc)
		if err != nil {
			continue
		}
		midnight := dateStart.AddDate(0, 0, 1)
		var session models.StudySession
		err = db.DB.Where("task_id = ? AND user_id = ? AND end_time IS NULL", task.ID, uid).First(&session).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			task.Status = models.TaskStatusPending
			task.NeedsDecision = true
			if saveErr := db.DB.Save(&task).Error; saveErr != nil {
				return closed, saveErr
			}
			closed++
			continue
		}
		if err != nil {
			return closed, err
		}
		seconds := int(midnight.Sub(session.StartTime).Seconds())
		if seconds < 0 {
			seconds = 0
		}
		minutes := seconds / 60
		session.EndTime = &midnight
		session.DurationMin = minutes
		session.DurationSec = seconds
		task.ActualEnd = &midnight
		task.StudyMinutes += minutes
		task.StudySeconds += seconds
		task.Status = models.TaskStatusPending
		task.NeedsDecision = true
		markStudyReviewFlags(&task, &session)
		if err := db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&session).Error; err != nil {
				return err
			}
			return tx.Save(&task).Error
		}); err != nil {
			return closed, err
		}
		closed++
	}
	return closed, nil
}

func markStudyReviewFlags(task *models.DailyTask, session *models.StudySession) {
	if session != nil && session.DurationMin > suspiciousSessionMinutes {
		session.Suspicious = true
		session.ReviewNote = "single study session exceeds MVP review threshold"
	}
	if task != nil && task.StudyMinutes > suspiciousDailyMinutes {
		task.Suspicious = true
		task.SuspiciousReason = "daily study minutes exceed MVP review threshold"
	}
}

func getOwnedTask(c *gin.Context, uid uint) (*models.DailyTask, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid task id")
		return nil, false
	}
	var task models.DailyTask
	if err := db.DB.First(&task, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "task not found")
		return nil, false
	}
	if task.UserID != uid {
		api.Fail(c, http.StatusForbidden, "not your task")
		return nil, false
	}
	return &task, true
}

func ensureDailyTask(uid uint, plan models.Plan, date string) (models.DailyTask, error) {
	var task models.DailyTask
	if plan.Status != models.PlanStatusActive {
		return task, gorm.ErrRecordNotFound
	}
	err := db.DB.Where("user_id = ? AND plan_id = ? AND date = ?", uid, plan.ID, date).First(&task).Error
	if err == nil {
		return task, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return task, err
	}
	day, err := time.Parse(dateLayout, date)
	if err != nil || !planStudiesOn(plan, day) {
		return task, gorm.ErrRecordNotFound
	}
	return task, gorm.ErrRecordNotFound
}

func defaultPlannedStart() string      { return "20:00" }
func defaultPlannedEnd() string        { return "21:00" }
func defaultPlannedMinutes() int       { return 60 }
func defaultPlannedDifficulty() string { return "medium" }

func parseMakeupDateTime(value, actualDate string, loc *time.Location) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.In(loc), nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04", value, loc); err == nil {
		return t, nil
	}
	if actualDate == "" {
		return time.Time{}, errors.New("actual_date is required for HH:mm values")
	}
	return time.ParseInLocation("2006-01-02 15:04", actualDate+" "+value, loc)
}

var (
	errTaskCompleted    = errors.New("task is completed")
	errTargetNotReached = errors.New("target duration not reached")
)

func validatePlannedRange(start, end string) error {
	if plannedRangeMinutes(start, end) <= 0 {
		return errors.New("planned time range is invalid, expect HH:mm with end after start")
	}
	return nil
}

func taskAccumulatedSeconds(tx *gorm.DB, task *models.DailyTask, uid uint, now time.Time) (int, error) {
	seconds := task.StudySeconds
	if seconds == 0 && task.StudyMinutes > 0 {
		seconds = task.StudyMinutes * 60
	}
	var session models.StudySession
	err := tx.Where("task_id = ? AND user_id = ? AND end_time IS NULL", task.ID, uid).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return seconds, nil
	}
	if err != nil {
		return 0, err
	}
	if live := int(now.Sub(session.StartTime).Seconds()); live > 0 {
		seconds += live
	}
	return seconds, nil
}

func isUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}

func closeActiveSession(tx *gorm.DB, task *models.DailyTask, uid uint, now time.Time) (*models.StudySession, error) {
	var session models.StudySession
	err := tx.Where("task_id = ? AND user_id = ? AND end_time IS NULL", task.ID, uid).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	seconds := int(now.Sub(session.StartTime).Seconds())
	if seconds < 0 {
		seconds = 0
	}
	session.EndTime = &now
	session.DurationSec = seconds
	session.DurationMin = seconds / 60
	task.ActualEnd = &now
	baseSeconds := task.StudySeconds
	if baseSeconds == 0 && task.StudyMinutes > 0 {
		baseSeconds = task.StudyMinutes * 60
	}
	task.StudySeconds = baseSeconds + seconds
	task.StudyMinutes = task.StudySeconds / 60
	markStudyReviewFlags(task, &session)
	if err := tx.Save(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func buildTaskTimerViews(tasks []models.DailyTask, now time.Time) ([]taskTimerView, error) {
	views := make([]taskTimerView, 0, len(tasks))
	for _, task := range tasks {
		view, err := buildTaskTimerView(task, now)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func buildTaskTimerView(task models.DailyTask, now time.Time) (taskTimerView, error) {
	target := plannedRangeMinutes(task.PlannedStart, task.PlannedEnd)
	if target <= 0 {
		target = task.EstimatedMinutes
	}
	if target <= 0 {
		target = defaultPlannedMinutes()
	}
	accumulated := task.StudySeconds
	if accumulated == 0 && task.StudyMinutes > 0 {
		accumulated = task.StudyMinutes * 60
	}
	var active models.StudySession
	activePtr := (*models.StudySession)(nil)
	err := db.DB.Where("task_id = ? AND user_id = ? AND end_time IS NULL", task.ID, task.UserID).First(&active).Error
	if err == nil {
		activePtr = &active
		live := int(now.Sub(active.StartTime).Seconds())
		if live > 0 {
			accumulated += live
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return taskTimerView{}, err
	}
	remaining := target*60 - accumulated
	overtime := 0
	if remaining < 0 {
		overtime = -remaining
		remaining = 0
	}
	state := "pending"
	if task.Status == models.TaskStatusCompleted {
		state = "completed"
	} else if accumulated >= target*60 {
		state = "achieved"
	} else if activePtr != nil {
		state = "running"
	} else if accumulated > 0 {
		state = "paused"
	}
	return taskTimerView{DailyTask: task, TargetMinutes: target, AccumulatedSeconds: accumulated, ActiveSession: activePtr, TimerState: state, RemainingSeconds: remaining, OvertimeSeconds: overtime}, nil
}

func validateObjective(title, objective string) error {
	trimmed := strings.TrimSpace(objective)
	if trimmed == "" {
		return errors.New("objective is required")
	}
	if utf8.RuneCountInString(trimmed) > 500 {
		return errors.New("objective cannot exceed 500 characters")
	}
	if strings.EqualFold(trimmed, strings.TrimSpace(title)) {
		return errors.New("objective must be more specific than the task title")
	}
	return nil
}
