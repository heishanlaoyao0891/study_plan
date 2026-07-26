package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const maxBlueprintStages = 12
const maxBlueprintTasks = 120
const maxGeneratedPreviewTasks = 120

type PlanningBlueprint struct {
	Title     string                   `json:"title"`
	Summary   string                   `json:"summary"`
	Rationale string                   `json:"rationale"`
	Stages    []PlanningBlueprintStage `json:"stages"`
	Tasks     []PlanningBlueprintTask  `json:"tasks"`
}

type PlanningBlueprintStage struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Objective string `json:"objective"`
	Order     int    `json:"order"`
}

type PlanningBlueprintTask struct {
	ID              string   `json:"id"`
	StageID         string   `json:"stage_id"`
	Title           string   `json:"title"`
	Objective       string   `json:"objective"`
	Description     string   `json:"description"`
	EffortMinutes   int      `json:"effort_minutes"`
	Difficulty      string   `json:"difficulty"`
	Order           int      `json:"order"`
	PrerequisiteIDs []string `json:"prerequisite_ids,omitempty"`
}

func ParsePlanningBlueprintJSON(raw string) (PlanningBlueprint, error) {
	var blueprint PlanningBlueprint
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	if err := json.Unmarshal([]byte(strings.TrimSpace(cleaned)), &blueprint); err != nil {
		return blueprint, err
	}
	return blueprint, nil
}

func ValidatePlanningBlueprint(blueprint PlanningBlueprint) error {
	if strings.TrimSpace(blueprint.Title) == "" || strings.TrimSpace(blueprint.Summary) == "" {
		return fmt.Errorf("blueprint title and summary are required")
	}
	if len(blueprint.Stages) < 1 || len(blueprint.Stages) > maxBlueprintStages {
		return fmt.Errorf("blueprint stages must be between 1 and %d", maxBlueprintStages)
	}
	if len(blueprint.Tasks) < 1 || len(blueprint.Tasks) > maxBlueprintTasks {
		return fmt.Errorf("blueprint tasks must be between 1 and %d", maxBlueprintTasks)
	}
	stageIDs := map[string]bool{}
	lastStageOrder := -1
	for _, stage := range blueprint.Stages {
		if !validBlueprintID(stage.ID) || stageIDs[stage.ID] || strings.TrimSpace(stage.Name) == "" || strings.TrimSpace(stage.Objective) == "" || stage.Order <= lastStageOrder {
			return fmt.Errorf("invalid or unordered blueprint stage")
		}
		stageIDs[stage.ID] = true
		lastStageOrder = stage.Order
	}
	taskIDs := map[string]int{}
	lastTaskOrder := -1
	for index, task := range blueprint.Tasks {
		if !validBlueprintID(task.ID) || taskIDs[task.ID] != 0 || !stageIDs[task.StageID] || strings.TrimSpace(task.Title) == "" || strings.TrimSpace(task.Objective) == "" || strings.EqualFold(strings.TrimSpace(task.Title), strings.TrimSpace(task.Objective)) {
			return fmt.Errorf("invalid blueprint task")
		}
		if task.EffortMinutes < 15 || task.EffortMinutes > 720 || task.Order <= lastTaskOrder {
			return fmt.Errorf("invalid blueprint task effort or order")
		}
		if task.Difficulty != "easy" && task.Difficulty != "medium" && task.Difficulty != "hard" {
			return fmt.Errorf("invalid blueprint task difficulty")
		}
		for _, prerequisite := range task.PrerequisiteIDs {
			prerequisiteIndex, exists := taskIDs[prerequisite]
			if !exists || prerequisiteIndex >= index+1 {
				return fmt.Errorf("invalid blueprint prerequisite reference")
			}
		}
		taskIDs[task.ID] = index + 1
		lastTaskOrder = task.Order
	}
	return nil
}

func BuildPlanningBlueprintPrompt(ctx PlanningContext) string {
	brief, _ := json.Marshal(struct {
		Contract           string             `json:"contract"`
		Goal               string             `json:"goal"`
		Days               int                `json:"days"`
		HoursPerDay        int                `json:"hours_per_day"`
		Refinement         string             `json:"refinement,omitempty"`
		LearningProfile    LearningProfile    `json:"learning_profile"`
		ActivePlanLoad     ActivePlanLoad     `json:"active_plan_load"`
		RecentTaskOutcomes RecentTaskOutcomes `json:"recent_task_outcomes"`
	}{"planning_blueprint_v1", ctx.Input.Goal, ctx.Input.Days, ctx.Input.HoursPerDay, ctx.Input.Refinement, ctx.LearningProfile, ctx.ActivePlanLoad, ctx.RecentTaskOutcomes})
	return "请作为学习任务拆解专家输出简体中文 JSON 蓝图。你负责学习阶段、任务数量、目标、描述、难度、工作量、顺序和前置关系；不要输出日期、时间、数据库 ID 或用户隐私。任务必须可执行、目标不能只是重复标题。\nNORMALIZED_DECOMPOSITION_BRIEF:\n" + string(brief)
}

