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
	EndTime string `json:"end_time" binding:"required"`
	Reason  string `json:"reason"`
}

type createTaskReq struct {
	Date        string `json:"date" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type updateTaskReq struct {
	Date        *string `json:"date"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	SortOrder   *int    `json:"sort_order"`
}

type reorderTaskReq struct {
	TaskIDs []uint `json:"task_ids" binding:"required"`
}

type postponeTaskReq struct {
	Date   string `json:"date" binding:"required"`
	Reason string `json:"reason"`
}

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
	task := models.DailyTask{UserID: uid, PlanID: plan.ID, Date: req.Date, Title: req.Title, Description: req.Description, SortOrder: req.SortOrder, Status: models.TaskStatusPending}
	if err := db.DB.Create(&task).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "create task failed: "+err.Error())
		return
	}
	api.OK(c, task)
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

	err = db.DB.Transaction(func(tx *gorm.DB) error {
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
	if err := db.DB.Save(task).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "complete task failed: "+err.Error())
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
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
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
	record := models.PostponeRecord{TaskID: task.ID, UserID: uid, PlanID: task.PlanID, OldDate: task.Date, NewDate: req.Date, Reason: req.Reason}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		task.Date = req.Date
		task.Status = models.TaskStatusPending
		return tx.Save(task).Error
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "postpone task failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"task": task, "record": record})
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
	end, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		if t, e := time.Parse("2006-01-02 15:04", req.EndTime); e == nil {
			end = t
		} else {
			api.Fail(c, http.StatusBadRequest, "invalid end_time, use RFC3339 or yyyy-mm-dd hh:mm")
			return
		}
	}
	if task.ActualStart == nil {
		task.ActualStart = &end
	}
	if end.Before(*task.ActualStart) {
		api.Fail(c, http.StatusBadRequest, "end_time before actual_start")
		return
	}
	minutes := int(end.Sub(*task.ActualStart).Minutes())
	if minutes < 1 {
		minutes = 1
	}
	task.ActualEnd = &end
	task.StudyMinutes = minutes
	task.Status = models.TaskStatusPending
	if err := db.DB.Save(task).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "makeup task failed: "+err.Error())
		return
	}
	api.OK(c, task)
}

func PendingDecisionTasks(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	date := c.DefaultQuery("date", time.Now().Format(dateLayout))
	var tasks []models.DailyTask
	if err := db.DB.Where("user_id = ? AND date = ? AND status = ?", uid, date, models.TaskStatusInProgress).Find(&tasks).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query tasks failed: "+err.Error())
		return
	}
	api.OK(c, tasks)
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
		UserID: uid,
		PlanID: plan.ID,
		Date:   date,
		Title:  plan.Title,
		Status: models.TaskStatusPending,
	}
	return task, db.DB.Create(&task).Error
}
