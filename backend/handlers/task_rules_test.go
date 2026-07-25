package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	config.App = &config.Config{DBPath: dsn}
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.DB = gdb
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

func TestFindTaskSlotConflictsSkipsCompletedTasks(t *testing.T) {
	setupTestDB(t)
	uid := uint(1)
	plan := models.Plan{UserID: uid, Title: "P"}
	if err := db.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&models.DailyTask{UserID: uid, PlanID: plan.ID, Date: "2026-07-20", Title: "A", PlannedStart: "20:00", PlannedEnd: "21:00", Status: models.TaskStatusPending}).Error; err != nil {
		t.Fatal(err)
	}
	plan2 := models.Plan{UserID: uid, Title: "P2"}
	if err := db.DB.Create(&plan2).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&models.DailyTask{UserID: uid, PlanID: plan2.ID, Date: "2026-07-20", Title: "B", PlannedStart: "20:30", PlannedEnd: "21:30", Status: models.TaskStatusCompleted}).Error; err != nil {
		t.Fatal(err)
	}
	conflicts, err := findTaskSlotConflicts(uid, 0, "2026-07-20", "20:15", "20:45")
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
}

func TestScheduleMutationExcludesCompletedButRetainsPendingConflicts(t *testing.T) {
	setupTestDB(t)
	uid := uint(11)
	plan := models.Plan{UserID: uid, Title: "Existing"}
	db.DB.Create(&plan)
	db.DB.Create(&models.DailyTask{UserID: uid, PlanID: plan.ID, Date: "2026-08-01", Title: "Done", PlannedStart: "20:00", PlannedEnd: "21:00", Status: models.TaskStatusCompleted})
	proposed := []models.DailyTask{{UserID: uid, Date: "2026-08-01", Title: "New", PlannedStart: "20:00", PlannedEnd: "21:00", Status: models.TaskStatusPending}}
	if err := validateScheduleMutation(db.DB, uid, proposed); err != nil {
		t.Fatalf("completed task must not occupy final validation: %v", err)
	}
	if err := db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND plan_id = ?", uid, plan.ID).Update("status", models.TaskStatusPending).Error; err != nil {
		t.Fatal(err)
	}
	if err := validateScheduleMutation(db.DB, uid, proposed); err == nil {
		t.Fatal("pending task must remain an active schedule conflict")
	}
}

func TestCloseOvernightTasks(t *testing.T) {
	setupTestDB(t)
	loc, _ := time.LoadLocation("Asia/Shanghai")
	user := models.User{OpenID: "u1", Nickname: "u1", SlackBalance: 0}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	plan := models.Plan{UserID: user.ID, Title: "P"}
	if err := db.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 7, 19, 23, 30, 0, 0, loc)
	task := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-19", Title: "A", Status: models.TaskStatusInProgress}
	if err := db.DB.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	session := models.StudySession{TaskID: task.ID, UserID: user.ID, StartTime: start}
	if err := db.DB.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	closed, err := closeOvernightTasks(user.ID, time.Date(2026, 7, 20, 0, 5, 0, 0, loc))
	if err != nil {
		t.Fatal(err)
	}
	if closed != 1 {
		t.Fatalf("expected 1 closed task, got %d", closed)
	}
	if err := db.DB.First(&task, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !task.NeedsDecision || task.Status != models.TaskStatusPending {
		t.Fatalf("unexpected task state: %+v", task)
	}
	if task.ActualEnd == nil || task.ActualEnd.In(loc).Hour() != 0 || task.ActualEnd.In(loc).Minute() != 0 {
		t.Fatalf("unexpected actual end: %+v", task.ActualEnd)
	}
}

