package handlers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

type banReq struct {
	DurationHours int    `json:"duration_hours"` // 0=永久封禁；>0=指定小时数
	Reason        string `json:"reason"`
}

type slackConfigReq struct {
	CheckinMinutes  int     `json:"checkin_minutes"`
	MakeupCostRatio float64 `json:"makeup_cost_ratio"`
	StreakBonus     int     `json:"streak_bonus"`
	QualityBonus    int     `json:"quality_bonus"`
	UserID          *uint   `json:"user_id"`
}

type suspiciousRecordResp struct {
	Tasks    []models.DailyTask    `json:"tasks"`
	Sessions []models.StudySession `json:"sessions"`
}

type aiConfigReq struct {
	Provider              string `json:"provider"`
	ModelName             string `json:"model_name"`
	BaseURL               string `json:"base_url"`
	RequestTimeoutSeconds int    `json:"request_timeout_seconds"`
	DailyGenerationLimit  int    `json:"daily_generation_limit"`
	Enabled               bool   `json:"enabled"`
	APIKey                string `json:"api_key"`
}

type subscriptionConfigReq struct {
	StudyStartTemplateID    string `json:"study_start_template_id"`
	CompletionTemplateID    string `json:"completion_template_id"`
	DecisionTemplateID      string `json:"decision_template_id"`
	MissedCheckinTemplateID string `json:"missed_checkin_template_id"`
	StudyStartEnabled       bool   `json:"study_start_enabled"`
	CompletionEnabled       bool   `json:"completion_enabled"`
	DecisionEnabled         bool   `json:"decision_enabled"`
	MissedCheckinEnabled    bool   `json:"missed_checkin_enabled"`
}

const farFutureYear = 2099

// ListUsers 管理员：列出所有用户
func ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))
	q := db.DB.Model(&models.User{})
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("nickname LIKE ? OR open_id LIKE ?", like, like)
	}
	if status == "banned" {
		q = q.Where("banned_until IS NOT NULL AND banned_until > ?", time.Now())
	} else if status == "active" {
		q = q.Where("banned_until IS NULL OR banned_until <= ?", time.Now())
	}

	var total int64
	var users []models.User
	q.Count(&total)
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&users).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query users failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{
		"total": total,
		"page":  page,
		"size":  size,
		"users": users,
	})
}

func GetAdminUserDetail(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	var user models.User
	if err := db.DB.First(&user, targetID).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	var planCount int64
	var checkinCount int64
	db.DB.Model(&models.Plan{}).Where("user_id = ?", user.ID).Count(&planCount)
	db.DB.Model(&models.Checkin{}).Where("user_id = ?", user.ID).Count(&checkinCount)
	api.OK(c, gin.H{
		"user":          user,
		"plan_count":    planCount,
		"checkin_count": checkinCount,
		"slack_balance": user.SlackBalance,
	})
}

func AdminOverview(c *gin.Context) {
	var users int64
	var activePlans int64
	var checkinsToday int64
	var bannedUsers int64
	today := time.Now().Format(dateLayout)
	db.DB.Model(&models.User{}).Count(&users)
	db.DB.Model(&models.Plan{}).Where("status = ?", models.PlanStatusActive).Count(&activePlans)
	db.DB.Model(&models.Checkin{}).Where("date = ?", today).Count(&checkinsToday)
	db.DB.Model(&models.User{}).Where("banned_until IS NOT NULL AND banned_until > ?", time.Now()).Count(&bannedUsers)
	api.OK(c, gin.H{"users": users, "active_plans": activePlans, "checkins_today": checkinsToday, "banned_users": bannedUsers})
}

// BanUser 管理员：封禁用户
func BanUser(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	var req banReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if uint(targetID) == uid {
		api.Fail(c, http.StatusBadRequest, "cannot ban yourself")
		return
	}

	var user models.User
	if err := db.DB.First(&user, targetID).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	if user.Role == models.RoleAdmin {
		api.Fail(c, http.StatusBadRequest, "cannot ban admin user")
		return
	}

	var until *time.Time
	if req.DurationHours == 0 {
		// 永久封禁：用一个远未来的时间戳
		t := time.Date(farFutureYear, 12, 31, 23, 59, 59, 0, time.UTC)
		until = &t
	} else {
		t := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)
		until = &t
	}
	if err := db.DB.Model(&user).Updates(map[string]interface{}{
		"banned_until":  until,
		"banned_reason": req.Reason,
	}).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "ban user failed: "+err.Error())
		return
	}
	user.BannedUntil = until
	user.BannedReason = req.Reason
	recordAdminAudit(uid, &user.ID, "ban_user", req.Reason)
	api.OK(c, user)
}

