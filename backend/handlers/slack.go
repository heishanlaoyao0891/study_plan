package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

const defaultCheckinRewardMinutes = 10

func SlackBalance(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query user failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"balance": user.SlackBalance, "unit": "minutes"})
}

func slackRewardMinutes(uid uint) int {
	var cfg models.SlackConfig
	err := db.DB.Where("user_id = ?", uid).First(&cfg).Error
	if err == nil && cfg.CheckinMinutes > 0 {
		return cfg.CheckinMinutes
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return defaultCheckinRewardMinutes
	}
	err = db.DB.Where("user_id IS NULL").First(&cfg).Error
	if err == nil && cfg.CheckinMinutes > 0 {
		return cfg.CheckinMinutes
	}
	return defaultCheckinRewardMinutes
}

func awardSlackIfNeeded(tx *gorm.DB, uid uint, checkin *models.Checkin) error {
	if !checkin.Completed || checkin.Rewarded {
		return nil
	}
	minutes := slackRewardMinutes(uid)
	if minutes <= 0 {
		return nil
	}
	if err := tx.Model(&models.User{}).Where("id = ?", uid).
		Update("slack_balance", gorm.Expr("slack_balance + ?", minutes)).Error; err != nil {
		return err
	}
	checkin.Rewarded = true
	return tx.Model(checkin).Update("rewarded", true).Error
}

func ensureDefaultSlackConfig() error {
	var cfg models.SlackConfig
	err := db.DB.Where("user_id IS NULL").First(&cfg).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	cfg = models.SlackConfig{CheckinMinutes: defaultCheckinRewardMinutes}
	return db.DB.Create(&cfg).Error
}

func EnsureSlackConfig(c *gin.Context) {
	if err := ensureDefaultSlackConfig(); err != nil {
		api.Fail(c, http.StatusInternalServerError, "init slack config failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"ok": true})
}