func TestTaskCompletionDoesNotCheckinAndExplicitRewardIsIdempotent(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "u1", Nickname: "u1"}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	plan := models.Plan{UserID: user.ID, Title: "P"}
	if err := db.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	task := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-20", Title: "A", Status: models.TaskStatusCompleted}
	if err := db.DB.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.DB.Model(&models.Checkin{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("task completion must not create checkin: count=%d err=%v", count, err)
	}
	checkin := models.Checkin{UserID: user.ID, PlanID: plan.ID, Date: task.Date, Completed: true}
	if err := db.DB.Create(&checkin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error { return awardSlackIfNeeded(tx, user.ID, &checkin) }); err != nil {
		t.Fatal(err)
	}
	checkin.Rewarded = false
	if err := db.DB.Transaction(func(tx *gorm.DB) error { return awardSlackIfNeeded(tx, user.ID, &checkin) }); err != nil {
		t.Fatal(err)
	}
	if err := db.DB.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.SlackBalance != 10 {
		t.Fatalf("expected 10 slack minutes, got %d", user.SlackBalance)
	}
	if err := db.DB.Where("user_id = ? AND plan_id = ? AND date = ?", user.ID, plan.ID, task.Date).First(&checkin).Error; err != nil {
		t.Fatal(err)
	}
	if !checkin.Completed || !checkin.Rewarded {
		t.Fatalf("unexpected checkin state: %+v", checkin)
	}
}

func TestPausePreservesAccumulatedTimeAndAchievedOvertime(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "timer", Nickname: "timer"}
	db.DB.Create(&user)
	plan := models.Plan{UserID: user.ID, Title: "P"}
	db.DB.Create(&plan)
	task := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-20", Title: "A", Objective: "finish lesson one", PlannedStart: "20:00", PlannedEnd: "21:00", Status: models.TaskStatusInProgress, StudySeconds: 3500}
	if err := db.DB.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	start := time.Now().Add(-200 * time.Second)
	if err := db.DB.Create(&models.StudySession{TaskID: task.ID, UserID: user.ID, StartTime: start}).Error; err != nil {
		t.Fatal(err)
	}
	view, err := buildTaskTimerView(task, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if view.TimerState != "achieved" || view.OvertimeSeconds < 90 {
		t.Fatalf("unexpected achieved view: %+v", view)
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		_, err := closeActiveSession(tx, &task, user.ID, time.Now())
		task.Status = models.TaskStatusPending
		if err != nil {
			return err
		}
		return tx.Save(&task).Error
	}); err != nil {
		t.Fatal(err)
	}
	if task.Status == models.TaskStatusCompleted || task.StudySeconds < 3690 {
		t.Fatalf("pause must preserve incomplete accumulated state: %+v", task)
	}
}

func TestScheduleResolutionUsesDateThenWeekdayThenDefault(t *testing.T) {
	plan := models.Plan{DefaultPlannedStart: "20:00", DefaultPlannedEnd: "21:00", StudyWeekdays: []int{1}}
	day, _ := time.Parse(dateLayout, "2026-07-20")
	rows := []models.PlanScheduleOverride{{Weekday: 1, PlannedStart: "19:00", PlannedEnd: "20:00"}, {Date: "2026-07-20", PlannedStart: "18:00", PlannedEnd: "19:30"}}
	start, end := resolvePlanSchedule(plan, rows, day)
	if start != "18:00" || end != "19:30" {
		t.Fatalf("date override should win, got %s-%s", start, end)
	}
	if planStudiesOn(plan, day.AddDate(0, 0, 1)) {
		t.Fatal("unselected weekday must not generate a task")
	}
}

