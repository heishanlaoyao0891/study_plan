package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"study_plan_backend/db"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

const CtxUserIDKey = "user_id"
const CtxRoleKey = "role"

// Auth 校验 JWT，并把 user_id / role 写入 context
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "missing Authorization header",
			})
			return
		}
		claims, err := services.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid or expired token",
			})
			return
		}
		// 同步检查当前用户的封禁状态（防止 token 内忽略封禁）
		var user models.User
		if err := db.DB.First(&user, claims.UserID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "user not found",
			})
			return
		}
		if user.BannedUntil != nil && user.BannedUntil.After(time.Now()) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":         403,
				"message":      "user banned",
				"banned_until": user.BannedUntil,
				"reason":       user.BannedReason,
			})
			return
		}
		c.Set(CtxUserIDKey, user.ID)
		c.Set(CtxRoleKey, user.Role)
		c.Next()
	}
}

// RequireAdmin 仅允许 admin 访问
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(CtxRoleKey)
		if role != models.RoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "admin only",
			})
			return
		}
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return auth
}