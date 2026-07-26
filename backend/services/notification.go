package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"study_plan_backend/models"
)

const notificationDateLayout = "2006-01-02"

type NotificationTemplate struct {
	ReminderType string
	TemplateID   string
	Enabled      bool
	Page         string
	FieldMapping string
}

type NotificationValues struct {
	Message      string
	Title        string
	Date         string
	PlannedStart string
	PlannedEnd   string
	Sender       string
}

type SubscriptionSender func(openid, templateID, page string, data map[string]any) error

type NotificationCycleResult struct {
	Due       int `json:"due"`
	Processed int `json:"processed"`
	Sent      int `json:"sent"`
	Skipped   int `json:"skipped"`
	Failed    int `json:"failed"`
}

func TemplateFor(cfg models.SubscriptionMessageConfig, reminderType string) NotificationTemplate {
	switch reminderType {
	case "study_start":
		return NotificationTemplate{reminderType, cfg.StudyStartTemplateID, cfg.StudyStartEnabled, cfg.StudyStartPage, cfg.StudyStartFieldMapping}
	case "completion":
		return NotificationTemplate{reminderType, cfg.CompletionTemplateID, cfg.CompletionEnabled, cfg.CompletionPage, cfg.CompletionFieldMapping}
	case "decision_2330":
		return NotificationTemplate{reminderType, cfg.DecisionTemplateID, cfg.DecisionEnabled, cfg.DecisionPage, cfg.DecisionFieldMapping}
	case "missed_checkin":
		return NotificationTemplate{reminderType, cfg.MissedCheckinTemplateID, cfg.MissedCheckinEnabled, cfg.MissedCheckinPage, cfg.MissedCheckinMapping}
	case "group_nudge":
		return NotificationTemplate{reminderType, cfg.GroupNudgeTemplateID, cfg.GroupNudgeEnabled, cfg.GroupNudgePage, cfg.GroupNudgeFieldMapping}
	case "slack_balance":
		return NotificationTemplate{reminderType, cfg.SlackBalanceTemplateID, cfg.SlackBalanceEnabled, cfg.SlackBalancePage, cfg.SlackBalanceMapping}
	default:
		return NotificationTemplate{ReminderType: reminderType}
	}
}

func ValidateTemplate(template NotificationTemplate) error {
	if !template.Enabled {
		return nil
	}
	missing := make([]string, 0, 3)
	if strings.TrimSpace(template.TemplateID) == "" {
		missing = append(missing, "template ID")
	}
	if strings.TrimSpace(template.Page) == "" {
		missing = append(missing, "page target")
	}
	var mapping map[string]string
	if err := json.Unmarshal([]byte(template.FieldMapping), &mapping); err != nil || len(mapping) == 0 {
		missing = append(missing, "valid non-empty JSON field mapping")
	} else {
		for field, source := range mapping {
			if strings.TrimSpace(field) == "" || !validNotificationSource(source) {
				return fmt.Errorf("%s mapping contains unsupported field/source %q:%q", template.ReminderType, field, source)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s cannot be enabled: missing %s", template.ReminderType, strings.Join(missing, ", "))
	}
	return nil
}

func DeliverNotification(database *gorm.DB, eventKey string, userID uint, reminderType string, values NotificationValues, sender SubscriptionSender) (models.NotificationDeliveryLog, bool, error) {
	delivery := models.NotificationDeliveryLog{EventKey: eventKey, UserID: userID, ReminderType: reminderType, Status: "processing"}
	if err := database.Create(&delivery).Error; err != nil {
		var existing models.NotificationDeliveryLog
		if lookupErr := database.Where("event_key = ?", eventKey).First(&existing).Error; lookupErr == nil {
			staleBefore := time.Now().Add(-5 * time.Minute)
			claim := database.Model(&models.NotificationDeliveryLog{}).
				Where("id = ? AND (status = ? OR (status = ? AND updated_at < ?))", existing.ID, "failed", "processing", staleBefore).
				Updates(map[string]any{"status": "processing", "message": ""})
			if claim.Error != nil {
				return existing, false, claim.Error
			}
			if claim.RowsAffected == 0 {
				return existing, false, nil
			}
			delivery = existing
			delivery.Status, delivery.Message = "processing", ""
		} else {
			return delivery, false, err
		}
	}

	status, message := "sent", ""
	var cfg models.SubscriptionMessageConfig
	if err := database.Order("id ASC").First(&cfg).Error; err != nil {
		status, message = "failed", "load subscription config: "+err.Error()
	} else {
		template := TemplateFor(cfg, reminderType)
		var subscribed int64
		database.Model(&models.NotificationSubscription{}).Where("user_id = ? AND reminder_type = ? AND template_id = ? AND subscribed = ?", userID, reminderType, template.TemplateID, true).Count(&subscribed)
		var user models.User
		switch {
		case !template.Enabled:
			status, message = "skipped_disabled", "reminder type disabled"
		case ValidateTemplate(template) != nil:
			status, message = "skipped_invalid_config", ValidateTemplate(template).Error()
		case subscribed == 0:
			status, message = "skipped_missing_subscription", "user has not authorized this template"
		case database.First(&user, userID).Error != nil || user.OpenID == "":
			status, message = "skipped_missing_openid", "user has no linked mini-program identity"
		default:
			data, err := buildTemplateData(template.FieldMapping, values)
			if err != nil {
				status, message = "failed", err.Error()
			} else if err := sender(user.OpenID, template.TemplateID, template.Page, data); err != nil {
				status, message = "failed", err.Error()
			} else if err := database.Where("user_id = ? AND reminder_type = ? AND template_id = ?", userID, reminderType, template.TemplateID).Delete(&models.NotificationSubscription{}).Error; err != nil {
				status, message = "failed", "consume subscription: "+err.Error()
			}
		}
	}
	delivery.Status, delivery.Message = status, message
	if err := database.Model(&delivery).Updates(map[string]any{"status": status, "message": message}).Error; err != nil {
		return delivery, true, err
	}
	return delivery, true, nil
}

func RunNotificationCycle(database *gorm.DB, now time.Time, sender SubscriptionSender) (NotificationCycleResult, error) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("CST", 8*60*60)
	}
	now = now.In(location)
	date := now.Format(notificationDateLayout)
	var tasks []models.DailyTask
	if err := database.Where("date = ?", date).Find(&tasks).Error; err != nil {
		return NotificationCycleResult{}, err
	}
	checked := map[uint]bool{}
	var checkins []models.DailyCheckin
	if err := database.Where("date = ? AND completed = ?", date, true).Find(&checkins).Error; err != nil {
		return NotificationCycleResult{}, err
	}
	for _, checkin := range checkins {
		checked[checkin.UserID] = true
	}

	result := NotificationCycleResult{}
	for _, task := range tasks {
		events := dueTaskEvents(task, checked[task.UserID], now)
		result.Due += len(events)
		for _, event := range events {
			values := NotificationValues{Message: event.message, Title: task.Title, Date: task.Date, PlannedStart: task.PlannedStart, PlannedEnd: task.PlannedEnd}
			key := fmt.Sprintf("%s:task:%d:%s", event.reminderType, task.ID, task.Date)
			delivery, claimed, deliverErr := DeliverNotification(database, key, task.UserID, event.reminderType, values, sender)
			if deliverErr != nil {
				return result, deliverErr
			}
			if !claimed {
				continue
			}
			result.Processed++
			switch {
			case delivery.Status == "sent":
				result.Sent++
			case delivery.Status == "failed":
				result.Failed++
			default:
				result.Skipped++
			}
		}
	}
	return result, nil
}

