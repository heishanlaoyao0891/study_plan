package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

type loginReq struct {
	Code       string `json:"code" binding:"required"`
	Nickname   string `json:"nickname"`
	AvatarURL  string `json:"avatar_url"`
}

type loginResp struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

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
			OpenID:       s.OpenID,
			Nickname:     req.Nickname,
			AvatarURL:    req.AvatarURL,
			Role:         models.RoleUser,
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