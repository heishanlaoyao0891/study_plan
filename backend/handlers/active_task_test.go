package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

func TestGetActiveTaskReturnsOriginalSessionElapsedTime(t *testing.T) {
	setupTestDB(t)
	user := createActiveTaskTestUser(t, "active-task-owner")
	plan := models.Plan{UserID: user.ID, Title: "Timer plan"}
	if err := db.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	task := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-31", Title: "Keep studying", Objective: "Finish the lesson", PlannedStart: "20:00", PlannedEnd: "21:00", Status: models.TaskStatusInProgress, StudySeconds: 120}
	if err := db.DB.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	startedAt := time.Now().Add(-90 * time.Second)
	if err := db.DB.Create(&models.StudySession{UserID: user.ID, TaskID: task.ID, StartTime: startedAt}).Error; err != nil {
		t.Fatal(err)
	}

	response := activeTaskResponse(t, user.ID)
	if response.Code != 0 || response.Data == nil {
		t.Fatalf("active task response missing: %+v", response)
	}
	if response.Data.ID != task.ID || response.Data.ActiveSession == nil || response.Data.ActiveSession.TaskID != task.ID {
		t.Fatalf("active task must return the owner's open session: %+v", response.Data)
	}
	if response.Data.AccumulatedSeconds < 208 || response.Data.TimerState != "running" {
		t.Fatalf("expected original session duration to continue, got %+v", response.Data)
	}
}

func TestGetActiveTaskReturnsEmptyAndNeverLeaksAnotherUserSession(t *testing.T) {
	setupTestDB(t)
	owner := createActiveTaskTestUser(t, "active-task-owner")
	other := createActiveTaskTestUser(t, "active-task-other")
	plan := models.Plan{UserID: owner.ID, Title: "Owner plan"}
	if err := db.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	task := models.DailyTask{UserID: owner.ID, PlanID: plan.ID, Date: "2026-07-31", Title: "Private task", Objective: "Finish privately", PlannedStart: "20:00", PlannedEnd: "21:00", Status: models.TaskStatusInProgress}
	if err := db.DB.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&models.StudySession{UserID: owner.ID, TaskID: task.ID, StartTime: time.Now().Add(-time.Minute)}).Error; err != nil {
		t.Fatal(err)
	}

	response := activeTaskResponse(t, other.ID)
	if response.Code != 0 || response.Data != nil {
		t.Fatalf("another user's active session must not be visible: %+v", response)
	}
}

type activeTaskEnvelope struct {
	Code int            `json:"code"`
	Data *taskTimerView `json:"data"`
}

func activeTaskResponse(t *testing.T, userID uint) activeTaskEnvelope {
	t.Helper()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/tasks/active", nil)
	context.Set(middleware.CtxUserIDKey, userID)
	GetActiveTask(context)
	var response activeTaskEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func createActiveTaskTestUser(t *testing.T, openID string) models.User {
	t.Helper()
	user := models.User{OpenID: openID, Nickname: openID, NicknameNormalized: openID, AccountStatus: models.AccountStatusActive}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user
}
