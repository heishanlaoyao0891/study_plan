package services

import (
	"context"
	"errors"
	"time"

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
	if err := db.DB.WithContext(ctx).Model(&models.AIGenerationUsage{}).Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, start, end).Count(&count).Error; err != nil {
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
	start, end := ShanghaiDayRange(time.Now())
	now := time.Now()
	result := db.DB.WithContext(ctx).Exec(`INSERT INTO ai_generation_usage (user_id, provider, status, message, created_at)
		SELECT ?, ?, 'attempt', 'external provider attempt', ?
		WHERE (SELECT COUNT(*) FROM ai_generation_usage WHERE user_id = ? AND created_at >= ? AND created_at < ?) < ?`, quota.UserID, quota.Provider, now, quota.UserID, start, end, quota.Limit)
	if result.Error != nil {
		return result.Error
	}
	var count int64
	if _, current, err := CanUseAIGenerationContext(ctx, quota.UserID, quota.Limit); err == nil {
		count = current
		if quota.Used != nil {
			*quota.Used = count
		}
	}
	if result.RowsAffected != 1 {
		return ErrAIQuotaExceeded
	}
	return nil
}