func PlanningBlueprintTokenAllowance(input PlanGenerationInput) int {
	estimatedTasks := input.Days * 2
	if estimatedTasks < 4 {
		estimatedTasks = 4
	}
	if estimatedTasks > 60 {
		estimatedTasks = 60
	}
	allowance := 512 + estimatedTasks*96
	if allowance > 8192 {
		return 8192
	}
	return allowance
}

func SchedulePlanningBlueprint(ctx PlanningContext, blueprint PlanningBlueprint) (PlanPreview, []string, error) {
	if err := ValidatePlanningBlueprint(blueprint); err != nil {
		return PlanPreview{}, nil, err
	}
	availableStart, availableEnd, err := parsePlanningSlot(ctx.Input.AvailableTimeSlot)
	if err != nil {
		return PlanPreview{}, nil, err
	}
	capacity := minInt(ctx.Input.HoursPerDay*60, availableEnd-availableStart)
	if capacity < 15 {
		return PlanPreview{}, nil, fmt.Errorf("available daily capacity is too small")
	}
	occupied := map[string][][2]int{}
	for _, row := range ctx.Occupancy {
		start, end, parseErr := parsePlanningRange(row.PlannedStart, row.PlannedEnd)
		if parseErr == nil {
			occupied[row.Date] = append(occupied[row.Date], [2]int{start, end})
		}
	}
	skip := map[string]bool{}
	for _, date := range ctx.Input.SkipDates {
		skip[date] = true
	}
	startDate, _ := time.Parse(aiPlanDateLayout, ctx.Input.StartDate)
	nextOffset := 0
	tasks := make([]PlanPreviewTask, 0, len(blueprint.Tasks))
	warnings := []string{}
	for _, semanticTask := range blueprint.Tasks {
		remaining := semanticTask.EffortMinutes
		part := 1
		parts := (remaining + capacity - 1) / capacity
		if parts > 1 {
			warnings = append(warnings, fmt.Sprintf("任务“%s”超过单日容量，已拆为 %d 个顺序部分。", semanticTask.Title, parts))
		}
		for remaining > 0 {
			duration := minInt(remaining, capacity)
			allocated := false
			for ; nextOffset < maxPlanningSearchDays; nextOffset++ {
				date := startDate.AddDate(0, 0, nextOffset).Format(aiPlanDateLayout)
				if skip[date] {
					continue
				}
				taskStart, ok := firstFreePlanningSlot(availableStart, availableEnd, duration, occupied[date])
				if !ok {
					continue
				}
				title := semanticTask.Title
				if parts > 1 {
					title = fmt.Sprintf("%s（%d/%d）", title, part, parts)
				}
				tasks = append(tasks, PlanPreviewTask{Date: date, PlannedStart: formatPlanningMinute(taskStart), PlannedEnd: formatPlanningMinute(taskStart + duration), Title: title, Objective: semanticTask.Objective, Description: semanticTask.Description, EstimatedMinutes: duration, Difficulty: semanticTask.Difficulty})
				occupied[date] = append(occupied[date], [2]int{taskStart, taskStart + duration})
				remaining -= duration
				part++
				allocated = true
				break
			}
			if !allocated {
				return PlanPreview{}, warnings, fmt.Errorf("could not schedule blueprint within %d days", maxPlanningSearchDays)
			}
		}
	}
	if len(tasks) > maxGeneratedPreviewTasks {
		return PlanPreview{}, warnings, fmt.Errorf("scheduled blueprint exceeds %d tasks", maxGeneratedPreviewTasks)
	}
	sort.SliceStable(tasks, func(i, j int) bool { return tasks[i].Date < tasks[j].Date })
	preview := PlanPreview{Title: strings.TrimSpace(blueprint.Title), Summary: strings.TrimSpace(blueprint.Summary), Rationale: strings.TrimSpace(blueprint.Rationale), Tasks: tasks}
	recomputePlanTotals(&preview)
	return preview, warnings, ValidatePlanPreview(preview, ctx.Input)
}

func validBlueprintID(value string) bool {
	if len(value) < 1 || len(value) > 32 {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_' && char != '-' {
			return false
		}
	}
	return true
}