// UnbanUser 管理员：解封用户
func UnbanUser(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	var user models.User
	if err := db.DB.First(&user, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.Fail(c, http.StatusNotFound, "user not found")
			return
		}
		api.Fail(c, http.StatusInternalServerError, "query user failed: "+err.Error())
		return
	}
	if err := db.DB.Model(&user).Updates(map[string]interface{}{
		"banned_until":  nil,
		"banned_reason": "",
	}).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "unban user failed: "+err.Error())
		return
	}
	user.BannedUntil = nil
	user.BannedReason = ""
	adminID := c.GetUint(middleware.CtxUserIDKey)
	recordAdminAudit(adminID, &user.ID, "unban_user", "")
	api.OK(c, user)
}

func GetSlackConfigs(c *gin.Context) {
	var configs []models.SlackConfig
	if err := db.DB.Order("user_id ASC, id ASC").Find(&configs).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query configs failed: "+err.Error())
		return
	}
	api.OK(c, configs)
}

func UpsertGlobalSlackConfig(c *gin.Context) {
	upsertSlackConfig(c, nil)
}

func UpsertUserSlackConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	uid := uint(id)
	upsertSlackConfig(c, &uid)
}

func upsertSlackConfig(c *gin.Context, targetUserID *uint) {
	adminID := c.GetUint(middleware.CtxUserIDKey)
	var req slackConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.CheckinMinutes < 0 || req.StreakBonus < 0 || req.QualityBonus < 0 {
		api.Fail(c, http.StatusBadRequest, "config values must be non-negative")
		return
	}
	if req.MakeupCostRatio < 0 {
		api.Fail(c, http.StatusBadRequest, "makeup_cost_ratio must be non-negative")
		return
	}
	var cfg models.SlackConfig
	q := db.DB.Where("user_id IS NULL")
	if targetUserID != nil {
		q = db.DB.Where("user_id = ?", *targetUserID)
	}
	err := q.First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = models.SlackConfig{UserID: targetUserID}
	} else if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query config failed: "+err.Error())
		return
	}
	cfg.CheckinMinutes = req.CheckinMinutes
	cfg.MakeupCostRatio = req.MakeupCostRatio
	cfg.StreakBonus = req.StreakBonus
	cfg.QualityBonus = req.QualityBonus
	cfg.UpdatedBy = &adminID
	if err := db.DB.Save(&cfg).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "save config failed: "+err.Error())
		return
	}
	reason := "global slack config"
	if targetUserID != nil {
		reason = "user slack config"
	}
	recordAdminAudit(adminID, targetUserID, "update_slack_config", reason)
	api.OK(c, cfg)
}

func GetSuspiciousRecords(c *gin.Context) {
	var tasks []models.DailyTask
	var sessions []models.StudySession
	if err := db.DB.Where("suspicious = ?", true).Order("updated_at DESC").Limit(100).Find(&tasks).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query suspicious tasks failed: "+err.Error())
		return
	}
	if err := db.DB.Where("suspicious = ?", true).Order("updated_at DESC").Limit(100).Find(&sessions).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query suspicious sessions failed: "+err.Error())
		return
	}
	api.OK(c, suspiciousRecordResp{Tasks: tasks, Sessions: sessions})
}

func GetAIConfig(c *gin.Context) {
	cfg, err := firstAIConfig()
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query ai config failed: "+err.Error())
		return
	}
	api.OK(c, aiConfigResp(cfg))
}

func UpdateAIConfig(c *gin.Context) {
	adminID := c.GetUint(middleware.CtxUserIDKey)
	var req aiConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.Provider == "" {
		req.Provider = "mock"
	}
	if req.RequestTimeoutSeconds <= 0 {
		req.RequestTimeoutSeconds = 30
	}
	if req.DailyGenerationLimit < 0 {
		api.Fail(c, http.StatusBadRequest, "daily_generation_limit must be non-negative")
		return
	}
	cfg, err := firstAIConfig()
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query ai config failed: "+err.Error())
		return
	}
	cfg.Provider = strings.TrimSpace(req.Provider)
	cfg.ModelName = strings.TrimSpace(req.ModelName)
	cfg.BaseURL = strings.TrimSpace(req.BaseURL)
	cfg.RequestTimeoutSeconds = req.RequestTimeoutSeconds
	cfg.DailyGenerationLimit = req.DailyGenerationLimit
	cfg.Enabled = req.Enabled
	cfg.UpdatedBy = &adminID
	if req.APIKey != "" {
		stored, encrypted, err := protectAIAPIKey(req.APIKey)
		if err != nil {
			api.Fail(c, http.StatusInternalServerError, "encrypt api key failed: "+err.Error())
			return
		}
		cfg.APIKeyCiphertext = stored
		cfg.APIKeyEncrypted = encrypted
	}
	if err := db.DB.Save(&cfg).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "save ai config failed: "+err.Error())
		return
	}
	recordAdminAudit(adminID, nil, "update_ai_config", "")
	api.OK(c, aiConfigResp(cfg))
}

