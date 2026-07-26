package services

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"study_plan_backend/db"
	"study_plan_backend/models"
)

const aiPlanDateLayout = "2006-01-02"
const defaultMaxPreviewDays = 30
const maxPlanningGoalRunes = 500
const maxPlanningRefinementRunes = 1000
const maxPlanningHoursPerDay = 12
const maxPlanningSearchDays = 120
const maxPlanningSkipDates = 120

type PlanGenerationInput struct {
	UserID            uint     `json:"-"`
	Goal              string   `json:"goal"`
	HoursPerDay       int      `json:"hours_per_day"`
	Days              int      `json:"days"`
	StartDate         string   `json:"start_date"`
	AvailableTimeSlot string   `json:"available_time_slot"`
	SkipDates         []string `json:"skip_dates"`
	Refinement        string   `json:"refinement,omitempty"`
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

type PlanningOccupancy struct {
	Date         string `json:"date"`
	PlannedStart string `json:"planned_start"`
	PlannedEnd   string `json:"planned_end"`
}

type PlanningContext struct {
	Input              PlanGenerationInput `json:"input"`
	LearningProfile    LearningProfile     `json:"learning_profile"`
	ActivePlanLoad     ActivePlanLoad      `json:"active_plan_load"`
	RecentTaskOutcomes RecentTaskOutcomes  `json:"recent_task_outcomes"`
	Occupancy          []PlanningOccupancy `json:"occupancy"`
	ConflictingDates   []string            `json:"conflicting_dates"`
	Warnings           []string            `json:"warnings,omitempty"`
	Prompt             string              `json:"prompt"`
}

type PlanPreview struct {
	Title               string            `json:"title"`
	Summary             string            `json:"summary"`
	EstimatedTotalHours float64           `json:"estimated_total_hours"`
	Rationale           string            `json:"rationale"`
	Tasks               []PlanPreviewTask `json:"tasks"`
}

type PlanPreviewTask struct {
	Identity         string `json:"identity"`
	Date             string `json:"date"`
	PlannedStart     string `json:"planned_start"`
	PlannedEnd       string `json:"planned_end"`
	Title            string `json:"title"`
	Objective        string `json:"objective"`
	Description      string `json:"description"`
	EstimatedMinutes int    `json:"estimated_minutes"`
	Difficulty       string `json:"difficulty"`
}

type planningStage struct {
	Name        string
	Objective   string
	Description string
}

var planningStageTemplates = map[string][]planningStage{
	"learning": {
		{Name: "夯实基础", Objective: "梳理核心概念并完成基础示例", Description: "建立知识框架，记录仍不清楚的概念。"},
		{Name: "跟练实践", Objective: "跟随示例完成一个可验证练习", Description: "对照示例操作，并解释每一步的作用。"},
		{Name: "独立实践", Objective: "脱离示例独立完成一个练习", Description: "独立应用所学内容，记录卡点和解决方法。"},
		{Name: "复盘巩固", Objective: "复盘错点并用自己的话总结", Description: "回顾笔记和练习结果，形成可复用总结。"},
	},
	"reading": {
		{Name: "明确范围", Objective: "确定阅读范围和关键问题", Description: "浏览目录并写下本轮阅读要回答的问题。"},
		{Name: "分段阅读", Objective: "完成一个连续章节并标记重点", Description: "按范围推进阅读，标注论点、证据和疑问。"},
		{Name: "整理笔记", Objective: "将本轮内容整理为结构化笔记", Description: "提炼主要观点并建立章节之间的联系。"},
		{Name: "综合输出", Objective: "输出摘要并回答关键问题", Description: "不看原文复述核心内容，补齐理解缺口。"},
	},
	"exam": {
		{Name: "诊断摸底", Objective: "完成一次诊断并列出薄弱主题", Description: "用样题识别知识缺口和时间分配问题。"},
		{Name: "专题循环", Objective: "复习一个薄弱主题并完成对应题目", Description: "先回顾方法，再用题目检验掌握程度。"},
		{Name: "限时训练", Objective: "在限定时间内完成一组模拟题", Description: "按考试节奏作答并记录耗时。"},
		{Name: "错题复盘", Objective: "订正错题并归纳错误模式", Description: "分析错误原因，写出下一次的检查策略。"},
		{Name: "缓冲查漏", Objective: "补齐遗留薄弱点并做轻量回顾", Description: "优先处理未掌握内容，不再引入大量新题。"},
	},
	"project": {
		{Name: "澄清需求", Objective: "写出范围、验收标准和风险清单", Description: "把目标转化为可检查的交付条件。"},
		{Name: "搭建基础", Objective: "完成最小环境和骨架并验证运行", Description: "建立可持续迭代的基础结构。"},
		{Name: "推进里程碑", Objective: "完成一个可演示的核心增量", Description: "围绕验收标准交付一个端到端增量。"},
		{Name: "集成验证", Objective: "集成已有增量并处理关键问题", Description: "验证主要流程，修复阻塞交付的问题。"},
		{Name: "交付复盘", Objective: "按验收标准检查并整理交付说明", Description: "完成最终检查，记录遗留项和后续行动。"},
	},
	"general": {
		{Name: "理解目标", Objective: "拆解目标并定义本轮完成标准", Description: "明确范围、产出和验证方式。"},
		{Name: "刻意练习", Objective: "完成一个聚焦关键能力的练习", Description: "针对最重要的能力进行可验证练习。"},
		{Name: "实际应用", Objective: "在真实或模拟场景中应用所学", Description: "产出一个可以检查的应用结果。"},
		{Name: "巩固复盘", Objective: "回顾结果并整理下一步改进", Description: "总结有效方法、问题和后续行动。"},
	},
}

func NormalizePlanGenerationInput(input PlanGenerationInput) (PlanGenerationInput, []string, error) {
	input.Goal = strings.TrimSpace(input.Goal)
	input.Refinement = strings.TrimSpace(input.Refinement)
	if input.Goal == "" {
		return input, nil, fmt.Errorf("goal is required")
	}
	if len([]rune(input.Goal)) > maxPlanningGoalRunes {
		return input, nil, fmt.Errorf("goal must not exceed %d characters", maxPlanningGoalRunes)
	}
	if len([]rune(input.Refinement)) > maxPlanningRefinementRunes {
		return input, nil, fmt.Errorf("refinement must not exceed %d characters", maxPlanningRefinementRunes)
	}
	if input.Days == 0 {
		input.Days = 7
	}
	if input.Days < 1 || input.Days > defaultMaxPreviewDays {
		return input, nil, fmt.Errorf("days must be between 1 and %d", defaultMaxPreviewDays)
	}
	if input.HoursPerDay == 0 {
		input.HoursPerDay = 1
	}
	if input.HoursPerDay < 1 || input.HoursPerDay > maxPlanningHoursPerDay {
		return input, nil, fmt.Errorf("hours_per_day must be between 1 and %d", maxPlanningHoursPerDay)
	}
	if len(input.SkipDates) > maxPlanningSkipDates {
		return input, nil, fmt.Errorf("skip_dates must not exceed %d entries", maxPlanningSkipDates)
	}
	if input.StartDate == "" {
		input.StartDate = shanghaiTimeNow().Format(aiPlanDateLayout)
	} else if _, err := time.Parse(aiPlanDateLayout, input.StartDate); err != nil {
		return input, nil, fmt.Errorf("start_date must use YYYY-MM-DD")
	}
	if strings.TrimSpace(input.AvailableTimeSlot) == "" {
		input.AvailableTimeSlot = "20:00-21:00"
	}
	startMinute, endMinute, err := parsePlanningSlot(input.AvailableTimeSlot)
	if err != nil {
		return input, nil, err
	}
	input.AvailableTimeSlot = formatPlanningMinute(startMinute) + "-" + formatPlanningMinute(endMinute)
	warnings := make([]string, 0, 1)
	if input.HoursPerDay*60 > endMinute-startMinute {
		warnings = append(warnings, fmt.Sprintf("每天期望学习 %d 小时，但可用时段只有 %d 分钟，计划将以实际可用时段为准。", input.HoursPerDay, endMinute-startMinute))
	}
	seen := map[string]bool{}
	normalizedSkip := make([]string, 0, len(input.SkipDates))
	for _, raw := range input.SkipDates {
		date := strings.TrimSpace(raw)
		if _, err := time.Parse(aiPlanDateLayout, date); err != nil {
			return input, nil, fmt.Errorf("skip_dates must use YYYY-MM-DD")
		}
		if !seen[date] {
			seen[date] = true
			normalizedSkip = append(normalizedSkip, date)
		}
	}
	sort.Strings(normalizedSkip)
	input.SkipDates = normalizedSkip
	return input, warnings, nil
}

func BuildPlanningContext(input PlanGenerationInput) (PlanningContext, error) {
	return BuildPlanningContextWithContext(context.Background(), input)
}

func BuildPlanningContextWithContext(requestContext context.Context, input PlanGenerationInput) (PlanningContext, error) {
	normalized, warnings, err := NormalizePlanGenerationInput(input)
	if err != nil {
		return PlanningContext{}, err
	}
	profile, err := GetUserLearningProfileWithContext(requestContext, normalized.UserID)
	if err != nil {
		return PlanningContext{}, err
	}
	load, err := GetActivePlanLoadWithContext(requestContext, normalized.UserID)
	if err != nil {
		return PlanningContext{}, err
	}
	outcomes, err := GetRecentTaskOutcomesWithContext(requestContext, normalized.UserID)
	if err != nil {
		return PlanningContext{}, err
	}
	occupancy, err := LoadPlanningOccupancyWithContext(requestContext, normalized.UserID, normalized.StartDate, maxPlanningSearchDays)
	if err != nil {
		return PlanningContext{}, err
	}
	conflictSet := map[string]bool{}
	for _, row := range occupancy {
		conflictSet[row.Date] = true
	}
	conflicts := make([]string, 0, len(conflictSet))
	for date := range conflictSet {
		conflicts = append(conflicts, date)
	}
	sort.Strings(conflicts)
	ctx := PlanningContext{Input: normalized, LearningProfile: profile, ActivePlanLoad: load, RecentTaskOutcomes: outcomes, Occupancy: occupancy, ConflictingDates: conflicts, Warnings: warnings}
	return ctx, nil
}

func BuildPlanningPrompt(ctx PlanningContext, local PlanPreview) string {
	brief, _ := json.Marshal(struct {
		Input              PlanGenerationInput `json:"input"`
		LearningProfile    LearningProfile     `json:"learning_profile"`
		ActivePlanLoad     ActivePlanLoad      `json:"active_plan_load"`
		RecentTaskOutcomes RecentTaskOutcomes  `json:"recent_task_outcomes"`
		OccupancyCount     int                 `json:"unfinished_occupancy_count"`
	}{ctx.Input, ctx.LearningProfile, ctx.ActivePlanLoad, ctx.RecentTaskOutcomes, len(ctx.Occupancy)})
	skeleton, _ := json.Marshal(local)
	return "You are a bounded semantic collaborator for a study planning agent. Write all user-facing content in Simplified Chinese. Enrich only the title and each existing task's title, objective, description, and difficulty. Return the complete JSON object. Preserve summary, rationale, task count, order, dates, planned_start, planned_end, estimated_minutes, and estimated_total_hours exactly. Do not add or remove tasks.\nNORMALIZED_BRIEF:\n" + string(brief) + "\nCANONICAL_LOCAL_SKELETON:\n" + string(skeleton) + "\nREFINEMENT_INSTRUCTIONS:\n" + defaultString(ctx.Input.Refinement, "无")
}

func LoadPlanningOccupancy(userID uint, startDate string, searchDays int) ([]PlanningOccupancy, error) {
	return LoadPlanningOccupancyWithContext(context.Background(), userID, startDate, searchDays)
}

func LoadPlanningOccupancyWithContext(requestContext context.Context, userID uint, startDate string, searchDays int) ([]PlanningOccupancy, error) {
	start, err := time.Parse(aiPlanDateLayout, startDate)
	if err != nil {
		return nil, err
	}
	end := start.AddDate(0, 0, searchDays-1).Format(aiPlanDateLayout)
	var tasks []models.DailyTask
	if err := db.DB.WithContext(requestContext).Select("date", "planned_start", "planned_end").Where("user_id = ? AND date >= ? AND date <= ? AND status <> ?", userID, startDate, end, models.TaskStatusCompleted).Order("date ASC, planned_start ASC, planned_end ASC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	result := make([]PlanningOccupancy, 0, len(tasks))
	for _, task := range tasks {
		if _, _, err := parsePlanningRange(task.PlannedStart, task.PlannedEnd); err == nil {
			result = append(result, PlanningOccupancy{Date: task.Date, PlannedStart: task.PlannedStart, PlannedEnd: task.PlannedEnd})
		}
	}
	return result, nil
}

func BuildLocalPlan(ctx PlanningContext) (PlanPreview, error) {
	stageType := classifyPlanningGoal(ctx.Input.Goal)
	stages := planningStageTemplates[stageType]
	minutes := planningTaskMinutes(ctx)
	start, end, _ := parsePlanningSlot(ctx.Input.AvailableTimeSlot)
	occupied := map[string][][2]int{}
	for _, row := range ctx.Occupancy {
		rowStart, rowEnd, err := parsePlanningRange(row.PlannedStart, row.PlannedEnd)
		if err == nil {
			occupied[row.Date] = append(occupied[row.Date], [2]int{rowStart, rowEnd})
		}
	}
	skip := map[string]bool{}
	for _, date := range ctx.Input.SkipDates {
		skip[date] = true
	}
	firstDate, _ := time.Parse(aiPlanDateLayout, ctx.Input.StartDate)
	tasks := make([]PlanPreviewTask, 0, ctx.Input.Days)
	for offset := 0; offset < maxPlanningSearchDays && len(tasks) < ctx.Input.Days; offset++ {
		date := firstDate.AddDate(0, 0, offset).Format(aiPlanDateLayout)
		if skip[date] {
			continue
		}
		taskStart, ok := firstFreePlanningSlot(start, end, minutes, occupied[date])
		if !ok {
			continue
		}
		index := len(tasks)
		stage := planningStageForTask(stages, index, ctx.Input.Days)
		tasks = append(tasks, PlanPreviewTask{
			Date: date, PlannedStart: formatPlanningMinute(taskStart), PlannedEnd: formatPlanningMinute(taskStart + minutes),
			Title: fmt.Sprintf("%s：%s", stage.Name, ctx.Input.Goal), Objective: stage.Objective, Description: stage.Description,
			EstimatedMinutes: minutes, Difficulty: planningDifficulty(index, ctx.Input.Days, ctx.LearningProfile.CompletionRate),
		})
	}
	if len(tasks) != ctx.Input.Days {
		return PlanPreview{}, fmt.Errorf("could not allocate %d conflict-free task dates within %d days", ctx.Input.Days, maxPlanningSearchDays)
	}
	preview := PlanPreview{
		Title:     ctx.Input.Goal,
		Summary:   fmt.Sprintf("这是一个为期 %d 天的渐进式学习计划，包含阶段练习、复盘和缓冲安排。", len(tasks)),
		Rationale: localPlanRationale(ctx, stageType, minutes), Tasks: tasks,
	}
	recomputePlanTotals(&preview)
	return preview, ValidatePlanPreview(preview, ctx.Input)
}

func MergePlanEnrichment(local, enriched PlanPreview) (PlanPreview, error) {
	if len(enriched.Tasks) != len(local.Tasks) {
		return PlanPreview{}, fmt.Errorf("enrichment changed task count")
	}
	for index := range local.Tasks {
		if enriched.Tasks[index].Date != local.Tasks[index].Date || enriched.Tasks[index].PlannedStart != local.Tasks[index].PlannedStart || enriched.Tasks[index].PlannedEnd != local.Tasks[index].PlannedEnd {
			return PlanPreview{}, fmt.Errorf("enrichment changed task schedule identity or order")
		}
	}
	merged := local
	merged.Title = semanticValue(enriched.Title, local.Title)
	merged.Tasks = append([]PlanPreviewTask(nil), local.Tasks...)
	for index := range merged.Tasks {
		merged.Tasks[index].Title = semanticValue(enriched.Tasks[index].Title, local.Tasks[index].Title)
		merged.Tasks[index].Objective = semanticValue(enriched.Tasks[index].Objective, local.Tasks[index].Objective)
		merged.Tasks[index].Description = semanticValue(enriched.Tasks[index].Description, local.Tasks[index].Description)
		merged.Tasks[index].Difficulty = semanticValue(enriched.Tasks[index].Difficulty, local.Tasks[index].Difficulty)
	}
	recomputePlanTotals(&merged)
	return merged, nil
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
	sort.Strings(result)
	return result, nil
}

func FallbackPlanPreview(ctx PlanningContext) PlanPreview {
	ctx.Input.Days = boundedDays(ctx.Input.Days)
	if strings.TrimSpace(ctx.Input.Goal) == "" {
		ctx.Input.Goal = "Study plan"
	}
	if strings.TrimSpace(ctx.Input.AvailableTimeSlot) == "" {
		ctx.Input.AvailableTimeSlot = "20:00-21:00"
	}
	if strings.TrimSpace(ctx.Input.StartDate) == "" {
		ctx.Input.StartDate = shanghaiTimeNow().Format(aiPlanDateLayout)
	}
	preview, _ := BuildLocalPlan(ctx)
	return preview
}

func GetUserLearningProfile(userID uint) (LearningProfile, error) {
	return GetUserLearningProfileWithContext(context.Background(), userID)
}

func GetUserLearningProfileWithContext(requestContext context.Context, userID uint) (LearningProfile, error) {
	now := shanghaiTimeNow()
	from := now.AddDate(0, 0, -30).Format(aiPlanDateLayout)
	today := now.Format(aiPlanDateLayout)
	var total, completed int64
	database := db.DB.WithContext(requestContext)
	if err := database.Model(&models.DailyTask{}).Distinct("date").Where("user_id = ? AND date >= ? AND date <= ?", userID, from, today).Count(&total).Error; err != nil {
		return LearningProfile{}, err
	}
	if err := database.Model(&models.DailyCheckin{}).Where("user_id = ? AND date >= ? AND date <= ? AND completed = ?", userID, from, today, true).Count(&completed).Error; err != nil {
		return LearningProfile{}, err
	}
	var sessions []models.StudySession
	if err := database.Where("user_id = ? AND created_at >= ?", userID, time.Now().AddDate(0, 0, -30)).Find(&sessions).Error; err != nil {
		return LearningProfile{}, err
	}
	totalMinutes := 0
	for _, session := range sessions {
		totalMinutes += session.DurationMin
	}
	average := 0
	if len(sessions) > 0 {
		average = totalMinutes / len(sessions)
	}
	var postponed int64
	if err := database.Model(&models.PostponeRecord{}).Where("user_id = ? AND created_at >= ?", userID, time.Now().AddDate(0, 0, -30)).Count(&postponed).Error; err != nil {
		return LearningProfile{}, err
	}
	rate := 0.0
	if total > 0 {
		rate = float64(completed) / float64(total)
	}
	return LearningProfile{CompletionRate: rate, AverageStudyMins: average, PostponeFrequency: postponed}, nil
}

func GetActivePlanLoad(userID uint) (ActivePlanLoad, error) {
	return GetActivePlanLoadWithContext(context.Background(), userID)
}

func GetActivePlanLoadWithContext(requestContext context.Context, userID uint) (ActivePlanLoad, error) {
	var plans []models.Plan
	if err := db.DB.WithContext(requestContext).Where("user_id = ? AND status = ?", userID, models.PlanStatusActive).Find(&plans).Error; err != nil {
		return ActivePlanLoad{}, err
	}
	weekly := int64(0)
	for _, plan := range plans {
		weekly += int64(plan.WeeklyTargetHours)
	}
	return ActivePlanLoad{ActivePlanCount: int64(len(plans)), WeeklyTargetHours: weekly}, nil
}

func GetRecentTaskOutcomes(userID uint) (RecentTaskOutcomes, error) {
	return GetRecentTaskOutcomesWithContext(context.Background(), userID)
}

func GetRecentTaskOutcomesWithContext(requestContext context.Context, userID uint) (RecentTaskOutcomes, error) {
	from := shanghaiTimeNow().AddDate(0, 0, -30)
	var completed, missed, postponed int64
	database := db.DB.WithContext(requestContext)
	if err := database.Model(&models.DailyTask{}).Where("user_id = ? AND updated_at >= ? AND status = ?", userID, from, models.TaskStatusCompleted).Count(&completed).Error; err != nil {
		return RecentTaskOutcomes{}, err
	}
	if err := database.Model(&models.DailyTask{}).Where("user_id = ? AND date < ? AND status <> ?", userID, shanghaiTimeNow().Format(aiPlanDateLayout), models.TaskStatusCompleted).Count(&missed).Error; err != nil {
		return RecentTaskOutcomes{}, err
	}
	if err := database.Model(&models.PostponeRecord{}).Where("user_id = ? AND created_at >= ?", userID, from).Count(&postponed).Error; err != nil {
		return RecentTaskOutcomes{}, err
	}
	return RecentTaskOutcomes{Completed: completed, Missed: missed, Postponed: postponed}, nil
}

func classifyPlanningGoal(goal string) string {
	lower := strings.ToLower(goal)
	for _, keyword := range []string{"考试", "备考", "考研", "考证", "exam", "test", "certification"} {
		if strings.Contains(lower, keyword) {
			return "exam"
		}
	}
	for _, keyword := range []string{"阅读", "读完", "书", "reading", "read", "book"} {
		if strings.Contains(lower, keyword) {
			return "reading"
		}
	}
	for _, keyword := range []string{"项目", "开发", "交付", "上线", "project", "build", "ship", "deliver"} {
		if strings.Contains(lower, keyword) {
			return "project"
		}
	}
	for _, keyword := range []string{"学习", "掌握", "课程", "教程", "learn", "study", "course"} {
		if strings.Contains(lower, keyword) {
			return "learning"
		}
	}
	return "general"
}

func planningTaskMinutes(ctx PlanningContext) int {
	start, end, _ := parsePlanningSlot(ctx.Input.AvailableTimeSlot)
	capacity := minInt(ctx.Input.HoursPerDay*60, end-start)
	minutes := capacity
	if ctx.LearningProfile.CompletionRate > 0 && ctx.LearningProfile.CompletionRate < 0.6 {
		minutes = minutes * 2 / 3
	} else if ctx.LearningProfile.CompletionRate > 0 && ctx.LearningProfile.CompletionRate < 0.8 {
		minutes = minutes * 4 / 5
	}
	if ctx.LearningProfile.AverageStudyMins > 0 && ctx.LearningProfile.AverageStudyMins < minutes {
		minutes = ctx.LearningProfile.AverageStudyMins
	}
	minimum := minInt(60, capacity)
	if minutes < minimum {
		minutes = minimum
	}
	if minutes < 30 && end-start >= 30 {
		minutes = 30
	}
	return minInt(minutes, end-start)
}

func firstFreePlanningSlot(availableStart, availableEnd, duration int, occupied [][2]int) (int, bool) {
	rows := make([]ScheduleInterval, 0, len(occupied))
	for _, interval := range occupied {
		rows = append(rows, ScheduleInterval{Start: interval[0], End: interval[1]})
	}
	return FirstFreeScheduleSlot(availableStart, availableEnd, duration, rows)
}

func stageIndex(index, total, stageCount int) int {
	if total <= 1 {
		return stageCount - 1
	}
	result := index * stageCount / total
	if result >= stageCount {
		return stageCount - 1
	}
	return result
}

func planningStageForTask(stages []planningStage, index, total int) planningStage {
	if index == total-1 {
		return stages[len(stages)-1]
	}
	if index > 0 && (index+1)%5 == 0 {
		return planningStage{Name: "复盘缓冲", Objective: "回顾前一阶段并补齐一个关键缺口", Description: "整理阶段成果，优先解决遗留问题，再继续推进。"}
	}
	return stages[stageIndex(index, total, len(stages))]
}

func planningDifficulty(index, total int, completionRate float64) string {
	if completionRate > 0 && completionRate < 0.6 {
		return "easy"
	}
	if total >= 4 && index >= total/2 && index < total-1 {
		return "medium"
	}
	return "easy"
}

func localPlanRationale(ctx PlanningContext, stageType string, minutes int) string {
	pace := "稳定"
	if ctx.LearningProfile.CompletionRate > 0 && ctx.LearningProfile.CompletionRate < 0.6 {
		pace = "适当放缓"
	}
	return fmt.Sprintf("计划采用渐进式任务安排，每次学习 %d 分钟，整体节奏%s；已避开不可用日期，并处理未完成任务的时间冲突。", minutes, pace)
}

func recomputePlanTotals(preview *PlanPreview) {
	total := 0
	for _, task := range preview.Tasks {
		total += task.EstimatedMinutes
	}
	preview.EstimatedTotalHours = float64(total) / 60
}

func NormalizeCommittedPlanPreview(preview PlanPreview) (PlanPreview, error) {
	if len(preview.Tasks) == 0 || len(preview.Tasks) > maxGeneratedPreviewTasks {
		return PlanPreview{}, fmt.Errorf("task count must be between 1 and %d", maxGeneratedPreviewTasks)
	}
	for index := range preview.Tasks {
		task := &preview.Tasks[index]
		if _, err := time.Parse(aiPlanDateLayout, task.Date); err != nil {
			return PlanPreview{}, fmt.Errorf("task date must use YYYY-MM-DD")
		}
		start, end, err := parsePlanningRange(task.PlannedStart, task.PlannedEnd)
		if err != nil {
			return PlanPreview{}, err
		}
		task.EstimatedMinutes = end - start
	}
	sort.SliceStable(preview.Tasks, func(i, j int) bool {
		if preview.Tasks[i].Date != preview.Tasks[j].Date {
			return preview.Tasks[i].Date < preview.Tasks[j].Date
		}
		if preview.Tasks[i].PlannedStart != preview.Tasks[j].PlannedStart {
			return preview.Tasks[i].PlannedStart < preview.Tasks[j].PlannedStart
		}
		return preview.Tasks[i].PlannedEnd < preview.Tasks[j].PlannedEnd
	})
	recomputePlanTotals(&preview)
	return preview, ValidatePlanPreview(preview, PlanGenerationInput{})
}

func AssignPlanTaskIdentities(preview *PlanPreview) error {
	for index := range preview.Tasks {
		value := make([]byte, 16)
		if _, err := rand.Read(value); err != nil {
			return err
		}
		preview.Tasks[index].Identity = fmt.Sprintf("%x", value)
	}
	return nil
}

func semanticValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func parsePlanningSlot(slot string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(slot), "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("available_time_slot must use HH:mm-HH:mm")
	}
	return parsePlanningRange(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
}

func parsePlanningRange(startValue, endValue string) (int, int, error) {
	start, startErr := time.Parse("15:04", startValue)
	end, endErr := time.Parse("15:04", endValue)
	if startErr != nil || endErr != nil || !end.After(start) {
		return 0, 0, fmt.Errorf("available_time_slot must be a same-day increasing HH:mm-HH:mm range")
	}
	return start.Hour()*60 + start.Minute(), end.Hour()*60 + end.Minute(), nil
}

func formatPlanningMinute(value int) string { return fmt.Sprintf("%02d:%02d", value/60, value%60) }

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func shanghaiTimeNow() time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Now()
	}
	return time.Now().In(loc)
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

func minInt(first, second int) int {
	if first < second {
		return first
	}
	return second
}
