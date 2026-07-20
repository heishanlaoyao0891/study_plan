package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
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
	var cfg models.AIConfig
	if err := db.DB.Order("id ASC").First(&cfg).Error; err == nil && !cfg.Enabled {
		api.Fail(c, http.StatusForbidden, "AI generation is disabled")
		return
	}
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
	ctx, err := services.BuildPlanningContext(services.PlanGenerationInput{UserID: uid, Goal: req.Goal, HoursPerDay: req.HoursPerDay, Days: req.Days, StartDate: req.StartDate, SkipDates: req.SkipDates})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "build planning context failed: "+err.Error())
		return
	}
	preview := services.FallbackPlanPreview(ctx)
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
	plan := models.Plan{UserID: uid, Title: preview.Title, Description: preview.Rationale, Status: models.PlanStatusActive, WeeklyTargetHours: req.HoursPerDay * 7, AIGenerated: true}
	var tasks []models.DailyTask
	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		day := 0
		for len(tasks) < len(preview.Tasks) {
			date := start.AddDate(0, 0, day).Format(dateLayout)
			day++
			if skip[date] {
				continue
			}
			previewTask := preview.Tasks[len(tasks)]
			task := models.DailyTask{UserID: uid, PlanID: plan.ID, Date: previewTask.Date, Title: previewTask.Title, Description: previewTask.Description, Status: models.TaskStatusPending}
			if err := tx.Create(&task).Error; err != nil {
				return err
			}
			tasks = append(tasks, task)
		}
		return nil
	})
	if txErr != nil {
		api.Fail(c, http.StatusInternalServerError, "generate plan failed: "+txErr.Error())
		return
	}
	api.OK(c, gin.H{"plan": plan, "tasks": tasks, "mode": "fallback", "rationale": preview.Rationale})
}

func RegeneratePlan(c *gin.Context) { GeneratePlan(c) }

func EditAIPlan(c *gin.Context) {
	api.OK(c, gin.H{"message": "AI plan edit is covered by normal plan/task APIs in MVP"})
}