func TestDefaultWeekdaysExcludeSaturday(t *testing.T) {
	plan := models.Plan{UserID: 1, Title: "Read", StartDate: "2026-07-25", EndDate: "2026-07-27", StudyWeekdays: []int{1, 2, 3, 4, 5}}
	tasks, err := draftTasksForPlan(plan, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Date != "2026-07-27" {
		t.Fatalf("default weekdays should generate only Monday, got %+v", tasks)
	}
}

func TestExplicitTaskDraftsPreservePerDateValues(t *testing.T) {
	req := createPlanReq{Title: "Read", StartDate: "2026-07-25", EndDate: "2026-07-27", StudyWeekdays: []int{6, 1}, PublicToGroup: true, TaskDrafts: []taskDraftReq{
		{Date: "2026-07-25", Title: "Chapter 1", Objective: "finish chapter one exercises", Description: "notes", PlannedStart: "09:00", PlannedEnd: "10:00"},
		{Date: "2026-07-27", Title: "Chapter 2", Objective: "finish chapter two exercises", PlannedStart: "19:30", PlannedEnd: "20:45"},
	}}
	tasks, err := explicitTasksForPlan(7, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 || tasks[0].Title != "Chapter 1" || tasks[1].PlannedStart != "19:30" || tasks[1].EstimatedMinutes != 75 || !tasks[0].PublicToGroup {
		t.Fatalf("explicit task values were not preserved: %+v", tasks)
	}
	req.TaskDrafts[1].Date = "2026-07-26"
	if _, err := explicitTasksForPlan(7, req); err == nil {
		t.Fatal("unselected draft date must be rejected")
	}
}

func TestTaskStructureCannotChangeAfterStudyStarts(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "locked-task-user", Nickname: "locked-task-user"}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	plan := models.Plan{UserID: user.ID, Title: "P"}
	if err := db.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	for index, status := range []string{models.TaskStatusInProgress, models.TaskStatusCompleted} {
		task := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: fmt.Sprintf("2026-07-%02d", 25+index), Title: status, Objective: "finish the assigned lesson", PlannedStart: "09:00", PlannedEnd: "10:00", Status: status}
		if err := db.DB.Create(&task).Error; err != nil {
			t.Fatal(err)
		}
		update := performJSONRequestWithContext(UpdateTask, map[string]any{"title": "changed"}, func(c *gin.Context) {
			c.Set(middleware.CtxUserIDKey, user.ID)
			c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(task.ID), 10)}}
		})
		if recorderResponseCode(t, update) != http.StatusConflict {
			t.Fatalf("%s task update was allowed: %s", status, update.Body.String())
		}
		remove := performJSONRequestWithContext(DeleteTask, nil, func(c *gin.Context) {
			c.Set(middleware.CtxUserIDKey, user.ID)
			c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(task.ID), 10)}}
		})
		if recorderResponseCode(t, remove) != http.StatusConflict {
			t.Fatalf("%s task deletion was allowed: %s", status, remove.Body.String())
		}
	}
}

func TestValidationLimits(t *testing.T) {
	if validateObjective("Read", "Read") == nil {
		t.Fatal("objective repeating title must be rejected")
	}
	if validMotivationContent(strings.Repeat("学", 33), "今日寄语") {
		t.Fatal("over-limit motivation must be rejected")
	}
	if !validMotivationContent(strings.Repeat("学", 32), "今日寄语") {
		t.Fatal("bounded motivation should pass")
	}
}

func TestShiftPlanTasksSkipsCompletedTasks(t *testing.T) {
	setupTestDB(t)
	uid := uint(1)
	plan := models.Plan{UserID: uid, Title: "P", StartDate: "2026-07-20", EndDate: "2026-07-21"}
	if err := db.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	pending := models.DailyTask{UserID: uid, PlanID: plan.ID, Date: "2026-07-20", Title: "A", Status: models.TaskStatusPending}
	completed := models.DailyTask{UserID: uid, PlanID: plan.ID, Date: "2026-07-21", Title: "B", Status: models.TaskStatusCompleted}
	if err := db.DB.Create(&pending).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&completed).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		return shiftPlanTasks(tx, uid, &plan, 3, "2026-07-20")
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.DB.First(&plan, plan.ID).Error; err != nil {
		t.Fatal(err)
	}
	if plan.StartDate != "2026-07-23" || plan.EndDate != "2026-07-24" {
		t.Fatalf("unexpected plan dates: %+v", plan)
	}
	if err := db.DB.First(&pending, pending.ID).Error; err != nil {
		t.Fatal(err)
	}
	if pending.Date != "2026-07-23" {
		t.Fatalf("expected pending task shifted, got %s", pending.Date)
	}
	if err := db.DB.First(&completed, completed.ID).Error; err != nil {
		t.Fatal(err)
	}
	if completed.Date != "2026-07-21" {
		t.Fatalf("completed task should not shift, got %s", completed.Date)
	}
}

