package services

import (
	"fmt"
	"strings"
	"time"

	"study_plan_backend/db"
	"study_plan_backend/models"
)

const aiPlanDateLayout = "2006-01-02"
const defaultMaxPreviewDays = 30

type PlanGenerationInput struct {
	UserID      uint
	Goal        string
	HoursPerDay int
	Days        int
	StartDate   string
	SkipDates   []string
	Refinement  string
}

type LearningProfile struct {
	CompletionRate    float64 `json:"completion_rate"`
	AverageStudyMins  int     `json:"average_study_minutes"`
	PostponeFrequency int64   `json:"postpone_frequency"`
}

type ActivePlanLoad struct {
	ActivePlanCount   int64 `json:"active_plan_count"`
	WeeklyTargetHours int64 `json:"weekly_target_hours"`
}

type RecentTaskOutcomes struct {
	Completed int64 `json:"completed"`
	Missed    int64 `json:"missed"`
	Postponed int64 `json:"postponed"`
	Makeup    int64 `json:"makeup"`
}

type PlanningContext struct {
	Input             PlanGenerationInput `json:"input"`
	LearningProfile   LearningProfile     `json:"learning_profile"`
	ActivePlanLoad    ActivePlanLoad      `json:"active_plan_load"`
	RecentTaskOutcomes RecentTaskOutcomes `json:"recent_task_outcomes"`
	ConflictingDates   []string           `json:"conflicting_dates"`
	Prompt             string             `json:"prompt"`
}

type PlanPreview struct {
	Title               string            `json:"title"`
	Summary             string            `json:"summary"`
	EstimatedTotalHours float64           `json:"estimated_total_hours"`
	Rationale           string            `json:"rationale"`
	Tasks               []PlanPreviewTask `json:"tasks"`
}

