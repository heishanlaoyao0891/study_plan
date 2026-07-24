package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

var defaultPlanActions = models.PlanActionLayout{Direct: []string{"toggle_status", "edit"}, Overflow: []string{"postpone", "invite", "delete"}}
var allowedPlanActions = map[string]bool{"toggle_status": true, "edit": true, "postpone": true, "invite": true, "delete": true}

func GetPlanActionLayout(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var layout models.PlanActionLayout
	err := db.DB.Where("user_id = ?", uid).First(&layout).Error
	if err == gorm.ErrRecordNotFound {
		layout = defaultPlanActions
		layout.UserID = uid
	} else if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query action layout failed: "+err.Error())
		return
	}
	layout.Direct, layout.Overflow = normalizePlanActionLayout(layout.Direct, layout.Overflow)
	api.OK(c, layout)
}

func UpdatePlanActionLayout(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req struct {
		Direct   []string `json:"direct"`
		Overflow []string `json:"overflow"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	direct, overflow := normalizePlanActionLayout(req.Direct, req.Overflow)
	layout := models.PlanActionLayout{UserID: uid, Direct: direct, Overflow: overflow}
	if err := db.DB.Where("user_id = ?", uid).Assign(models.PlanActionLayout{Direct: direct, Overflow: overflow}).FirstOrCreate(&layout).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "save action layout failed: "+err.Error())
		return
	}
	api.OK(c, layout)
}

func normalizePlanActionLayout(requestedDirect, requestedOverflow []string) ([]string, []string) {
	seen := map[string]bool{}
	direct, overflow := []string{}, []string{}
	for _, id := range requestedDirect {
		if allowedPlanActions[id] && !seen[id] {
			direct = append(direct, id)
			seen[id] = true
		}
	}
	for _, id := range requestedOverflow {
		if allowedPlanActions[id] && !seen[id] {
			overflow = append(overflow, id)
			seen[id] = true
		}
	}
	for _, id := range defaultPlanActions.Overflow {
		if !seen[id] {
			overflow = append(overflow, id)
			seen[id] = true
		}
	}
	for _, id := range defaultPlanActions.Direct {
		if !seen[id] {
			overflow = append(overflow, id)
			seen[id] = true
		}
	}
	return direct, overflow
}