func TestAIProvider(c *gin.Context) {
	var req aiConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	cfg, err := firstAIConfig()
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query ai config failed: "+err.Error())
		return
	}
	if req.Provider != "" {
		cfg.Provider = req.Provider
	}
	if req.ModelName != "" {
		cfg.ModelName = req.ModelName
	}
	if req.BaseURL != "" {
		cfg.BaseURL = req.BaseURL
	}
	if req.RequestTimeoutSeconds > 0 {
		cfg.RequestTimeoutSeconds = req.RequestTimeoutSeconds
	}
	if req.APIKey != "" {
		cfg.APIKeyCiphertext = req.APIKey
		cfg.APIKeyEncrypted = false
	}
	if err := services.ProviderTestError(cfg); err != nil {
		api.OK(c, gin.H{"ok": false, "message": err.Error()})
		return
	}
	api.OK(c, gin.H{"ok": true, "message": "provider test passed"})
}

func GetSubscriptionMessageConfig(c *gin.Context) {
	cfg, err := firstSubscriptionConfig()
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query subscription config failed: "+err.Error())
		return
	}
	api.OK(c, subscriptionConfigResp(cfg))
}

func UpdateSubscriptionMessageConfig(c *gin.Context) {
	adminID := c.GetUint(middleware.CtxUserIDKey)
	var req subscriptionConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	cfg, err := firstSubscriptionConfig()
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query subscription config failed: "+err.Error())
		return
	}
	cfg.StudyStartTemplateID = strings.TrimSpace(req.StudyStartTemplateID)
	cfg.CompletionTemplateID = strings.TrimSpace(req.CompletionTemplateID)
	cfg.DecisionTemplateID = strings.TrimSpace(req.DecisionTemplateID)
	cfg.MissedCheckinTemplateID = strings.TrimSpace(req.MissedCheckinTemplateID)
	cfg.StudyStartEnabled = req.StudyStartEnabled
	cfg.CompletionEnabled = req.CompletionEnabled
	cfg.DecisionEnabled = req.DecisionEnabled
	cfg.MissedCheckinEnabled = req.MissedCheckinEnabled
	cfg.UpdatedBy = &adminID
	if err := db.DB.Save(&cfg).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "save subscription config failed: "+err.Error())
		return
	}
	recordAdminAudit(adminID, nil, "update_subscription_config", "")
	api.OK(c, subscriptionConfigResp(cfg))
}

func ListAuditLogs(c *gin.Context) {
	var logs []models.AdminAuditLog
	if err := db.DB.Order("id DESC").Limit(100).Find(&logs).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query audit logs failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"logs": logs})
}

func recordAdminAudit(adminID uint, targetUserID *uint, actionType, reason string) {
	if adminID == 0 || actionType == "" {
		return
	}
	db.DB.Create(&models.AdminAuditLog{AdminUserID: adminID, TargetUserID: targetUserID, ActionType: actionType, Reason: reason})
}

func firstAIConfig() (models.AIConfig, error) {
	var cfg models.AIConfig
	err := db.DB.Order("id ASC").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = models.AIConfig{Provider: "mock", RequestTimeoutSeconds: 30, DailyGenerationLimit: 5, Enabled: true}
		err = db.DB.Create(&cfg).Error
	}
	return cfg, err
}

func firstSubscriptionConfig() (models.SubscriptionMessageConfig, error) {
	var cfg models.SubscriptionMessageConfig
	err := db.DB.Order("id ASC").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = models.SubscriptionMessageConfig{StudyStartEnabled: true, CompletionEnabled: true, DecisionEnabled: true, MissedCheckinEnabled: true}
		err = db.DB.Create(&cfg).Error
	}
	return cfg, err
}

func aiConfigResp(cfg models.AIConfig) gin.H {
	return gin.H{
		"provider":                cfg.Provider,
		"model_name":              cfg.ModelName,
		"base_url":                cfg.BaseURL,
		"request_timeout_seconds": cfg.RequestTimeoutSeconds,
		"daily_generation_limit":  cfg.DailyGenerationLimit,
		"enabled":                 cfg.Enabled,
		"has_api_key":             cfg.APIKeyCiphertext != "",
		"api_key_masked":          maskSecret(cfg.APIKeyCiphertext),
	}
}

func subscriptionConfigResp(cfg models.SubscriptionMessageConfig) gin.H {
	var recent []models.NotificationDeliveryLog
	db.DB.Order("id DESC").Limit(20).Find(&recent)
	return gin.H{
		"study_start_template_id":    cfg.StudyStartTemplateID,
		"completion_template_id":     cfg.CompletionTemplateID,
		"decision_template_id":       cfg.DecisionTemplateID,
		"missed_checkin_template_id": cfg.MissedCheckinTemplateID,
		"study_start_enabled":        cfg.StudyStartEnabled,
		"completion_enabled":         cfg.CompletionEnabled,
		"decision_enabled":           cfg.DecisionEnabled,
		"missed_checkin_enabled":     cfg.MissedCheckinEnabled,
		"recent_status":              recent,
	}
}

func protectAIAPIKey(secret string) (string, bool, error) {
	if config.App.AIKeySecret == "" {
		return secret, false, nil
	}
	key := sha256.Sum256([]byte(config.App.AIKeySecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", false, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", false, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", false, err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(secret), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), true, nil
}

func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	return "********"
}
