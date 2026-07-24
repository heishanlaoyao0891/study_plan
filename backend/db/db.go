package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"study_plan_backend/config"
	"study_plan_backend/identity"
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
		&models.PlanScheduleOverride{},
		&models.StudyGroup{},
		&models.StudyGroupMember{},
		&models.StudyGroupInvitation{},
		&models.StudyGroupNudge{},
		&models.DailyTask{},
		&models.StudySession{},
		&models.PostponeRecord{},
		&models.DailyMotivation{},
		&models.Checkin{},
		&models.DailyCheckin{},
		&models.PlanActionLayout{},
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
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_schedule_override_weekday ON plan_schedule_overrides (plan_id, weekday) WHERE weekday > 0").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_schedule_override_date ON plan_schedule_overrides (plan_id, date) WHERE date <> ''").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_motivations_user_date ON daily_motivations (user_id, date)").Error; err != nil {
		return err
	}
	if err := DB.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_tasks_user_plan_date ON daily_tasks (user_id, plan_id, date)",
	).Error; err != nil {
		return err
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DROP INDEX IF EXISTS idx_study_sessions_active_task").Error; err != nil {
			return err
		}
		if err := resolveDuplicateOpenSessions(tx); err != nil {
			return err
		}
		return tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_study_sessions_active_user ON study_sessions (user_id) WHERE end_time IS NULL").Error
	}); err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_checkins_user_date ON daily_checkins (user_id, date)").Error; err != nil {
		return err
	}
	if err := auditLegacyNicknames(); err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_nickname_normalized ON users (nickname_normalized) WHERE nickname_normalized <> '' AND deleted_at IS NULL").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_invite_target_id ON users (invite_target_id) WHERE invite_target_id <> ''").Error; err != nil {
		return err
	}
	if err := migrateLegacyCheckins(); err != nil {
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

func resolveDuplicateOpenSessions(tx *gorm.DB) error {
	type duplicate struct{ UserID uint }
	var users []duplicate
	if err := tx.Raw("SELECT user_id FROM study_sessions WHERE end_time IS NULL GROUP BY user_id HAVING COUNT(*) > 1").Scan(&users).Error; err != nil {
		return err
	}
	for _, row := range users {
		var sessions []models.StudySession
		if err := tx.Where("user_id = ? AND end_time IS NULL", row.UserID).Order("start_time DESC, id DESC").Find(&sessions).Error; err != nil {
			return err
		}
		for _, session := range sessions[1:] {
			end := time.Now()
			seconds := int(end.Sub(session.StartTime).Seconds())
			if seconds < 0 {
				seconds = 0
			}
			session.EndTime = &end
			session.DurationSec = seconds
			session.DurationMin = seconds / 60
			session.Note = "migration closed duplicate open session; duration requires decision"
			if err := tx.Save(&session).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.DailyTask{}).Where("id = ?", session.TaskID).Updates(map[string]interface{}{
				"status":         models.TaskStatusPending,
				"needs_decision": true,
				"study_seconds":  gorm.Expr("study_seconds + ?", session.DurationSec),
				"study_minutes":  gorm.Expr("study_minutes + ?", session.DurationMin),
				"actual_end":     &end,
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func auditLegacyNicknames() error {
	var users []models.User
	if err := DB.Unscoped().Order("id ASC").Find(&users).Error; err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, user := range users {
		updates := map[string]interface{}{}
		if user.InviteTargetID == "" {
			targetID, err := identity.NewInviteTargetID()
			if err != nil {
				return err
			}
			updates["invite_target_id"] = targetID
		}
		_, key, err := identity.Validate(user.Nickname)
		if err == nil && !seen[key] && user.DeletedAt.Valid == false {
			updates["nickname_normalized"] = key
			seen[key] = true
		} else if user.NicknameNormalized != "" {
			updates["nickname_normalized"] = ""
		}
		if len(updates) > 0 {
			if err := DB.Unscoped().Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func migrateLegacyCheckins() error {
	type legacy struct {
		UserID   uint
		Date     string
		Rewarded bool
	}
	var rows []legacy
	if err := DB.Model(&models.Checkin{}).Select("user_id, date, MAX(CASE WHEN rewarded THEN 1 ELSE 0 END) AS rewarded").Where("completed = ?", true).Group("user_id, date").Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		checkin := models.DailyCheckin{UserID: row.UserID, Date: row.Date, Completed: true, Rewarded: row.Rewarded, Migrated: true, CreatedAt: time.Now()}
		if err := DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "date"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"completed": true,
				"migrated":  true,
				"rewarded":  gorm.Expr("daily_checkins.rewarded OR excluded.rewarded"),
			}),
		}).Create(&checkin).Error; err != nil {
			return err
		}
	}
	return nil
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
