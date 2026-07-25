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
	ReminderType string `json:"reminder_type"`
	TemplateID   string `json:"template_id"`
	Result       string `json:"result"`
}

func NotificationTemplateMetadata(c *gin.Context) {
	cfg, err := firstSubscriptionConfig()
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query subscription config failed: "+err.Error())
		return
	}
	templates := make([]gin.H, 0, len(reminderTypes))
	for _, reminderType := range reminderTypes {
		template := services.TemplateFor(cfg, reminderType)
		if template.Enabled && services.ValidateTemplate(template) == nil {
			templates = append(templates, gin.H{"reminder_type": reminderType, "template_id": template.TemplateID})
		}
	}
	api.OK(c, gin.H{"platform": "mp-weixin", "templates": templates})
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
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req subscriptionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	req.ReminderType = strings.TrimSpace(req.ReminderType)
	req.TemplateID = strings.TrimSpace(req.TemplateID)
	if req.ReminderType == "" || req.TemplateID == "" || (req.Result != "accept" && req.Result != "reject" && req.Result != "ban") {
		api.Fail(c, http.StatusBadRequest, "reminder_type, template_id, and a valid authorization result are required")
		return
	}
	cfg, err := firstSubscriptionConfig()
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query subscription config failed: "+err.Error())
		return
	}
	template := services.TemplateFor(cfg, req.ReminderType)
	if !template.Enabled || services.ValidateTemplate(template) != nil || template.TemplateID != req.TemplateID {
		api.Fail(c, http.StatusBadRequest, "template is not the current enabled template for reminder_type")
		return
	}
	accepted := make([]string, 0, 1)
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if req.Result != "accept" {
			return tx.Where("user_id = ? AND reminder_type = ?", uid, req.ReminderType).Delete(&models.NotificationSubscription{}).Error
		}
		sub := models.NotificationSubscription{UserID: uid, ReminderType: req.ReminderType, TemplateID: req.TemplateID, Subscribed: true}
		if err := tx.Where("user_id = ? AND reminder_type = ?", uid, req.ReminderType).Assign(models.NotificationSubscription{TemplateID: req.TemplateID, Subscribed: true}).FirstOrCreate(&sub).Error; err != nil {
			return err
		}
		accepted = append(accepted, req.ReminderType)
		return nil
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "save subscription results failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"accepted": accepted})
}

func UnsubscribeNotification(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	if err := db.DB.Where("user_id = ?", uid).Delete(&models.NotificationSubscription{}).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "remove subscriptions failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"subscribed": false})
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
	api.OK(c, gin.H{"date": date, "events": events})
}

type notificationEvent struct {
	Type    string           `json:"type"`
	Task    models.DailyTask `json:"task"`
	Message string           `json:"message"`
}

func userSubscribed(userID uint, reminderType string) bool {
	var count int64
	db.DB.Model(&models.NotificationSubscription{}).Where("user_id = ? AND reminder_type = ? AND subscribed = ?", userID, reminderType, true).Count(&count)
	return count > 0
}
