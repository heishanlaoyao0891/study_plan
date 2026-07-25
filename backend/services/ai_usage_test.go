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
	if successes != limit {
		t.Fatalf("expected exactly %d reservations, got %d", limit, successes)
	}
	_, count, err := CanUseAIGeneration(9, limit)
	if err != nil || count != limit {
		t.Fatalf("expected %d durable attempts, count=%d err=%v", limit, count, err)
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
