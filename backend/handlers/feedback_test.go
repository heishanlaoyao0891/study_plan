package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

func TestSubmitFeedbackValidationAndThrottling(t *testing.T) {
	setupGroupTestDB(t)
	for _, test := range []struct {
		name    string
		payload map[string]string
		message string
	}{
		{"category", map[string]string{"category": "bug", "content": "help"}, "category"},
		{"empty content", map[string]string{"category": "issue", "content": "  "}, "content required"},
		{"content length", map[string]string{"category": "issue", "content": strings.Repeat("界", feedbackContentMax+1)}, "content"},
		{"contact length", map[string]string{"category": "issue", "content": "help", "contact": strings.Repeat("a", feedbackContactMax+1)}, "contact"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := feedbackRequest(SubmitFeedback, test.payload, 1, "")
			if recorderResponseCode(t, response) != http.StatusBadRequest || !strings.Contains(response.Body.String(), test.message) {
				t.Fatalf("unexpected response: %s", response.Body.String())
			}
		})
	}

	for index := 0; index < feedbackRateLimit; index++ {
		response := feedbackRequest(SubmitFeedback, map[string]string{"category": "suggestion", "content": "idea"}, 2, "")
		if recorderResponseCode(t, response) != 0 {
			t.Fatalf("submission %d failed: %s", index, response.Body.String())
		}
	}
	blocked := feedbackRequest(SubmitFeedback, map[string]string{"category": "other", "content": "more"}, 2, "")
	if recorderResponseCode(t, blocked) != http.StatusTooManyRequests {
		t.Fatalf("expected throttling: %s", blocked.Body.String())
	}
	var count int64
	db.DB.Model(&models.FeedbackReport{}).Where("user_id = ?", 2).Count(&count)
	if count != feedbackRateLimit {
		t.Fatalf("throttled submission created a row: %d", count)
	}
}

func TestFeedbackRateLimitTriggerGuardsDirectInsert(t *testing.T) {
	setupGroupTestDB(t)
	for index := 0; index < feedbackRateLimit; index++ {
		if err := db.DB.Create(&models.FeedbackReport{UserID: 7, Category: "issue", Content: "report", Status: "open"}).Error; err != nil {
			t.Fatalf("seed feedback %d: %v", index, err)
		}
	}
	if err := db.DB.Create(&models.FeedbackReport{UserID: 7, Category: "issue", Content: "blocked", Status: "open"}).Error; err == nil || !strings.Contains(err.Error(), "feedback rate limit exceeded") {
		t.Fatalf("database guard did not reject fourth insert: %v", err)
	}
}

