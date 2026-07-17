package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

func NotificationSubscriptions(c *gin.Context) {
	api.OK(c, []gin.H{})
}

func SubscribeNotification(c *gin.Context) {
	api.OK(c, gin.H{"subscribed": true, "mode": "placeholder"})
}

func UnsubscribeNotification(c *gin.Context) {
	api.OK(c, gin.H{"subscribed": false, "mode": "placeholder"})
}

func DueNotificationEvents(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	date := c.DefaultQuery("date", time.Now().Format(dateLayout))
	if _, err := time.Parse(dateLayout, date); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid date, expect YYYY-MM-DD")
		return
	}
	var tasks []models.DailyTask
	if err := db.DB.Where("user_id = ? AND date = ?", uid, date).Order("sort_order ASC, id ASC").Find(&tasks).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query notification tasks failed: "+err.Error())
		return
	}
	var checkins []models.Checkin
	if err := db.DB.Where("user_id = ? AND date = ?", uid, date).Find(&checkins).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query notification checkins failed: "+err.Error())
		return
	}
	checked := map[uint]bool{}
	for _, row := range checkins {
		checked[row.PlanID] = row.Completed
	}
	events := make([]gin.H, 0)
	for _, task := range tasks {
		if task.Status == models.TaskStatusPending && task.ActualStart == nil {
			events = append(events, gin.H{"type": "study_start", "task": task, "message": "到点学习提醒"})
		}
		if task.Status == models.TaskStatusInProgress {
			events = append(events, gin.H{"type": "study_end", "task": task, "message": "计划完成提醒"})
			events = append(events, gin.H{"type": "decision_2330", "task": task, "message": "23:30 超时决策提醒"})
		}
		if !checked[task.PlanID] && task.Status != models.TaskStatusCompleted {
			events = append(events, gin.H{"type": "missed_checkin", "task": task, "message": "未打卡提醒"})
		}
	}
	api.OK(c, gin.H{"date": date, "events": events, "mode": "placeholder"})
}
