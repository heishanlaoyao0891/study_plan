package handlers

import (
	"errors"
	"net/http"
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

var reminderTypes = []string{"study_start", "completion", "decision_2330", "missed_checkin", "group_nudge"}

type subscriptionReq struct {
	ReminderTypes []string `json:"reminder_types"`
}

func NotificationSubscriptions(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var subs []models.NotificationSubscription
	if err := db.DB.Where("user_id = ?", uid).Order("reminder_type ASC").Find(&subs).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query subscriptions failed: "+err.Error())
		return
	}
	api.OK(c, subs)
}

func SubscribeNotification(c *gin.Context) {
	upsertNotificationSubscriptions(c, true)
}

func UnsubscribeNotification(c *gin.Context) {
	upsertNotificationSubscriptions(c, false)
}

func DueNotificationEvents(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	date := c.DefaultQuery("date", shanghaiToday())
	if _, err := time.Parse(dateLayout, date); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid date, expect YYYY-MM-DD")
		return
	}
	var tasks []models.DailyTask
	if err := db.DB.Where("user_id = ? AND date = ?", uid, date).Order("sort_order ASC, id ASC").Find(&tasks).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query notification tasks failed: "+err.Error())
		return
	}
	var checkin models.DailyCheckin
	checkinErr := db.DB.Where("user_id = ? AND date = ? AND completed = ?", uid, date, true).First(&checkin).Error
	if checkinErr != nil && !errors.Is(checkinErr, gorm.ErrRecordNotFound) {
		api.Fail(c, http.StatusInternalServerError, "query notification checkins failed: "+checkinErr.Error())
		return
	}
	checked := checkinErr == nil
	events := make([]notificationEvent, 0)
	for _, task := range tasks {
		if task.Status == models.TaskStatusPending && task.ActualStart == nil {
			events = append(events, notificationEvent{Type: "study_start", Task: task, Message: "到点学习提醒"})
		}
		if task.Status == models.TaskStatusInProgress {
			events = append(events, notificationEvent{Type: "completion", Task: task, Message: "计划完成提醒"})
			events = append(events, notificationEvent{Type: "decision_2330", Task: task, Message: "23:30 超时决策提醒"})
		}
		if !checked && task.Status != models.TaskStatusCompleted {
			events = append(events, notificationEvent{Type: "missed_checkin", Task: task, Message: "未打卡提醒"})
		}
	}
	deliveries := []models.NotificationDeliveryLog{}
	if strings.EqualFold(c.Query("send"), "true") {
		deliveries = deliverNotificationEvents(uid, events)
	}
	api.OK(c, gin.H{"date": date, "events": events, "deliveries": deliveries})
}

type notificationEvent struct {
	Type    string           `json:"type"`
	Task    models.DailyTask `json:"task"`
	Message string           `json:"message"`
}

func upsertNotificationSubscriptions(c *gin.Context, subscribed bool) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req subscriptionReq
	_ = c.ShouldBindJSON(&req)
	types := normalizeReminderTypes(req.ReminderTypes)
	subs := make([]models.NotificationSubscription, 0, len(types))
	for _, reminderType := range types {
		var sub models.NotificationSubscription
		err := db.DB.Where("user_id = ? AND reminder_type = ?", uid, reminderType).First(&sub).Error
		if err != nil {
			sub = models.NotificationSubscription{UserID: uid, ReminderType: reminderType}
		}
		sub.Subscribed = subscribed
		if err := db.DB.Save(&sub).Error; err != nil {
			api.Fail(c, http.StatusInternalServerError, "save subscription failed: "+err.Error())
			return
		}
		subs = append(subs, sub)
	}
	api.OK(c, gin.H{"subscribed": subscribed, "subscriptions": subs})
}

func normalizeReminderTypes(types []string) []string {
	allowed := map[string]bool{}
	for _, item := range reminderTypes {
		allowed[item] = true
	}
	result := make([]string, 0)
	for _, item := range types {
		if allowed[item] {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		return reminderTypes
	}
	return result
}

func deliverNotificationEvents(userID uint, events []notificationEvent) []models.NotificationDeliveryLog {
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		return nil
	}
	cfg, err := firstSubscriptionConfig()
	if err != nil {
		return nil
	}
	logs := make([]models.NotificationDeliveryLog, 0, len(events))
	for _, event := range events {
		status := "sent"
		message := ""
		templateID, enabled := notificationTemplate(cfg, event.Type)
		if !enabled {
			status = "skipped_disabled"
			message = "reminder type disabled"
		} else if templateID == "" {
			status = "skipped_missing_template"
			message = "template id is not configured"
		} else if !userSubscribed(userID, event.Type) {
			status = "skipped_missing_subscription"
			message = "user has not subscribed to this template"
		} else if err := services.SendSubscriptionMessage(user.OpenID, templateID, event.Message); err != nil {
			status = "failed"
			message = err.Error()
		}
		log := models.NotificationDeliveryLog{UserID: userID, ReminderType: event.Type, Status: status, Message: message}
		db.DB.Create(&log)
		logs = append(logs, log)
	}
	return logs
}

func userSubscribed(userID uint, reminderType string) bool {
	var count int64
	db.DB.Model(&models.NotificationSubscription{}).Where("user_id = ? AND reminder_type = ? AND subscribed = ?", userID, reminderType, true).Count(&count)
	return count > 0
}

func notificationTemplate(cfg models.SubscriptionMessageConfig, reminderType string) (string, bool) {
	switch reminderType {
	case "study_start":
		return cfg.StudyStartTemplateID, cfg.StudyStartEnabled
	case "completion":
		return cfg.CompletionTemplateID, cfg.CompletionEnabled
	case "decision_2330":
		return cfg.DecisionTemplateID, cfg.DecisionEnabled
	case "missed_checkin":
		return cfg.MissedCheckinTemplateID, cfg.MissedCheckinEnabled
	default:
		return "", false
	}
}
