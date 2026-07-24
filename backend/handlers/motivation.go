package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

var motivationFallbacks = []struct{ Text, Source string }{
	{"不积跬步，无以至千里。", "荀子"},
	{"学而时习之，不亦说乎。", "论语"},
	{"知之者不如好之者。", "论语"},
}

func DailyMotivation(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	date := c.Query("date")
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if date == "" {
		date = time.Now().In(loc).Format(dateLayout)
	}
	if _, err := time.ParseInLocation(dateLayout, date, loc); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid date, expect YYYY-MM-DD")
		return
	}

	var cached models.DailyMotivation
	if err := db.DB.Where("user_id = ? AND date = ?", uid, date).First(&cached).Error; err == nil {
		api.OK(c, cached)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		api.Fail(c, http.StatusInternalServerError, "query motivation failed: "+err.Error())
		return
	}

	message := fallbackMotivation(uid, date)
	cfg, provider, providerErr := services.CurrentAIProvider()
	if providerErr == nil && cfg.Enabled {
		completionRate, streak := motivationSignals(uid)
		prompt := "生成一句简洁、积极但不夸张的中文学习寄语。只返回正文，不要作者、引号或换行，最多32个汉字。统计信号：连续打卡" + streak + "天，近30天完成率" + completionRate + "%。不得提及任务内容或个人信息。"
		if text, err := provider.Generate(prompt, 64); err == nil && validMotivationContent(text, "今日寄语") {
			message = models.DailyMotivation{UserID: uid, Date: date, Text: strings.TrimSpace(text), Source: "今日寄语", Origin: "ai"}
		}
	}
	if err := db.DB.Create(&message).Error; err != nil {
		if queryErr := db.DB.Where("user_id = ? AND date = ?", uid, date).First(&message).Error; queryErr != nil {
			api.Fail(c, http.StatusInternalServerError, "cache motivation failed: "+err.Error())
			return
		}
	}
	api.OK(c, message)
}

func motivationSignals(uid uint) (string, string) {
	profile, _ := services.GetUserLearningProfile(uid)
	return strconvFormatFloat(profile.CompletionRate * 100), currentStreakString(uid)
}

func currentStreakString(uid uint) string {
	streak := 0
	for day := shanghaiNow(); streak < 366; day = day.AddDate(0, 0, -1) {
		var count int64
		db.DB.Model(&models.DailyCheckin{}).Where("user_id = ? AND date = ? AND completed = ?", uid, day.Format(dateLayout), true).Count(&count)
		if count == 0 {
			break
		}
		streak++
	}
	return strconv.Itoa(streak)
}

func strconvFormatFloat(value float64) string { return strconv.FormatFloat(value, 'f', 0, 64) }

func fallbackMotivation(uid uint, date string) models.DailyMotivation {
	index := int(uid) % len(motivationFallbacks)
	if len(date) > 0 {
		index = (index + int(date[len(date)-1]-'0')) % len(motivationFallbacks)
	}
	row := motivationFallbacks[index]
	return models.DailyMotivation{UserID: uid, Date: date, Text: row.Text, Source: row.Source, Origin: "library"}
}

func validMotivationContent(text, source string) bool {
	text = strings.TrimSpace(text)
	return text != "" && !strings.ContainsAny(text, "\r\n") && utf8.RuneCountInString(text) <= 32 && utf8.RuneCountInString(source) <= 12
}
