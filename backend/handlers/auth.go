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
	"gorm.io/gorm/clause"

	"study_plan_backend/api"
	"study_plan_backend/banstate"
	"study_plan_backend/db"
	"study_plan_backend/identity"
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
	Token            string      `json:"token"`
	User             models.User `json:"user"`
	NicknameRequired bool        `json:"nickname_required"`
}

type registrationRequiredResp struct {
	RegistrationRequired bool   `json:"registration_required"`
	RegistrationToken    string `json:"registration_token"`
}

type updateNicknameReq struct {
	Nickname string `json:"nickname" binding:"required"`
}

var userSearchLimits = struct {
	sync.Mutex
	Items map[uint][]time.Time
}{Items: map[uint][]time.Time{}}

type adminLoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
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
		registrationToken, signErr := services.SignRegistrationToken(s.OpenID)
		if signErr != nil {
			api.Fail(c, http.StatusInternalServerError, "sign registration token failed: "+signErr.Error())
			return
		}
		api.OK(c, registrationRequiredResp{RegistrationRequired: true, RegistrationToken: registrationToken})
		return
	} else if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query user failed: "+err.Error())
		return
	} else {
		if !allowActiveUser(c, &user) {
			return
		}
		// WeChat login does not authoritatively provide the application nickname.
		if req.AvatarURL != "" {
			updates := map[string]interface{}{}
			if req.AvatarURL != "" {
				updates["avatar_url"] = req.AvatarURL
				user.AvatarURL = req.AvatarURL
			}
			db.DB.Model(&user).Updates(updates)
		}
	}
	if user.UsernameNormalized == "" || user.NicknameNormalized == "" || user.PasswordHash == nil {
		registrationToken, signErr := services.SignRegistrationToken(s.OpenID)
		if signErr != nil {
			api.Fail(c, http.StatusInternalServerError, "sign registration token failed: "+signErr.Error())
			return
		}
		api.OK(c, registrationRequiredResp{RegistrationRequired: true, RegistrationToken: registrationToken})
		return
	}

	token, err := services.SignToken(user.ID, s.OpenID, user.Role, user.SecurityVersion)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "sign token failed: "+err.Error())
		return
	}
	recordUserLogin(&user, "wechat")
	api.OK(c, loginResp{Token: token, User: user, NicknameRequired: user.NicknameNormalized == ""})
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
	if banstate.Block(c, &cred.User, time.Now()) {
		recordAdminLoginFailure(key)
		return
	}

	if cred.User.AccountStatus != models.AccountStatusActive {
		api.Fail(c, http.StatusForbidden, "account inactive")
		return
	}
	token, err := services.SignToken(cred.User.ID, cred.User.OpenID, cred.User.Role, cred.User.SecurityVersion)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "sign token failed: "+err.Error())
		return
	}
	now := time.Now()
	db.DB.Model(&cred).Update("last_login_at", &now)
	clearAdminLoginFailure(key)
	recordAdminAudit(cred.User.ID, nil, "admin_login", "")
	api.OK(c, loginResp{Token: token, User: cred.User, NicknameRequired: false})
}

func UpdateNickname(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req updateNicknameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	display, key, err := identity.Validate(req.Nickname)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	if err := db.DB.Model(&user).Updates(map[string]interface{}{"nickname": display, "nickname_normalized": key}).Error; err != nil {
		if isUniqueConstraintError(err) {
			api.Conflict(c, "nickname is already in use", gin.H{"nickname_conflict": true})
			return
		}
		api.Fail(c, http.StatusInternalServerError, "update nickname failed: "+err.Error())
		return
	}
	user.Nickname, user.NicknameNormalized = display, key
	api.OK(c, user)
}

func SearchUsers(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	query := identity.Normalize(c.Query("q"))
	if len([]rune(query)) < 2 {
		api.Fail(c, http.StatusBadRequest, "search query must contain at least 2 characters")
		return
	}
	if !allowUserSearch(uid, time.Now()) {
		api.Fail(c, http.StatusTooManyRequests, "too many searches, try again later")
		return
	}
	escaped := escapeLikePattern(query)
	contains, prefix := "%"+escaped+"%", escaped+"%"
	var users []models.User
	if err := db.DB.Where("id <> ? AND account_status = ? AND nickname_normalized <> '' AND (banned_until IS NULL OR banned_until <= ?) AND nickname_normalized LIKE ? ESCAPE '\\'", uid, models.AccountStatusActive, time.Now(), contains).
		Order(clause.Expr{SQL: "CASE WHEN nickname_normalized = ? THEN 0 WHEN nickname_normalized LIKE ? ESCAPE '\\' THEN 1 ELSE 2 END, nickname_normalized ASC, id ASC", Vars: []interface{}{query, prefix}}).
		Limit(10).Find(&users).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "search users failed: "+err.Error())
		return
	}
	rows := make([]gin.H, 0, len(users))
	for _, user := range users {
		rows = append(rows, gin.H{"invite_target_id": user.InviteTargetID, "nickname": user.Nickname, "avatar_url": user.AvatarURL})
	}
	api.OK(c, rows)
}

func escapeLikePattern(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "%", "\\%")
	return strings.ReplaceAll(value, "_", "\\_")
}

func allowUserSearch(uid uint, now time.Time) bool {
	userSearchLimits.Lock()
	defer userSearchLimits.Unlock()
	cutoff := now.Add(-time.Minute)
	recent := userSearchLimits.Items[uid][:0]
	for _, value := range userSearchLimits.Items[uid] {
		if value.After(cutoff) {
			recent = append(recent, value)
		}
	}
	if len(recent) >= 20 {
		userSearchLimits.Items[uid] = recent
		return false
	}
	userSearchLimits.Items[uid] = append(recent, now)
	return true
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
		if err := db.DB.Model(&user).Updates(map[string]interface{}{"account_status": models.AccountStatusInactive, "security_version": gorm.Expr("security_version + 1")}).Error; err != nil {
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
			"open_id":             "",
			"username":            "",
			"username_normalized": "",
			"password_hash":       nil,
			"nickname":            "",
			"nickname_normalized": "",
			"avatar_url":          "",
			"weekly_hours":        0,
			"slack_balance":       0,
			"account_status":      models.AccountStatusDeleted,
			"security_version":    gorm.Expr("security_version + 1"),
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
	if err := tx.Where("user_id = ?", uid).Delete(&models.PasswordResetCode{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.PostponeRecord{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.Checkin{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.DailyCheckin{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", uid).Delete(&models.PlanActionLayout{}).Error; err != nil {
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
