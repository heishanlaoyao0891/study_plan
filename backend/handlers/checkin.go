package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

type checkinReq struct {
	PlanID    uint   `json:"plan_id" binding:"required"`
	Date      string `json:"date" binding:"required"` // YYYY-MM-DD
	Completed *bool  `json:"completed"`               // nil=toggle, true=打勾, false=取消
}

const dateLayout = "2006-01-02"

// ListCheckins 获取指定日期的打卡状态（返回该用户所有计划在 date 的完成情况）
func ListCheckins(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format(dateLayout)
	}
	if _, err := time.Parse(dateLayout, date); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid date, expect YYYY-MM-DD")
		return
	}

	// 拉取用户所有计划
	var plans []models.Plan
	if err := db.DB.Where("user_id = ?", uid).Order("sort_order ASC, id ASC").Find(&plans).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query plans failed: "+err.Error())
		return
	}

	planIDs := make([]uint, 0, len(plans))
	for _, p := range plans {
		planIDs = append(planIDs, p.ID)
	}
	type checkinfo struct {
		PlanID         uint           `json:"plan_id"`
		TaskID         uint           `json:"task_id"`
		Title          string         `json:"title"`
		Status         string         `json:"status"`
		TaskStatus     string         `json:"task_status"`
		Date           string         `json:"date"`
		StudyMinutes   int            `json:"study_minutes"`
		Completed      bool           `json:"completed"`
		Eligible       bool           `json:"eligible"`
		RemainingTasks int            `json:"remaining_tasks"`
		Task           *taskTimerView `json:"task,omitempty"`
	}
	out := make([]checkinfo, 0, len(plans))

	if len(planIDs) > 0 {
		var rows []models.Checkin
		if err := db.DB.Where("user_id = ? AND date = ? AND plan_id IN ?", uid, date, planIDs).Find(&rows).Error; err != nil {
			api.Fail(c, http.StatusInternalServerError, "query checkins failed: "+err.Error())
			return
		}
		checked := map[uint]bool{}
		for _, r := range rows {
			if r.Completed {
				checked[r.PlanID] = true
			}
		}
		for _, p := range plans {
			var tasks []models.DailyTask
			err := db.DB.Where("user_id = ? AND plan_id = ? AND date = ?", uid, p.ID, date).Order("sort_order ASC, id ASC").Find(&tasks).Error
			if err != nil {
				api.Fail(c, http.StatusInternalServerError, "query daily tasks failed: "+err.Error())
				return
			}
			if len(tasks) == 0 {
				task, ensureErr := ensureDailyTask(uid, p, date)
				if ensureErr == nil {
					tasks = append(tasks, task)
				} else if !errors.Is(ensureErr, gorm.ErrRecordNotFound) {
					api.Fail(c, http.StatusInternalServerError, "ensure daily task failed: "+ensureErr.Error())
					return
				}
			}
			if len(tasks) == 0 {
				continue
			}
			remaining := 0
			studyMinutes := 0
			for _, task := range tasks {
				studyMinutes += task.StudyMinutes
				if task.Status != models.TaskStatusCompleted {
					remaining++
				}
			}
			view, viewErr := buildTaskTimerView(tasks[0], time.Now())
			if viewErr != nil {
				api.Fail(c, http.StatusInternalServerError, viewErr.Error())
				return
			}
			out = append(out, checkinfo{
				PlanID:         p.ID,
				TaskID:         tasks[0].ID,
				Title:          p.Title,
				Status:         p.Status,
				TaskStatus:     tasks[0].Status,
				Date:           date,
				StudyMinutes:   studyMinutes,
				Completed:      checked[p.ID],
				Eligible:       remaining == 0,
				RemainingTasks: remaining,
				Task:           &view,
			})
		}
	}
	api.OK(c, out)
}