func TestMakeupCostDeductsSlackAndRecordsDelta(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "u1", Nickname: "u1", SlackBalance: 50}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&models.SlackConfig{UserID: &user.ID, MakeupCostRatio: 0.5}).Error; err != nil {
		t.Fatal(err)
	}
	cost := makeupSlackCost(user.ID, 40)
	if cost != 20 {
		t.Fatalf("expected cost 20, got %d", cost)
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", user.ID).Update("slack_balance", gorm.Expr("slack_balance - ?", cost)).Error; err != nil {
			return err
		}
		return recordSlackDelta(tx, user.ID, "补录消耗: A", -cost)
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.DB.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.SlackBalance != 30 {
		t.Fatalf("expected slack balance 30, got %d", user.SlackBalance)
	}
	var record models.SlackRecord
	if err := db.DB.Where("user_id = ? AND delta_min = ?", user.ID, -cost).First(&record).Error; err != nil {
		t.Fatal(err)
	}
	if record.Activity == "" || record.DeltaMin != -cost {
		t.Fatalf("unexpected record: %+v", record)
	}
}

func TestMarkStudyReviewFlags(t *testing.T) {
	task := models.DailyTask{StudyMinutes: suspiciousDailyMinutes + 1}
	session := models.StudySession{DurationMin: suspiciousSessionMinutes + 1}
	markStudyReviewFlags(&task, &session)
	if !task.Suspicious || !session.Suspicious {
		t.Fatalf("expected both records to be flagged: task=%+v session=%+v", task, session)
	}
}

func TestTaskStateActionsAreIdempotentAndGuarded(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "state", Nickname: "state"}
	db.DB.Create(&user)
	plan := models.Plan{UserID: user.ID, Title: "P", Status: models.PlanStatusActive}
	db.DB.Create(&plan)
	task := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-20", Title: "Read", Objective: "finish chapter one", PlannedStart: "20:00", PlannedEnd: "21:00", Status: models.TaskStatusPending, StudySeconds: 3600}
	if err := db.DB.Create(&task).Error; err != nil {
		t.Fatal(err)
	}

	if code := callTaskHandler(t, CompleteTask, user.ID, task.ID, nil); code != 0 {
		t.Fatalf("empty-body complete failed with code %d", code)
	}
	if err := db.DB.First(&task, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	completedAt := *task.ActualEnd
	if code := callTaskHandler(t, CompleteTask, user.ID, task.ID, nil); code != 0 {
		t.Fatalf("repeated complete failed with code %d", code)
	}
	db.DB.First(&task, task.ID)
	if !task.ActualEnd.Equal(completedAt) {
		t.Fatal("repeated complete must preserve the original completion time")
	}
	if code := callTaskHandler(t, PauseTask, user.ID, task.ID, nil); code != http.StatusConflict {
		t.Fatalf("expected completed pause conflict, got %d", code)
	}
}

func TestUpdateTaskCannotBypassStateOrInvalidateObjective(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "update", Nickname: "update"}
	db.DB.Create(&user)
	plan := models.Plan{UserID: user.ID, Title: "P"}
	db.DB.Create(&plan)
	task := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-20", Title: "Read", Objective: "finish chapter one", PlannedStart: "20:00", PlannedEnd: "21:00", Status: models.TaskStatusPending}
	db.DB.Create(&task)

	if code := callTaskHandler(t, UpdateTask, user.ID, task.ID, gin.H{"status": models.TaskStatusCompleted}); code != http.StatusBadRequest {
		t.Fatalf("expected status update rejection, got %d", code)
	}
	if code := callTaskHandler(t, UpdateTask, user.ID, task.ID, gin.H{"title": "finish chapter one"}); code != http.StatusBadRequest {
		t.Fatalf("expected final objective validation failure, got %d", code)
	}
	if code := callTaskHandler(t, UpdateTask, user.ID, task.ID, gin.H{"planned_end": "19:00"}); code != http.StatusBadRequest {
		t.Fatalf("expected invalid planned range rejection, got %d", code)
	}
	task.Status = models.TaskStatusInProgress
	db.DB.Save(&task)
	if code := callTaskHandler(t, PostponeTask, user.ID, task.ID, gin.H{"date": "2026-07-21"}); code != http.StatusConflict {
		t.Fatalf("expected running postpone rejection, got %d", code)
	}
}

