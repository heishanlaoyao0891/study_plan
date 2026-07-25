package handlers

import (
	"net/http"
	"strings"

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
	if category == "" {
		category = "feedback"
	}
	report := models.FeedbackReport{UserID: uid, Category: category, Content: strings.TrimSpace(req.Content), Contact: strings.TrimSpace(req.Contact), Status: "open"}
	if report.Content == "" {
		api.Fail(c, http.StatusBadRequest, "content required")
		return
	}
	if err := db.DB.Create(&report).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "submit feedback failed: "+err.Error())
		return
	}
	api.OK(c, report)
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
	var rows []models.FeedbackReport
	if err := db.DB.Order("id DESC").Limit(100).Find(&rows).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query feedback failed: "+err.Error())
		return
	}
	api.OK(c, rows)
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
