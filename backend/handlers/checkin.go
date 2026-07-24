package handlers

import (
	"errors"
	"net/http"
	"strconv"
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

type dailyCheckinReq struct {
	Date      string `json:"date" binding:"required"`
	Completed bool   `json:"completed"`
}

const dateLayout = "2006-01-02"

// ListCheckins 获取指定日期的打卡状态（返回该用户所有计划在 date 的完成情况）
func ListCheckins(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	date := c.Query("date")
	if date == "" {
		date = shanghaiToday()
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
		var daily models.DailyCheckin
		dailyErr := db.DB.Where("user_id = ? AND date = ? AND completed = ?", uid, date, true).First(&daily).Error
		if dailyErr != nil && !errors.Is(dailyErr, gorm.ErrRecordNotFound) {
			api.Fail(c, http.StatusInternalServerError, "query checkins failed: "+dailyErr.Error())
			return
		}
		var completedToday int64
		if err := db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND date = ? AND status = ?", uid, date, models.TaskStatusCompleted).Count(&completedToday).Error; err != nil {
			api.Fail(c, http.StatusInternalServerError, "query completed tasks failed: "+err.Error())
			return
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
				Completed:      dailyErr == nil,
				Eligible:       completedToday > 0,
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

	if req.Completed != nil && !*req.Completed {
		api.Fail(c, http.StatusBadRequest, "completed check-in cannot be reopened")
		return
	}
	finalizeDailyCheckin(c, uid, req.Date, req.PlanID)
}

func GetDailyCheckin(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	date := c.DefaultQuery("date", shanghaiToday())
	if _, err := time.Parse(dateLayout, date); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid date, expect YYYY-MM-DD")
		return
	}
	view, err := dailyCheckinView(db.DB, uid, date)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query daily checkin failed: "+err.Error())
		return
	}
	api.OK(c, view)
}

func CompleteDailyCheckin(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req dailyCheckinReq
	if err := c.ShouldBindJSON(&req); err != nil || !req.Completed {
		api.Fail(c, http.StatusBadRequest, "date and completed=true required")
		return
	}
	if _, err := time.Parse(dateLayout, req.Date); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid date, expect YYYY-MM-DD")
		return
	}
	finalizeDailyCheckin(c, uid, req.Date, 0)
}

func finalizeDailyCheckin(c *gin.Context, uid uint, date string, legacyPlanID uint) {
	var existing models.DailyCheckin
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if legacyPlanID != 0 {
			var plan models.Plan
			if err := tx.First(&plan, legacyPlanID).Error; err != nil {
				return err
			}
			if plan.UserID != uid {
				return errNotPlanOwner
			}
		}
		queryErr := tx.Where("user_id = ? AND date = ?", uid, date).First(&existing).Error
		if queryErr == nil && existing.Completed {
			return nil
		}
		if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return queryErr
		}
		var completed int64
		if err := tx.Model(&models.DailyTask{}).Where("user_id = ? AND date = ? AND status = ?", uid, date, models.TaskStatusCompleted).Count(&completed).Error; err != nil {
			return err
		}
		if completed == 0 {
			return errTasksRemaining
		}
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			existing = models.DailyCheckin{UserID: uid, Date: date, Completed: true}
			if err := tx.Create(&existing).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&existing).Update("completed", true).Error; err != nil {
				return err
			}
			existing.Completed = true
		}
		return awardDailySlackIfNeeded(tx, uid, &existing)
	})
	if err != nil && isUniqueConstraintError(err) {
		if queryErr := db.DB.Where("user_id = ? AND date = ? AND completed = ?", uid, date, true).First(&existing).Error; queryErr == nil {
			view, _ := dailyCheckinView(db.DB, uid, date)
			api.OK(c, view)
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
	if errors.Is(err, errTasksRemaining) {
		api.Fail(c, http.StatusBadRequest, "complete at least one task before daily check-in")
		return
	}
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "create checkin failed: "+err.Error())
		return
	}
	view, _ := dailyCheckinView(db.DB, uid, date)
	api.OK(c, view)
}

var (
	errNotPlanOwner      = errors.New("not plan owner")
	errDailyTaskNotFound = errors.New("daily task not found")
	errTasksRemaining    = errors.New("tasks remaining")
)

// Streak 连续打卡天数（MVP：连续多少天所有 active 计划都打满了卡）
func Streak(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	streak, todayQualified, err := consecutiveCheckins(db.DB, uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query consecutive checkins failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"streak": streak, "consecutive_checkin_days": streak, "today_qualified": todayQualified, "display_text": strconv.Itoa(streak) + " 天"})
}

func dailyCheckinView(tx *gorm.DB, uid uint, date string) (gin.H, error) {
	var completedCount int64
	if err := tx.Model(&models.DailyTask{}).Where("user_id = ? AND date = ? AND status = ?", uid, date, models.TaskStatusCompleted).Count(&completedCount).Error; err != nil {
		return nil, err
	}
	var row models.DailyCheckin
	err := tx.Where("user_id = ? AND date = ? AND completed = ?", uid, date, true).First(&row).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return gin.H{"date": date, "completed_task_count": completedCount, "eligible": completedCount > 0, "completed": err == nil, "rewarded": err == nil && row.Rewarded}, nil
}

func consecutiveCheckins(tx *gorm.DB, uid uint) (int, bool, error) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	today := time.Now().In(loc).Format(dateLayout)
	var dates []string
	if err := tx.Model(&models.DailyTask{}).Distinct("date").Where("user_id = ? AND date <= ?", uid, today).Order("date DESC").Pluck("date", &dates).Error; err != nil {
		return 0, false, err
	}
	checked := map[string]bool{}
	var rows []models.DailyCheckin
	if err := tx.Where("user_id = ? AND completed = ?", uid, true).Find(&rows).Error; err != nil {
		return 0, false, err
	}
	for _, row := range rows {
		checked[row.Date] = true
	}
	var completedDates []string
	if err := tx.Model(&models.DailyTask{}).Distinct("date").Where("user_id = ? AND status = ?", uid, models.TaskStatusCompleted).Pluck("date", &completedDates).Error; err != nil {
		return 0, false, err
	}
	completed := map[string]bool{}
	for _, date := range completedDates {
		completed[date] = true
	}
	todayQualified := checked[today] && completed[today]
	streak := 0
	for _, date := range dates {
		if date == today && (!checked[date] || !completed[date]) {
			continue
		}
		if !checked[date] || !completed[date] {
			break
		}
		streak++
	}
	return streak, todayQualified, nil
}

func shanghaiToday() string {
	return shanghaiNow().Format(dateLayout)
}

func shanghaiNow() time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Now()
	}
	return time.Now().In(loc)
}
