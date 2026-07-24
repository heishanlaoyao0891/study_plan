package services

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/models"
)

func StartArchiveSync() {
	cfg := config.App
	if !cfg.ArchiveEnabled || strings.TrimSpace(cfg.ArchiveDSN) == "" || strings.ToLower(cfg.ArchiveDriver) != "mysql" {
		return
	}
	interval := time.Duration(cfg.ArchiveIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	go func() {
		archiveOnce()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			archiveOnce()
		}
	}()
}

func archiveOnce() {
	conn, err := gorm.Open(mysql.Open(config.App.ArchiveDSN), &gorm.Config{})
	if err != nil {
		log.Printf("[archive] open mysql failed: %v", err)
		return
	}
	if err := conn.AutoMigrate(
		&models.User{},
		&models.Plan{},
		&models.DailyTask{},
		&models.Checkin{},
		&models.DailyCheckin{},
		&models.StudySession{},
		&models.SlackRecord{},
	); err != nil {
		log.Printf("[archive] migrate mysql failed: %v", err)
		return
	}
	for _, table := range parseArchiveTables(config.App.ArchiveTables) {
		if err := archiveTable(conn, table); err != nil {
			log.Printf("[archive] table=%s failed: %v", table, err)
		}
	}
}

func parseArchiveTables(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if s := strings.TrimSpace(part); s != "" {
			result = append(result, s)
		}
	}
	return result
}

func archiveTable(dst *gorm.DB, table string) error {
	switch table {
	case "users":
		return upsertAll(dst, "users", &[]models.User{})
	case "plans":
		return upsertAll(dst, "plans", &[]models.Plan{})
	case "daily_tasks":
		return upsertAll(dst, "daily_tasks", &[]models.DailyTask{})
	case "checkins":
		return upsertAll(dst, "checkins", &[]models.Checkin{})
	case "daily_checkins":
		return upsertAll(dst, "daily_checkins", &[]models.DailyCheckin{})
	case "study_sessions":
		return upsertAll(dst, "study_sessions", &[]models.StudySession{})
	case "slack_records":
		return upsertAll(dst, "slack_records", &[]models.SlackRecord{})
	default:
		return fmt.Errorf("unsupported archive table %s", table)
	}
}

func upsertAll(dst *gorm.DB, table string, target any) error {
	if err := db.DB.Table(table).Find(target).Error; err != nil {
		return err
	}
	if reflect.ValueOf(target).Elem().Len() == 0 {
		return nil
	}
	return dst.Clauses(clause.OnConflict{UpdateAll: true}).Create(target).Error
}
