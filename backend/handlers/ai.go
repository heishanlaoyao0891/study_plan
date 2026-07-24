package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

type generatePlanReq struct {
	Goal              string   `json:"goal" binding:"required"`
	HoursPerDay       int      `json:"hours_per_day"`
	Days              int      `json:"days"`
	StartDate         string   `json:"start_date"`
	AvailableTimeSlot string   `json:"available_time_slot"`
	SkipDates         []string `json:"skip_dates"`
	Refinement        string   `json:"refinement"`
}

type commitAIPlanReq struct {
	Preview services.PlanPreview `json:"preview" binding:"required"`
}

func GeneratePlan(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var cfg models.AIConfig
	if err := db.DB.Order("id ASC").First(&cfg).Error; err == nil && !cfg.Enabled {
		services.RecordAIGenerationUsage(uid, cfg.Provider, "disabled", "AI generation is disabled")
		api.Fail(c, http.StatusForbidden, "AI generation is disabled")
		return
	}
	canUse, usedToday, err := services.CanUseAIGeneration(uid, cfg.DailyGenerationLimit)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query ai usage failed: "+err.Error())
		return
	}
	if !canUse {
		services.RecordAIGenerationUsage(uid, cfg.Provider, "limit_exceeded", "daily generation limit exceeded")
		api.Fail(c, http.StatusTooManyRequests, "daily AI generation limit exceeded")
		return
	}
	var req generatePlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		services.RecordAIGenerationUsage(uid, cfg.Provider, "failed", err.Error())
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.Days <= 0 {
		req.Days = 7
	}
	if req.HoursPerDay <= 0 {
		req.HoursPerDay = 1
	}
	ctx, err := services.BuildPlanningContext(services.PlanGenerationInput{UserID: uid, Goal: req.Goal, HoursPerDay: req.HoursPerDay, Days: req.Days, StartDate: req.StartDate, AvailableTimeSlot: req.AvailableTimeSlot, SkipDates: req.SkipDates, Refinement: req.Refinement})
	if err != nil {
		services.RecordAIGenerationUsage(uid, cfg.Provider, "failed", err.Error())
		api.Fail(c, http.StatusInternalServerError, "build planning context failed: "+err.Error())
		return
	}
	preview := services.PlanPreview{}
	mode := "ai"
	_, provider, providerErr := services.CurrentAIProvider()
	if providerErr == nil {
		raw, generateErr := provider.Generate(ctx.Prompt, 4096)
		if generateErr == nil {
			preview, err = services.ParsePlanPreviewJSON(raw)
			if err != nil {
				services.RecordAIGenerationUsage(uid, cfg.Provider, "failed", "invalid provider response: "+err.Error())
				api.Fail(c, http.StatusBadGateway, "AI returned an invalid plan: "+err.Error())
				return
			}
		} else {
			mode = "fallback"
			preview = services.FallbackPlanPreview(ctx)
		}
	} else {
		mode = "fallback"
		preview = services.FallbackPlanPreview(ctx)
	}
	if err := services.ValidatePlanPreview(preview, ctx.Input); err != nil {
		services.RecordAIGenerationUsage(uid, cfg.Provider, "failed", err.Error())
		api.Fail(c, http.StatusBadGateway, "AI plan validation failed: "+err.Error())
		return
	}
	services.RecordAIGenerationUsage(uid, cfg.Provider, "success", mode+" preview generated")
	api.OK(c, gin.H{"preview": preview, "mode": mode, "usage": gin.H{"used_today": usedToday + 1, "daily_limit": maxPositive(cfg.DailyGenerationLimit, 5)}})
}

func CommitAIPlan(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req commitAIPlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	input := services.PlanGenerationInput{UserID: uid, Goal: req.Preview.Title, Days: len(req.Preview.Tasks)}
	if err := services.ValidatePlanPreview(req.Preview, input); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid preview: "+err.Error())
		return
	}
	first, last := req.Preview.Tasks[0], req.Preview.Tasks[len(req.Preview.Tasks)-1]
	plan := models.Plan{UserID: uid, Title: req.Preview.Title, Description: req.Preview.Summary, Status: models.PlanStatusActive, WeeklyTargetHours: int(req.Preview.EstimatedTotalHours), AIGenerated: true, StartDate: first.Date, EndDate: last.Date, DefaultPlannedStart: first.PlannedStart, DefaultPlannedEnd: first.PlannedEnd}
	for _, previewTask := range req.Preview.Tasks {
		plan.StudyDates = append(plan.StudyDates, previewTask.Date)
	}
	var tasks []models.DailyTask
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		for i, previewTask := range req.Preview.Tasks {
			task := models.DailyTask{UserID: uid, PlanID: plan.ID, Date: previewTask.Date, Title: previewTask.Title, Objective: previewTask.Objective, Description: previewTask.Description, SortOrder: i, PlannedStart: previewTask.PlannedStart, PlannedEnd: previewTask.PlannedEnd, EstimatedMinutes: previewTask.EstimatedMinutes, Difficulty: previewTask.Difficulty, Status: models.TaskStatusPending}
			if err := tx.Create(&task).Error; err != nil {
				return err
			}
			tasks = append(tasks, task)
		}
		return nil
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "commit ai plan failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"plan": plan, "tasks": tasks})
}

func RegeneratePlan(c *gin.Context) { GeneratePlan(c) }

func EditAIPlan(c *gin.Context) {
	api.OK(c, gin.H{"message": "AI plan edit is covered by normal plan/task APIs in MVP"})
}

func maxPositive(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}
