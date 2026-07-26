package handlers

import (
	"errors"
	"net/http"
	"strconv"
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
	Enabled               *bool  `json:"enabled"`
	APIKey                string `json:"api_key"`
}

type subscriptionConfigReq struct {
	StudyStartTemplateID    string `json:"study_start_template_id"`
	CompletionTemplateID    string `json:"completion_template_id"`
	DecisionTemplateID      string `json:"decision_template_id"`
	MissedCheckinTemplateID string `json:"missed_checkin_template_id"`
	GroupNudgeTemplateID    string `json:"group_nudge_template_id"`
	SlackBalanceTemplateID  string `json:"slack_balance_template_id"`
	StudyStartEnabled       bool   `json:"study_start_enabled"`
	CompletionEnabled       bool   `json:"completion_enabled"`
	DecisionEnabled         bool   `json:"decision_enabled"`
	MissedCheckinEnabled    bool   `json:"missed_checkin_enabled"`
	GroupNudgeEnabled       bool   `json:"group_nudge_enabled"`
	SlackBalanceEnabled     bool   `json:"slack_balance_enabled"`
	StudyStartPage          string `json:"study_start_page"`
	CompletionPage          string `json:"completion_page"`
	DecisionPage            string `json:"decision_page"`
	MissedCheckinPage       string `json:"missed_checkin_page"`
	GroupNudgePage          string `json:"group_nudge_page"`
	SlackBalancePage        string `json:"slack_balance_page"`
	StudyStartFieldMapping  string `json:"study_start_field_mapping"`
	CompletionFieldMapping  string `json:"completion_field_mapping"`
	DecisionFieldMapping    string `json:"decision_field_mapping"`
	MissedCheckinMapping    string `json:"missed_checkin_field_mapping"`
	GroupNudgeFieldMapping  string `json:"group_nudge_field_mapping"`
	SlackBalanceMapping     string `json:"slack_balance_field_mapping"`
}

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
	db.DB.Model(&models.DailyCheckin{}).Where("user_id = ? AND completed = ?", user.ID, true).Count(&checkinCount)
	api.OK(c, gin.H{
		"user":          user,
		"plan_count":    planCount,
		"checkin_count": checkinCount,
		"slack_balance": user.SlackBalance,
	})
}

func AdminOverview(c *gin.Context) {
	now := shanghaiNow()
	today := now.Format(dateLayout)
	start := now.AddDate(0, 0, -29).Format(dateLayout)
	var users []models.User
	if err := db.DB.Where("role = ?", models.RoleUser).Find(&users).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query dashboard users failed")
		return
	}
	segments := map[string]int{"active": 0, "general": 0, "zombie": 0, "banned": 0, "inactive": 0, "deleted": 0}
	newUsers := map[string]int{}
	activeUsers := 0
	for _, user := range users {
		segment := adminUserSegment(user, now)
		segments[segment]++
		if segment == "active" {
			activeUsers++
		}
		createdDate := user.CreatedAt.In(now.Location()).Format(dateLayout)
		if createdDate >= start && createdDate <= today {
			newUsers[createdDate]++
		}
	}
	var plans []models.Plan
	if err := db.DB.Select("status").Find(&plans).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query dashboard plans failed")
		return
	}
	planStatuses := map[string]int{"active": 0, "paused": 0, "archived": 0}
	for _, plan := range plans {
		planStatuses[plan.Status]++
	}
	type learningRow struct {
		Date         string `json:"date"`
		StudyMinutes int    `json:"study_minutes"`
		CheckinUsers int    `json:"checkin_users"`
	}
	learning := make([]learningRow, 0, 30)
	studyByDate := map[string]int{}
	checkinsByDate := map[string]int{}
	var taskRows []struct {
		Date    string
		Minutes int
	}
	if err := db.DB.Model(&models.DailyTask{}).Select("date, COALESCE(SUM(study_minutes),0) AS minutes").Where("date >= ? AND date <= ?", start, today).Group("date").Scan(&taskRows).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query dashboard study failed")
		return
	}
	for _, row := range taskRows {
		studyByDate[row.Date] = row.Minutes
	}
	var checkinRows []struct {
		Date  string
		Count int
	}
	if err := db.DB.Model(&models.DailyCheckin{}).Select("date, COUNT(DISTINCT user_id) AS count").Where("date >= ? AND date <= ? AND completed = ?", start, today, true).Group("date").Scan(&checkinRows).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query dashboard checkins failed")
		return
	}
	for _, row := range checkinRows {
		checkinsByDate[row.Date] = row.Count
	}
	registrations := make([]gin.H, 0, 30)
	for offset := 29; offset >= 0; offset-- {
		date := now.AddDate(0, 0, -offset).Format(dateLayout)
		registrations = append(registrations, gin.H{"date": date, "count": newUsers[date]})
		learning = append(learning, learningRow{Date: date, StudyMinutes: studyByDate[date], CheckinUsers: checkinsByDate[date]})
	}
	segmentRows := []gin.H{}
	for _, item := range []struct{ Key, Label string }{{"active", "7日活跃"}, {"general", "8-30日一般"}, {"zombie", "30日未登录"}, {"banned", "封禁"}, {"inactive", "停用"}, {"deleted", "已删除"}} {
		segmentRows = append(segmentRows, gin.H{"key": item.Key, "label": item.Label, "count": segments[item.Key]})
	}
	planRows := []gin.H{{"key": "active", "label": "进行中", "count": planStatuses[models.PlanStatusActive]}, {"key": "paused", "label": "已暂停", "count": planStatuses[models.PlanStatusPaused]}, {"key": "archived", "label": "已归档", "count": planStatuses[models.PlanStatusArchived]}}
	api.OK(c, gin.H{
		"as_of": now, "range": gin.H{"start": start, "end": today, "days": 30},
		"summary": gin.H{"user_accounts": len(users), "active_users_7d": activeUsers, "active_plans": planStatuses[models.PlanStatusActive], "checkins_today": checkinsByDate[today], "study_minutes_today": studyByDate[today]},
		"charts":  gin.H{"segments": segmentRows, "registrations": registrations, "learning_activity": learning, "plan_statuses": planRows},
	})
}

