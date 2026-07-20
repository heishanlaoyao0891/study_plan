package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

type makeupTaskReq struct {
	ActualStart string `json:"actual_start"`
	ActualEnd   string `json:"actual_end" binding:"required"`
	Reason      string `json:"reason"`
}

type createTaskReq struct {
	Date             string `json:"date" binding:"required"`
	Title            string `json:"title" binding:"required"`
	Description      string `json:"description"`
	SortOrder        int    `json:"sort_order"`
	PlannedStart     string `json:"planned_start"`
	PlannedEnd       string `json:"planned_end"`
	EstimatedMinutes int    `json:"estimated_minutes"`
	Difficulty       string `json:"difficulty"`
}

type updateTaskReq struct {
	Date             *string `json:"date"`
	Title            *string `json:"title"`
	Description      *string `json:"description"`
	Status           *string `json:"status"`
	SortOrder        *int    `json:"sort_order"`
	PlannedStart     *string `json:"planned_start"`
	PlannedEnd       *string `json:"planned_end"`
	EstimatedMinutes *int    `json:"estimated_minutes"`
	Difficulty       *string `json:"difficulty"`
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
	api.OK(c, tasks)
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
	task := models.DailyTask{UserID: uid, PlanID: plan.ID, Date: req.Date, Title: req.Title, Description: req.Description, SortOrder: req.SortOrder, PlannedStart: defaultPlannedStart(), PlannedEnd: defaultPlannedEnd(), EstimatedMinutes: defaultPlannedMinutes(), Difficulty: defaultPlannedDifficulty(), Status: models.TaskStatusPending}
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
	if err := db.DB.Create(&task).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "create task failed: "+err.Error())
		return
	}
	api.OK(c, task)
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
	api.OK(c, gin.H{"task": task, "plan": plan, "history": history})
}

func StartTask(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}

	var active models.StudySession
	err := db.DB.Where("task_id = ? AND user_id = ? AND end_time IS NULL", task.ID, uid).First(&active).Error
	if err == nil {
		api.OK(c, gin.H{"task": task, "session": active})
		return
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		api.Fail(c, http.StatusInternalServerError, "query session failed: "+err.Error())
		return
	}

	now := time.Now()
	session := models.StudySession{TaskID: task.ID, UserID: uid, StartTime: now}
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if task.ActualStart == nil {
			task.ActualStart = &now
		}
		task.Status = models.TaskStatusInProgress
		if e := tx.Save(task).Error; e != nil {
			return e
		}
		return tx.Create(&session).Error
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "start task failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"task": task, "session": session})
}

func StopTask(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}

	var session models.StudySession
	err := db.DB.Where("task_id = ? AND user_id = ? AND end_time IS NULL", task.ID, uid).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		api.Fail(c, http.StatusBadRequest, "no active study session")
		return
	}
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query session failed: "+err.Error())
		return
	}

	now := time.Now()
	dur := int(now.Sub(session.StartTime).Minutes())
	if dur < 1 {
		dur = 1
	}
	session.EndTime = &now
	session.DurationMin = dur
	task.ActualEnd = &now
	task.StudyMinutes += dur
	if task.Status == models.TaskStatusInProgress {
		task.Status = models.TaskStatusPending
	}
	task.NeedsDecision = false

	err = db.DB.Transaction(func(tx *gorm.DB) error {
		markStudyReviewFlags(task, &session)
		if e := tx.Save(&session).Error; e != nil {
			return e
		}
		return tx.Save(task).Error
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "stop task failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"task": task, "session": session})
}

func CompleteTask(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	task, ok := getOwnedTask(c, uid)
	if !ok {
		return
	}
	now := time.Now()
	if task.ActualEnd == nil {
		task.ActualEnd = &now
	}
	task.Status = models.TaskStatusCompleted
	task.NeedsDecision = false
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(task).Error; err != nil {
			return err
		}
		return autoCompleteCheckinIfPlanDateDone(tx, uid, task.PlanID, task.Date)
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "complete task failed: "+err.Error())
		return
	}
	api.OK(c, task)
}

