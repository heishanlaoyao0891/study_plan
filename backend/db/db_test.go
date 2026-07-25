package db

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"study_plan_backend/config"
	"study_plan_backend/models"
)

func openMigrationTestDB(t *testing.T) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	connection, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	DB = connection
}

func TestMigrateLegacyCheckinsUpsertsMarkers(t *testing.T) {
	openMigrationTestDB(t)
	if err := DB.AutoMigrate(&models.Checkin{}, &models.DailyCheckin{}); err != nil {
		t.Fatal(err)
	}
	if err := DB.Exec("CREATE UNIQUE INDEX idx_daily_checkins_user_date ON daily_checkins (user_id, date)").Error; err != nil {
		t.Fatal(err)
	}
	DB.Create(&models.Checkin{UserID: 1, PlanID: 1, Date: "2026-07-20", Completed: true, Rewarded: false})
	existing := models.DailyCheckin{UserID: 1, Date: "2026-07-20", Completed: false, Rewarded: false, Migrated: false}
	DB.Create(&existing)
	if err := migrateLegacyCheckins(); err != nil {
		t.Fatal(err)
	}
	DB.First(&existing, existing.ID)
	if !existing.Completed || existing.Rewarded || !existing.Migrated {
		t.Fatalf("unexpected migration markers: %+v", existing)
	}
}

func TestResolveDuplicateOpenSessionsPreservesDurationForDecision(t *testing.T) {
	openMigrationTestDB(t)
	if err := DB.AutoMigrate(&models.DailyTask{}, &models.StudySession{}); err != nil {
		t.Fatal(err)
	}
	task := models.DailyTask{UserID: 1, PlanID: 1, Date: "2026-07-20", Title: "Task"}
	DB.Create(&task)
	DB.Create(&models.StudySession{UserID: 1, TaskID: task.ID, StartTime: time.Now().Add(-10 * time.Minute)})
	DB.Create(&models.StudySession{UserID: 1, TaskID: task.ID + 1, StartTime: time.Now().Add(-5 * time.Minute)})
	if err := DB.Transaction(resolveDuplicateOpenSessions); err != nil {
		t.Fatal(err)
	}
	DB.First(&task, task.ID)
	if !task.NeedsDecision || task.StudySeconds <= 0 || task.StudyMinutes <= 0 {
		t.Fatalf("closed session duration must be retained for decision: %+v", task)
	}
}

func TestAutoMigrateGuardsOneActiveStudyGroupPerUser(t *testing.T) {
	openMigrationTestDB(t)
	config.App = &config.Config{}
	if err := AutoMigrate(); err != nil {
		t.Fatal(err)
	}
	first := models.StudyGroup{Name: "First", LeaderUserID: 1, Status: models.StudyGroupStatusActive}
	second := models.StudyGroup{Name: "Second", LeaderUserID: 2, Status: models.StudyGroupStatusActive}
	if err := DB.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	active := models.StudyGroupMember{GroupID: first.ID, UserID: 1, Role: models.GroupMemberRoleLeader, Status: models.GroupMemberStatusActive, JoinedAt: time.Now()}
	if err := DB.Create(&active).Error; err != nil {
		t.Fatal(err)
	}
	duplicate := models.StudyGroupMember{GroupID: second.ID, UserID: 1, Role: models.GroupMemberRoleMember, Status: models.GroupMemberStatusActive, JoinedAt: time.Now()}
	if err := DB.Create(&duplicate).Error; err == nil {
		t.Fatal("expected active membership unique index to reject a second active group")
	}
	duplicate.Status = models.GroupMemberStatusLeft
	if err := DB.Create(&duplicate).Error; err != nil {
		t.Fatalf("non-active history membership should remain valid: %v", err)
	}
}

func TestAutoMigrateAddsTemplateIDToExistingSubscriptions(t *testing.T) {
	openMigrationTestDB(t)
	config.App = &config.Config{}
	if err := DB.Exec(`CREATE TABLE notification_subscriptions (
		id integer PRIMARY KEY AUTOINCREMENT,
		user_id integer NOT NULL,
		reminder_type text NOT NULL,
		subscribed numeric DEFAULT true,
		created_at datetime,
		updated_at datetime
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Exec("INSERT INTO notification_subscriptions (user_id, reminder_type, subscribed) VALUES (1, 'study_start', 1)").Error; err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(); err != nil {
		t.Fatalf("migrate existing subscriptions: %v", err)
	}
	var templateID string
	if err := DB.Raw("SELECT template_id FROM notification_subscriptions WHERE user_id = 1").Scan(&templateID).Error; err != nil {
		t.Fatal(err)
	}
	if templateID != "" {
		t.Fatalf("legacy authorization must require reauthorization, got %q", templateID)
	}
}