// ToggleCheckin 切换打卡状态
func ToggleCheckin(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req checkinReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if _, err := time.Parse(dateLayout, req.Date); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid date, expect YYYY-MM-DD")
		return
	}

	newVal := true
	if req.Completed != nil {
		newVal = *req.Completed
	}
	if !newVal {
		api.Fail(c, http.StatusBadRequest, "completed check-in cannot be reopened")
		return
	}
	var existing models.Checkin
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		var plan models.Plan
		if err := tx.First(&plan, req.PlanID).Error; err != nil {
			return err
		}
		if plan.UserID != uid {
			return errNotPlanOwner
		}
		queryErr := tx.Where("user_id = ? AND plan_id = ? AND date = ?", uid, req.PlanID, req.Date).First(&existing).Error
		if queryErr == nil && existing.Completed {
			return nil
		}
		if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return queryErr
		}
		var total, remaining int64
		if err := tx.Model(&models.DailyTask{}).Where("user_id = ? AND plan_id = ? AND date = ?", uid, req.PlanID, req.Date).Count(&total).Error; err != nil {
			return err
		}
		if total == 0 {
			return errDailyTaskNotFound
		}
		if err := tx.Model(&models.DailyTask{}).Where("user_id = ? AND plan_id = ? AND date = ? AND status <> ?", uid, req.PlanID, req.Date, models.TaskStatusCompleted).Count(&remaining).Error; err != nil {
			return err
		}
		if remaining > 0 {
			return fmt.Errorf("%w: %d", errTasksRemaining, remaining)
		}
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			existing = models.Checkin{UserID: uid, PlanID: req.PlanID, Date: req.Date, Completed: true}
			if err := tx.Create(&existing).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&existing).Update("completed", true).Error; err != nil {
				return err
			}
			existing.Completed = true
		}
		return awardSlackIfNeeded(tx, uid, &existing)
	})
	if err != nil && isUniqueConstraintError(err) {
		if queryErr := db.DB.Where("user_id = ? AND plan_id = ? AND date = ? AND completed = ?", uid, req.PlanID, req.Date, true).First(&existing).Error; queryErr == nil {
			api.OK(c, existing)
			return
		}
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		api.Fail(c, http.StatusNotFound, "plan not found")
		return
	}
	if errors.Is(err, errNotPlanOwner) {
		api.Fail(c, http.StatusForbidden, "not your plan")
		return
	}
	if errors.Is(err, errDailyTaskNotFound) {
		api.Fail(c, http.StatusBadRequest, "daily task not found")
		return
	}
	if errors.Is(err, errTasksRemaining) {
		api.Fail(c, http.StatusBadRequest, "complete today's tasks before check-in; remaining_tasks="+strings.TrimPrefix(err.Error(), errTasksRemaining.Error()+": "))
		return
	}
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "create checkin failed: "+err.Error())
		return
	}
	api.OK(c, existing)
}

var (
	errNotPlanOwner      = errors.New("not plan owner")
	errDailyTaskNotFound = errors.New("daily task not found")
	errTasksRemaining    = errors.New("tasks remaining")
)

// Streak 连续打卡天数（MVP：连续多少天所有 active 计划都打满了卡）
func Streak(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)

	var plans []models.Plan
	if err := db.DB.Where("user_id = ? AND status = ?", uid, models.PlanStatusActive).Find(&plans).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query plans failed: "+err.Error())
		return
	}
	if len(plans) == 0 {
		api.OK(c, gin.H{"streak": 0})
		return
	}
	planIDs := make([]uint, 0, len(plans))
	for _, p := range plans {
		planIDs = append(planIDs, p.ID)
	}

	streak := 0
	date := time.Now()
	for {
		dateStr := date.Format(dateLayout)
		var cnt int64
		db.DB.Model(&models.Checkin{}).
			Where("user_id = ? AND date = ? AND plan_id IN ? AND completed = ?", uid, dateStr, planIDs, true).
			Count(&cnt)
		if int(cnt) < len(planIDs) {
			break
		}
		streak++
		date = date.AddDate(0, 0, -1)
		// 安全上限，避免历史数据太多时无限循环
		if streak > 366 {
			break
		}
	}
	api.OK(c, gin.H{"streak": streak, "streak_str": strconv.Itoa(streak) + " 天"})
}
