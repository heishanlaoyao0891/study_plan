package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

type recoveryAction struct {
	TaskID       uint   `json:"task_id"`
	Title        string `json:"title"`
	PlanID       uint   `json:"plan_id"`
	PlanTitle    string `json:"plan_title"`
	OldDate      string `json:"old_date"`
	NewDate      string `json:"new_date"`
	PlannedStart string `json:"planned_start"`
	PlannedEnd   string `json:"planned_end"`
	Reason       string `json:"reason"`
	Valid        bool   `json:"valid"`
	Version      int64  `json:"version"`
}

type applyRecoveryReq struct {
	Token   string           `json:"token" binding:"required"`
	Actions []recoveryAction `json:"actions"`
}
type recoverySnapshot struct {
	UserID    uint
	Actions   map[uint]recoveryAction
	ExpiresAt time.Time
	Mode      string
	PlanID    uint
	PlanDays  int
}

var recoveryPreviews = struct {
	sync.Mutex
	Items map[string]recoverySnapshot
}{Items: map[string]recoverySnapshot{}}

func RecoveryPreview(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	preview, err := buildRecoveryPreview(uid)
	if err != nil {
		if respondScheduleError(c, err) {
			return
		}
		api.Fail(c, http.StatusInternalServerError, "build recovery preview failed: "+err.Error())
		return
	}
	api.OK(c, preview)
}

func ApplyRecovery(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req applyRecoveryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "token and actions required")
		return
	}
	recoveryPreviews.Lock()
	defer recoveryPreviews.Unlock()
	snapshot, ok := recoveryPreviewLocked(req.Token, uid)
	if !ok {
		api.Conflict(c, "recovery preview is stale", gin.H{"stale": true})
		return
	}
	skipped := make([]gin.H, 0)
	applied := 0
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		candidates := make([]models.DailyTask, 0, len(req.Actions))
		for _, action := range req.Actions {
			original, exists := snapshot.Actions[action.TaskID]
			if !exists || !original.Valid {
				skipped = append(skipped, gin.H{"task_id": action.TaskID, "reason": "not_in_preview"})
				continue
			}
			var task models.DailyTask
			if err := tx.Where("id = ? AND user_id = ?", action.TaskID, uid).First(&task).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					skipped = append(skipped, gin.H{"task_id": action.TaskID, "reason": "not_owned_or_deleted"})
					continue
				}
				return err
			}
			if task.UpdatedAt.UnixNano() != original.Version || task.Date != original.OldDate {
				skipped = append(skipped, gin.H{"task_id": task.ID, "reason": "stale"})
				continue
			}
			if task.Status == models.TaskStatusCompleted {
				skipped = append(skipped, gin.H{"task_id": task.ID, "reason": "completed"})
				continue
			}
			var active int64
			if err := tx.Model(&models.StudySession{}).Where("task_id = ? AND end_time IS NULL", task.ID).Count(&active).Error; err != nil {
				return err
			}
			if active > 0 {
				skipped = append(skipped, gin.H{"task_id": task.ID, "reason": "active"})
				continue
			}
			date, dateErr := time.Parse(dateLayout, action.NewDate)
			var plan models.Plan
			planErr := tx.First(&plan, task.PlanID).Error
			if dateErr != nil || validatePlannedRange(action.PlannedStart, action.PlannedEnd) != nil || planErr != nil || !planStudiesOn(plan, date) {
				skipped = append(skipped, gin.H{"task_id": task.ID, "reason": "invalid_schedule"})
				continue
			}
			task.Date, task.PlannedStart, task.PlannedEnd = action.NewDate, action.PlannedStart, action.PlannedEnd
			candidates = append(candidates, task)
		}
		if err := validateScheduleMutation(tx, uid, candidates); err != nil {
			return err
		}
		for _, task := range candidates {
			original := snapshot.Actions[task.ID]
			result := tx.Model(&models.DailyTask{}).Where("id = ? AND user_id = ? AND updated_at = ? AND status <> ? AND NOT EXISTS (SELECT 1 FROM study_sessions WHERE study_sessions.task_id = daily_tasks.id AND study_sessions.end_time IS NULL)", task.ID, uid, time.Unix(0, original.Version), models.TaskStatusCompleted).Updates(map[string]interface{}{"date": task.Date, "planned_start": task.PlannedStart, "planned_end": task.PlannedEnd, "status": models.TaskStatusPending, "needs_decision": false})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				skipped = append(skipped, gin.H{"task_id": task.ID, "reason": "stale"})
				continue
			}
			applied++
		}
		return nil
	})
	if err != nil {
		if respondScheduleError(c, err) {
			return
		}
		api.Fail(c, http.StatusInternalServerError, "apply recovery failed: "+err.Error())
		return
	}
	delete(recoveryPreviews.Items, req.Token)
	api.OK(c, gin.H{"applied": applied, "skipped": len(skipped), "skipped_actions": skipped})
}

