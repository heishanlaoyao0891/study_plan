package services

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"study_plan_backend/db"
	"study_plan_backend/models"
)

var ErrAIQuotaExceeded = errors.New("daily enrichment limit reached")

type aiQuotaContext struct {
	UserID   uint
	Provider string
	Limit    int
	Used     *int64
}

func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func ShanghaiDayRange(t time.Time) (time.Time, time.Time) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		t = t.In(location)
	}
	start := StartOfDay(t)
	return start, start.AddDate(0, 0, 1)
}

func CanUseAIGeneration(userID uint, limit int) (bool, int64, error) {
	return CanUseAIGenerationContext(context.Background(), userID, limit)
}

func CanUseAIGenerationContext(ctx context.Context, userID uint, limit int) (bool, int64, error) {
	if limit <= 0 {
		limit = 5
	}
	var count int64
	start, end := ShanghaiDayRange(time.Now())
	if err := db.DB.WithContext(ctx).Model(&models.AIGenerationUsage{}).Where("user_id = ? AND status = ? AND created_at >= ? AND created_at < ?", userID, "success", start, end).Count(&count).Error; err != nil {
		return false, 0, err
	}
	return count < int64(limit), count, nil
}

func WithAIQuota(ctx context.Context, userID uint, provider string, limit int, used *int64) context.Context {
	return context.WithValue(ctx, aiQuotaContextKey{}, aiQuotaContext{UserID: userID, Provider: provider, Limit: maxInt(limit, 5), Used: used})
}

type aiQuotaContextKey struct{}

func ReserveAIProviderAttempt(ctx context.Context) error {
	quota, ok := ctx.Value(aiQuotaContextKey{}).(aiQuotaContext)
	if !ok {
		return nil
	}
	now := time.Now()
	result := db.DB.WithContext(ctx).Create(&models.AIGenerationUsage{UserID: quota.UserID, Provider: quota.Provider, Status: "attempt", Message: "external provider attempt", CreatedAt: now})
	if result.Error != nil {
		return result.Error
	}
	if _, current, err := CanUseAIGenerationContext(ctx, quota.UserID, quota.Limit); err == nil {
		if quota.Used != nil {
			*quota.Used = current
		}
	}
	return nil
}

func RecordSuccessfulAIGeneration(ctx context.Context, database *gorm.DB, userID uint, provider, referenceID string, limit int) error {
	if limit <= 0 {
		limit = 5
	}
	start, end := ShanghaiDayRange(time.Now())
	usage := models.AIGenerationUsage{UserID: userID, Provider: provider, Status: "success", ReferenceID: referenceID, Message: "valid ai_decomposed preview published", CreatedAt: time.Now()}
	result := database.WithContext(ctx).Exec(`INSERT INTO ai_generation_usage (user_id, provider, status, reference_id, message, created_at)
		SELECT ?, ?, 'success', ?, ?, ?
		WHERE NOT EXISTS (SELECT 1 FROM ai_generation_usage WHERE user_id = ? AND status = 'success' AND reference_id = ?)
		AND (SELECT COUNT(*) FROM ai_generation_usage WHERE user_id = ? AND status = 'success' AND created_at >= ? AND created_at < ?) < ?`,
		usage.UserID, usage.Provider, usage.ReferenceID, usage.Message, usage.CreatedAt,
		usage.UserID, usage.ReferenceID, usage.UserID, start, end, limit)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var existing int64
	if err := database.WithContext(ctx).Model(&models.AIGenerationUsage{}).Where("user_id = ? AND status = 'success' AND reference_id = ?", userID, referenceID).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}
	return ErrAIQuotaExceeded
}
