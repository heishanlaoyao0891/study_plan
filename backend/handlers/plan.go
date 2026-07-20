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

const (
	maxActivePlans      = 3
	maxWeeklyHoursTotal = 56
	errParamPlanID      = "invalid plan id"
)

type createPlanReq struct {
	Title             string `json:"title" binding:"required"`
	Description       string `json:"description"`
	WeeklyTargetHours int    `json:"weekly_target_hours"`
	StartDate         string `json:"start_date"`
	EndDate           string `json:"end_date"`
	ConfirmOverload   bool   `json:"confirm_overload"`
}

type updatePlanReq struct {
	Title             *string `json:"title"`
	Description       *string `json:"description"`
	WeeklyTargetHours *int    `json:"weekly_target_hours"`
	StartDate         *string `json:"start_date"`
	EndDate           *string `json:"end_date"`
	Status            *string `json:"status"`
}

type shiftPlanReq struct {
	Days      int    `json:"days" binding:"required"`
	StartDate string `json:"start_date"`
}

type invitePlanReq struct {
	UserID uint `json:"user_id" binding:"required"`
}

// ListPlans 当前用户的计划列表
func ListPlans(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	status := c.Query("status")

	var plans []models.Plan
	q := db.DB.Where("user_id = ?", uid)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	q = q.Order("sort_order ASC, id ASC")
	if err := q.Find(&plans).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query plans failed: "+err.Error())
		return
	}
	api.OK(c, plans)
}

// GetPlan 获取单个计划详情
func GetPlan(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	plan, err := mustGetOwnedPlan(c, uid)
	if err != nil {
		return
	}
	api.OK(c, plan)
}

// CreatePlan 创建学习计划（含超负荷校验）
func CreatePlan(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req createPlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	warnings, err := checkOverload(uid, req.WeeklyTargetHours, req.ConfirmOverload)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.StartDate != "" && req.EndDate != "" {
		if slotWarnings, slotErr := checkTaskSlotConflicts(uid, req.StartDate, req.EndDate, "20:00", "21:00"); slotErr == nil && len(slotWarnings) > 0 {
			warnings = append(warnings, slotWarnings...)
		} else if slotErr != nil {
			api.Fail(c, http.StatusBadRequest, slotErr.Error())
			return
		}
	}

	plan := models.Plan{
		UserID:            uid,
		Title:             req.Title,
		Description:       req.Description,
		WeeklyTargetHours: req.WeeklyTargetHours,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		Status:            models.PlanStatusActive,
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		return generateTasksForPlan(tx, uid, plan)
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "create plan failed: "+err.Error())
		return
	}
	api.Warn(c, plan, warnings)
}

// UpdatePlan 编辑计划
func UpdatePlan(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	plan, err := mustGetOwnedPlan(c, uid)
	if err != nil {
		return
	}
	var req updatePlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.WeeklyTargetHours != nil {
		updates["weekly_target_hours"] = *req.WeeklyTargetHours
	}
	if req.StartDate != nil {
		updates["start_date"] = *req.StartDate
	}
	if req.EndDate != nil {
		updates["end_date"] = *req.EndDate
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if len(updates) == 0 {
		api.OK(c, plan)
		return
	}
	if err := db.DB.Model(&plan).Updates(updates).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "update plan failed: "+err.Error())
		return
	}
	// 返回最新数据
	db.DB.First(&plan, plan.ID)
	api.OK(c, plan)
}

// DeletePlan 删除计划（连带删除关联打卡记录）
func DeletePlan(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	plan, err := mustGetOwnedPlan(c, uid)
	if err != nil {
		return
	}
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("plan_id = ?", plan.ID).Delete(&models.Checkin{}).Error; e != nil {
			return e
		}
		var taskIDs []uint
		if e := tx.Model(&models.DailyTask{}).Where("plan_id = ?", plan.ID).Pluck("id", &taskIDs).Error; e != nil {
			return e
		}
		if len(taskIDs) > 0 {
			if e := tx.Where("task_id IN ?", taskIDs).Delete(&models.StudySession{}).Error; e != nil {
				return e
			}
			if e := tx.Where("task_id IN ?", taskIDs).Delete(&models.PostponeRecord{}).Error; e != nil {
				return e
			}
		}
		if e := tx.Where("plan_id = ?", plan.ID).Delete(&models.DailyTask{}).Error; e != nil {
			return e
		}
		return tx.Delete(&plan).Error
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "delete plan failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"deleted": plan.ID})
}