func buildRecoveryPreview(uid uint) (gin.H, error) {
	today := shanghaiToday()
	var overdue []models.DailyTask
	if err := db.DB.Where("user_id = ? AND date < ? AND status <> ?", uid, today, models.TaskStatusCompleted).Order("date ASC, sort_order ASC, id ASC").Find(&overdue).Error; err != nil {
		return nil, err
	}
	var plans []models.Plan
	if err := db.DB.Preload("ScheduleOverrides").Where("user_id = ?", uid).Find(&plans).Error; err != nil {
		return nil, err
	}
	planByID := map[uint]models.Plan{}
	for _, plan := range plans {
		planByID[plan.ID] = plan
	}
	actions := make([]recoveryAction, 0, len(overdue))
	proposed := make([]models.DailyTask, 0, len(overdue))
	cursor := shanghaiNow()
	for _, task := range overdue {
		plan, found := planByID[task.PlanID]
		next, foundDay := nextPlanStudyDay(plan, cursor.AddDate(0, 0, 1))
		if !found || !foundDay {
			actions = append(actions, recoveryAction{TaskID: task.ID, Title: task.Title, OldDate: task.Date, Reason: "plan has no future study day", Valid: false, Version: task.UpdatedAt.UnixNano()})
			continue
		}
		start, end := resolvePlanSchedule(plan, plan.ScheduleOverrides, next)
		candidate := task
		candidate.Date, candidate.PlannedStart, candidate.PlannedEnd = next.Format(dateLayout), start, end
		valid := false
		for attempts := 0; attempts < 60; attempts++ {
			trial := append(append([]models.DailyTask{}, proposed...), candidate)
			if validateScheduleMutation(db.DB, uid, trial) == nil {
				valid = true
				break
			}
			var ok bool
			next, ok = nextPlanStudyDay(plan, next.AddDate(0, 0, 1))
			if !ok {
				break
			}
			candidate.Date = next.Format(dateLayout)
			candidate.PlannedStart, candidate.PlannedEnd = resolvePlanSchedule(plan, plan.ScheduleOverrides, next)
		}
		if valid {
			proposed = append(proposed, candidate)
		}
		cursor = next
		reason := "overdue task moved to next plan study day"
		if !valid {
			reason = "no conflict-free future study day"
		}
		actions = append(actions, recoveryAction{TaskID: task.ID, Title: task.Title, OldDate: task.Date, NewDate: candidate.Date, PlannedStart: candidate.PlannedStart, PlannedEnd: candidate.PlannedEnd, Reason: reason, Valid: valid, Version: task.UpdatedAt.UnixNano()})
	}
	if err := validateScheduleMutation(db.DB, uid, proposed); err != nil {
		return nil, err
	}
	token, err := storeRecoveryPreview(uid, actions)
	if err != nil {
		return nil, err
	}
	mode := "rule"
	if cfg, err := firstAIConfig(); err == nil && cfg.Enabled {
		mode = "ai_fallback"
	}
	return gin.H{"mode": mode, "token": token, "version": 1, "overdue_tasks": len(overdue), "actions": actions}, nil
}

func nextPlanStudyDay(plan models.Plan, day time.Time) (time.Time, bool) {
	if len(plan.StudyDates) == 0 && len(plan.StudyWeekdays) == 0 {
		return day, true
	}
	for offset := 0; offset < 370; offset++ {
		candidate := day.AddDate(0, 0, offset)
		if planStudiesOn(plan, candidate) {
			return candidate, true
		}
	}
	return time.Time{}, false
}

func storeRecoveryPreview(uid uint, actions []recoveryAction) (string, error) {
	return storeRecoveryPreviewForMode(uid, actions, "recovery", 0, 0)
}

func storeRecoveryPreviewForMode(uid uint, actions []recoveryAction, mode string, planID uint, planDays int) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	rows := map[uint]recoveryAction{}
	for _, action := range actions {
		rows[action.TaskID] = action
	}
	recoveryPreviews.Lock()
	recoveryPreviews.Items[token] = recoverySnapshot{UserID: uid, Actions: rows, ExpiresAt: time.Now().Add(15 * time.Minute), Mode: mode, PlanID: planID, PlanDays: planDays}
	recoveryPreviews.Unlock()
	return token, nil
}

func recoveryPreviewLocked(token string, uid uint) (recoverySnapshot, bool) {
	snapshot, ok := recoveryPreviews.Items[token]
	if !ok || snapshot.UserID != uid || snapshot.ExpiresAt.Before(time.Now()) {
		if ok {
			delete(recoveryPreviews.Items, token)
		}
		return recoverySnapshot{}, false
	}
	return snapshot, true
}
