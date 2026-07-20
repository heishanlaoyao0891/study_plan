package handlers

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"study_plan_backend/config"
	"study_plan_backend/db"
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

func TestAutoCompleteCheckinRewardsOnce(t *testing.T) {
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
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		return autoCompleteCheckinIfPlanDateDone(tx, user.ID, plan.ID, task.Date)
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		return autoCompleteCheckinIfPlanDateDone(tx, user.ID, plan.ID, task.Date)
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.DB.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.SlackBalance != 10 {
		t.Fatalf("expected 10 slack minutes, got %d", user.SlackBalance)
	}
	var checkin models.Checkin
	if err := db.DB.Where("user_id = ? AND plan_id = ? AND date = ?", user.ID, plan.ID, task.Date).First(&checkin).Error; err != nil {
		t.Fatal(err)
	}
	if !checkin.Completed || !checkin.Rewarded {
		t.Fatalf("unexpected checkin state: %+v", checkin)
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
