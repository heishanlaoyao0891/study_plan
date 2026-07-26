package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

type statsMetrics struct {
	StudyMinutes    int  `json:"study_minutes"`
	PlannedMinutes  int  `json:"planned_minutes"`
	OvertimeMinutes int  `json:"overtime_minutes"`
	CompletedTasks  int  `json:"completed_tasks"`
	TotalTasks      int  `json:"total_tasks"`
	CompletionRate  *int `json:"completion_rate"`
}

type statsPoint struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Start     string `json:"start"`
	End       string `json:"end"`
	PlanID    uint   `json:"plan_id,omitempty"`
	PlanTitle string `json:"plan_title,omitempty"`
	statsMetrics
}

func StatsTrend(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	period, dimension := c.DefaultQuery("period", "7d"), c.DefaultQuery("dimension", "time")
	if (period != "7d" && period != "1m" && period != "1y") || (dimension != "time" && dimension != "plan") {
		api.Fail(c, http.StatusBadRequest, "period or dimension is invalid")
		return
	}
	now := shanghaiNow()
	end := now.Format(dateLayout)
	startTime := now.AddDate(0, 0, -6)
	if period == "1m" {
		startTime = now.AddDate(0, 0, -29)
	} else if period == "1y" {
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -11, 0)
	}
	start := startTime.Format(dateLayout)
	var tasks []models.DailyTask
	if err := db.DB.Where("user_id = ? AND date >= ? AND date <= ?", uid, start, end).Order("date ASC, id ASC").Find(&tasks).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query trend failed")
		return
	}
	planTitles := map[uint]string{}
	var plans []models.Plan
	if err := db.DB.Select("id", "title").Where("user_id = ?", uid).Find(&plans).Error; err == nil {
		for _, plan := range plans {
			planTitles[plan.ID] = plan.Title
		}
	}
	points := map[string]int{}
	series := make([]statsPoint, 0)
	if dimension == "time" {
		if period == "1y" {
			for offset := 0; offset < 12; offset++ {
				month := startTime.AddDate(0, offset, 0)
				key := month.Format("2006-01")
				pointEnd := month.AddDate(0, 1, -1)
				if pointEnd.After(now) {
					pointEnd = now
				}
				series = append(series, statsPoint{Key: key, Label: month.Format("06-01"), Start: month.Format(dateLayout), End: pointEnd.Format(dateLayout)})
			}
		} else {
			days := 7
			if period == "1m" {
				days = 30
			}
			for offset := 0; offset < days; offset++ {
				date := startTime.AddDate(0, 0, offset)
				series = append(series, statsPoint{Key: date.Format(dateLayout), Label: date.Format("01-02"), Start: date.Format(dateLayout), End: date.Format(dateLayout)})
			}
		}
		for index := range series {
			points[series[index].Key] = index
		}
	} else {
		for _, task := range tasks {
			key := strconv.FormatUint(uint64(task.PlanID), 10)
			if _, exists := points[key]; !exists {
				series = append(series, statsPoint{Key: key, Label: planTitles[task.PlanID], Start: start, End: end, PlanID: task.PlanID, PlanTitle: planTitles[task.PlanID]})
				points[key] = len(series) - 1
			}
		}
	}
	summary := statsMetrics{}
	for _, task := range tasks {
		key := task.Date
		if period == "1y" && dimension == "time" {
			key = task.Date[:7]
		}
		if dimension == "plan" {
			key = strconv.FormatUint(uint64(task.PlanID), 10)
		}
		pointIndex, exists := points[key]
		if !exists {
			continue
		}
		planned := plannedRangeMinutes(task.PlannedStart, task.PlannedEnd)
		if planned <= 0 {
			planned = task.EstimatedMinutes
		}
		if planned <= 0 {
			planned = 60
		}
		applyStatsTask(&series[pointIndex].statsMetrics, task, planned)
		applyStatsTask(&summary, task, planned)
	}
	for index := range series {
		finalizeStatsMetrics(&series[index].statsMetrics)
	}
	finalizeStatsMetrics(&summary)
	if dimension == "plan" {
		sort.Slice(series, func(i, j int) bool {
			if series[i].StudyMinutes != series[j].StudyMinutes {
				return series[i].StudyMinutes > series[j].StudyMinutes
			}
			return series[i].PlanID < series[j].PlanID
		})
	}
	api.OK(c, gin.H{"period": period, "dimension": dimension, "timezone": "Asia/Shanghai", "start": start, "end": end, "bucket_unit": map[bool]string{true: "plan", false: map[bool]string{true: "month", false: "day"}[period == "1y"]}[dimension == "plan"], "summary": summary, "series": series})
}

func applyStatsTask(metrics *statsMetrics, task models.DailyTask, planned int) {
	metrics.StudyMinutes += task.StudyMinutes
	metrics.PlannedMinutes += planned
	if task.StudyMinutes > planned {
		metrics.OvertimeMinutes += task.StudyMinutes - planned
	}
	metrics.TotalTasks++
	if task.Status == models.TaskStatusCompleted {
		metrics.CompletedTasks++
	}
}

func finalizeStatsMetrics(metrics *statsMetrics) {
	if metrics.TotalTasks == 0 {
		return
	}
	rate := metrics.CompletedTasks * 100 / metrics.TotalTasks
	metrics.CompletionRate = &rate
}

