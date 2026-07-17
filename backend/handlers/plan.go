package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

const (
	maxActivePlans       = 3
	maxWeeklyHoursTotal  = 56
	errParamPlanID       = "invalid plan id"
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

	plan := models.Plan{
		UserID:            uid,
		Title:             req.Title,
		Description:       req.Description,
		WeeklyTargetHours: req.WeeklyTargetHours,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		Status:            models.PlanStatusActive,
	}
	if err := db.DB.Create(&plan).Error; err != nil {
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