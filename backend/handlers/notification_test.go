package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

func TestSubscribeNotificationBindsResultToCurrentReminderTemplate(t *testing.T) {
	setupGroupTestDB(t)
	var cfg models.SubscriptionMessageConfig
	if err := db.DB.First(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	cfg.StudyStartEnabled = true
	cfg.StudyStartTemplateID = "shared-template"
	cfg.StudyStartPage = "pages/checkin/checkin"
	cfg.StudyStartFieldMapping = `{"thing1":"title"}`
	cfg.CompletionEnabled = true
	cfg.CompletionTemplateID = "shared-template"
	cfg.CompletionPage = "pages/checkin/checkin"
	cfg.CompletionFieldMapping = `{"thing1":"title"}`
	if err := db.DB.Save(&cfg).Error; err != nil {
		t.Fatal(err)
	}

	accepted := notificationRequest(map[string]string{"reminder_type": "study_start", "template_id": "shared-template", "result": "accept"}, 4)
	if recorderResponseCode(t, accepted) != 0 {
		t.Fatal(accepted.Body.String())
	}
	var subscriptions []models.NotificationSubscription
	db.DB.Where("user_id = ?", 4).Find(&subscriptions)
	if len(subscriptions) != 1 || subscriptions[0].ReminderType != "study_start" {
		t.Fatalf("authorization affected the wrong reminder: %+v", subscriptions)
	}

	stale := notificationRequest(map[string]string{"reminder_type": "study_start", "template_id": "old-template", "result": "accept"}, 4)
	if recorderResponseCode(t, stale) != http.StatusBadRequest {
		t.Fatalf("stale template accepted: %s", stale.Body.String())
	}
}

func notificationRequest(payload interface{}, userID uint) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/notifications/subscribe", bytes.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set(middleware.CtxUserIDKey, userID)
	SubscribeNotification(context)
	return recorder
}
