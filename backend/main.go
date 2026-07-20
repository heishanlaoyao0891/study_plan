package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/handlers"
	"study_plan_backend/middleware"
	"study_plan_backend/services"
)

func main() {
	// 加载 .env（可选）
	_ = godotenv.Load()
	config.Load()
	if err := config.App.Validate(); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}

	if err := db.Init(); err != nil {
		log.Fatalf("init db: %v", err)
	}
	services.StartArchiveSync()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.StructuredLogger())
	r.Use(cors())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "now": time.Now().Unix()})
	})

	apiGroup := r.Group("/api")
	// 认证
	apiGroup.POST("/auth/login", handlers.Login)
	apiGroup.POST("/admin/auth/login", handlers.AdminLogin)

	// 需要登录
	auth := apiGroup.Group("", middleware.Auth())
	{
		auth.GET("/auth/me", handlers.CurrentUser)
		auth.POST("/auth/phone", handlers.BindPhoneNumber)
		auth.PUT("/auth/avatar", handlers.UpdateAvatar)
	}

	bound := apiGroup.Group("", middleware.Auth(), middleware.RequirePhoneBound())
	{
		// 计划
		bound.GET("/plans", handlers.ListPlans)
		bound.POST("/plans", handlers.CreatePlan)
		bound.GET("/plans/:id", handlers.GetPlan)
		bound.PUT("/plans/:id", handlers.UpdatePlan)
		bound.DELETE("/plans/:id", handlers.DeletePlan)
		bound.PUT("/plans/:id/pause", handlers.PausePlan)
		bound.PUT("/plans/:id/resume", handlers.ResumePlan)
		bound.PUT("/plans/:id/shift", handlers.ShiftPlan)
		bound.POST("/plans/:id/invite", handlers.InvitePlanMember)
		bound.POST("/plans/:id/join", handlers.JoinPlan)

		// AI 计划
		bound.POST("/ai/generate-plan", handlers.GeneratePlan)
		bound.POST("/ai/regenerate", handlers.RegeneratePlan)
		bound.POST("/ai/commit-plan", handlers.CommitAIPlan)
		bound.PUT("/ai/plan/:id/edit", handlers.EditAIPlan)

		// 打卡
		bound.GET("/checkins", handlers.ListCheckins)
		bound.POST("/checkins", handlers.ToggleCheckin)
		bound.GET("/checkins/streak", handlers.Streak)

		// 学习任务
		bound.GET("/plans/:id/tasks", handlers.ListPlanTasks)
		bound.POST("/plans/:id/tasks", handlers.CreatePlanTask)
		bound.PUT("/plans/:id/tasks/reorder", handlers.ReorderPlanTasks)
		bound.PUT("/tasks/:id/start", handlers.StartTask)
		bound.PUT("/tasks/:id/stop", handlers.StopTask)
		bound.PUT("/tasks/:id/pause", handlers.StopTask)
		bound.PUT("/tasks/:id/resume", handlers.StartTask)
		bound.PUT("/tasks/:id/extend", handlers.StartTask)
		bound.GET("/tasks/:id", handlers.GetTask)
		bound.PUT("/tasks/:id", handlers.UpdateTask)
		bound.DELETE("/tasks/:id", handlers.DeleteTask)
		bound.PUT("/tasks/:id/postpone", handlers.PostponeTask)
		bound.PUT("/tasks/:id/makeup", handlers.MakeupTask)
		bound.PUT("/tasks/:id/complete", handlers.CompleteTask)
		bound.GET("/tasks/pending-decision", handlers.PendingDecisionTasks)
		bound.POST("/tasks/midnight-compensate", handlers.CompensateMidnightTasks)

		// 躺平币
		bound.GET("/slack/balance", handlers.SlackBalance)
		bound.POST("/slack/start", handlers.StartSlack)
		bound.PUT("/slack/stop", handlers.StopSlack)
		bound.GET("/slack/records", handlers.SlackRecords)

		// 统计
		bound.GET("/stats/calendar", handlers.StatsCalendar)
		bound.GET("/stats/streak", handlers.Streak)
		bound.GET("/stats/daily-distribution", handlers.DailyDistribution)
		bound.GET("/stats/weekly-report", handlers.WeeklyReport)
		bound.GET("/stats/monthly-report", handlers.MonthlyReport)
		bound.GET("/stats/slack-distribution", handlers.SlackDistribution)
		bound.GET("/stats/efficiency", handlers.EfficiencyStats)

		// 提醒占位
		bound.GET("/notifications/subscriptions", handlers.NotificationSubscriptions)
		bound.GET("/notifications/due", handlers.DueNotificationEvents)
		bound.POST("/notifications/subscribe", handlers.SubscribeNotification)
		bound.DELETE("/notifications/subscribe", handlers.UnsubscribeNotification)
	}

	// 管理员
	admin := apiGroup.Group("/admin", middleware.Auth(), middleware.RequireAdmin())
	{
		admin.GET("/overview", handlers.AdminOverview)
		admin.GET("/users", handlers.ListUsers)
		admin.GET("/users/:id", handlers.GetAdminUserDetail)
		admin.POST("/users/:id/ban", handlers.BanUser)
		admin.POST("/users/:id/unban", handlers.UnbanUser)
		admin.GET("/slack-config", handlers.GetSlackConfigs)
		admin.PUT("/slack-config", handlers.UpsertGlobalSlackConfig)
		admin.PUT("/slack-config/:userId", handlers.UpsertUserSlackConfig)
		admin.GET("/ai-config", handlers.GetAIConfig)
		admin.PUT("/ai-config", handlers.UpdateAIConfig)
		admin.POST("/ai-config/test", handlers.TestAIProvider)
		admin.GET("/subscription-config", handlers.GetSubscriptionMessageConfig)
		admin.PUT("/subscription-config", handlers.UpdateSubscriptionMessageConfig)
		admin.GET("/audit-logs", handlers.ListAuditLogs)
	}

	addr := ":" + config.App.Port
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 15 * time.Second,
	}

	// 优雅启停
	go func() {
		log.Printf("[study_plan] listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("\nshutting down...")
	_ = srv.Close()
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
