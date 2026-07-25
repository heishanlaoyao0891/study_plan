package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"study_plan_backend/models"
)

func notificationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&models.User{}, &models.DailyTask{}, &models.DailyCheckin{}, &models.SubscriptionMessageConfig{}, &models.NotificationSubscription{}, &models.NotificationDeliveryLog{}); err != nil {
		t.Fatal(err)
	}
	if err := database.Exec("CREATE UNIQUE INDEX idx_notification_delivery_event_key ON notification_delivery_logs (event_key) WHERE event_key <> ''").Error; err != nil {
		t.Fatal(err)
	}
	return database
}

func TestRunNotificationCycleSendsDueEventOnce(t *testing.T) {
	database := notificationTestDB(t)
	user := models.User{OpenID: "openid", Nickname: "Test"}
	database.Create(&user)
	database.Create(&models.SubscriptionMessageConfig{StudyStartEnabled: true, StudyStartTemplateID: "template", StudyStartPage: "pages/checkin/checkin", StudyStartFieldMapping: `{"thing1":"title"}`})
	database.Create(&models.NotificationSubscription{UserID: user.ID, ReminderType: "study_start", TemplateID: "template", Subscribed: true})
	database.Create(&models.DailyTask{UserID: user.ID, PlanID: 1, Date: "2026-07-25", Title: "Read", PlannedStart: "09:00", PlannedEnd: "10:00", Status: models.TaskStatusPending})

	sends := 0
	sender := func(openid, templateID, page string, data map[string]any) error {
		sends++
		if openid != "openid" || templateID != "template" || page != "pages/checkin/checkin" {
			t.Fatalf("unexpected send arguments: %s %s %s", openid, templateID, page)
		}
		return nil
	}
	now := time.Date(2026, 7, 25, 9, 0, 30, 0, time.FixedZone("CST", 8*60*60))
	first, err := RunNotificationCycle(database, now, sender)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RunNotificationCycle(database, now, sender)
	if err != nil {
		t.Fatal(err)
	}
	if sends != 1 || first.Sent != 1 || second.Processed != 0 {
		t.Fatalf("expected one idempotent send, sends=%d first=%+v second=%+v", sends, first, second)
	}
}

func TestDeliverNotificationRequiresAuthorizationForCurrentTemplate(t *testing.T) {
	database := notificationTestDB(t)
	user := models.User{OpenID: "openid", Nickname: "Test"}
	database.Create(&user)
	database.Create(&models.SubscriptionMessageConfig{StudyStartEnabled: true, StudyStartTemplateID: "new-template", StudyStartPage: "pages/checkin/checkin", StudyStartFieldMapping: `{"thing1":"title"}`})
	database.Create(&models.NotificationSubscription{UserID: user.ID, ReminderType: "study_start", TemplateID: "old-template", Subscribed: true})
	called := false
	delivery, _, err := DeliverNotification(database, "template-change", user.ID, "study_start", NotificationValues{Title: "Read"}, func(string, string, string, map[string]any) error {
		called = true
		return nil
	})
	if err != nil || called || delivery.Status != "skipped_missing_subscription" {
		t.Fatalf("stale template authorization was used: delivery=%+v called=%v err=%v", delivery, called, err)
	}
}

func TestDeliverNotificationRetriesFailedEvent(t *testing.T) {
	database := notificationTestDB(t)
	user := models.User{OpenID: "openid", Nickname: "Test"}
	database.Create(&user)
	database.Create(&models.SubscriptionMessageConfig{StudyStartEnabled: true, StudyStartTemplateID: "template", StudyStartPage: "pages/checkin/checkin", StudyStartFieldMapping: `{"thing1":"title"}`})
	database.Create(&models.NotificationSubscription{UserID: user.ID, ReminderType: "study_start", TemplateID: "template", Subscribed: true})
	attempts := 0
	sender := func(string, string, string, map[string]any) error {
		attempts++
		if attempts == 1 {
			return fmt.Errorf("temporary provider failure")
		}
		return nil
	}
	first, firstClaimed, firstErr := DeliverNotification(database, "retry-event", user.ID, "study_start", NotificationValues{Title: "Read"}, sender)
	if firstErr != nil || !firstClaimed || first.Status != "failed" {
		t.Fatalf("first failure was not recorded: %+v claimed=%v err=%v", first, firstClaimed, firstErr)
	}
	database.Create(&models.NotificationSubscription{UserID: user.ID, ReminderType: "study_start", TemplateID: "template", Subscribed: true})
	second, secondClaimed, secondErr := DeliverNotification(database, "retry-event", user.ID, "study_start", NotificationValues{Title: "Read"}, sender)
	if secondErr != nil || !secondClaimed || second.Status != "sent" || attempts != 2 {
		t.Fatalf("failed event was not retried: %+v claimed=%v attempts=%d err=%v", second, secondClaimed, attempts, secondErr)
	}
}

func TestDeliverNotificationRecordsSkippedWithoutCallingSender(t *testing.T) {
	database := notificationTestDB(t)
	database.Create(&models.SubscriptionMessageConfig{})
	called := false
	delivery, claimed, err := DeliverNotification(database, "group_nudge:1", 42, "group_nudge", NotificationValues{}, func(string, string, string, map[string]any) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !claimed || called || delivery.Status != "skipped_disabled" {
		t.Fatalf("unexpected delivery: claimed=%v called=%v delivery=%+v", claimed, called, delivery)
	}
}
