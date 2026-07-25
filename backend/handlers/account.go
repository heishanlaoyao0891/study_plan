package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

type onboardingReq struct {
	Status string `json:"status" binding:"required"`
}

type changePasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

type redeemPasswordResetReq struct {
	Username    string `json:"username" binding:"required"`
	Code        string `json:"code" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func UpdateOnboarding(c *gin.Context) {
	var req onboardingReq
	if err := c.ShouldBindJSON(&req); err != nil || (req.Status != models.OnboardingStatusCompleted && req.Status != models.OnboardingStatusSkipped) {
		api.Fail(c, http.StatusBadRequest, "status must be completed or skipped")
		return
	}
	uid := c.GetUint(middleware.CtxUserIDKey)
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	if user.OnboardingVersion == models.CurrentOnboardingVersion && user.OnboardingStatus != models.OnboardingStatusNotStarted {
		api.OK(c, user)
		return
	}
	now := time.Now()
	if err := db.DB.Model(&user).Updates(map[string]interface{}{
		"onboarding_status": req.Status, "onboarding_version": models.CurrentOnboardingVersion, "onboarding_completed_at": &now,
	}).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "update onboarding failed: "+err.Error())
		return
	}
	user.OnboardingStatus = req.Status
	user.OnboardingVersion = models.CurrentOnboardingVersion
	user.OnboardingCompletedAt = &now
	api.OK(c, user)
}

func ChangePassword(c *gin.Context) {
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	passwordHash, err := hashPassword(req.NewPassword)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	uid := c.GetUint(middleware.CtxUserIDKey)
	var user models.User
	if err := db.DB.First(&user, uid).Error; err != nil || user.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.CurrentPassword)) != nil {
		api.Fail(c, http.StatusUnauthorized, "current password is incorrect")
		return
	}
	if err := db.DB.Model(&user).Updates(map[string]interface{}{"password_hash": passwordHash, "security_version": gorm.Expr("security_version + 1")}).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "change password failed: "+err.Error())
		return
	}
	user.PasswordHash = &passwordHash
	user.SecurityVersion++
	token, err := services.SignToken(user.ID, user.OpenID, user.Role, user.SecurityVersion)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "sign token failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"token": token, "user": user})
}

func CreatePasswordResetCode(c *gin.Context) {
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
	if user.AccountStatus != models.AccountStatusActive || user.UsernameNormalized == "" || user.PasswordHash == nil {
		api.Fail(c, http.StatusBadRequest, "password reset requires an active password account")
		return
	}
	codeBytes := make([]byte, 18)
	if _, err := rand.Read(codeBytes); err != nil {
		api.Fail(c, http.StatusInternalServerError, "generate reset code failed")
		return
	}
	code := base64.RawURLEncoding.EncodeToString(codeBytes)
	now := time.Now()
	expiresAt := now.Add(30 * time.Minute)
	reset := models.PasswordResetCode{
		UserID: user.ID, CodeHash: hashResetCode(code), ExpiresAt: expiresAt, CreatedByAdminID: c.GetUint(middleware.CtxUserIDKey),
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.PasswordResetCode{}).Where("user_id = ? AND consumed_at IS NULL", user.ID).Update("consumed_at", now).Error; err != nil {
			return err
		}
		return tx.Create(&reset).Error
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "save reset code failed: "+err.Error())
		return
	}
	recordAdminAudit(reset.CreatedByAdminID, &user.ID, "create_password_reset", "expires in 30 minutes")
	api.OK(c, gin.H{"code": code, "expires_at": expiresAt})
}

func RedeemPasswordReset(c *gin.Context) {
	var req redeemPasswordResetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}
	passwordHash, err := hashPassword(req.NewPassword)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))
	codeHash := hashResetCode(strings.TrimSpace(req.Code))
	now := time.Now()
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.Where("username_normalized = ? AND account_status = ?", username, models.AccountStatusActive).First(&user).Error; err != nil {
			return err
		}
		result := tx.Model(&models.PasswordResetCode{}).
			Where("user_id = ? AND code_hash = ? AND consumed_at IS NULL AND expires_at > ?", user.ID, codeHash, now).
			Update("consumed_at", now)
		if result.Error != nil || result.RowsAffected != 1 {
			if result.Error != nil {
				return result.Error
			}
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&user).Updates(map[string]interface{}{"password_hash": passwordHash, "security_version": gorm.Expr("security_version + 1")}).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.Fail(c, http.StatusBadRequest, "invalid or expired reset code")
			return
		}
		api.Fail(c, http.StatusInternalServerError, "reset password failed")
		return
	}
	api.OK(c, gin.H{"reset": true})
}

func hashResetCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
