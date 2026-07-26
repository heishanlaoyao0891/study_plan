package banstate

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/models"
)

const Message = "账号访问已暂停，请查看封禁说明"

type Data struct {
	AccountBanned bool   `json:"account_banned"`
	Reason        string `json:"reason"`
	BannedUntil   string `json:"banned_until"`
	Permanent     bool   `json:"permanent"`
	ServerNow     string `json:"server_now"`
}

// Block evaluates the persisted ban once, clears expired bans, and writes the
// canonical response when access must remain blocked.
func Block(c *gin.Context, user *models.User, now time.Time) bool {
	if user.BannedUntil == nil {
		return false
	}
	if !user.BannedUntil.After(now) {
		if err := db.DB.Model(user).Updates(map[string]interface{}{
			"banned_until":  nil,
			"banned_reason": "",
		}).Error; err != nil {
			api.Fail(c, http.StatusInternalServerError, "clear expired ban failed")
			return true
		}
		user.BannedUntil = nil
		user.BannedReason = ""
		return false
	}
	c.JSON(http.StatusForbidden, gin.H{
		"code":    http.StatusForbidden,
		"message": Message,
		"data": Data{
			AccountBanned: true,
			Reason:        user.BannedReason,
			BannedUntil:   user.BannedUntil.UTC().Format(time.RFC3339),
			Permanent:     models.IsPermanentBan(user.BannedUntil),
			ServerNow:     now.UTC().Format(time.RFC3339),
		},
	})
	return true
}
