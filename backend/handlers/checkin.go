package handlers

import (
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
	Completed *bool  `json:"completed"`              // nil=toggle, true=打勾, false=取消
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
		PlanID    uint   `json:"plan_id"`
		Title     string `json:"title"`
		Status    string `json:"status"`
		Date      string `json:"date"`
		Completed bool   `json:"completed"`
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
			out = append(out, checkinfo{
				PlanID:    p.ID,
				Title:     p.Title,
				Status:    p.Status,
				Date:      date,
				Completed: checked[p.ID],
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

	// 校验收计划归属
	var plan models.Plan
	if err := db.DB.First(&plan, req.PlanID).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "plan not found")
		return
	}
	if plan.UserID != uid {
		api.Fail(c, http.StatusForbidden, "not your plan")
		return
	}

	// 查询现有记录
	var existing models.Checkin
	err := db.DB.Where("user_id = ? AND plan_id = ? AND date = ?", uid, req.PlanID, req.Date).First(&existing).Error

	newVal := true
	if req.Completed != nil {
		newVal = *req.Completed
	}

	if err == gorm.ErrRecordNotFound {
		existing = models.Checkin{
			UserID:    uid,
			PlanID:    req.PlanID,
			Date:      req.Date,
			Completed: newVal,
		}
		if e := db.DB.Create(&existing).Error; e != nil {
			api.Fail(c, http.StatusInternalServerError, "create checkin failed: "+e.Error())
			return
		}
	} else if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query checkin failed: "+err.Error())
		return
	} else {
		// toggle or set
		if req.Completed == nil {
			newVal = !existing.Completed
		}
		if e := db.DB.Model(&existing).Update("completed", newVal).Error; e != nil {
			api.Fail(c, http.StatusInternalServerError, "update checkin failed: "+e.Error())
			return
		}
		existing.Completed = newVal
	}
	api.OK(c, existing)
}

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