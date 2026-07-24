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

func TestCoveredMinutesUsesUnionAndRejectsAtSixty(t *testing.T) {
	base := models.DailyTask{ID: 2, Date: "2026-07-20", Title: "B", PlannedStart: "10:00", PlannedEnd: "12:00"}
	duplicateCoverage := []models.DailyTask{
		{ID: 1, Date: base.Date, Title: "A", PlannedStart: "10:00", PlannedEnd: "10:30"},
		base,
		{ID: 3, Date: base.Date, Title: "C", PlannedStart: "10:00", PlannedEnd: "10:30"},
	}
	if err := validateScheduleTasks(duplicateCoverage); err != nil {
		t.Fatalf("duplicate 30-minute coverage must count once: %v", err)
	}
	belowLimit := append([]models.DailyTask{}, duplicateCoverage[:2]...)
	belowLimit = append(belowLimit, models.DailyTask{ID: 3, Date: base.Date, Title: "C", PlannedStart: "11:30", PlannedEnd: "11:59"})
	if err := validateScheduleTasks(belowLimit); err != nil {
		t.Fatalf("59 covered minutes must pass: %v", err)
	}
	atLimit := append([]models.DailyTask{}, duplicateCoverage[:2]...)
	atLimit = append(atLimit, models.DailyTask{ID: 3, Date: base.Date, Title: "C", PlannedStart: "11:30", PlannedEnd: "12:00"})
	err := validateScheduleTasks(atLimit)
	conflict, ok := err.(*scheduleConflictError)
	if !ok || len(conflict.InvalidTasks) != 1 || conflict.InvalidTasks[0].TaskID != base.ID || conflict.InvalidTasks[0].CoveredMinutes != 60 {
		t.Fatalf("expected only middle task B to fail at 60 minutes: %#v", err)
	}
}

func TestStartTaskConflictsWithAnotherActiveTask(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "active-user", Nickname: "Active User", NicknameNormalized: "active user"}
	db.DB.Create(&user)
	plan := models.Plan{UserID: user.ID, Title: "P"}
	db.DB.Create(&plan)
	taskA := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-20", Title: "A", Objective: "finish A", Status: models.TaskStatusInProgress}
	taskB := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-21", Title: "B", Objective: "finish B", Status: models.TaskStatusPending}
	db.DB.Create(&taskA)
	db.DB.Create(&taskB)
	session := models.StudySession{UserID: user.ID, TaskID: taskA.ID, StartTime: time.Now()}
	db.DB.Create(&session)
	body := callJSONHandler(t, StartTask, user.ID, "/tasks/:id", strconv.FormatUint(uint64(taskB.ID), 10), gin.H{})
	var response struct {
		Code int `json:"code"`
		Data struct {
			ActiveTaskID    uint `json:"active_task_id"`
			ActiveSessionID uint `json:"active_session_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusConflict || response.Data.ActiveTaskID != taskA.ID || response.Data.ActiveSessionID != session.ID {
		t.Fatalf("unexpected conflict: %s", body)
	}
	db.DB.First(&taskB, taskB.ID)
	if taskB.Status != models.TaskStatusPending {
		t.Fatal("conflicting task must not be mutated")
	}
}

func TestDailyCheckinNeedsAnyCompletedTaskAcrossPlans(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "daily", Nickname: "Daily User", NicknameNormalized: "daily user"}
	db.DB.Create(&user)
	planA := models.Plan{UserID: user.ID, Title: "A"}
	planB := models.Plan{UserID: user.ID, Title: "B"}
	db.DB.Create(&planA)
	db.DB.Create(&planB)
	date := "2026-07-20"
	db.DB.Create(&models.DailyTask{UserID: user.ID, PlanID: planA.ID, Date: date, Title: "done", Status: models.TaskStatusCompleted})
	db.DB.Create(&models.DailyTask{UserID: user.ID, PlanID: planB.ID, Date: date, Title: "pending", Status: models.TaskStatusPending})
	request := gin.H{"date": date, "completed": true}
	if code := responseCode(t, callJSONHandler(t, CompleteDailyCheckin, user.ID, "/checkins/daily", "", request)); code != 0 {
		t.Fatalf("daily checkin should succeed, code=%d", code)
	}
	if code := responseCode(t, callJSONHandler(t, CompleteDailyCheckin, user.ID, "/checkins/daily", "", request)); code != 0 {
		t.Fatalf("retry should succeed, code=%d", code)
	}
	var count int64
	db.DB.Model(&models.DailyCheckin{}).Where("user_id = ? AND date = ?", user.ID, date).Count(&count)
	db.DB.First(&user, user.ID)
	if count != 1 || user.SlackBalance != 10 {
		t.Fatalf("expected one record and one reward, count=%d balance=%d", count, user.SlackBalance)
	}
}

func TestPendingDecisionExcludesNormalRunningTask(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "decision", Nickname: "Decision User", NicknameNormalized: "decision user"}
	db.DB.Create(&user)
	plan := models.Plan{UserID: user.ID, Title: "P"}
	db.DB.Create(&plan)
	date := "2026-07-20"
	db.DB.Create(&models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: date, Title: "running", Status: models.TaskStatusInProgress})
	db.DB.Create(&models.DailyTask{UserID: user.ID, PlanID: plan.ID + 1, Date: date, Title: "decision", Status: models.TaskStatusPending, NeedsDecision: true})
	g := gin.New()
	g.GET("/tasks/pending-decision", func(c *gin.Context) { c.Set("user_id", user.ID) }, PendingDecisionTasks)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/tasks/pending-decision?date="+date, nil)
	g.ServeHTTP(recorder, request)
	response := recorder.Body.Bytes()
	var payload struct {
		Data []struct {
			Task models.DailyTask `json:"task"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Data) != 1 || !payload.Data[0].Task.NeedsDecision {
		t.Fatalf("unexpected pending decisions: %s", response)
	}
}