// PausePlan 暂停计划
func PausePlan(c *gin.Context) {
	changePlanStatus(c, models.PlanStatusPaused)
}

// ResumePlan 恢复计划
func ResumePlan(c *gin.Context) {
	changePlanStatus(c, models.PlanStatusActive)
}

func ShiftPlan(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	plan, err := mustGetOwnedPlan(c, uid)
	if err != nil {
		return
	}
	var req shiftPlanReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Days == 0 {
		api.Fail(c, http.StatusBadRequest, "invalid request: days required")
		return
	}
	startDate := req.StartDate
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, 1).Format(dateLayout)
	}
	if _, err := time.Parse(dateLayout, startDate); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid start_date, expect YYYY-MM-DD")
		return
	}
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		return shiftPlanTasks(tx, uid, plan, req.Days, startDate)
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "shift plan failed: "+err.Error())
		return
	}
	api.OK(c, plan)
}

func InvitePlanMember(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	plan, err := mustGetOwnedPlan(c, uid)
	if err != nil {
		return
	}
	var req invitePlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.UserID == uid {
		api.Fail(c, http.StatusBadRequest, "cannot invite yourself")
		return
	}
	var user models.User
	if err := db.DB.First(&user, req.UserID).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "user not found")
		return
	}
	now := time.Now()
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(plan).Update("is_shared", true).Error; e != nil {
			return e
		}
		owner := models.PlanMember{PlanID: plan.ID, UserID: uid, Role: "owner", JoinedAt: now}
		_ = tx.Where("plan_id = ? AND user_id = ?", plan.ID, uid).FirstOrCreate(&owner).Error
		member := models.PlanMember{PlanID: plan.ID, UserID: req.UserID, Role: "member", JoinedAt: now}
		return tx.Where("plan_id = ? AND user_id = ?", plan.ID, req.UserID).FirstOrCreate(&member).Error
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "invite failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"plan_id": plan.ID, "user_id": req.UserID})
}

func JoinPlan(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	pid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, errParamPlanID)
		return
	}
	now := time.Now()
	member := models.PlanMember{PlanID: uint(pid), UserID: uid, Role: "member", JoinedAt: now}
	if err := db.DB.Where("plan_id = ? AND user_id = ?", pid, uid).FirstOrCreate(&member).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "join failed: "+err.Error())
		return
	}
	api.OK(c, member)
}

func changePlanStatus(c *gin.Context, status string) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	plan, err := mustGetOwnedPlan(c, uid)
	if err != nil {
		return
	}
	if err := db.DB.Model(&plan).Update("status", status).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "update status failed: "+err.Error())
		return
	}
	plan.Status = status
	api.OK(c, plan)
}

// checkOverload 校验是否超负荷。返回 warnings 空切片表示无警告。
// 若有警告但未 confirm_overload，则返回错误阻止创建。
func checkOverload(uid uint, newHours int, confirmed bool) ([]string, error) {
	var activePlans []models.Plan
	if err := db.DB.Where("user_id = ? AND status = ?", uid, models.PlanStatusActive).Find(&activePlans).Error; err != nil {
		return nil, err
	}

	var warnings []string
	// 1) 活跃计划数量
	if len(activePlans) >= maxActivePlans {
		warnings = append(warnings, "已有 "+itoa(len(activePlans))+" 个活跃计划，不建议同时进行过多学习计划")
	}
	// 2) 每周总学时
	total := newHours
	for _, p := range activePlans {
		total += p.WeeklyTargetHours
	}
	if total > maxWeeklyHoursTotal {
		warnings = append(warnings, "所有计划每周总学时已达 "+itoa(total)+" 小时，压力可能过大")
	}

	if len(warnings) > 0 && !confirmed {
		return warnings, errors.New("overload detected, confirm_overload required")
	}
	return warnings, nil
}

