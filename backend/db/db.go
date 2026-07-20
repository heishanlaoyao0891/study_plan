package db

import (
	"fmt"
	"strings"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"study_plan_backend/config"
	"study_plan_backend/models"
)

var DB *gorm.DB

func Init() error {
	cfg := config.App
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", cfg.DBPath)
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	DB = gdb
	return AutoMigrate()
}

func AutoMigrate() error {
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Plan{},
		&models.PlanMember{},
		&models.StudyGroup{},
		&models.StudyGroupMember{},
		&models.StudyGroupInvitation{},
		&models.StudyGroupNudge{},
		&models.DailyTask{},
		&models.StudySession{},
		&models.PostponeRecord{},
		&models.Checkin{},
		&models.SlackConfig{},
		&models.SlackRecord{},
		&models.AdminCredential{},
		&models.AdminAuditLog{},
		&models.AIConfig{},
		&models.SubscriptionMessageConfig{},
		&models.NotificationDeliveryLog{},
		&models.NotificationSubscription{},
		&models.AIGenerationUsage{},
		&models.OpsContent{},
		&models.FeedbackReport{},
		&models.AccountEvent{},
	); err != nil {
		return err
	}
	// 复合唯一索引：同一用户/同一天/同一计划 只能有一条打卡记录
	idx := &models.Checkin{}
	if err := DB.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_checkins_user_plan_date ON " + idx.TableName() + " (user_id, plan_id, date)",
	).Error; err != nil {
		return err
	}
	if err := DB.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_tasks_user_plan_date ON daily_tasks (user_id, plan_id, date)",
	).Error; err != nil {
		return err
	}

	var count int64
	if err := DB.Model(&models.SlackConfig{}).Where("user_id IS NULL").Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := DB.Create(&models.SlackConfig{CheckinMinutes: 10, MakeupCostRatio: 1}).Error; err != nil {
			return err
		}
	}
	if err := ensureDefaultAdminConfigs(); err != nil {
		return err
	}
	return bootstrapAdminCredential()
}

func ensureDefaultAdminConfigs() error {
	var count int64
	if err := DB.Model(&models.AIConfig{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		if err := DB.Create(&models.AIConfig{Provider: "mock", RequestTimeoutSeconds: 30, DailyGenerationLimit: 5, Enabled: true}).Error; err != nil {
			return err
		}
	}
	if err := DB.Model(&models.SubscriptionMessageConfig{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return DB.Create(&models.SubscriptionMessageConfig{StudyStartEnabled: true, CompletionEnabled: true, DecisionEnabled: true, MissedCheckinEnabled: true}).Error
	}
	return nil
}

func bootstrapAdminCredential() error {
	username := strings.TrimSpace(config.App.AdminUsername)
	password := config.App.AdminPassword
	if username == "" || password == "" {
		return nil
	}

	var existing models.AdminCredential
	if err := DB.Where("username = ?", username).First(&existing).Error; err == nil {
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	openid := "admin:" + username
	var user models.User
	if err := DB.Where("open_id = ?", openid).First(&user).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		user = models.User{OpenID: openid, Nickname: username, Role: models.RoleAdmin}
		if err := DB.Create(&user).Error; err != nil {
			return err
		}
	} else if user.Role != models.RoleAdmin {
		user.Role = models.RoleAdmin
		if err := DB.Save(&user).Error; err != nil {
			return err
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return DB.Create(&models.AdminCredential{
		UserID:       user.ID,
		Username:     username,
		PasswordHash: string(hash),
	}).Error
}
