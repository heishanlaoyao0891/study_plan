package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxBlueprintStages = 24
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

func NormalizePlanningBlueprint(blueprint PlanningBlueprint) (PlanningBlueprint, []string) {
	warnings := []string{}
	stageIDMap := make(map[string]string, len(blueprint.Stages))
	for index := range blueprint.Stages {
		oldID := blueprint.Stages[index].ID
		newID := "stage_" + strconv.Itoa(index+1)
		stageIDMap[oldID] = newID
		blueprint.Stages[index].ID = newID
		blueprint.Stages[index].Order = index + 1
		blueprint.Stages[index].Name = strings.TrimSpace(blueprint.Stages[index].Name)
		blueprint.Stages[index].Objective = strings.TrimSpace(blueprint.Stages[index].Objective)
	}
	taskIDMap := make(map[string]string, len(blueprint.Tasks))
	for index := range blueprint.Tasks {
		oldID := blueprint.Tasks[index].ID
		taskIDMap[oldID] = "task_" + strconv.Itoa(index+1)
	}
	for index := range blueprint.Tasks {
		task := &blueprint.Tasks[index]
		oldStageID := task.StageID
		task.ID = taskIDMap[task.ID]
		if mapped := stageIDMap[oldStageID]; mapped != "" {
			task.StageID = mapped
		}
		task.Order = index + 1
		task.Title = strings.TrimSpace(task.Title)
		task.Objective = strings.TrimSpace(task.Objective)
		task.Description = strings.TrimSpace(task.Description)
		if task.EffortMinutes < 15 {
			task.EffortMinutes = 15
			warnings = append(warnings, "effort_clamped")
		} else if task.EffortMinutes > 720 {
			task.EffortMinutes = 720
			warnings = append(warnings, "effort_clamped")
		}
		switch strings.ToLower(strings.TrimSpace(task.Difficulty)) {
		case "easy", "简单", "beginner", "basic":
			task.Difficulty = "easy"
		case "hard", "困难", "advanced", "difficult":
			task.Difficulty = "hard"
		default:
			task.Difficulty = "medium"
		}
		prerequisites := make([]string, 0, len(task.PrerequisiteIDs))
		for _, prerequisite := range task.PrerequisiteIDs {
			if mapped := taskIDMap[prerequisite]; mapped != "" {
				prerequisites = append(prerequisites, mapped)
			}
		}
		task.PrerequisiteIDs = prerequisites
	}
	return blueprint, warnings
}

func ValidatePlanningBlueprintForInput(blueprint PlanningBlueprint, input PlanGenerationInput) error {
	if err := ValidatePlanningBlueprint(blueprint); err != nil {
		return err
	}
	capacity := input.Days * input.HoursPerDay * 60
	if capacity <= 0 {
		return fmt.Errorf("invalid requested plan capacity")
	}
	total := 0
	for _, task := range blueprint.Tasks {
		total += task.EffortMinutes
	}
	if total > capacity {
		return fmt.Errorf("blueprint effort %d exceeds requested capacity %d minutes", total, capacity)
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
		CapacityMinutes    int                `json:"capacity_minutes"`
	}{"planning_blueprint_v1", ctx.Input.Goal, ctx.Input.Days, ctx.Input.HoursPerDay, ctx.Input.Refinement, ctx.LearningProfile, ctx.ActivePlanLoad, ctx.RecentTaskOutcomes, ctx.Input.Days * ctx.Input.HoursPerDay * 60})
	return "请作为学习任务拆解专家输出简体中文 JSON 蓝图。你负责学习阶段、任务数量、目标、描述、难度、工作量、顺序和前置关系；不要输出日期、时间、数据库 ID 或用户隐私。任务必须可执行、目标不能只是重复标题。所有任务 effort_minutes 总和不得超过 capacity_minutes。每个批次最多 8 个阶段。stage.order 和 task.order 都必须从 1 开始全局连续递增，不能在新阶段重置。尽量使用 30-90 分钟的任务，30 天计划也必须输出完整且精炼的内容。仅输出一个完整 JSON 对象。" + ActivePromptPlaybookGuidance(context.Background()) + "\nNORMALIZED_DECOMPOSITION_BRIEF:\n" + string(brief)
}

func PlanningBlueprintTokenAllowance(input PlanGenerationInput) int {
	estimatedTasks := input.Days * 2
	if estimatedTasks < 4 {
		estimatedTasks = 4
	}
	if estimatedTasks > 60 {
		estimatedTasks = 60
	}
	allowance := 1024 + estimatedTasks*160
	if allowance > 8192 {
		return 8192
	}
	return allowance
}

type PlanningBlueprintCheckpoint struct {
	Version       int               `json:"version"`
	CompletedDays int               `json:"completed_days"`
	Blueprint     PlanningBlueprint `json:"blueprint"`
}

func GeneratePlanningBlueprintWithRepair(ctx context.Context, provider AIProvider, planningContext PlanningContext, maxAttempts int) (PlanningBlueprint, error) {
	return GeneratePlanningBlueprintWithCheckpoint(ctx, provider, planningContext, maxAttempts, nil, nil)
}

