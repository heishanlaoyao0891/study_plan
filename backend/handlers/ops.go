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

var defaultOpsContent = map[string]models.OpsContent{
	"privacy":      {Kind: "privacy", Title: "隐私政策", Body: "我们仅为账号登录、计划、打卡、提醒、统计和小组协作处理必要数据。用户名用于账号识别，微信登录标识用于关联小程序账号；学习记录用于提供计划、统计和协作功能。AI 计划仅提供可编辑建议。"},
	"agreement":    {Kind: "agreement", Title: "用户协议", Body: "请合理安排学习和休息。本服务不承诺任何考试、成绩或技能结果。躺平时间为应用内休息记录额度，不具有现金或交易价值。"},
	"announcement": {Kind: "announcement", Title: "公告", Body: "暂无公告。"},
	"version":      {Kind: "version", Title: "版本说明", Body: "当前版本提供计划、任务、打卡、提醒、小组和恢复安排功能。"},
}

const legacyPhonePrivacyBody = "我们仅为登录、计划、打卡、提醒和统计功能处理必要数据。手机号用于账号识别和安全校验。AI 计划仅提供可编辑建议。"

type feedbackReq struct {
	Category string `json:"category"`
	Content  string `json:"content" binding:"required"`
	Contact  string `json:"contact"`
}

type feedbackUpdateReq struct {
	Status              string  `json:"status"`
	PublicResponse      *string `json:"public_response"`
	ClearPublicResponse bool    `json:"clear_public_response"`
}

