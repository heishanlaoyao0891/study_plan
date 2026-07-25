package db

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"study_plan_backend/aikey"
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

func TestAutoMigrateNormalizesLegacyAIProviderAliases(t *testing.T) {
	openMigrationTestDB(t)
	config.App = &config.Config{}
	if err := DB.AutoMigrate(&models.AIConfig{}); err != nil {
		t.Fatal(err)
	}
	for _, provider := range []string{"openai", "openai-compatible"} {
		if err := DB.Create(&models.AIConfig{Provider: provider}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := AutoMigrate(); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := DB.Model(&models.AIConfig{}).Where("provider = ?", "openai_compatible").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected both aliases normalized, got %d rows", count)
	}
}

func TestAutoMigrateConvertsLegacyDeepSeekConfigWithoutOverwritingCustomValues(t *testing.T) {
	openMigrationTestDB(t)
	config.App = &config.Config{AIKeySecret: "migration-test-secret"}
	if err := DB.AutoMigrate(&models.AIConfig{}); err != nil {
		t.Fatal(err)
	}
	configs := []models.AIConfig{
		{Provider: "deepseek", BaseURL: "https://api.deepseek.com", ModelName: "deepseek-chat"},
		{Provider: "deepseek", BaseURL: "", ModelName: "deepseek-chat"},
		{Provider: "deepseek", BaseURL: "https://gateway.example/v1", ModelName: "deepseek-chat"},
		{Provider: "deepseek", BaseURL: "https://api.deepseek.com", ModelName: "custom-model"},
		{Provider: "deepseek", BaseURL: "https://gateway.example/v1", ModelName: "custom-model"},
	}
	for i := range configs {
		if err := DB.Create(&configs[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := AutoMigrate(); err != nil {
		t.Fatal(err)
	}
	wants := []struct{ provider, baseURL, modelName string }{
		{"siliconflow", "https://api.siliconflow.cn/v1", "deepseek-ai/DeepSeek-V3.2"},
		{"siliconflow", "https://api.siliconflow.cn/v1", "deepseek-ai/DeepSeek-V3.2"},
		{"openai_compatible", "https://gateway.example/v1", "deepseek-chat"},
		{"siliconflow", "https://api.siliconflow.cn/v1", "custom-model"},
		{"openai_compatible", "https://gateway.example/v1", "custom-model"},
	}
	for i := range configs {
		var migrated models.AIConfig
		if err := DB.First(&migrated, configs[i].ID).Error; err != nil {
			t.Fatal(err)
		}
		if migrated.Provider != wants[i].provider || migrated.BaseURL != wants[i].baseURL || migrated.ModelName != wants[i].modelName {
			t.Fatalf("row %d was corrupted during migration: %+v", i, migrated)
		}
	}
}

func TestAutoMigrateEncryptsLegacyPlaintextAIKeys(t *testing.T) {
	openMigrationTestDB(t)
	config.App = &config.Config{AIKeySecret: "migration-test-secret"}
	if err := DB.AutoMigrate(&models.AIConfig{}); err != nil {
		t.Fatal(err)
	}
	legacy := []models.AIConfig{
		{Provider: "openai_compatible", APIKeyCiphertext: "sk-legacy-one"},
		{Provider: "openai_compatible", APIKeyCiphertext: "sk-legacy-two"},
	}
	for i := range legacy {
		if err := DB.Create(&legacy[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := AutoMigrate(); err != nil {
		t.Fatal(err)
	}
	for i := range legacy {
		var migrated models.AIConfig
		if err := DB.First(&migrated, legacy[i].ID).Error; err != nil {
			t.Fatal(err)
		}
		if !migrated.APIKeyEncrypted || migrated.APIKeyCiphertext == legacy[i].APIKeyCiphertext {
			t.Fatalf("legacy key %d was not encrypted: %+v", i, migrated)
		}
		plaintext, err := aikey.Decrypt(migrated.APIKeyCiphertext, config.App.AIKeySecret)
		if err != nil || plaintext != legacy[i].APIKeyCiphertext {
			t.Fatalf("migrated key %d did not decrypt: value=%q err=%v", i, plaintext, err)
		}
	}
}

func TestAutoMigrateRejectsLegacyPlaintextAIKeyWithoutSecret(t *testing.T) {
	openMigrationTestDB(t)
	config.App = &config.Config{}
	if err := DB.AutoMigrate(&models.AIConfig{}); err != nil {
		t.Fatal(err)
	}
	legacy := models.AIConfig{Provider: "deepseek", BaseURL: "https://api.deepseek.com", APIKeyCiphertext: "sk-legacy-plaintext"}
	if err := DB.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(); err == nil {
		t.Fatal("expected startup migration to fail without encryption secret")
	}
	var unchanged models.AIConfig
	if err := DB.First(&unchanged, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if unchanged.Provider != "deepseek" || unchanged.APIKeyEncrypted || unchanged.APIKeyCiphertext != legacy.APIKeyCiphertext {
		t.Fatalf("failed migration was not rolled back: %+v", unchanged)
	}
}

func TestAutoMigrateAllowsAIConfigWithoutKeyAndSecret(t *testing.T) {
	openMigrationTestDB(t)
	config.App = &config.Config{}
	if err := DB.AutoMigrate(&models.AIConfig{}); err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&models.AIConfig{Provider: "mock"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := AutoMigrate(); err != nil {
		t.Fatalf("config without an API key should migrate without a secret: %v", err)
	}
}
