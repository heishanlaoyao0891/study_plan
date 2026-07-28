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

	"study_plan_backend/aikey"
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
		&models.RegistrationInvite{},
		&models.PasswordResetCode{},
		&models.Plan{},
		&models.AIPlanCommit{},
		&models.AIPlanGenerationJob{},
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
		&models.PlanningPreviewVersion{},
		&models.PlanningJob{},
		&models.SubscriptionMessageConfig{},
		&models.NotificationDeliveryLog{},
		&models.NotificationSubscription{},
		&models.AIGenerationUsage{},
		&models.AIPromptPattern{},
		&models.AIInvocationLog{},
		&models.OpsContent{},
		&models.FeedbackReport{},
		&models.AccountEvent{},
	); err != nil {
		return err
	}
	if err := migrateAIConfigs(); err != nil {
		return err
	}
	if err := DB.Model(&models.Plan{}).Where("generation_source = '' OR generation_source IS NULL").Update("generation_source", "manual").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_planning_jobs_active_request ON planning_jobs (user_id, request_fingerprint) WHERE status IN ('queued', 'decomposing', 'scheduling')").Error; err != nil {
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
	if err := DB.Exec("DROP INDEX IF EXISTS idx_daily_tasks_user_plan_date").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_daily_tasks_user_date_schedule ON daily_tasks (user_id, date, planned_start, planned_end)").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_plan_jobs_user_key ON ai_plan_generation_jobs (user_id, idempotency_key) WHERE idempotency_key <> ''").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_plan_jobs_active_user ON ai_plan_generation_jobs (user_id) WHERE status IN ('pending', 'running')").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE INDEX IF NOT EXISTS idx_ai_plan_jobs_claim ON ai_plan_generation_jobs (status, lease_expires_at, created_at)").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_usage_success_reference ON ai_generation_usage (user_id, reference_id) WHERE status = 'success' AND reference_id <> ''").Error; err != nil {
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
	if err := DB.Exec(`CREATE TRIGGER IF NOT EXISTS trg_study_group_members_capacity_insert
		BEFORE INSERT ON study_group_members
		WHEN NEW.status = 'active' AND (SELECT COUNT(*) FROM study_group_members WHERE group_id = NEW.group_id AND status = 'active') >= 10
		BEGIN SELECT RAISE(ABORT, 'study group is full'); END`).Error; err != nil {
		return err
	}
	if err := DB.Exec(`CREATE TRIGGER IF NOT EXISTS trg_study_group_members_capacity_update
		BEFORE UPDATE OF status, group_id ON study_group_members
		WHEN NEW.status = 'active' AND OLD.status <> 'active' AND (SELECT COUNT(*) FROM study_group_members WHERE group_id = NEW.group_id AND status = 'active') >= 10
		BEGIN SELECT RAISE(ABORT, 'study group is full'); END`).Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_checkins_user_date ON daily_checkins (user_id, date)").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_delivery_event_key ON notification_delivery_logs (event_key) WHERE event_key <> ''").Error; err != nil {
		return err
	}
	if err := DB.Exec(`CREATE TRIGGER IF NOT EXISTS trg_feedback_reports_rate_limit
		BEFORE INSERT ON feedback_reports
		WHEN (SELECT COUNT(*) FROM feedback_reports WHERE user_id = NEW.user_id AND created_at >= datetime('now', '-10 minutes')) >= 3
		BEGIN SELECT RAISE(ABORT, 'feedback rate limit exceeded'); END`).Error; err != nil {
		return err
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		if err := resolveDuplicateActiveGroupMemberships(tx); err != nil {
			return err
		}
		return tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_study_group_members_active_user ON study_group_members (user_id) WHERE status = 'active'").Error
	}); err != nil {
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
	if err := DB.Migrator().DropIndex(&models.User{}, "idx_users_open_id"); err != nil && !strings.Contains(strings.ToLower(err.Error()), "no such index") {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_open_id_nonempty ON users (open_id) WHERE open_id IS NOT NULL AND open_id <> '' AND deleted_at IS NULL").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_normalized ON users (username_normalized) WHERE username_normalized IS NOT NULL AND username_normalized <> '' AND deleted_at IS NULL").Error; err != nil {
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

func migrateAIConfigs() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("UPDATE ai_configs SET provider = 'openai_compatible' WHERE lower(trim(provider)) IN ('openai', 'openai-compatible', 'openai_compatible')").Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE ai_configs SET
			provider = 'siliconflow',
			base_url = 'https://api.siliconflow.cn/v1',
			model_name = CASE WHEN trim(model_name) = 'deepseek-chat' THEN 'deepseek-ai/DeepSeek-V3.2' ELSE model_name END
			WHERE lower(trim(provider)) = 'deepseek' AND (trim(base_url) = '' OR trim(base_url) = 'https://api.deepseek.com')`).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE ai_configs SET provider = 'openai_compatible' WHERE lower(trim(provider)) = 'deepseek'").Error; err != nil {
			return err
		}

		var plaintextConfigs []models.AIConfig
		if err := tx.Where("api_key_ciphertext <> '' AND api_key_encrypted = ?", false).Find(&plaintextConfigs).Error; err != nil {
			return err
		}
		secret := ""
		if config.App != nil {
			secret = config.App.AIKeySecret
		}
		for _, cfg := range plaintextConfigs {
			encrypted, err := aikey.Encrypt(cfg.APIKeyCiphertext, secret)
			if err != nil {
				return fmt.Errorf("encrypt legacy AI API key for config %d: %w", cfg.ID, err)
			}
			result := tx.Model(&models.AIConfig{}).Where("id = ? AND api_key_encrypted = ?", cfg.ID, false).Updates(map[string]interface{}{
				"api_key_ciphertext": encrypted,
				"api_key_encrypted":  true,
			})
			if result.Error != nil {
				return fmt.Errorf("update encrypted AI API key for config %d: %w", cfg.ID, result.Error)
			}
			if result.RowsAffected != 1 {
				return fmt.Errorf("update encrypted AI API key for config %d: row changed during migration", cfg.ID)
			}
		}
		return nil
	})
}

func resolveDuplicateActiveGroupMemberships(tx *gorm.DB) error {
	today := time.Now()
	if location, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		today = today.In(location)
	}
	now := time.Now()
	if err := tx.Exec("UPDATE study_groups SET status = 'ended', ended_at = ? WHERE status = 'active' AND end_date <> '' AND end_date < ?", now, today.Format("2006-01-02")).Error; err != nil {
		return err
	}
	if err := tx.Exec(`UPDATE study_group_invitations
		SET revoked_at = ?
		WHERE revoked_at IS NULL AND group_id IN (SELECT id FROM study_groups WHERE status = 'ended')`, now).Error; err != nil {
		return err
	}
	if err := tx.Exec(`UPDATE study_group_members
		SET status = 'left', left_at = ?
		WHERE status = 'active' AND group_id IN (
			SELECT id FROM study_groups WHERE status = 'ended' OR (end_date <> '' AND end_date < ?)
		)`, now, today.Format("2006-01-02")).Error; err != nil {
		return err
	}
	return tx.Exec(`UPDATE study_group_members
		SET status = 'left', left_at = ?
		WHERE status = 'active' AND id NOT IN (
			SELECT MAX(id) FROM study_group_members WHERE status = 'active' GROUP BY user_id
		)`, now).Error
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
		return DB.Create(&models.SubscriptionMessageConfig{}).Error
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
