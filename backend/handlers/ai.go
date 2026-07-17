package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

type generatePlanReq struct {
	Goal        string   `json:"goal" binding:"required"`
	HoursPerDay int      `json:"hours_per_day"`
	Days        int      `json:"days"`
	StartDate   string   `json:"start_date"`
	SkipDates   []string `json:"skip_dates"`
}

func GeneratePlan(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req generatePlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.Days <= 0 {
		req.Days = 7
	}
	if req.HoursPerDay <= 0 {
		req.HoursPerDay = 1
	}
	start := time.Now()
	if req.StartDate != "" {
		if t, err := time.Parse(dateLayout, req.StartDate); err == nil {
			start = t
		}
	}
	skip := map[string]bool{}
	for _, d := range req.SkipDates {
		skip[d] = true
	}
	plan := models.Plan{UserID: uid, Title: req.Goal, Description: "AI mock 生成计划", Status: models.PlanStatusActive, WeeklyTargetHours: req.HoursPerDay * 7, AIGenerated: true}
	var tasks []models.DailyTask
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		day := 0
		for len(tasks) < req.Days {
			date := start.AddDate(0, 0, day).Format(dateLayout)
			day++
			if skip[date] {
				continue
			}
			task := models.DailyTask{UserID: uid, PlanID: plan.ID, Date: date, Title: fmt.Sprintf("Day %d：%s", len(tasks)+1, req.Goal), Description: fmt.Sprintf("学习 %s 的第 %d 天任务，建议投入 %d 小时。", req.Goal, len(tasks)+1, req.HoursPerDay), Status: models.TaskStatusPending}
			if err := tx.Create(&task).Error; err != nil {
				return err
			}
			tasks = append(tasks, task)
		}
		return nil
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "generate plan failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"plan": plan, "tasks": tasks, "mode": "mock"})
}

func RegeneratePlan(c *gin.Context) { GeneratePlan(c) }

func EditAIPlan(c *gin.Context) {
	api.OK(c, gin.H{"message": "AI plan edit is covered by normal plan/task APIs in MVP"})
}
