package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

func TestListPlansNormalizesMissingScheduleArrays(t *testing.T) {
	setupGroupTestDB(t)
	user := models.User{OpenID: "legacy-plan-user", Nickname: "Legacy Plan", NicknameNormalized: "legacy plan", Role: models.RoleUser, AccountStatus: models.AccountStatusActive}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	plan := models.Plan{UserID: user.ID, Title: "Legacy schedule", Status: models.PlanStatusActive}
	if err := db.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/plans", nil)
	context.Set(middleware.CtxUserIDKey, user.ID)
	ListPlans(context)
	if recorderResponseCode(t, recorder) != 0 {
		t.Fatalf("list plans failed: %s", recorder.Body.String())
	}
	var envelope struct {
		Data []struct {
			StudyWeekdays     []int                         `json:"study_weekdays"`
			StudyDates        []string                      `json:"study_dates"`
			ScheduleOverrides []models.PlanScheduleOverride `json:"schedule_overrides"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != 1 || envelope.Data[0].StudyWeekdays == nil || envelope.Data[0].StudyDates == nil || envelope.Data[0].ScheduleOverrides == nil {
		t.Fatalf("legacy schedule fields must be arrays, got: %s", recorder.Body.String())
	}
}

func TestGetPlanNormalizesMissingScheduleArrays(t *testing.T) {
	setupGroupTestDB(t)
	user := models.User{OpenID: "legacy-plan-detail-user", Nickname: "Legacy Detail", NicknameNormalized: "legacy detail", Role: models.RoleUser, AccountStatus: models.AccountStatusActive}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	plan := models.Plan{UserID: user.ID, Title: "Legacy schedule", Status: models.PlanStatusActive}
	if err := db.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/plans/1", nil)
	context.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(plan.ID), 10)}}
	context.Set(middleware.CtxUserIDKey, user.ID)
	GetPlan(context)
	if recorderResponseCode(t, recorder) != 0 {
		t.Fatalf("get plan failed: %s", recorder.Body.String())
	}
	var envelope struct {
		Data struct {
			StudyWeekdays     []int                         `json:"study_weekdays"`
			StudyDates        []string                      `json:"study_dates"`
			ScheduleOverrides []models.PlanScheduleOverride `json:"schedule_overrides"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.StudyWeekdays == nil || envelope.Data.StudyDates == nil || envelope.Data.ScheduleOverrides == nil {
		t.Fatalf("legacy schedule fields must be arrays, got: %s", recorder.Body.String())
	}
}