func TestOwnFeedbackIsIsolatedAndPrivate(t *testing.T) {
	setupGroupTestDB(t)
	adminID := uint(99)
	response := "fixed"
	now := time.Now()
	rows := []models.FeedbackReport{
		{UserID: 1, Category: "issue", Content: "mine", Contact: "secret", Status: "resolved", PublicResponse: &response, RespondedAt: &now, RespondedBy: &adminID},
		{UserID: 2, Category: "other", Content: "theirs", Contact: "other secret", Status: "open"},
	}
	if err := db.DB.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	recorder := feedbackRequest(ListOwnFeedback, nil, 1, "")
	if recorderResponseCode(t, recorder) != 0 {
		t.Fatal(recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "mine") || !strings.Contains(body, "fixed") || strings.Contains(body, "theirs") || strings.Contains(body, "secret") || strings.Contains(body, "responded_by") || strings.Contains(body, "user_id") {
		t.Fatalf("owner response leaked private data: %s", body)
	}
}

func TestAdminFeedbackFiltersAndUpdate(t *testing.T) {
	setupGroupTestDB(t)
	user := models.User{OpenID: "feedback-user", Nickname: "Feedback User"}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	reports := []models.FeedbackReport{
		{UserID: user.ID, Category: "issue", Content: "broken", Status: "open"},
		{UserID: user.ID, Category: "suggestion", Content: "idea", Status: "closed"},
	}
	if err := db.DB.Create(&reports).Error; err != nil {
		t.Fatal(err)
	}
	list := feedbackRequest(AdminListFeedback, nil, 10, "category=issue&status=open")
	if recorderResponseCode(t, list) != 0 || !strings.Contains(list.Body.String(), "broken") || strings.Contains(list.Body.String(), "idea") || !strings.Contains(list.Body.String(), "Feedback User") {
		t.Fatalf("filtered list failed: %s", list.Body.String())
	}
	contextID := strconv.FormatUint(uint64(reports[0].ID), 10)
	update := feedbackRequest(AdminUpdateFeedback, map[string]string{"status": "resolved", "public_response": "  已处理  "}, 10, contextID)
	if recorderResponseCode(t, update) != 0 {
		t.Fatal(update.Body.String())
	}
	var saved models.FeedbackReport
	db.DB.First(&saved, reports[0].ID)
	if saved.Status != "resolved" || saved.PublicResponse == nil || *saved.PublicResponse != "已处理" || saved.RespondedBy == nil || *saved.RespondedBy != 10 || saved.RespondedAt == nil {
		t.Fatalf("feedback update not recorded: %+v", saved)
	}
	var audits int64
	db.DB.Model(&models.AdminAuditLog{}).Where("admin_user_id = ? AND action_type = ?", 10, "update_feedback").Count(&audits)
	if audits != 1 {
		t.Fatalf("expected feedback audit, got %d", audits)
	}
}

func TestAdminFeedbackStatusOnlyPreservesResponseMetadataAndExplicitClearRemovesIt(t *testing.T) {
	setupGroupTestDB(t)
	response := "existing response"
	respondedAt := time.Now().Add(-time.Hour).Truncate(time.Second)
	respondedBy := uint(8)
	report := models.FeedbackReport{UserID: 1, Category: "issue", Content: "broken", Status: "open", PublicResponse: &response, RespondedAt: &respondedAt, RespondedBy: &respondedBy}
	if err := db.DB.Create(&report).Error; err != nil {
		t.Fatal(err)
	}

	statusOnly := feedbackRequest(AdminUpdateFeedback, map[string]interface{}{"status": "processing"}, 10, strconv.FormatUint(uint64(report.ID), 10))
	if recorderResponseCode(t, statusOnly) != 0 {
		t.Fatal(statusOnly.Body.String())
	}
	var saved models.FeedbackReport
	db.DB.First(&saved, report.ID)
	if saved.PublicResponse == nil || *saved.PublicResponse != response || saved.RespondedAt == nil || !saved.RespondedAt.Equal(respondedAt) || saved.RespondedBy == nil || *saved.RespondedBy != respondedBy {
		t.Fatalf("status update changed response metadata: %+v", saved)
	}

	cleared := feedbackRequest(AdminUpdateFeedback, map[string]interface{}{"status": "closed", "clear_public_response": true}, 10, strconv.FormatUint(uint64(report.ID), 10))
	if recorderResponseCode(t, cleared) != 0 {
		t.Fatal(cleared.Body.String())
	}
	if strings.Contains(cleared.Body.String(), response) || strings.Contains(cleared.Body.String(), "responded_at") {
		t.Fatalf("clear response returned stale metadata: %s", cleared.Body.String())
	}
	saved = models.FeedbackReport{}
	db.DB.First(&saved, report.ID)
	if saved.PublicResponse != nil || saved.RespondedAt != nil || saved.RespondedBy != nil {
		t.Fatalf("explicit clear retained response metadata: %+v", saved)
	}
}

func TestAdminFeedbackRejectsAmbiguousResponseMutation(t *testing.T) {
	setupGroupTestDB(t)
	report := models.FeedbackReport{UserID: 1, Category: "issue", Content: "broken", Status: "open"}
	if err := db.DB.Create(&report).Error; err != nil {
		t.Fatal(err)
	}
	id := strconv.FormatUint(uint64(report.ID), 10)
	for _, payload := range []map[string]interface{}{
		{"status": "resolved", "public_response": "  "},
		{"status": "resolved", "public_response": "fixed", "clear_public_response": true},
	} {
		response := feedbackRequest(AdminUpdateFeedback, payload, 10, id)
		if recorderResponseCode(t, response) != http.StatusBadRequest {
			t.Fatalf("ambiguous response mutation accepted: %s", response.Body.String())
		}
	}
}

func TestFeedbackAdminAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := performJSONRequestWithContext(middleware.RequireAdmin(), nil, func(context *gin.Context) {
		context.Set(middleware.CtxRoleKey, models.RoleUser)
	})
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("non-admin accepted: %s", recorder.Body.String())
	}
}

func feedbackRequest(handler gin.HandlerFunc, payload interface{}, userID uint, rawQuery string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/?"+rawQuery, bytes.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set(middleware.CtxUserIDKey, userID)
	if rawQuery != "" && !strings.Contains(rawQuery, "=") {
		context.Params = gin.Params{{Key: "id", Value: rawQuery}}
	}
	handler(context)
	return recorder
}