type feedbackPublicResp struct {
	ID             uint       `json:"id"`
	Category       string     `json:"category"`
	Content        string     `json:"content"`
	Status         string     `json:"status"`
	PublicResponse *string    `json:"public_response,omitempty"`
	RespondedAt    *time.Time `json:"responded_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type feedbackAdminResp struct {
	models.FeedbackReport
	UserNickname string `json:"user_nickname"`
}

const (
	feedbackContentMax  = 1000
	feedbackContactMax  = 100
	feedbackResponseMax = 1000
	feedbackRateLimit   = 3
)

var feedbackCategories = map[string]bool{"issue": true, "suggestion": true, "content": true, "account": true, "other": true}
var feedbackStatuses = map[string]bool{"open": true, "processing": true, "resolved": true, "closed": true}

type opsContentReq struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func GetOpsContent(c *gin.Context) {
	kind := strings.TrimSpace(c.Param("kind"))
	content, err := firstOpsContent(kind)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query content failed: "+err.Error())
		return
	}
	api.OK(c, content)
}

func SubmitFeedback(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req feedbackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	category := strings.TrimSpace(req.Category)
	if !feedbackCategories[category] {
		api.Fail(c, http.StatusBadRequest, "category must be one of issue, suggestion, content, account, other")
		return
	}
	report := models.FeedbackReport{UserID: uid, Category: category, Content: strings.TrimSpace(req.Content), Contact: strings.TrimSpace(req.Contact), Status: "open"}
	if report.Content == "" {
		api.Fail(c, http.StatusBadRequest, "content required")
		return
	}
	if len([]rune(report.Content)) > feedbackContentMax {
		api.Fail(c, http.StatusBadRequest, "content must be at most 1000 characters")
		return
	}
	if len([]rune(report.Contact)) > feedbackContactMax {
		api.Fail(c, http.StatusBadRequest, "contact must be at most 100 characters")
		return
	}
	var recent int64
	if err := db.DB.Model(&models.FeedbackReport{}).Where("user_id = ? AND created_at >= ?", uid, time.Now().Add(-10*time.Minute)).Count(&recent).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "check feedback limit failed: "+err.Error())
		return
	}
	if recent >= feedbackRateLimit {
		api.Fail(c, http.StatusTooManyRequests, "too many feedback reports; try again later")
		return
	}
	if err := db.DB.Create(&report).Error; err != nil {
		if strings.Contains(err.Error(), "feedback rate limit exceeded") {
			api.Fail(c, http.StatusTooManyRequests, "too many feedback reports; try again later")
			return
		}
		api.Fail(c, http.StatusInternalServerError, "submit feedback failed: "+err.Error())
		return
	}
	api.OK(c, publicFeedback(report))
}

func ListOwnFeedback(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var rows []models.FeedbackReport
	if err := db.DB.Where("user_id = ?", uid).Order("id DESC").Limit(100).Find(&rows).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query feedback failed: "+err.Error())
		return
	}
	result := make([]feedbackPublicResp, 0, len(rows))
	for _, row := range rows {
		result = append(result, publicFeedback(row))
	}
	api.OK(c, result)
}

func AdminListOpsContents(c *gin.Context) {
	for kind := range defaultOpsContent {
		_, _ = firstOpsContent(kind)
	}
	var rows []models.OpsContent
	if err := db.DB.Order("kind ASC").Find(&rows).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query contents failed: "+err.Error())
		return
	}
	api.OK(c, rows)
}

func AdminSaveOpsContent(c *gin.Context) {
	adminID := c.GetUint(middleware.CtxUserIDKey)
	kind := strings.TrimSpace(c.Param("kind"))
	var req opsContentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	content, err := firstOpsContent(kind)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query content failed: "+err.Error())
		return
	}
	content.Title = strings.TrimSpace(req.Title)
	content.Body = strings.TrimSpace(req.Body)
	content.UpdatedBy = &adminID
	if content.Title == "" {
		content.Title = kind
	}
	if err := db.DB.Save(&content).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "save content failed: "+err.Error())
		return
	}
	api.OK(c, content)
}

func AdminListFeedback(c *gin.Context) {
	query := db.DB.Model(&models.FeedbackReport{})
	if category := strings.TrimSpace(c.Query("category")); category != "" {
		if !feedbackCategories[category] {
			api.Fail(c, http.StatusBadRequest, "invalid category filter")
			return
		}
		query = query.Where("feedback_reports.category = ?", category)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		if !feedbackStatuses[status] {
			api.Fail(c, http.StatusBadRequest, "invalid status filter")
			return
		}
		query = query.Where("feedback_reports.status = ?", status)
	}
	var rows []feedbackAdminResp
	if err := query.Select("feedback_reports.*, users.nickname AS user_nickname").Joins("LEFT JOIN users ON users.id = feedback_reports.user_id").Order("feedback_reports.id DESC").Limit(100).Scan(&rows).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query feedback failed: "+err.Error())
		return
	}
	api.OK(c, rows)
}

func AdminUpdateFeedback(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid feedback id")
		return
	}
	var req feedbackUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	status := strings.TrimSpace(req.Status)
	if !feedbackStatuses[status] {
		api.Fail(c, http.StatusBadRequest, "status must be one of open, processing, resolved, closed")
		return
	}
	if req.PublicResponse != nil && req.ClearPublicResponse {
		api.Fail(c, http.StatusBadRequest, "public_response and clear_public_response cannot be used together")
		return
	}
	var response string
	if req.PublicResponse != nil {
		response = strings.TrimSpace(*req.PublicResponse)
		if response == "" {
			api.Fail(c, http.StatusBadRequest, "use clear_public_response to clear the public response")
			return
		}
		if len([]rune(response)) > feedbackResponseMax {
			api.Fail(c, http.StatusBadRequest, "public_response must be at most 1000 characters")
			return
		}
	}
	var report models.FeedbackReport
	if err := db.DB.First(&report, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			api.Fail(c, http.StatusNotFound, "feedback not found")
			return
		}
		api.Fail(c, http.StatusInternalServerError, "query feedback failed: "+err.Error())
		return
	}
	adminID := c.GetUint(middleware.CtxUserIDKey)
	updates := map[string]interface{}{"status": status}
	if req.PublicResponse != nil {
		now := time.Now()
		updates["public_response"] = response
		updates["responded_at"] = &now
		updates["responded_by"] = adminID
	} else if req.ClearPublicResponse {
		updates["public_response"] = nil
		updates["responded_at"] = nil
		updates["responded_by"] = nil
	}
	if err := db.DB.Model(&report).Updates(updates).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "update feedback failed: "+err.Error())
		return
	}
	recordAdminAudit(adminID, &report.UserID, "update_feedback", "feedback #"+strconv.FormatUint(uint64(report.ID), 10)+" status "+status)
	updatedReport := models.FeedbackReport{}
	if err := db.DB.First(&updatedReport, report.ID).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "reload feedback failed: "+err.Error())
		return
	}
	api.OK(c, updatedReport)
}

func publicFeedback(report models.FeedbackReport) feedbackPublicResp {
	return feedbackPublicResp{
		ID: report.ID, Category: report.Category, Content: report.Content, Status: report.Status,
		PublicResponse: report.PublicResponse, RespondedAt: report.RespondedAt,
		CreatedAt: report.CreatedAt, UpdatedAt: report.UpdatedAt,
	}
}

func firstOpsContent(kind string) (models.OpsContent, error) {
	if _, ok := defaultOpsContent[kind]; !ok {
		return models.OpsContent{}, gorm.ErrRecordNotFound
	}
	var content models.OpsContent
	err := db.DB.Where("kind = ?", kind).First(&content).Error
	if err == nil {
		if kind == "privacy" && content.Body == legacyPhonePrivacyBody {
			content.Body = defaultOpsContent[kind].Body
			err = db.DB.Model(&content).Update("body", content.Body).Error
		}
		return content, nil
	}
	if err != gorm.ErrRecordNotFound {
		return models.OpsContent{}, err
	}
	content = defaultOpsContent[kind]
	err = db.DB.Create(&content).Error
	return content, err
}
