package db

import (
	"fmt"

	"github.com/glebarez/sqlite"
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
		&models.DailyTask{},
		&models.StudySession{},
		&models.PostponeRecord{},
		&models.Checkin{},
		&models.SlackConfig{},
		&models.SlackRecord{},
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
		return DB.Create(&models.SlackConfig{CheckinMinutes: 10}).Error
	}
	return nil
}
