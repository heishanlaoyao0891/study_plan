package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

type loginReq struct {
	Code      string `json:"code" binding:"required"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

type loginResp struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

type adminLoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type bindPhoneReq struct {
	Code        string `json:"code"`
	PhoneNumber string `json:"phone_number"`
}

type updateAvatarReq struct {
	AvatarURL string `json:"avatar_url" binding:"required"`
}

type deactivateAccountReq struct {
	Retain bool   `json:"retain"`
	Note   string `json:"note"`
}

type adminLoginFailure struct {
	Count     int
	LockedTil time.Time
}

var adminLoginFailures = struct {
	sync.Mutex
	items map[string]adminLoginFailure
}{items: map[string]adminLoginFailure{}}

const (
	adminLoginMaxFailures = 5
	adminLoginWindow      = 15 * time.Minute
)

// Login 微信登录
func Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	s, err := services.Code2Session(req.Code)
	if err != nil {
		api.Fail(c, http.StatusBadGateway, "wechat login failed: "+err.Error())
		return
	}

	var user models.User
	err = db.DB.Where("open_id = ?", s.OpenID).First(&user).Error

	if err == gorm.ErrRecordNotFound {
		// 新用户：自动注册；首名注册用户设为 admin
		user = models.User{
			OpenID:    s.OpenID,
			Nickname:  req.Nickname,
			AvatarURL: req.AvatarURL,
			Role:      models.RoleUser,
		}
		// 检查是否已有用户
		var count int64
		db.DB.Model(&models.User{}).Count(&count)
		if count == 0 {
			user.Role = models.RoleAdmin
		}
		if err := db.DB.Create(&user).Error; err != nil {
			api.Fail(c, http.StatusInternalServerError, "create user failed: "+err.Error())
			return
		}
	} else if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query user failed: "+err.Error())
		return
	} else {
		// 老用户：检查封禁
		if user.AccountStatus != models.AccountStatusActive {
			db.DB.Model(&user).Updates(map[string]interface{}{"account_status": models.AccountStatusActive})
			user.AccountStatus = models.AccountStatusActive
			db.DB.Create(&models.AccountEvent{UserID: user.ID, EventType: "restore", Detail: "reactivated on login"})
		}
		if user.BannedUntil != nil && user.BannedUntil.After(time.Now()) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":         403,
				"message":      "user banned",
				"banned_until": user.BannedUntil,
				"reason":       user.BannedReason,
			})
			return
		}
		// 封禁时间已过，自动解封
		if user.BannedUntil != nil {
			db.DB.Model(&user).Updates(map[string]interface{}{
				"banned_until":  nil,
				"banned_reason": "",
			})
			user.BannedUntil = nil
			user.BannedReason = ""
		}
		// 客户端可每次刷新昵称/头像
		if req.Nickname != "" || req.AvatarURL != "" {
			updates := map[string]interface{}{}
			if req.Nickname != "" {
				updates["nickname"] = req.Nickname
				user.Nickname = req.Nickname
			}
			if req.AvatarURL != "" {
				updates["avatar_url"] = req.AvatarURL
				user.AvatarURL = req.AvatarURL
			}
			db.DB.Model(&user).Updates(updates)
		}
	}

	token, err := services.SignToken(user.ID, user.OpenID, user.Role)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "sign token failed: "+err.Error())
		return
	}
	api.OK(c, loginResp{Token: token, User: user})
}

func AdminLogin(c *gin.Context) {
	var req adminLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	username := strings.TrimSpace(req.Username)
	key := c.ClientIP() + ":" + strings.ToLower(username)
	if isAdminLoginLocked(key) {
		api.Fail(c, http.StatusTooManyRequests, "too many login attempts, try again later")
		return
	}

	var cred models.AdminCredential
	if err := db.DB.Preload("User").Where("username = ?", username).First(&cred).Error; err != nil {
		recordAdminLoginFailure(key)
		api.Fail(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(req.Password)); err != nil {
		recordAdminLoginFailure(key)
		api.Fail(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if cred.User.Role != models.RoleAdmin {
		recordAdminLoginFailure(key)
		api.Fail(c, http.StatusForbidden, "admin only")
		return
	}
	if cred.User.BannedUntil != nil && cred.User.BannedUntil.After(time.Now()) {
		recordAdminLoginFailure(key)
		api.Fail(c, http.StatusForbidden, "user banned")
		return
	}

	token, err := services.SignToken(cred.User.ID, cred.User.OpenID, cred.User.Role)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "sign token failed: "+err.Error())
		return
	}
	now := time.Now()
	db.DB.Model(&cred).Update("last_login_at", &now)
	clearAdminLoginFailure(key)
	recordAdminAudit(cred.User.ID, nil, "admin_login", "")
	api.OK(c, loginResp{Token: token, User: cred.User})
}

func CurrentUser(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	api.OK(c, user)
}

func BindPhoneNumber(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req bindPhoneReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	phone := strings.TrimSpace(req.PhoneNumber)
	if req.Code != "" {
		wxPhone, err := services.GetPhoneNumber(req.Code)
		if err != nil {
			api.Fail(c, http.StatusBadGateway, "wechat phone binding failed: "+err.Error())
			return
		}
		phone = wxPhone
	} else if !config.App.WeChatLoginMock {
		api.Fail(c, http.StatusBadRequest, "wechat phone binding code is required")
		return
	}
	if phone == "" {
		api.Fail(c, http.StatusBadRequest, "phone binding code is required")
		return
	}
	now := time.Now()
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	if err := db.DB.Model(&user).Updates(map[string]interface{}{"phone_number": phone, "phone_verified_at": &now}).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "bind phone failed: "+err.Error())
		return
	}
	if user.PhoneNumber != "" && user.PhoneNumber != phone {
		db.DB.Create(&models.AccountEvent{UserID: user.ID, EventType: "phone_rebind", Detail: user.PhoneNumber + " -> " + phone})
	}
	user.PhoneNumber = phone
	user.PhoneVerifiedAt = &now
	api.OK(c, user)
}

func DeactivateAccount(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req deactivateAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	if req.Retain {
		if err := db.DB.Model(&user).Updates(map[string]interface{}{"account_status": models.AccountStatusInactive}).Error; err != nil {
			api.Fail(c, http.StatusInternalServerError, "deactivate failed: "+err.Error())
			return
		}
		db.DB.Create(&models.AccountEvent{UserID: user.ID, EventType: "deactivate_retain", Detail: req.Note})
		user.AccountStatus = models.AccountStatusInactive
		api.OK(c, user)
		return
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := cleanupUserData(tx, user.ID); err != nil {
			return err
		}
		return tx.Model(&user).Updates(map[string]interface{}{
			"nickname":          "",
			"avatar_url":        "",
			"phone_number":      "",
			"phone_verified_at": nil,
			"weekly_hours":      0,
			"slack_balance":     0,
			"account_status":    models.AccountStatusDeleted,
		}).Error
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "delete account failed: "+err.Error())
		return
	}
	db.DB.Create(&models.AccountEvent{UserID: user.ID, EventType: "deactivate_delete", Detail: req.Note})
	user.AccountStatus = models.AccountStatusDeleted
	api.OK(c, user)
}

func cleanupUserData(tx *gorm.DB, uid uint) error {
	if err := tx.Where("user_id = ?", uid).Delete(&models.PostponeRecord{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.Checkin{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.StudySession{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.SlackRecord{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.NotificationSubscription{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.NotificationDeliveryLog{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.AIGenerationUsage{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.FeedbackReport{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.PlanMember{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.StudyGroupMember{}).Error; err != nil {
		return err
	}
	if err := tx.Where("created_by = ?", uid).Delete(&models.StudyGroupInvitation{}).Error; err != nil {
		return err
	}
	if err := tx.Where("sender_user_id = ? OR target_user_id = ?", uid, uid).Delete(&models.StudyGroupNudge{}).Error; err != nil {
		return err
	}
	if err := tx.Where("leader_user_id = ?", uid).Delete(&models.StudyGroup{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.AccountEvent{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.DailyTask{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.Plan{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.SlackConfig{}).Error; err != nil {
		return err
	}
	return nil
}

func UpdateAvatar(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req updateAvatarReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	avatarURL := strings.TrimSpace(req.AvatarURL)
	if avatarURL == "" {
		api.Fail(c, http.StatusBadRequest, "avatar_url is required")
		return
	}
	if parsed, err := url.Parse(avatarURL); err != nil || parsed.Scheme == "" {
		api.Fail(c, http.StatusBadRequest, "avatar_url must be a URL or object-storage HTTPS URL")
		return
	}
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	if err := db.DB.Model(&user).Update("avatar_url", avatarURL).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "update avatar failed: "+err.Error())
		return
	}
	user.AvatarURL = avatarURL
	api.OK(c, user)
}

func isAdminLoginLocked(key string) bool {
	adminLoginFailures.Lock()
	defer adminLoginFailures.Unlock()
	failure, ok := adminLoginFailures.items[key]
	if !ok {
		return false
	}
	if !failure.LockedTil.IsZero() && failure.LockedTil.After(time.Now()) {
		return true
	}
	if !failure.LockedTil.IsZero() {
		delete(adminLoginFailures.items, key)
	}
	return false
}

func recordAdminLoginFailure(key string) {
	adminLoginFailures.Lock()
	defer adminLoginFailures.Unlock()
	failure := adminLoginFailures.items[key]
	failure.Count++
	if failure.Count >= adminLoginMaxFailures {
		failure.LockedTil = time.Now().Add(adminLoginWindow)
	}
	adminLoginFailures.items[key] = failure
}

func clearAdminLoginFailure(key string) {
	adminLoginFailures.Lock()
	defer adminLoginFailures.Unlock()
	delete(adminLoginFailures.items, key)
}
