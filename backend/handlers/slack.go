package handlers

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

const defaultCheckinRewardMinutes = 10
const defaultMakeupCostRatio = 1.0
const slackLowBalanceMinutes = 10

var errSlackAlreadyStopped = errors.New("slack session already stopped")

func SlackBalance(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query user failed: "+err.Error())
		return
	}
	var active models.SlackRecord
	hasActive := db.DB.Where("user_id = ? AND end_time IS NULL", uid).First(&active).Error == nil
	api.OK(c, gin.H{"balance": user.SlackBalance, "unit": "minutes", "can_start": user.SlackBalance > 0 && !hasActive, "blocked_reason": slackBlockedReason(user.SlackBalance, hasActive), "low_balance": user.SlackBalance <= slackLowBalanceMinutes, "active_session": func() interface{} {
		if hasActive {
			return active
		}
		return nil
	}()})
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
	activity := strings.TrimSpace(req.Activity)
	if activity == "" || len([]rune(activity)) > 128 {
		api.Fail(c, http.StatusBadRequest, "activity must contain 1 to 128 characters")
		return
	}
	rec := models.SlackRecord{UserID: uid, StartTime: time.Now(), Activity: activity}
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
	rec.DeltaMin = -dur
	previousBalance := 0
	newBalance := 0
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		claim := tx.Model(&models.SlackRecord{}).Where("id = ? AND end_time IS NULL", rec.ID).Updates(map[string]interface{}{"end_time": now, "duration_min": dur, "delta_min": -dur})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected == 0 {
			return errSlackAlreadyStopped
		}
		var user models.User
		if e := tx.First(&user, uid).Error; e != nil {
			return e
		}
		previousBalance = user.SlackBalance
		newBalance = previousBalance - dur
		if e := tx.Model(&models.User{}).Where("id = ?", uid).Update("slack_balance", newBalance).Error; e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errSlackAlreadyStopped) {
			api.Fail(c, http.StatusConflict, "slack session already stopped")
			return
		}
		api.Fail(c, http.StatusInternalServerError, "stop slack failed: "+err.Error())
		return
	}
	if previousBalance > 0 && newBalance <= slackLowBalanceMinutes {
		go notifyLowSlackBalance(uid, newBalance, now)
	}
	api.OK(c, rec)
}

func slackBlockedReason(balance int, active bool) string {
	if active {
		return "已有进行中的躺平记录"
	}
	if balance < 0 {
		return "躺平币为负，请先通过完成任务并打卡补回"
	}
	if balance == 0 {
		return "躺平币已用完，请先完成任务并打卡"
	}
	return ""
}

func notifyLowSlackBalance(uid uint, balance int, now time.Time) {
	message := fmt.Sprintf("躺平币仅剩 %d 分钟，请及时完成任务并打卡补回", balance)
	if balance < 0 {
		message = fmt.Sprintf("躺平币已透支 %d 分钟，补回前不能继续躺平", -balance)
	}
	key := fmt.Sprintf("slack_balance:user:%d:%s", uid, now.UTC().Format("20060102150405"))
	_, _, _ = services.DeliverNotification(db.DB, key, uid, "slack_balance", services.NotificationValues{Message: message, Title: "躺平币余额"}, services.SendSubscriptionMessage)
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
	result := tx.Model(&models.Checkin{}).Where("id = ? AND completed = ? AND rewarded = ?", checkin.ID, true, false).Update("rewarded", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	if err := tx.Model(&models.User{}).Where("id = ?", uid).Update("slack_balance", gorm.Expr("slack_balance + ?", minutes)).Error; err != nil {
		return err
	}
	checkin.Rewarded = true
	return recordSlackDelta(tx, uid, "完成今日打卡", minutes)
}

func awardDailySlackIfNeeded(tx *gorm.DB, uid uint, checkin *models.DailyCheckin) error {
	if !checkin.Completed || checkin.Rewarded {
		return nil
	}
	minutes := slackRewardMinutes(uid)
	result := tx.Model(&models.DailyCheckin{}).Where("id = ? AND completed = ? AND rewarded = ?", checkin.ID, true, false).Update("rewarded", true)
	if result.Error != nil || result.RowsAffected == 0 {
		return result.Error
	}
	if minutes > 0 {
		if err := tx.Model(&models.User{}).Where("id = ?", uid).Update("slack_balance", gorm.Expr("slack_balance + ?", minutes)).Error; err != nil {
			return err
		}
		if err := recordSlackDelta(tx, uid, "完成今日打卡", minutes); err != nil {
			return err
		}
	}
	checkin.Rewarded = true
	return nil
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
