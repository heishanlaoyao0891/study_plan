package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

type banReq struct {
	DurationHours int    `json:"duration_hours"` // 0=永久封禁；>0=指定小时数
	Reason        string `json:"reason"`
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

	var (
		total int64
		users []models.User
	)
	db.DB.Model(&models.User{}).Count(&total)
	if err := db.DB.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&users).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query users failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{
		"total":   total,
		"page":    page,
		"size":    size,
		"users":   users,
	})
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
	api.OK(c, user)
}