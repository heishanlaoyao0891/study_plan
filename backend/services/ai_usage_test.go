package services

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"study_plan_backend/db"
	"study_plan_backend/models"
)

func setupAIUsageTestDB(t *testing.T) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "usage.db") + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	connection, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	db.DB = connection
	sqlDB, err := connection.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := connection.AutoMigrate(&models.AIGenerationUsage{}); err != nil {
		t.Fatal(err)
	}
}

func TestAIQuotaReservationIsAtomic(t *testing.T) {
	setupAIUsageTestDB(t)
	const limit = 5
	var wait sync.WaitGroup
	results := make(chan error, 20)
	for index := 0; index < 20; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results <- ReserveAIProviderAttempt(WithAIQuota(context.Background(), 9, "provider", limit, nil))
		}()
	}
	wait.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if err != ErrAIQuotaExceeded {
			t.Fatalf("unexpected reservation error: %v", err)
		}
	}
	if successes != 20 {
		t.Fatalf("expected all provider attempts to be recorded, got %d", successes)
	}
	_, count, err := CanUseAIGeneration(9, limit)
	var attempts int64
	db.DB.Model(&models.AIGenerationUsage{}).Where("status = ?", "attempt").Count(&attempts)
	if err != nil || count != 0 || attempts != 20 {
		t.Fatalf("attempts must not consume successful-generation quota: attempts=%d count=%d err=%v", attempts, count, err)
	}
}

func TestShanghaiDayRangeUsesLocalMidnight(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	start, end := ShanghaiDayRange(time.Date(2026, 8, 1, 16, 30, 0, 0, time.UTC))
	if start.In(location).Format("2006-01-02 15:04") != "2026-08-02 00:00" || end.Sub(start) != 24*time.Hour {
		t.Fatalf("unexpected Shanghai day range: %v - %v", start, end)
	}
}

func TestSuccessfulGenerationQuotaIsIdempotentAndAttemptsAreFree(t *testing.T) {
	setupAIUsageTestDB(t)
	ctx := WithAIQuota(context.Background(), 12, "provider", 2, nil)
	for index := 0; index < 4; index++ {
		if err := ReserveAIProviderAttempt(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := RecordSuccessfulAIGeneration(context.Background(), db.DB, 12, "provider", "job-1", 2); err != nil {
		t.Fatal(err)
	}
	if err := RecordSuccessfulAIGeneration(context.Background(), db.DB, 12, "provider", "job-1", 2); err != nil {
		t.Fatal(err)
	}
	_, count, err := CanUseAIGeneration(12, 2)
	if err != nil || count != 1 {
		t.Fatalf("idempotent publication should charge once: count=%d err=%v", count, err)
	}
}
