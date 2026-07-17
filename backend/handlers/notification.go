package handlers

import (
	"github.com/gin-gonic/gin"

	"study_plan_backend/api"
)

func NotificationSubscriptions(c *gin.Context) {
	api.OK(c, []gin.H{})
}

func SubscribeNotification(c *gin.Context) {
	api.OK(c, gin.H{"subscribed": true, "mode": "placeholder"})
}

func UnsubscribeNotification(c *gin.Context) {
	api.OK(c, gin.H{"subscribed": false, "mode": "placeholder"})
}