func TestMakeupDateTimeUsesShanghaiDateAndStoresSeconds(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "makeup", Nickname: "makeup", SlackBalance: 200}
	db.DB.Create(&user)
	plan := models.Plan{UserID: user.ID, Title: "P"}
	db.DB.Create(&plan)
	task := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2020-01-02", Title: "Read", Objective: "finish chapter one", PlannedStart: "20:00", PlannedEnd: "21:00", Status: models.TaskStatusPending}
	db.DB.Create(&task)

	code := callTaskHandler(t, MakeupTask, user.ID, task.ID, gin.H{"actual_date": "2020-01-02", "actual_start": "20:00", "actual_end": "20:01"})
	if code != 0 {
		t.Fatalf("makeup failed with code %d", code)
	}
	db.DB.First(&task, task.ID)
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if task.StudySeconds != 60 || task.StudyMinutes != 1 || task.ActualStart.In(loc).Format("2006-01-02 15:04") != "2020-01-02 20:00" {
		t.Fatalf("unexpected makeup result: %+v", task)
	}
}

func TestEnsureDailyTaskRequiresActivePlanAndDoesNotInventObjective(t *testing.T) {
	setupTestDB(t)
	plan := models.Plan{UserID: 1, Title: "P", Description: "not an objective", Status: models.PlanStatusPaused, StudyDates: []string{"2026-07-20"}}
	db.DB.Create(&plan)
	if _, err := ensureDailyTask(1, plan, "2026-07-20"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("paused plan should not ensure a task: %v", err)
	}
	plan.Status = models.PlanStatusActive
	if _, err := ensureDailyTask(1, plan, "2026-07-20"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("plan without persisted objective cannot auto-create a task: %v", err)
	}
	var count int64
	db.DB.Model(&models.DailyTask{}).Count(&count)
	if count != 0 {
		t.Fatal("ensureDailyTask must not use description as objective")
	}
}

func TestPlanDateValidationAndActiveSessionIndex(t *testing.T) {
	if validatePlanDates("2026-07-20", "", nil, nil) == nil {
		t.Fatal("partial plan date range must be rejected")
	}
	if validatePlanDates("2026-07-20", "2026-07-21", []string{"2026-07-22"}, nil) == nil {
		t.Fatal("out-of-range study date must be rejected")
	}
	if validatePlanDates("2026-07-20", "2026-07-21", nil, []scheduleOverrideReq{{Date: "2026-07-22"}}) == nil {
		t.Fatal("out-of-range override date must be rejected")
	}

	setupTestDB(t)
	if !db.DB.Migrator().HasIndex(&models.StudySession{}, "idx_study_sessions_active_user") {
		t.Fatal("expected user-level active session partial unique index")
	}
	session := models.StudySession{TaskID: 1, UserID: 1, StartTime: time.Now()}
	if err := db.DB.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&models.StudySession{TaskID: 1, UserID: 1, StartTime: time.Now()}).Error; err == nil {
		t.Fatal("expected duplicate active session rejection")
	}
	if err := db.DB.Create(&models.StudySession{TaskID: 2, UserID: 1, StartTime: time.Now()}).Error; err == nil {
		t.Fatal("expected another task for the same user to be rejected")
	}
	end := time.Now()
	db.DB.Model(&session).Update("end_time", end)
	if err := db.DB.Create(&models.StudySession{TaskID: 1, UserID: 1, StartTime: time.Now()}).Error; err != nil {
		t.Fatalf("closed sessions must not block a new active session: %v", err)
	}
}

