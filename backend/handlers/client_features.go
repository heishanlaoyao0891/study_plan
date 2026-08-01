package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/models"
)

const miniProgramPlatform = "mp-weixin"

// GetClientFeatures exposes only effective client capabilities. Provider details
// remain restricted to the admin configuration API.
func GetClientFeatures(c *gin.Context) {
	api.OK(c, gin.H{"mini_program_ai_enabled": loadMiniProgramAIEnabled()})
}

func loadMiniProgramAIEnabled() bool {
	if db.DB == nil {
		return false
	}
	var cfg models.AIConfig
	if err := db.DB.Order("id ASC").First(&cfg).Error; err != nil {
		return false
	}
	return cfg.MiniProgramAIEnabled
}

func allowMiniProgramAIRequest(c *gin.Context) bool {
	if !strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Client-Platform")), miniProgramPlatform) {
		return true
	}
	if loadMiniProgramAIEnabled() {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{
		"code":    http.StatusForbidden,
		"message": "mini-program AI plan generation is disabled",
		"data":    gin.H{"feature": "mini_program_ai", "enabled": false},
	})
	return false
}
