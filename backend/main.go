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
)

func main() {
	// 加载 .env（可选）
	_ = godotenv.Load()
	config.Load()

	if err := db.Init(); err != nil {
		log.Fatalf("init db: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{SkipPaths: []string{"/health"}}))
	r.Use(cors())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "now": time.Now().Unix()})
	})

	apiGroup := r.Group("/api")
	// 认证
	apiGroup.POST("/auth/login", handlers.Login)

	// 需要登录
	auth := apiGroup.Group("", middleware.Auth())
	{
		// 计划
		auth.GET("/plans", handlers.ListPlans)
		auth.POST("/plans", handlers.CreatePlan)
		auth.GET("/plans/:id", handlers.GetPlan)
		auth.PUT("/plans/:id", handlers.UpdatePlan)
		auth.DELETE("/plans/:id", handlers.DeletePlan)
		auth.PUT("/plans/:id/pause", handlers.PausePlan)
		auth.PUT("/plans/:id/resume", handlers.ResumePlan)

		// 打卡
		auth.GET("/checkins", handlers.ListCheckins)
		auth.POST("/checkins", handlers.ToggleCheckin)
		auth.GET("/checkins/streak", handlers.Streak)
	}

	// 管理员
	admin := apiGroup.Group("/admin", middleware.Auth(), middleware.RequireAdmin())
	{
		admin.GET("/users", handlers.ListUsers)
		admin.POST("/users/:id/ban", handlers.BanUser)
		admin.POST("/users/:id/unban", handlers.UnbanUser)
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