func TestPlanResponsesPreloadScheduleOverrides(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "plan-response", Nickname: "plan-response"}
	db.DB.Create(&user)
	create := callJSONHandler(t, CreatePlan, user.ID, "/plans", "", gin.H{
		"title": "Read", "objective": "finish chapter one", "start_date": "2026-07-20", "end_date": "2026-07-20",
		"study_dates": []string{"2026-07-20"}, "schedule_overrides": []gin.H{{"date": "2026-07-20", "planned_start": "19:00", "planned_end": "20:00"}},
		"task_drafts": []gin.H{{"date": "2026-07-20", "title": "Read", "objective": "finish chapter one", "planned_start": "19:00", "planned_end": "20:00"}},
	})
	var created struct {
		Code int         `json:"code"`
		Data models.Plan `json:"data"`
	}
	if err := json.Unmarshal(create, &created); err != nil {
		t.Fatal(err)
	}
	if created.Code != 0 || len(created.Data.ScheduleOverrides) != 1 {
		t.Fatalf("create response missing overrides: %s", create)
	}

	update := callJSONHandler(t, UpdatePlan, user.ID, "/plans/:id", strconv.FormatUint(uint64(created.Data.ID), 10), gin.H{
		"schedule_overrides": []gin.H{{"date": "2026-07-20", "planned_start": "18:00", "planned_end": "19:00"}},
	})
	var updated struct {
		Code int         `json:"code"`
		Data models.Plan `json:"data"`
	}
	if err := json.Unmarshal(update, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Code != 0 || len(updated.Data.ScheduleOverrides) != 1 || updated.Data.ScheduleOverrides[0].PlannedStart != "18:00" {
		t.Fatalf("update response did not preload latest overrides: %s", update)
	}
}

func TestCheckinRechecksEligibilityAndRetryIsIdempotent(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "checkin", Nickname: "checkin"}
	db.DB.Create(&user)
	plan := models.Plan{UserID: user.ID, Title: "P", Status: models.PlanStatusActive}
	db.DB.Create(&plan)
	task := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-20", Title: "Read", Objective: "finish chapter one", Status: models.TaskStatusPending}
	db.DB.Create(&task)
	body := gin.H{"plan_id": plan.ID, "date": task.Date, "completed": true}
	response := callJSONHandler(t, ToggleCheckin, user.ID, "/checkins", "", body)
	if responseCode(t, response) != http.StatusBadRequest {
		t.Fatalf("ineligible checkin should fail: %s", response)
	}
	db.DB.Model(&task).Update("status", models.TaskStatusCompleted)
	if responseCode(t, callJSONHandler(t, ToggleCheckin, user.ID, "/checkins", "", body)) != 0 {
		t.Fatal("eligible checkin should succeed")
	}
	if responseCode(t, callJSONHandler(t, ToggleCheckin, user.ID, "/checkins", "", body)) != 0 {
		t.Fatal("checkin retry should succeed")
	}
	db.DB.First(&user, user.ID)
	if user.SlackBalance != 10 {
		t.Fatalf("checkin retry awarded twice: balance=%d", user.SlackBalance)
	}
}

func callTaskHandler(t *testing.T, handler gin.HandlerFunc, uid, taskID uint, body interface{}) int {
	t.Helper()
	g := gin.New()
	g.PUT("/tasks/:id", func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, uid) }, handler)
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/tasks/"+strconv.FormatUint(uint64(taskID), 10), reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	g.ServeHTTP(rec, req)
	var response struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return response.Code
}

func callJSONHandler(t *testing.T, handler gin.HandlerFunc, uid uint, route, id string, body interface{}) []byte {
	t.Helper()
	g := gin.New()
	g.POST(route, func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, uid) }, handler)
	g.PUT(route, func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, uid) }, handler)
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	path := strings.Replace(route, ":id", id, 1)
	method := http.MethodPost
	if strings.Contains(route, ":id") {
		method = http.MethodPut
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	g.ServeHTTP(rec, req)
	return rec.Body.Bytes()
}

func responseCode(t *testing.T, body []byte) int {
	t.Helper()
	var response struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("decode response %q: %v", body, err)
	}
	return response.Code
}