func adminUserSegment(user models.User, now time.Time) string {
	if user.AccountStatus == models.AccountStatusDeleted {
		return "deleted"
	}
	if user.AccountStatus == models.AccountStatusInactive {
		return "inactive"
	}
	if user.BannedUntil != nil && user.BannedUntil.After(now) {
		return "banned"
	}
	last := user.CreatedAt
	if user.LastLoginAt != nil {
		last = *user.LastLoginAt
	}
	age := now.Sub(last)
	if age <= 7*24*time.Hour {
		return "active"
	}
	if age <= 30*24*time.Hour {
		return "general"
	}
	return "zombie"
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
		t := models.PermanentBanUntil()
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
		req.Provider = services.AIProviderMock
	}
	cfg, err := firstAIConfig()
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query ai config failed: "+err.Error())
		return
	}
	cfg.Provider = req.Provider
	cfg.ModelName = strings.TrimSpace(req.ModelName)
	cfg.BaseURL = strings.TrimSpace(req.BaseURL)
	services.NormalizeAIConfig(&cfg)
	cfg.RequestTimeoutSeconds = req.RequestTimeoutSeconds
	cfg.DailyGenerationLimit = req.DailyGenerationLimit
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	cfg.UpdatedBy = &adminID
	if req.APIKey != "" {
		stored, err := services.ProtectAIAPIKey(req.APIKey)
		if err != nil {
			api.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		cfg.APIKeyCiphertext = stored
		cfg.APIKeyEncrypted = true
	}
	if err := services.ValidateAIConfig(cfg, true); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
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
	persistedBaseURL := cfg.BaseURL
	if req.Provider != "" {
		cfg.Provider = req.Provider
	}
	if req.ModelName != "" {
		cfg.ModelName = req.ModelName
	}
	if req.BaseURL != "" {
		cfg.BaseURL = req.BaseURL
	}
	services.NormalizeAIConfig(&cfg)
	if req.RequestTimeoutSeconds > 0 {
		cfg.RequestTimeoutSeconds = req.RequestTimeoutSeconds
	}
	if req.APIKey != "" {
		cfg.APIKeyCiphertext = req.APIKey
		cfg.APIKeyEncrypted = false
	}
	if req.APIKey == "" && cfg.APIKeyCiphertext != "" && !services.SameAIProviderOrigin(persistedBaseURL, cfg.BaseURL) {
		api.OK(c, gin.H{"ok": false, "message": "a new API key is required when testing a different provider origin"})
		return
	}
	if req.DailyGenerationLimit > 0 {
		cfg.DailyGenerationLimit = req.DailyGenerationLimit
	}
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	if err := services.ProviderTestError(cfg); err != nil {
		api.OK(c, gin.H{"ok": false, "message": err.Error()})
		return
	}
	api.OK(c, gin.H{"ok": true, "message": "provider returned a valid structured plan"})
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
	cfg.GroupNudgeTemplateID = strings.TrimSpace(req.GroupNudgeTemplateID)
	cfg.SlackBalanceTemplateID = strings.TrimSpace(req.SlackBalanceTemplateID)
	cfg.StudyStartEnabled = req.StudyStartEnabled
	cfg.CompletionEnabled = req.CompletionEnabled
	cfg.DecisionEnabled = req.DecisionEnabled
	cfg.MissedCheckinEnabled = req.MissedCheckinEnabled
	cfg.GroupNudgeEnabled = req.GroupNudgeEnabled
	cfg.SlackBalanceEnabled = req.SlackBalanceEnabled
	cfg.StudyStartPage = strings.TrimSpace(req.StudyStartPage)
	cfg.CompletionPage = strings.TrimSpace(req.CompletionPage)
	cfg.DecisionPage = strings.TrimSpace(req.DecisionPage)
	cfg.MissedCheckinPage = strings.TrimSpace(req.MissedCheckinPage)
	cfg.GroupNudgePage = strings.TrimSpace(req.GroupNudgePage)
	cfg.SlackBalancePage = strings.TrimSpace(req.SlackBalancePage)
	cfg.StudyStartFieldMapping = strings.TrimSpace(req.StudyStartFieldMapping)
	cfg.CompletionFieldMapping = strings.TrimSpace(req.CompletionFieldMapping)
	cfg.DecisionFieldMapping = strings.TrimSpace(req.DecisionFieldMapping)
	cfg.MissedCheckinMapping = strings.TrimSpace(req.MissedCheckinMapping)
	cfg.GroupNudgeFieldMapping = strings.TrimSpace(req.GroupNudgeFieldMapping)
	cfg.SlackBalanceMapping = strings.TrimSpace(req.SlackBalanceMapping)
	for _, reminderType := range reminderTypes {
		if err := services.ValidateTemplate(services.TemplateFor(cfg, reminderType)); err != nil {
			api.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
	}
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
		cfg = models.AIConfig{Provider: services.AIProviderMock, RequestTimeoutSeconds: 30, DailyGenerationLimit: 5, Enabled: true}
		err = db.DB.Create(&cfg).Error
	} else if err == nil {
		original := cfg
		services.NormalizeAIConfig(&cfg)
		if cfg.Provider != original.Provider || cfg.BaseURL != original.BaseURL || cfg.ModelName != original.ModelName {
			err = db.DB.Model(&cfg).Updates(map[string]interface{}{"provider": cfg.Provider, "base_url": cfg.BaseURL, "model_name": cfg.ModelName}).Error
		}
	}
	return cfg, err
}