func StatsCalendar(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	month := c.Query("month")
	if month == "" {
		month = shanghaiNow().Format("2006-01")
	}
	start, err := time.ParseInLocation("2006-01", month, shanghaiNow().Location())
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid month")
		return
	}
	end := start.AddDate(0, 1, 0)
	type row struct {
		Date         string `json:"date"`
		StudyMinutes int    `json:"study_minutes"`
		Completed    int    `json:"completed"`
		Total        int    `json:"total"`
	}
	var out []row
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		date := d.Format(dateLayout)
		var minutes int64
		db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND date = ?", uid, date).Select("COALESCE(SUM(study_minutes),0)").Scan(&minutes)
		var completed, taskCount int64
		db.DB.Model(&models.DailyCheckin{}).Where("user_id = ? AND date = ? AND completed = ?", uid, date, true).Count(&completed)
		db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND date = ?", uid, date).Count(&taskCount)
		total := 0
		if taskCount > 0 {
			total = 1
		}
		out = append(out, row{Date: date, StudyMinutes: int(minutes), Completed: int(completed), Total: total})
	}
	api.OK(c, out)
}

func DailyDistribution(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	date := c.DefaultQuery("date", shanghaiToday())
	type row struct {
		PlanID       uint   `json:"plan_id"`
		Title        string `json:"title"`
		StudyMinutes int    `json:"study_minutes"`
		Status       string `json:"status"`
	}
	var out []row
	db.DB.Table("daily_tasks").Select("daily_tasks.plan_id, plans.title, daily_tasks.study_minutes, daily_tasks.status").
		Joins("LEFT JOIN plans ON plans.id = daily_tasks.plan_id").
		Where("daily_tasks.user_id = ? AND daily_tasks.date = ?", uid, date).
		Scan(&out)
	api.OK(c, out)
}

func WeeklyReport(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	now := shanghaiNow()
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(now.Year())))
	week, _ := strconv.Atoi(c.DefaultQuery("week", strconv.Itoa(weekNumber(now))))
	start := firstDayOfISOWeek(year, week)
	end := start.AddDate(0, 0, 7)
	reportRange(c, uid, start, end)
}

func MonthlyReport(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	now := shanghaiNow()
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(now.Year())))
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(now.Month()))))
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0)
	reportRange(c, uid, start, end)
}

func SlackDistribution(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	month := c.Query("month")
	if month == "" {
		month = shanghaiNow().Format("2006-01")
	}
	start, err := time.ParseInLocation("2006-01", month, shanghaiNow().Location())
	if err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid month")
		return
	}
	end := start.AddDate(0, 1, 0)
	type row struct {
		Activity string `json:"activity"`
		Minutes  int    `json:"minutes"`
	}
	var out []row
	db.DB.Model(&models.SlackRecord{}).
		Select("activity, COALESCE(SUM(duration_min),0) as minutes").
		Where("user_id = ? AND start_time >= ? AND start_time < ?", uid, start, end).
		Group("activity").Scan(&out)
	api.OK(c, out)
}

func EfficiencyStats(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 || days > 366 {
		days = 30
	}
	start := shanghaiNow().AddDate(0, 0, -days+1).Format(dateLayout)
	today := shanghaiToday()
	var total, completed int64
	db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND date >= ? AND date <= ?", uid, start, today).Count(&total)
	db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND date >= ? AND date <= ? AND status = ?", uid, start, today, models.TaskStatusCompleted).Count(&completed)
	var minutes int64
	db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND date >= ? AND date <= ?", uid, start, today).Select("COALESCE(SUM(study_minutes),0)").Scan(&minutes)
	completionRate := 0
	if total > 0 {
		completionRate = int(completed * 100 / total)
	}
	type trendRow struct {
		Date         string `json:"date"`
		Completed    int    `json:"completed"`
		Total        int    `json:"total"`
		StudyMinutes int    `json:"study_minutes"`
	}
	var trend []trendRow
	db.DB.Model(&models.DailyTask{}).
		Select("date, SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) AS completed, COUNT(*) AS total, COALESCE(SUM(study_minutes),0) AS study_minutes", models.TaskStatusCompleted).
		Where("user_id = ? AND date >= ? AND date <= ?", uid, start, today).
		Group("date").Order("date ASC").Scan(&trend)
	api.OK(c, gin.H{
		"days":                days,
		"start":               start,
		"end":                 today,
		"total_tasks":         total,
		"completed_tasks":     completed,
		"completion_rate":     completionRate,
		"study_minutes":       minutes,
		"avg_minutes_per_day": int(minutes) / days,
		"trend":               trend,
	})
}

func reportRange(c *gin.Context, uid uint, start, end time.Time) {
	var studyMinutes int64
	db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND date >= ? AND date < ?", uid, start.Format(dateLayout), end.Format(dateLayout)).Select("COALESCE(SUM(study_minutes),0)").Scan(&studyMinutes)
	var slackMinutes int64
	db.DB.Model(&models.SlackRecord{}).Where("user_id = ? AND start_time >= ? AND start_time < ?", uid, start, end).Select("COALESCE(SUM(duration_min),0)").Scan(&slackMinutes)
	var completed int64
	db.DB.Model(&models.DailyCheckin{}).Where("user_id = ? AND date >= ? AND date < ? AND completed = ?", uid, start.Format(dateLayout), end.Format(dateLayout), true).Count(&completed)
	api.OK(c, gin.H{
		"start":               start.Format(dateLayout),
		"end":                 end.AddDate(0, 0, -1).Format(dateLayout),
		"total_study_minutes": studyMinutes,
		"slack_minutes":       slackMinutes,
		"completed_checkins":  completed,
	})
}

func weekNumber(t time.Time) int {
	_, w := t.ISOWeek()
	return w
}

func firstDayOfISOWeek(year int, week int) time.Time {
	date := time.Date(year, 1, 4, 0, 0, 0, 0, shanghaiNow().Location())
	isoWeekday := int(date.Weekday())
	if isoWeekday == 0 {
		isoWeekday = 7
	}
	monday := date.AddDate(0, 0, -isoWeekday+1)
	return monday.AddDate(0, 0, (week-1)*7)
}
