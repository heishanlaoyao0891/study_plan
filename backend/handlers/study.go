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
