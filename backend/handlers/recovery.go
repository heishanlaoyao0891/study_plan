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
)

type recoveryAction struct {
	TaskID  uint   `json:"task_id"`
	Title   string `json:"title"`
	OldDate string `json:"old_date"`
	NewDate string `json:"new_date"`
}

type applyRecoveryReq struct {
	Actions []recoveryAction `json:"actions"`
}

func RecoveryPreview(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	preview, err := buildRecoveryPreview(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "build recovery preview failed: "+err.Error())
		return
	}
	api.OK(c, preview)
}

func ApplyRecovery(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req applyRecoveryReq
	_ = c.ShouldBindJSON(&req)
	if len(req.Actions) == 0 {
		preview, err := buildRecoveryPreview(uid)
		if err != nil {
			api.Fail(c, http.StatusInternalServerError, "build recovery preview failed: "+err.Error())
			return
		}
		if actions, ok := preview["actions"].([]recoveryAction); ok {
			req.Actions = actions
		}
	}
	if len(req.Actions) == 0 {
		api.OK(c, gin.H{"applied": 0})
		return
	}
	applied := 0
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		for _, action := range req.Actions {
			res := tx.Model(&models.DailyTask{}).Where("id = ? AND user_id = ? AND status <> ?", action.TaskID, uid, models.TaskStatusCompleted).Updates(map[string]interface{}{"date": action.NewDate, "status": models.TaskStatusPending, "needs_decision": false})
			if res.Error != nil {
				return res.Error
			}
			applied += int(res.RowsAffected)
		}
		return nil
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "apply recovery failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"applied": applied})
}

func buildRecoveryPreview(uid uint) (gin.H, error) {
	today := time.Now().Format(dateLayout)
	var overdue []models.DailyTask
	if err := db.DB.Where("user_id = ? AND date < ? AND status <> ?", uid, today, models.TaskStatusCompleted).Order("date ASC, sort_order ASC, id ASC").Find(&overdue).Error; err != nil {
		return nil, err
	}
	var pendingDecision int64
	if err := db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND needs_decision = ?", uid, true).Count(&pendingDecision).Error; err != nil {
		return nil, err
	}
	missedDays := map[string]bool{}
	actions := make([]recoveryAction, 0, len(overdue))
	for index, task := range overdue {
		missedDays[task.Date] = true
		newDate := time.Now().AddDate(0, 0, index+1).Format(dateLayout)
		actions = append(actions, recoveryAction{TaskID: task.ID, Title: task.Title, OldDate: task.Date, NewDate: newDate})
	}
	mode := "rule"
	if cfg, err := firstAIConfig(); err == nil && cfg.Enabled {
		mode = "ai_fallback"
	}
	return gin.H{"mode": mode, "missed_days": len(missedDays), "overdue_tasks": len(overdue), "pending_decisions": pendingDecision, "actions": actions}, nil
}