func GeneratePlanningBlueprintWithCheckpoint(ctx context.Context, provider AIProvider, planningContext PlanningContext, maxAttempts int, checkpoint *PlanningBlueprintCheckpoint, save func(PlanningBlueprintCheckpoint) error) (PlanningBlueprint, error) {
	if planningContext.Input.Days <= 10 {
		return generatePlanningBlueprintBatchWithRepair(ctx, provider, planningContext, maxAttempts)
	}
	completedDays := 0
	combined := PlanningBlueprint{}
	if checkpoint != nil && checkpoint.Version == 1 && checkpoint.CompletedDays > 0 && checkpoint.CompletedDays <= planningContext.Input.Days {
		completedDays = checkpoint.CompletedDays
		combined = checkpoint.Blueprint
	}
	if completedDays == planningContext.Input.Days {
		combined, _ = NormalizePlanningBlueprint(combined)
		if err := ValidatePlanningBlueprintForInput(combined, planningContext.Input); err != nil {
			return PlanningBlueprint{}, fmt.Errorf("validate completed blueprint checkpoint: %w", err)
		}
		return combined, nil
	}
	remaining := planningContext.Input.Days - completedDays
	batchIndex := completedDays / 10
	for remaining > 0 {
		batchIndex++
		batchDays := minInt(remaining, 10)
		batchContext := planningContext
		batchContext.Input.Days = batchDays
		batchContext.Input.Refinement = strings.TrimSpace(planningContext.Input.Refinement + fmt.Sprintf("\n这是完整 %d 天计划的第 %d 个连续批次；保持与前序批次递进，当前批次覆盖 %d 个学习日。", planningContext.Input.Days, batchIndex, batchDays))
		batch, err := generatePlanningBlueprintBatchWithRepair(ctx, provider, batchContext, maxAttempts)
		if err != nil {
			return PlanningBlueprint{}, fmt.Errorf("expand blueprint batch %d: %w", batchIndex, err)
		}
		if combined.Title == "" {
			combined.Title, combined.Summary, combined.Rationale = batch.Title, batch.Summary, batch.Rationale
		}
		stageMap := map[string]string{}
		for index := range batch.Stages {
			oldID := batch.Stages[index].ID
			batch.Stages[index].ID = fmt.Sprintf("b%d_%s", batchIndex, oldID)
			stageMap[oldID] = batch.Stages[index].ID
		}
		taskMap := map[string]string{}
		for index := range batch.Tasks {
			oldID := batch.Tasks[index].ID
			batch.Tasks[index].ID = fmt.Sprintf("b%d_%s", batchIndex, oldID)
			taskMap[oldID] = batch.Tasks[index].ID
		}
		for index := range batch.Tasks {
			batch.Tasks[index].StageID = stageMap[batch.Tasks[index].StageID]
			for prerequisiteIndex, prerequisite := range batch.Tasks[index].PrerequisiteIDs {
				batch.Tasks[index].PrerequisiteIDs[prerequisiteIndex] = taskMap[prerequisite]
			}
		}
		combined.Stages = append(combined.Stages, batch.Stages...)
		combined.Tasks = append(combined.Tasks, batch.Tasks...)
		remaining -= batchDays
		completedDays += batchDays
		if save != nil {
			checkpointBlueprint, _ := NormalizePlanningBlueprint(combined)
			if err := save(PlanningBlueprintCheckpoint{Version: 1, CompletedDays: completedDays, Blueprint: checkpointBlueprint}); err != nil {
				return PlanningBlueprint{}, fmt.Errorf("persist blueprint checkpoint: %w", err)
			}
		}
	}
	combined, _ = NormalizePlanningBlueprint(combined)
	if err := ValidatePlanningBlueprintForInput(combined, planningContext.Input); err != nil {
		return PlanningBlueprint{}, fmt.Errorf("combine blueprint batches: %w", err)
	}
	return combined, nil
}

func generatePlanningBlueprintBatchWithRepair(ctx context.Context, provider AIProvider, planningContext PlanningContext, maxAttempts int) (PlanningBlueprint, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if maxAttempts > 6 {
		maxAttempts = 6
	}
	prompt := BuildPlanningBlueprintPrompt(planningContext)
	tokens := PlanningBlueprintTokenAllowance(planningContext.Input)
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		raw, err := provider.GenerateContext(ctx, prompt, tokens)
		if err != nil {
			lastErr = err
			if !IsAIOutputTruncated(err) {
				return PlanningBlueprint{}, err
			}
		} else {
			blueprint, parseErr := ParsePlanningBlueprintJSON(raw)
			if parseErr == nil {
				blueprint, _ = NormalizePlanningBlueprint(blueprint)
				if validateErr := ValidatePlanningBlueprintForInput(blueprint, planningContext.Input); validateErr == nil {
					return blueprint, nil
				} else {
					lastErr = validateErr
				}
			} else {
				lastErr = fmt.Errorf("invalid JSON: %w", parseErr)
			}
		}
		if attempt == maxAttempts {
			break
		}
		RecordPromptPattern(context.Background(), ClassifyBlueprintFailure(lastErr))
		if tokens < 8192 {
			tokens = minInt(8192, tokens+1536)
		}
		prompt = BuildPlanningBlueprintPrompt(planningContext) + "\nREPAIR_REQUIRED: 上一次输出未通过校验：" + boundedBlueprintDiagnostic(lastErr) + "。请重新输出完整 JSON；只修复该问题，不要解释，不要使用 Markdown。"
	}
	return PlanningBlueprint{}, fmt.Errorf("provider did not produce a valid planning blueprint after %d attempts: %w", maxAttempts, lastErr)
}

func boundedBlueprintDiagnostic(err error) string {
	if err == nil {
		return "unknown output error"
	}
	message := strings.ReplaceAll(strings.TrimSpace(err.Error()), "\n", " ")
	if len(message) > 240 {
		message = message[:240]
	}
	return message
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
