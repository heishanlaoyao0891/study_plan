package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/banstate"
	"study_plan_backend/db"
	"study_plan_backend/identity"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{4,24}$`)

var (
	errInvalidInvite      = errors.New("invitation is invalid, expired, used, or disabled")
	errOpenIDInUse        = errors.New("WeChat identity is already linked")
	errInvalidCredentials = errors.New("invalid username or password")
)

type h5RegisterReq struct {
	InviteCode string `json:"invite_code" binding:"required"`
	Username   string `json:"username" binding:"required"`
	Nickname   string `json:"nickname" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

type h5LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type weChatRegisterReq struct {
	RegistrationToken string `json:"registration_token" binding:"required"`
	InviteCode        string `json:"invite_code" binding:"required"`
	Username          string `json:"username" binding:"required"`
	Nickname          string `json:"nickname" binding:"required"`
	Password          string `json:"password" binding:"required"`
}

type weChatLinkReq struct {
	RegistrationToken string `json:"registration_token" binding:"required"`
	Username          string `json:"username" binding:"required"`
	Password          string `json:"password" binding:"required"`
}

func H5Register(c *gin.Context) {
	var req h5RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	user, err := registerInvitedAccount(req.InviteCode, req.Username, req.Nickname, req.Password, "")
	if err != nil {
		handleRegistrationError(c, err)
		return
	}
	respondWithUserToken(c, user)
}

func H5Login(c *gin.Context) {
	var req h5LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))
	var user models.User
	if err := db.DB.Where("username_normalized = ?", username).First(&user).Error; err != nil || user.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)) != nil {
		api.Fail(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if !allowActiveUser(c, &user) {
		return
	}
	respondWithUserToken(c, user)
}

func WeChatRegister(c *gin.Context) {
	var req weChatRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	claims, err := services.ParseRegistrationToken(req.RegistrationToken)
	if err != nil {
		api.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	user, err := registerInvitedAccount(req.InviteCode, req.Username, req.Nickname, req.Password, claims.OpenID)
	if err != nil {
		handleRegistrationError(c, err)
		return
	}
	respondWithUserToken(c, user)
}

func WeChatLink(c *gin.Context) {
	var req weChatLinkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	claims, err := services.ParseRegistrationToken(req.RegistrationToken)
	if err != nil {
		api.Fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))
	var user models.User
	if err := db.DB.Where("username_normalized = ?", username).First(&user).Error; err != nil || user.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)) != nil {
		api.Fail(c, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if !allowActiveUser(c, &user) {
		return
	}
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if user.OpenID != "" && user.OpenID != claims.OpenID {
			return errOpenIDInUse
		}
		var count int64
		if err := tx.Model(&models.User{}).Where("open_id = ? AND id <> ?", claims.OpenID, user.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errOpenIDInUse
		}
		user.OpenID = claims.OpenID
		return tx.Model(&user).Update("open_id", claims.OpenID).Error
	})
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			api.Fail(c, http.StatusUnauthorized, errInvalidCredentials.Error())
			return
		}
		handleRegistrationError(c, err)
		return
	}
	respondWithUserToken(c, user)
}

func registerInvitedAccount(inviteCode, username, nickname, password, openID string) (models.User, error) {
	username = strings.TrimSpace(username)
	if !usernamePattern.MatchString(username) {
		return models.User{}, errors.New("username must contain 4 to 24 ASCII letters, digits, or underscores")
	}
	usernameNormalized := strings.ToLower(username)
	display, nicknameNormalized, err := identity.Validate(nickname)
	if err != nil {
		return models.User{}, err
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return models.User{}, err
	}
	codeHash := hashInviteCode(inviteCode)
	if strings.TrimSpace(inviteCode) == "" {
		return models.User{}, errInvalidInvite
	}

	var user models.User
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if openID != "" {
			err := tx.Where("open_id = ?", openID).First(&user).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err == nil && user.UsernameNormalized != "" {
				return errOpenIDInUse
			}
		}
		if user.ID == 0 {
			targetID, err := identity.NewInviteTargetID()
			if err != nil {
				return err
			}
			user = models.User{OpenID: openID, InviteTargetID: targetID, Role: models.RoleUser}
		}
		user.Username = username
		user.UsernameNormalized = usernameNormalized
		user.Nickname = display
		user.NicknameNormalized = nicknameNormalized
		user.PasswordHash = &passwordHash
		user.AccountStatus = models.AccountStatusActive
		if user.ID == 0 {
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		} else if err := tx.Model(&user).Updates(map[string]interface{}{
			"username": username, "username_normalized": usernameNormalized,
			"nickname": display, "nickname_normalized": nicknameNormalized,
			"password_hash": passwordHash, "account_status": models.AccountStatusActive,
		}).Error; err != nil {
			return err
		}
		result := tx.Model(&models.RegistrationInvite{}).
			Where("code_hash = ? AND used_at IS NULL AND disabled_at IS NULL AND expires_at > ?", codeHash, now).
			Updates(map[string]interface{}{"used_at": now, "user_id": user.ID})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errInvalidInvite
		}
		return nil
	})
	return user, err
}

func hashInviteCode(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(sum[:])
}

func hashPassword(password string) (string, error) {
	if len(password) < 8 || len(password) > 72 {
		return "", errors.New("password must contain 8 to 72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func respondWithUserToken(c *gin.Context, user models.User) {
	token, err := services.SignToken(user.ID, user.OpenID, user.Role, user.SecurityVersion)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "sign token failed: "+err.Error())
		return
	}
	api.OK(c, loginResp{Token: token, User: user, NicknameRequired: false})
}

func allowActiveUser(c *gin.Context, user *models.User) bool {
	if banstate.Block(c, user, time.Now()) {
		return false
	}
	if user.AccountStatus == models.AccountStatusDeleted {
		api.Fail(c, http.StatusForbidden, "account is deleted")
		return false
	}
	if user.AccountStatus == models.AccountStatusInactive {
		if err := db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(user).Updates(map[string]interface{}{"account_status": models.AccountStatusActive, "security_version": gorm.Expr("security_version + 1")}).Error; err != nil {
				return err
			}
			return tx.Create(&models.AccountEvent{UserID: user.ID, EventType: "restore", Detail: "reactivated after credential login"}).Error
		}); err != nil {
			api.Fail(c, http.StatusInternalServerError, "restore account failed: "+err.Error())
			return false
		}
		user.AccountStatus = models.AccountStatusActive
		user.SecurityVersion++
	}
	return true
}

func handleRegistrationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errInvalidInvite):
		api.Fail(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, errOpenIDInUse), isUniqueConstraintError(err):
		api.Conflict(c, "username, nickname, or WeChat identity is already in use", nil)
	default:
		if strings.Contains(err.Error(), "username must") || strings.Contains(err.Error(), "password must") || strings.Contains(err.Error(), "nickname") {
			api.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
		api.Fail(c, http.StatusInternalServerError, "registration failed: "+err.Error())
	}
}