type PlanPreviewTask struct {
	Date             string `json:"date"`
	PlannedStart     string `json:"planned_start"`
	PlannedEnd       string `json:"planned_end"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	EstimatedMinutes int    `json:"estimated_minutes"`
	Difficulty       string `json:"difficulty"`
}

func BuildPlanningContext(input PlanGenerationInput) (PlanningContext, error) {
	profile, err := GetUserLearningProfile(input.UserID)
	if err != nil {
		return PlanningContext{}, err
	}
	load, err := GetActivePlanLoad(input.UserID)
	if err != nil {
		return PlanningContext{}, err
	}
	outcomes, err := GetRecentTaskOutcomes(input.UserID)
	if err != nil {
		return PlanningContext{}, err
	}
	previewDates := previewDateList(input)
	conflicts, err := CheckScheduleConflicts(input.UserID, previewDates)
	if err != nil {
		return PlanningContext{}, err
	}
	ctx := PlanningContext{Input: input, LearningProfile: profile, ActivePlanLoad: load, RecentTaskOutcomes: outcomes, ConflictingDates: conflicts}
	ctx.Prompt = BuildPlanningPrompt(ctx)
	return ctx, nil
}

func BuildPlanningPrompt(ctx PlanningContext) string {
	return fmt.Sprintf(
		"You are a study planning agent. Return only JSON for an editable plan preview. Goal: %s. Days: %d. Hours per day: %d. Completion rate: %.2f. Average study minutes: %d. Active plans: %d. Weekly target hours: %d. Postponed tasks: %d. Conflicting dates: %s. Respect skip dates and max 30 days.",
		ctx.Input.Goal,
		boundedDays(ctx.Input.Days),
		ctx.Input.HoursPerDay,
		ctx.LearningProfile.CompletionRate,
		ctx.LearningProfile.AverageStudyMins,
		ctx.ActivePlanLoad.ActivePlanCount,
		ctx.ActivePlanLoad.WeeklyTargetHours,
		ctx.RecentTaskOutcomes.Postponed,
		strings.Join(ctx.ConflictingDates, ","),
	)
}

func GetUserLearningProfile(userID uint) (LearningProfile, error) {
	from := time.Now().AddDate(0, 0, -30).Format(aiPlanDateLayout)
	var total int64
	var completed int64
	if err := db.DB.Model(&models.Checkin{}).Where("user_id = ? AND date >= ?", userID, from).Count(&total).Error; err != nil {
		return LearningProfile{}, err
	}
	if err := db.DB.Model(&models.Checkin{}).Where("user_id = ? AND date >= ? AND completed = ?", userID, from, true).Count(&completed).Error; err != nil {
		return LearningProfile{}, err
	}
	var sessions []models.StudySession
	if err := db.DB.Where("user_id = ? AND created_at >= ?", userID, time.Now().AddDate(0, 0, -30)).Find(&sessions).Error; err != nil {
		return LearningProfile{}, err
	}
	totalMinutes := 0
	for _, session := range sessions {
		totalMinutes += session.DurationMin
	}
	avg := 0
	if len(sessions) > 0 {
		avg = totalMinutes / len(sessions)
	}
	var postponed int64
	if err := db.DB.Model(&models.PostponeRecord{}).Where("user_id = ? AND created_at >= ?", userID, time.Now().AddDate(0, 0, -30)).Count(&postponed).Error; err != nil {
		return LearningProfile{}, err
	}
	rate := 0.0
	if total > 0 {
		rate = float64(completed) / float64(total)
	}
	return LearningProfile{CompletionRate: rate, AverageStudyMins: avg, PostponeFrequency: postponed}, nil
}

func GetActivePlanLoad(userID uint) (ActivePlanLoad, error) {
	var plans []models.Plan
	if err := db.DB.Where("user_id = ? AND status = ?", userID, models.PlanStatusActive).Find(&plans).Error; err != nil {
		return ActivePlanLoad{}, err
	}
	weekly := int64(0)
	for _, plan := range plans {
		weekly += int64(plan.WeeklyTargetHours)
	}
	return ActivePlanLoad{ActivePlanCount: int64(len(plans)), WeeklyTargetHours: weekly}, nil
}

func GetRecentTaskOutcomes(userID uint) (RecentTaskOutcomes, error) {
	from := time.Now().AddDate(0, 0, -30)
	var completed int64
	var missed int64
	var postponed int64
	if err := db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND updated_at >= ? AND status = ?", userID, from, models.TaskStatusCompleted).Count(&completed).Error; err != nil {
		return RecentTaskOutcomes{}, err
	}
	if err := db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND date < ? AND status <> ?", userID, time.Now().Format(aiPlanDateLayout), models.TaskStatusCompleted).Count(&missed).Error; err != nil {
		return RecentTaskOutcomes{}, err
	}
	if err := db.DB.Model(&models.PostponeRecord{}).Where("user_id = ? AND created_at >= ?", userID, from).Count(&postponed).Error; err != nil {
		return RecentTaskOutcomes{}, err
	}
	return RecentTaskOutcomes{Completed: completed, Missed: missed, Postponed: postponed, Makeup: 0}, nil
}

func CheckScheduleConflicts(userID uint, dates []string) ([]string, error) {
	if len(dates) == 0 {
		return nil, nil
	}
	var rows []models.DailyTask
	if err := db.DB.Where("user_id = ? AND date IN ? AND status <> ?", userID, dates, models.TaskStatusCompleted).Find(&rows).Error; err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, row := range rows {
		seen[row.Date] = true
	}
	result := make([]string, 0, len(seen))
	for date := range seen {
		result = append(result, date)
	}
	return result, nil
}

func FallbackPlanPreview(ctx PlanningContext) PlanPreview {
	input := ctx.Input
	minutes := maxInt(input.HoursPerDay, 1) * 60
	if ctx.LearningProfile.CompletionRate > 0 && ctx.LearningProfile.CompletionRate < 0.6 {
		minutes = maxInt(minutes*3/4, 30)
	}
	dates := previewDateList(input)
	tasks := make([]PlanPreviewTask, 0, len(dates))
	for i, date := range dates {
		tasks = append(tasks, PlanPreviewTask{
			Date:             date,
			PlannedStart:     "20:00",
			PlannedEnd:       "21:00",
			Title:            fmt.Sprintf("Day %d: %s", i+1, input.Goal),
			Description:      fmt.Sprintf("Focus on a concrete subtopic for %s and record blockers after study.", input.Goal),
			EstimatedMinutes: minutes,
			Difficulty:       fallbackDifficulty(ctx.LearningProfile.CompletionRate),
		})
	}
	return PlanPreview{Title: input.Goal, Summary: "Rule-based fallback preview", EstimatedTotalHours: float64(minutes*len(tasks)) / 60, Rationale: fallbackRationale(ctx), Tasks: tasks}
}

func previewDateList(input PlanGenerationInput) []string {
	start := time.Now()
	if input.StartDate != "" {
		if parsed, err := time.Parse(aiPlanDateLayout, input.StartDate); err == nil {
			start = parsed
		}
	}
	skip := map[string]bool{}
	for _, date := range input.SkipDates {
		skip[date] = true
	}
	days := boundedDays(input.Days)
	result := make([]string, 0, days)
	for offset := 0; len(result) < days && offset < defaultMaxPreviewDays*2; offset++ {
		date := start.AddDate(0, 0, offset).Format(aiPlanDateLayout)
		if skip[date] {
			continue
		}
		result = append(result, date)
	}
	return result
}

func boundedDays(days int) int {
	if days <= 0 {
		return 7
	}
	if days > defaultMaxPreviewDays {
		return defaultMaxPreviewDays
	}
	return days
}

func fallbackDifficulty(rate float64) string {
	if rate > 0.8 {
		return "medium"
	}
	return "easy"
}

func fallbackRationale(ctx PlanningContext) string {
	if ctx.LearningProfile.CompletionRate > 0 && ctx.LearningProfile.CompletionRate < 0.6 {
		return "Recent completion rate is low, so the fallback preview reduces daily load and keeps a predictable evening slot."
	}
	return "Fallback preview uses a steady daily workload and avoids configured skip dates."
}