func TestUpdatePlanTitleDoesNotRescheduleTasks(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "plan-update", Nickname: "Plan User", NicknameNormalized: "plan user"}
	db.DB.Create(&user)
	plan := models.Plan{UserID: user.ID, Title: "Old", DefaultPlannedStart: "20:00", DefaultPlannedEnd: "21:00"}
	db.DB.Create(&plan)
	task := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-20", Title: "Task", Objective: "finish task", PlannedStart: "09:15", PlannedEnd: "10:15", Status: models.TaskStatusPending}
	db.DB.Create(&task)

	body := callJSONHandler(t, UpdatePlan, user.ID, "/plans/:id", strconv.FormatUint(uint64(plan.ID), 10), gin.H{"title": "New"})
	if code := responseCode(t, body); code != 0 {
		t.Fatalf("update failed: %s", body)
	}
	db.DB.First(&task, task.ID)
	if task.PlannedStart != "09:15" || task.PlannedEnd != "10:15" {
		t.Fatalf("non-schedule update changed task slot: %+v", task)
	}
}

func TestSearchUsersEscapesWildcardsAndRanksInSQL(t *testing.T) {
	setupTestDB(t)
	requester := models.User{OpenID: "searcher", Nickname: "Searcher", NicknameNormalized: "searcher"}
	db.DB.Create(&requester)
	for index, nickname := range []string{"a_b", "axb", "x-a_b"} {
		db.DB.Create(&models.User{OpenID: "target-" + strconv.Itoa(index), Nickname: nickname, NicknameNormalized: nickname, InviteTargetID: "opaque-" + strconv.Itoa(index), AccountStatus: models.AccountStatusActive})
	}
	g := gin.New()
	g.GET("/users/search", func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, requester.ID) }, SearchUsers)
	recorder := httptest.NewRecorder()
	g.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/users/search?q=a_b", nil))
	var response struct {
		Code int `json:"code"`
		Data []struct {
			Nickname string `json:"nickname"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != 0 || len(response.Data) != 2 || response.Data[0].Nickname != "a_b" || response.Data[1].Nickname != "x-a_b" {
		t.Fatalf("unexpected escaped/ranked search: %s", recorder.Body.String())
	}
}

func TestPlanActionLayoutNormalizationPreservesDirectOrder(t *testing.T) {
	direct, overflow := normalizePlanActionLayout([]string{"delete", "edit", "invite"}, []string{"postpone"})
	if got := strings.Join(direct, ","); got != "delete,edit,invite" {
		t.Fatalf("direct order was truncated: %s", got)
	}
	if got := strings.Join(overflow, ","); got != "postpone,toggle_status" {
		t.Fatalf("missing actions must be appended to overflow: %s", got)
	}
}

func TestRecoveryEmptyActionsConsumesTokenOnlyAfterSuccess(t *testing.T) {
	setupTestDB(t)
	token, err := storeRecoveryPreview(42, nil)
	if err != nil {
		t.Fatal(err)
	}
	requestBody, _ := json.Marshal(gin.H{"token": token, "actions": []interface{}{}})
	g := gin.New()
	g.POST("/recovery/apply", func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, uint(42)) }, ApplyRecovery)
	first := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/recovery/apply", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	g.ServeHTTP(first, request)
	if responseCode(t, first.Body.Bytes()) != 0 {
		t.Fatalf("empty recovery should be a no-op: %s", first.Body.String())
	}
	second := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/recovery/apply", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	g.ServeHTTP(second, request)
	if responseCode(t, second.Body.Bytes()) != http.StatusConflict {
		t.Fatalf("successful no-op must consume token: %s", second.Body.String())
	}
}
