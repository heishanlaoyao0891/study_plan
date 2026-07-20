package handlers

import (
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

const defaultCheckinRewardMinutes = 10
const defaultMakeupCostRatio = 1.0

func SlackBalance(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query user failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"balance": user.SlackBalance, "unit": "minutes"})
}

type slackStartReq struct {
	Activity string `json:"activity" binding:"required"`
}

func StartSlack(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req slackStartReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "activity required")
		return
	}
	var active models.SlackRecord
	if err := db.DB.Where("user_id = ? AND end_time IS NULL", uid).First(&active).Error; err == nil {
		api.OK(c, active)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		api.Fail(c, http.StatusInternalServerError, "query slack failed: "+err.Error())
		return
	}
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query user failed: "+err.Error())
		return
	}
	if user.SlackBalance <= 0 {
		api.Fail(c, http.StatusBadRequest, "slack balance is empty")
		return
	}
	rec := models.SlackRecord{UserID: uid, StartTime: time.Now(), Activity: req.Activity}
	if err := db.DB.Create(&rec).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "start slack failed: "+err.Error())
		return
	}
	api.OK(c, rec)
}

func StopSlack(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var rec models.SlackRecord
	if err := db.DB.Where("user_id = ? AND end_time IS NULL", uid).First(&rec).Error; err != nil {
		api.Fail(c, http.StatusBadRequest, "no active slack session")
		return
	}
	now := time.Now()
	dur := int(now.Sub(rec.StartTime).Minutes())
	if dur < 1 {
		dur = 1
	}
	rec.EndTime = &now
	rec.DurationMin = dur
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&models.User{}).Where("id = ?", uid).
			Update("slack_balance", gorm.Expr("CASE WHEN slack_balance >= ? THEN slack_balance - ? ELSE 0 END", dur, dur)).Error; e != nil {
			return e
		}
		return tx.Save(&rec).Error
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "stop slack failed: "+err.Error())
		return
	}
	api.OK(c, rec)
}

func SlackRecords(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var records []models.SlackRecord
	if err := db.DB.Where("user_id = ?", uid).Order("id DESC").Limit(50).Find(&records).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query records failed: "+err.Error())
		return
	}
	api.OK(c, records)
}

func slackMakeupCostRatio(uid uint) float64 {
	var cfg models.SlackConfig
	err := db.DB.Where("user_id = ?", uid).First(&cfg).Error
	if err == nil {
		return cfg.MakeupCostRatio
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return defaultMakeupCostRatio
	}
	err = db.DB.Where("user_id IS NULL").First(&cfg).Error
	if err == nil {
		return cfg.MakeupCostRatio
	}
	return defaultMakeupCostRatio
}

func recordSlackDelta(tx *gorm.DB, uid uint, activity string, delta int) error {
	now := time.Now()
	return tx.Create(&models.SlackRecord{UserID: uid, StartTime: now, EndTime: &now, DeltaMin: delta, Activity: activity}).Error
}

func makeupSlackCost(uid uint, minutes int) int {
	if minutes <= 0 {
		return 0
	}
	return int(math.Ceil(float64(minutes) * slackMakeupCostRatio(uid)))
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
	cfg = models.SlackConfig{CheckinMinutes: defaultCheckinRewardMinutes, MakeupCostRatio: defaultMakeupCostRatio}
	return db.DB.Create(&cfg).Error
}

func EnsureSlackConfig(c *gin.Context) {
	if err := ensureDefaultSlackConfig(); err != nil {
		api.Fail(c, http.StatusInternalServerError, "init slack config failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"ok": true})
}
