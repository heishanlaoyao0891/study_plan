package services

import (
	"time"

	"study_plan_backend/db"
	"study_plan_backend/models"
)

func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func CanUseAIGeneration(userID uint, limit int) (bool, int64, error) {
	if limit <= 0 {
		limit = 5
	}
	var count int64
	if err := db.DB.Model(&models.AIGenerationUsage{}).Where("user_id = ? AND created_at >= ?", userID, StartOfDay(time.Now())).Count(&count).Error; err != nil {
		return false, 0, err
	}
	return count < int64(limit), count, nil
}

func RecordAIGenerationUsage(userID uint, provider, status, message string) {
	db.DB.Create(&models.AIGenerationUsage{UserID: userID, Provider: provider, Status: status, Message: message})
}
