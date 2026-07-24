package handlers

import (
	"errors"
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

const (
	maxActivePlans      = 3
	maxWeeklyHoursTotal = 56
	errParamPlanID      = "invalid plan id"
)

type createPlanReq struct {
	Title               string                `json:"title" binding:"required"`
	Description         string                `json:"description"`
	WeeklyTargetHours   int                   `json:"weekly_target_hours"`
	StartDate           string                `json:"start_date"`
	EndDate             string                `json:"end_date"`
	PublicToGroup       bool                  `json:"public_to_group"`
	ConfirmOverload     bool                  `json:"confirm_overload"`
	Objective           string                `json:"objective"`
	DefaultPlannedStart string                `json:"default_planned_start"`
	DefaultPlannedEnd   string                `json:"default_planned_end"`
	StudyWeekdays       []int                 `json:"study_weekdays"`
	StudyDates          []string              `json:"study_dates"`
	ScheduleOverrides   []scheduleOverrideReq `json:"schedule_overrides"`
}

type scheduleOverrideReq struct {
	Weekday      int    `json:"weekday"`
	Date         string `json:"date"`
	PlannedStart string `json:"planned_start"`
	PlannedEnd   string `json:"planned_end"`
}

type updatePlanReq struct {
	Title               *string                `json:"title"`
	Description         *string                `json:"description"`
	WeeklyTargetHours   *int                   `json:"weekly_target_hours"`
	StartDate           *string                `json:"start_date"`
	EndDate             *string                `json:"end_date"`
	Status              *string                `json:"status"`
	PublicToGroup       *bool                  `json:"public_to_group"`
	DefaultPlannedStart *string                `json:"default_planned_start"`
	DefaultPlannedEnd   *string                `json:"default_planned_end"`
	StudyWeekdays       *[]int                 `json:"study_weekdays"`
	StudyDates          *[]string              `json:"study_dates"`
	ScheduleOverrides   *[]scheduleOverrideReq `json:"schedule_overrides"`
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
	q := db.DB.Preload("ScheduleOverrides").Where("user_id = ?", uid)
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
	db.DB.Preload("ScheduleOverrides").First(plan, plan.ID)
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
		if slotWarnings, slotErr := checkTaskSlotConflicts(uid, req.StartDate, req.EndDate, defaultString(req.DefaultPlannedStart, defaultPlannedStart()), defaultString(req.DefaultPlannedEnd, defaultPlannedEnd())); slotErr == nil && len(slotWarnings) > 0 {
			warnings = append(warnings, slotWarnings...)
		} else if slotErr != nil {
			api.Fail(c, http.StatusBadRequest, slotErr.Error())
			return
		}
	}
	if err := validatePlanSchedule(req.DefaultPlannedStart, req.DefaultPlannedEnd, req.StudyWeekdays, req.StudyDates, req.ScheduleOverrides); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validatePlanDates(req.StartDate, req.EndDate, req.StudyDates, req.ScheduleOverrides); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.StudyWeekdays) > 0 || len(req.StudyDates) > 0 {
		if err := validateObjective(req.Title, req.Objective); err != nil {
			api.Fail(c, http.StatusBadRequest, err.Error())
			return
		}
	}

	plan := models.Plan{
		UserID:              uid,
		Title:               req.Title,
		Description:         req.Description,
		WeeklyTargetHours:   req.WeeklyTargetHours,
		StartDate:           req.StartDate,
		EndDate:             req.EndDate,
		PublicToGroup:       req.PublicToGroup,
		Status:              models.PlanStatusActive,
		DefaultPlannedStart: defaultString(req.DefaultPlannedStart, defaultPlannedStart()),
		DefaultPlannedEnd:   defaultString(req.DefaultPlannedEnd, defaultPlannedEnd()),
		StudyWeekdays:       req.StudyWeekdays,
		StudyDates:          req.StudyDates,
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		if err := replaceScheduleOverrides(tx, plan.ID, req.ScheduleOverrides); err != nil {
			return err
		}
		return generateTasksForPlan(tx, uid, plan, strings.TrimSpace(req.Objective))
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "create plan failed: "+err.Error())
		return
	}
	if err := db.DB.Preload("ScheduleOverrides").First(&plan, plan.ID).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query created plan failed: "+err.Error())
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
	if req.PublicToGroup != nil {
		updates["public_to_group"] = *req.PublicToGroup
	}
	if req.DefaultPlannedStart != nil {
		updates["default_planned_start"] = *req.DefaultPlannedStart
	}
	if req.DefaultPlannedEnd != nil {
		updates["default_planned_end"] = *req.DefaultPlannedEnd
	}
	if req.StudyWeekdays != nil {
		updates["study_weekdays"] = *req.StudyWeekdays
	}
	if req.StudyDates != nil {
		updates["study_dates"] = *req.StudyDates
	}
	start, end := plan.DefaultPlannedStart, plan.DefaultPlannedEnd
	if req.DefaultPlannedStart != nil {
		start = *req.DefaultPlannedStart
	}
	if req.DefaultPlannedEnd != nil {
		end = *req.DefaultPlannedEnd
	}
	weekdays, dates := plan.StudyWeekdays, plan.StudyDates
	if req.StudyWeekdays != nil {
		weekdays = *req.StudyWeekdays
	}
	if req.StudyDates != nil {
		dates = *req.StudyDates
	}
	overrides := make([]scheduleOverrideReq, 0, len(plan.ScheduleOverrides))
	for _, row := range plan.ScheduleOverrides {
		overrides = append(overrides, scheduleOverrideReq{Weekday: row.Weekday, Date: row.Date, PlannedStart: row.PlannedStart, PlannedEnd: row.PlannedEnd})
	}
	if req.ScheduleOverrides != nil {
		overrides = *req.ScheduleOverrides
	}
	if err := validatePlanSchedule(start, end, weekdays, dates, overrides); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	startDate, endDate := plan.StartDate, plan.EndDate
	if req.StartDate != nil {
		startDate = *req.StartDate
	}
	if req.EndDate != nil {
		endDate = *req.EndDate
	}
	if err := validatePlanDates(startDate, endDate, dates, overrides); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(updates) == 0 && req.ScheduleOverrides == nil {
		api.OK(c, plan)
		return
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&plan).Updates(updates).Error; err != nil {
			return err
		}
		if req.ScheduleOverrides != nil {
			return replaceScheduleOverrides(tx, plan.ID, *req.ScheduleOverrides)
		}
		return nil
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "update plan failed: "+err.Error())
		return
	}
	// 返回最新数据
	db.DB.Preload("ScheduleOverrides").First(&plan, plan.ID)
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
		if e := tx.Where("plan_id = ?", plan.ID).Delete(&models.PlanScheduleOverride{}).Error; e != nil {
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
	if err := db.DB.Preload("ScheduleOverrides").First(&plan, pid).Error; err != nil {
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

func generateTasksForPlan(tx *gorm.DB, uid uint, plan models.Plan, objective string) error {
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
	var overrides []models.PlanScheduleOverride
	if err := tx.Where("plan_id = ?", plan.ID).Find(&overrides).Error; err != nil {
		return err
	}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		date := d.Format(dateLayout)
		if !planStudiesOn(plan, d) {
			continue
		}
		plannedStart, plannedEnd := resolvePlanSchedule(plan, overrides, d)
		task := models.DailyTask{
			UserID:           uid,
			PlanID:           plan.ID,
			Date:             date,
			Title:            plan.Title,
			Description:      plan.Description,
			Objective:        objective,
			PlannedStart:     plannedStart,
			PlannedEnd:       plannedEnd,
			EstimatedMinutes: plannedRangeMinutes(plannedStart, plannedEnd),
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

func validatePlanSchedule(start, end string, weekdays []int, dates []string, overrides []scheduleOverrideReq) error {
	if start == "" {
		start = defaultPlannedStart()
	}
	if end == "" {
		end = defaultPlannedEnd()
	}
	if plannedRangeMinutes(start, end) <= 0 {
		return errors.New("default planned time range is invalid")
	}
	seenWeekdays := map[int]bool{}
	for _, day := range weekdays {
		if day < 1 || day > 7 || seenWeekdays[day] {
			return errors.New("study_weekdays must contain unique ISO weekdays 1..7")
		}
		seenWeekdays[day] = true
	}
	seenDates := map[string]bool{}
	for _, date := range dates {
		if _, err := time.Parse(dateLayout, date); err != nil {
			return errors.New("study_dates must use YYYY-MM-DD")
		}
		if seenDates[date] {
			return errors.New("study_dates must be unique")
		}
		seenDates[date] = true
	}
	seenOverrideWeekdays, seenOverrideDates := map[int]bool{}, map[string]bool{}
	for _, row := range overrides {
		if (row.Weekday == 0) == (row.Date == "") {
			return errors.New("schedule override requires exactly one weekday or date")
		}
		if row.Weekday != 0 {
			if row.Weekday < 1 || row.Weekday > 7 || seenOverrideWeekdays[row.Weekday] {
				return errors.New("override weekdays must be unique ISO weekdays 1..7")
			}
			seenOverrideWeekdays[row.Weekday] = true
		}
		if row.Date != "" {
			if _, err := time.Parse(dateLayout, row.Date); err != nil {
				return errors.New("override date must use YYYY-MM-DD")
			}
			if seenOverrideDates[row.Date] {
				return errors.New("override dates must be unique")
			}
			seenOverrideDates[row.Date] = true
		}
		if plannedRangeMinutes(row.PlannedStart, row.PlannedEnd) <= 0 {
			return errors.New("schedule override time range is invalid")
		}
	}
	return nil
}

func validatePlanDates(startDate, endDate string, studyDates []string, overrides []scheduleOverrideReq) error {
	if (startDate == "") != (endDate == "") {
		return errors.New("start_date and end_date must be provided together")
	}
	if startDate == "" {
		return nil
	}
	start, err := time.Parse(dateLayout, startDate)
	if err != nil {
		return errors.New("invalid start_date, expect YYYY-MM-DD")
	}
	end, err := time.Parse(dateLayout, endDate)
	if err != nil {
		return errors.New("invalid end_date, expect YYYY-MM-DD")
	}
	if end.Before(start) {
		return errors.New("end_date cannot be earlier than start_date")
	}
	for _, value := range studyDates {
		date, _ := time.Parse(dateLayout, value)
		if date.Before(start) || date.After(end) {
			return errors.New("study_dates must be within the plan date range")
		}
	}
	for _, row := range overrides {
		if row.Date == "" {
			continue
		}
		date, _ := time.Parse(dateLayout, row.Date)
		if date.Before(start) || date.After(end) {
			return errors.New("override dates must be within the plan date range")
		}
	}
	return nil
}

func replaceScheduleOverrides(tx *gorm.DB, planID uint, rows []scheduleOverrideReq) error {
	if err := tx.Where("plan_id = ?", planID).Delete(&models.PlanScheduleOverride{}).Error; err != nil {
		return err
	}
	for _, row := range rows {
		override := models.PlanScheduleOverride{PlanID: planID, Weekday: row.Weekday, Date: row.Date, PlannedStart: row.PlannedStart, PlannedEnd: row.PlannedEnd}
		if err := tx.Create(&override).Error; err != nil {
			return err
		}
	}
	return nil
}

func planStudiesOn(plan models.Plan, day time.Time) bool {
	date := day.Format(dateLayout)
	for _, selected := range plan.StudyDates {
		if selected == date {
			return true
		}
	}
	weekday := int(day.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	for _, selected := range plan.StudyWeekdays {
		if selected == weekday {
			return true
		}
	}
	return false
}

func resolvePlanSchedule(plan models.Plan, overrides []models.PlanScheduleOverride, day time.Time) (string, string) {
	date := day.Format(dateLayout)
	weekday := int(day.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	for _, row := range overrides {
		if row.Date == date {
			return row.PlannedStart, row.PlannedEnd
		}
	}
	for _, row := range overrides {
		if row.Date == "" && row.Weekday == weekday {
			return row.PlannedStart, row.PlannedEnd
		}
	}
	return defaultString(plan.DefaultPlannedStart, defaultPlannedStart()), defaultString(plan.DefaultPlannedEnd, defaultPlannedEnd())
}

func plannedRangeMinutes(start, end string) int {
	s, err1 := time.Parse("15:04", strings.TrimSpace(start))
	e, err2 := time.Parse("15:04", strings.TrimSpace(end))
	if err1 != nil || err2 != nil || !e.After(s) {
		return 0
	}
	return int(e.Sub(s).Minutes())
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