func autoCompleteCheckinIfPlanDateDone(tx *gorm.DB, uid, planID uint, date string) error {
	var remaining int64
	if err := tx.Model(&models.DailyTask{}).
		Where("user_id = ? AND plan_id = ? AND date = ? AND status <> ?", uid, planID, date, models.TaskStatusCompleted).
		Count(&remaining).Error; err != nil {
		return err
	}
	if remaining > 0 {
		return nil
	}

	var checkin models.Checkin
	err := tx.Where("user_id = ? AND plan_id = ? AND date = ?", uid, planID, date).First(&checkin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		checkin = models.Checkin{UserID: uid, PlanID: planID, Date: date, Completed: true}
		if err := tx.Create(&checkin).Error; err != nil {
			return err
		}
		return awardSlackIfNeeded(tx, uid, &checkin)
	}
	if err != nil {
		return err
	}
	if !checkin.Completed {
		if err := tx.Model(&checkin).Update("completed", true).Error; err != nil {
			return err
		}
		checkin.Completed = true
	}
	return awardSlackIfNeeded(tx, uid, &checkin)
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
	if req.Status != nil {
		updates["status"] = *req.Status
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
	plannedStart := task.PlannedStart
	plannedEnd := task.PlannedEnd
	if req.PlannedStart != "" {
		plannedStart = req.PlannedStart
	}
	if req.PlannedEnd != "" {
		plannedEnd = req.PlannedEnd
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
	end, err := parseTaskDateTime(req.ActualEnd)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid actual_end, use RFC3339 or yyyy-mm-dd hh:mm")
		return
	}
	start := end.Add(-time.Duration(defaultPlannedMinutes()) * time.Minute)
	if task.ActualStart != nil {
		start = *task.ActualStart
	}
	if req.ActualStart != "" {
		start, err = parseTaskDateTime(req.ActualStart)
		if err != nil {
			api.Fail(c, http.StatusBadRequest, "invalid actual_start, use RFC3339 or yyyy-mm-dd hh:mm")
			return
		}
	}
	if !end.After(start) {
		api.Fail(c, http.StatusBadRequest, "actual_end must be later than actual_start")
		return
	}
	if end.After(time.Now()) {
		api.Fail(c, http.StatusBadRequest, "actual_end cannot be in the future")
		return
	}
	minutes := int(end.Sub(start).Minutes())
	if minutes > 8*60 {
		api.Fail(c, http.StatusBadRequest, "corrected session cannot exceed 8 hours")
		return
	}
	task.ActualStart = &start
	task.ActualEnd = &end
	task.StudyMinutes = minutes
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
		minutes := int(midnight.Sub(session.StartTime).Minutes())
		if minutes < 1 {
			minutes = 1
		}
		session.EndTime = &midnight
		session.DurationMin = minutes
		task.ActualEnd = &midnight
		task.StudyMinutes += minutes
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
	err := db.DB.Where("user_id = ? AND plan_id = ? AND date = ?", uid, plan.ID, date).First(&task).Error
	if err == nil {
		return task, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return task, err
	}
	task = models.DailyTask{
		UserID:           uid,
		PlanID:           plan.ID,
		Date:             date,
		Title:            plan.Title,
		PlannedStart:     defaultPlannedStart(),
		PlannedEnd:       defaultPlannedEnd(),
		EstimatedMinutes: defaultPlannedMinutes(),
		Difficulty:       defaultPlannedDifficulty(),
		Status:           models.TaskStatusPending,
	}
	return task, db.DB.Create(&task).Error
}

func defaultPlannedStart() string      { return "20:00" }
func defaultPlannedEnd() string        { return "21:00" }
func defaultPlannedMinutes() int       { return 60 }
func defaultPlannedDifficulty() string { return "medium" }

func parseTaskDateTime(value string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04", value)
}