func StartNotificationScheduler(database *gorm.DB) {
	go func() {
		run := func() {
			result, err := RunNotificationCycle(database, time.Now(), SendSubscriptionMessage)
			if err != nil {
				log.Printf("[notifications] cycle failed: %v", err)
			} else if result.Processed > 0 {
				log.Printf("[notifications] processed=%d sent=%d skipped=%d failed=%d", result.Processed, result.Sent, result.Skipped, result.Failed)
			}
		}
		run()
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()
}

type dueEvent struct {
	reminderType string
	message      string
}

func dueTaskEvents(task models.DailyTask, checked bool, now time.Time) []dueEvent {
	minute := now.Hour()*60 + now.Minute()
	start, startOK := parseMinute(task.PlannedStart)
	end, endOK := parseMinute(task.PlannedEnd)
	events := make([]dueEvent, 0, 4)
	if startOK && minute == start && task.Status == models.TaskStatusPending && task.ActualStart == nil {
		events = append(events, dueEvent{"study_start", "学习计划已到开始时间"})
	}
	if endOK && minute == end && task.Status == models.TaskStatusInProgress {
		events = append(events, dueEvent{"completion", "学习计划已到完成时间"})
	}
	if minute == 23*60+30 && (task.Status == models.TaskStatusInProgress || task.NeedsDecision) {
		events = append(events, dueEvent{"decision_2330", "请处理今天未结束的学习任务"})
	}
	if endOK && minute == end && !checked && task.Status != models.TaskStatusCompleted {
		events = append(events, dueEvent{"missed_checkin", "今天的学习任务尚未打卡"})
	}
	return events
}

func parseMinute(value string) (int, bool) {
	parsed, err := time.Parse("15:04", value)
	return parsed.Hour()*60 + parsed.Minute(), err == nil
}

func validNotificationSource(source string) bool {
	switch source {
	case "message", "title", "date", "planned_start", "planned_end", "sender":
		return true
	default:
		return false
	}
}

func buildTemplateData(raw string, values NotificationValues) (map[string]any, error) {
	var mapping map[string]string
	if err := json.Unmarshal([]byte(raw), &mapping); err != nil {
		return nil, err
	}
	sources := map[string]string{"message": values.Message, "title": values.Title, "date": values.Date, "planned_start": values.PlannedStart, "planned_end": values.PlannedEnd, "sender": values.Sender}
	data := make(map[string]any, len(mapping))
	for field, source := range mapping {
		value, ok := sources[source]
		if !ok {
			return nil, errors.New("unsupported template mapping source: " + source)
		}
		data[field] = map[string]string{"value": value}
	}
	return data, nil
}