func checkTaskSlotConflicts(uid uint, startDate, endDate, plannedStart, plannedEnd string) ([]string, error) {
	start, err := time.Parse(dateLayout, startDate)
	if err != nil {
		return nil, errors.New("invalid start_date, expect YYYY-MM-DD")
	}
	end, err := time.Parse(dateLayout, endDate)
	if err != nil {
		return nil, errors.New("invalid end_date, expect YYYY-MM-DD")
	}
	if end.Before(start) {
		return nil, errors.New("end_date cannot be earlier than start_date")
	}
	var tasks []models.DailyTask
	if err := db.DB.Where(
		"user_id = ? AND date >= ? AND date <= ? AND planned_start < ? AND planned_end > ?",
		uid, startDate, endDate, plannedEnd, plannedStart,
	).Find(&tasks).Error; err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	warnings := make([]string, 0, len(tasks))
	for _, task := range tasks {
		warnings = append(warnings, "任务 "+task.Date+" "+task.Title+" 的计划时段与新计划默认时段冲突")
	}
	return warnings, nil
}

func shiftPlanTasks(tx *gorm.DB, uid uint, plan *models.Plan, days int, startDate string) error {
	if plan.StartDate != "" {
		if t, e := time.Parse(dateLayout, plan.StartDate); e == nil {
			plan.StartDate = t.AddDate(0, 0, days).Format(dateLayout)
		}
	}
	if plan.EndDate != "" {
		if t, e := time.Parse(dateLayout, plan.EndDate); e == nil {
			plan.EndDate = t.AddDate(0, 0, days).Format(dateLayout)
		}
	}
	if e := tx.Save(plan).Error; e != nil {
		return e
	}
	return tx.Exec("UPDATE daily_tasks SET date = date(date, ? || ' day') WHERE plan_id = ? AND user_id = ? AND status <> ? AND date >= ?", days, plan.ID, uid, models.TaskStatusCompleted, startDate).Error
}

// mustGetOwnedPlan 取回路径参数对应的计划，并校验归属当前用户
func mustGetOwnedPlan(c *gin.Context, uid uint) (*models.Plan, error) {
	pid, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, errParamPlanID)
		return nil, err
	}
	var plan models.Plan
	if err := db.DB.First(&plan, pid).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "plan not found")
		return nil, err
	}
	if plan.UserID != uid {
		api.Fail(c, http.StatusForbidden, "not your plan")
		return nil, errors.New("not owner")
	}
	return &plan, nil
}

func itoa(n int) string { return strconv.Itoa(n) }

func generateTasksForPlan(tx *gorm.DB, uid uint, plan models.Plan) error {
	if plan.StartDate == "" || plan.EndDate == "" {
		return nil
	}
	start, err := time.Parse(dateLayout, plan.StartDate)
	if err != nil {
		return nil
	}
	end, err := time.Parse(dateLayout, plan.EndDate)
	if err != nil || end.Before(start) {
		return nil
	}
	order := 1
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		task := models.DailyTask{
			UserID:           uid,
			PlanID:           plan.ID,
			Date:             d.Format(dateLayout),
			Title:            plan.Title,
			Description:      plan.Description,
			PlannedStart:     "20:00",
			PlannedEnd:       "21:00",
			EstimatedMinutes: 60,
			Difficulty:       "medium",
			Status:           models.TaskStatusPending,
			SortOrder:        order,
		}
		if err := tx.Where("user_id = ? AND plan_id = ? AND date = ?", uid, plan.ID, task.Date).FirstOrCreate(&task).Error; err != nil {
			return err
		}
		order++
	}
	return nil
}
