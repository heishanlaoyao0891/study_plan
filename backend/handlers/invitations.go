package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

const registrationInviteLifetime = 7 * 24 * time.Hour

type createInvitationsReq struct {
	Count int `json:"count" binding:"required,min=1,max=100"`
}

func CreateRegistrationInvites(c *gin.Context) {
	var req createInvitationsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "count must be between 1 and 100")
		return
	}
	adminID := c.GetUint(middleware.CtxUserIDKey)
	codes := make([]string, 0, req.Count)
	invites := make([]models.RegistrationInvite, 0, req.Count)
	for len(codes) < req.Count {
		randomBytes := make([]byte, 24)
		if _, err := rand.Read(randomBytes); err != nil {
			api.Fail(c, http.StatusInternalServerError, "generate invitation failed: "+err.Error())
			return
		}
		code := base64.RawURLEncoding.EncodeToString(randomBytes)
		codes = append(codes, code)
		invites = append(invites, models.RegistrationInvite{
			CodeHash: hashInviteCode(code), CodePrefix: code[:8], ExpiresAt: time.Now().Add(registrationInviteLifetime), CreatedByAdminID: &adminID,
		})
	}
	if err := db.DB.Create(&invites).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "create invitations failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"codes": codes, "invitations": invites})
}

func ListRegistrationInvites(c *gin.Context) {
	var invites []models.RegistrationInvite
	if err := db.DB.Order("id DESC").Find(&invites).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "list invitations failed: "+err.Error())
		return
	}
	now := time.Now()
	userIDs := make([]uint, 0)
	for _, invite := range invites {
		if invite.UserID != nil {
			userIDs = append(userIDs, *invite.UserID)
		}
	}
	users := map[uint]models.User{}
	if len(userIDs) > 0 {
		var usedBy []models.User
		if err := db.DB.Select("id", "username", "nickname").Where("id IN ?", userIDs).Find(&usedBy).Error; err != nil {
			api.Fail(c, http.StatusInternalServerError, "list invitation users failed: "+err.Error())
			return
		}
		for _, user := range usedBy {
			users[user.ID] = user
		}
	}
	rows := make([]gin.H, 0, len(invites))
	for _, invite := range invites {
		status := "active"
		if invite.UsedAt != nil {
			status = "used"
		} else if invite.DisabledAt != nil {
			status = "disabled"
		} else if !invite.ExpiresAt.After(now) {
			status = "expired"
		}
		row := gin.H{
			"id": invite.ID, "code_prefix": invite.CodePrefix, "expires_at": invite.ExpiresAt,
			"used_at": invite.UsedAt, "user_id": invite.UserID, "disabled_at": invite.DisabledAt,
			"created_at": invite.CreatedAt, "created_by_admin_id": invite.CreatedByAdminID, "status": status,
		}
		if invite.UserID != nil {
			if user, ok := users[*invite.UserID]; ok {
				row["used_by_user"] = gin.H{"id": user.ID, "username": user.Username, "nickname": user.Nickname}
			}
		}
		rows = append(rows, row)
	}
	api.OK(c, rows)
}

func DisableRegistrationInvite(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid invitation id")
		return
	}
	now := time.Now()
	result := db.DB.Model(&models.RegistrationInvite{}).Where("id = ? AND used_at IS NULL AND disabled_at IS NULL", id).Update("disabled_at", now)
	if result.Error != nil {
		api.Fail(c, http.StatusInternalServerError, "disable invitation failed: "+result.Error.Error())
		return
	}
	if result.RowsAffected != 1 {
		api.Fail(c, http.StatusNotFound, "active invitation not found")
		return
	}
	api.OK(c, gin.H{"disabled": true})
}