func firstSubscriptionConfig() (models.SubscriptionMessageConfig, error) {
	var cfg models.SubscriptionMessageConfig
	err := db.DB.Order("id ASC").First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = models.SubscriptionMessageConfig{}
		err = db.DB.Create(&cfg).Error
	}
	return cfg, err
}

func aiConfigResp(cfg models.AIConfig) gin.H {
	effectiveMode := "ai"
	if !cfg.Enabled {
		effectiveMode = "disabled"
	} else if cfg.Provider == services.AIProviderMock {
		effectiveMode = "fallback"
	}
	keyStorage := "missing"
	if cfg.APIKeyCiphertext != "" {
		keyStorage = "plaintext"
		if cfg.APIKeyEncrypted {
			keyStorage = "encrypted"
		}
	}
	return gin.H{
		"provider":                cfg.Provider,
		"model_name":              cfg.ModelName,
		"base_url":                cfg.BaseURL,
		"request_timeout_seconds": cfg.RequestTimeoutSeconds,
		"daily_generation_limit":  cfg.DailyGenerationLimit,
		"enabled":                 cfg.Enabled,
		"has_api_key":             cfg.APIKeyCiphertext != "",
		"api_key_masked":          maskSecret(cfg.APIKeyCiphertext),
		"key_storage":             keyStorage,
		"effective_mode":          effectiveMode,
		"fallback_enabled":        true,
	}
}

func subscriptionConfigResp(cfg models.SubscriptionMessageConfig) gin.H {
	var recent []models.NotificationDeliveryLog
	db.DB.Order("id DESC").Limit(20).Find(&recent)
	return gin.H{
		"study_start_template_id":      cfg.StudyStartTemplateID,
		"completion_template_id":       cfg.CompletionTemplateID,
		"decision_template_id":         cfg.DecisionTemplateID,
		"missed_checkin_template_id":   cfg.MissedCheckinTemplateID,
		"group_nudge_template_id":      cfg.GroupNudgeTemplateID,
		"slack_balance_template_id":    cfg.SlackBalanceTemplateID,
		"study_start_enabled":          cfg.StudyStartEnabled,
		"completion_enabled":           cfg.CompletionEnabled,
		"decision_enabled":             cfg.DecisionEnabled,
		"missed_checkin_enabled":       cfg.MissedCheckinEnabled,
		"group_nudge_enabled":          cfg.GroupNudgeEnabled,
		"slack_balance_enabled":        cfg.SlackBalanceEnabled,
		"study_start_page":             cfg.StudyStartPage,
		"completion_page":              cfg.CompletionPage,
		"decision_page":                cfg.DecisionPage,
		"missed_checkin_page":          cfg.MissedCheckinPage,
		"group_nudge_page":             cfg.GroupNudgePage,
		"slack_balance_page":           cfg.SlackBalancePage,
		"study_start_field_mapping":    cfg.StudyStartFieldMapping,
		"completion_field_mapping":     cfg.CompletionFieldMapping,
		"decision_field_mapping":       cfg.DecisionFieldMapping,
		"missed_checkin_field_mapping": cfg.MissedCheckinMapping,
		"group_nudge_field_mapping":    cfg.GroupNudgeFieldMapping,
		"slack_balance_field_mapping":  cfg.SlackBalanceMapping,
		"recent_status":                recent,
	}
}

func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	return "********"
}
