package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxBlueprintStages = 24
const maxBlueprintTasks = 120
const maxGeneratedPreviewTasks = 120
const planningBlueprintPromptVersion = "planning_blueprint_v2"
const planningBlueprintSystemContract = `You produce planning_blueprint_v2. Return exactly one JSON object and no Markdown, comments, prose, dates, time slots, persisted IDs, or private data. Use only this schema:
{"title":"string","summary":"string","rationale":"string","stages":[{"id":"stage_1","name":"string","objective":"string","order":1}],"tasks":[{"id":"task_1","stage_id":"stage_1","title":"string","objective":"specific action different from title","description":"string","effort_minutes":60,"difficulty":"easy|medium|hard","order":1,"prerequisite_ids":[]}]}`

var planningBlueprintOutputRules = []string{
	"只输出一个完整 JSON 对象，不要 Markdown、注释、前后说明或额外字段",
	"严格输出 title、summary、rationale、stages、tasks；stages 和 tasks 必须是非空数组",
	"stage.id 和 task.id 使用小写字母、数字、下划线或连字符，且各自唯一",
	"stage.order 与 task.order 都从 1 开始连续递增，跨阶段不得重置",
	"task.stage_id 必须引用已输出的 stage.id，prerequisite_ids 只能引用更早的 task.id",
	"task.objective 必须是可验收动作，不能与 task.title 相同",
	"difficulty 只能是 easy、medium、hard",
	"每个 effort_minutes 必须在 15 到 720 之间，所有任务总和不得超过 total_capacity_minutes",
	"任务颗粒度必须遵循 planning_strategy.granularity_rule，不能因容量有限而只输出学习目标的前半段",
	"start_date、available_time_slot、skip_dates 仅是只读排程约束，不得出现在输出任务字段中",
	"title 仅描述学习主题，不得包含计划天数、日期、时间段或安排跨度；后端会在排程完成后写入实际安排跨度",
}

var planTitleDurationPattern = regexp.MustCompile(`[0-9]+\s*天`)

var planningBlueprintOutputSchema = json.RawMessage(`{"title":"string","summary":"string","rationale":"string","stages":[{"id":"stage_1","name":"string","objective":"string","order":1}],"tasks":[{"id":"task_1","stage_id":"stage_1","title":"string","objective":"specific action different from title","description":"string","effort_minutes":60,"difficulty":"easy|medium|hard","order":1,"prerequisite_ids":[]}]}`)

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
	_, capacity := planningCapacityMinutes(input)
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
	dailyCapacity, totalCapacity := planningCapacityMinutes(ctx.Input)
	scope := resolvedPlanningPromptScope(ctx)
	granularity, granularityRule := planningGranularity(scope.TotalPlanLearningDays)
	batchRole, batchRule := planningBatchStrategy(scope)
	type planningStrategy struct {
		CoveragePriority string `json:"coverage_priority"`
		Granularity      string `json:"granularity"`
		BatchRole        string `json:"batch_role"`
		CoverageRule     string `json:"coverage_rule"`
		GranularityRule  string `json:"granularity_rule"`
		BatchRule        string `json:"batch_rule"`
	}
	type previousProgress struct {
		PreviousStageNames []string `json:"previous_stage_names"`
		RecentTaskTitles   []string `json:"recent_task_titles"`
	}
	brief, _ := json.Marshal(struct {
		Contract                      string             `json:"contract"`
		Goal                          string             `json:"goal"`
		RequestedLearningDays         int                `json:"requested_learning_days"`
		HoursPerDay                   int                `json:"hours_per_day"`
		StartDate                     string             `json:"start_date"`
		AvailableTimeSlot             string             `json:"available_time_slot"`
		SkipDates                     []string           `json:"skip_dates"`
		Refinement                    string             `json:"refinement,omitempty"`
		EffectiveDailyCapacityMinutes int                `json:"effective_daily_capacity_minutes"`
		TotalCapacityMinutes          int                `json:"total_capacity_minutes"`
		TotalPlanLearningDays         int                `json:"total_plan_learning_days"`
		BatchIndex                    int                `json:"batch_index"`
		CompletedLearningDays         int                `json:"completed_learning_days"`
		BatchLearningDays             int                `json:"batch_learning_days"`
		LearningProfile               LearningProfile    `json:"learning_profile"`
		ActivePlanLoad                ActivePlanLoad     `json:"active_plan_load"`
		RecentTaskOutcomes            RecentTaskOutcomes `json:"recent_task_outcomes"`
		PreviousProgress              previousProgress   `json:"previous_progress"`
		OutputSchema                  json.RawMessage    `json:"output_schema"`
		PlanningStrategy              planningStrategy   `json:"planning_strategy"`
		OutputRules                   []string           `json:"output_rules"`
	}{
		Contract:                      planningBlueprintPromptVersion,
		Goal:                          ctx.Input.Goal,
		RequestedLearningDays:         ctx.Input.Days,
		HoursPerDay:                   ctx.Input.HoursPerDay,
		StartDate:                     ctx.Input.StartDate,
		AvailableTimeSlot:             ctx.Input.AvailableTimeSlot,
		SkipDates:                     append([]string{}, ctx.Input.SkipDates...),
		Refinement:                    ctx.Input.Refinement,
		EffectiveDailyCapacityMinutes: dailyCapacity,
		TotalCapacityMinutes:          totalCapacity,
		TotalPlanLearningDays:         scope.TotalPlanLearningDays,
		BatchIndex:                    scope.BatchIndex,
		CompletedLearningDays:         scope.CompletedLearningDays,
		BatchLearningDays:             scope.BatchLearningDays,
		LearningProfile:               ctx.LearningProfile,
		ActivePlanLoad:                ctx.ActivePlanLoad,
		RecentTaskOutcomes:            ctx.RecentTaskOutcomes,
		PreviousProgress: previousProgress{
			PreviousStageNames: append([]string{}, scope.PreviousStageNames...),
			RecentTaskTitles:   append([]string{}, scope.RecentTaskTitles...),
		},
		OutputSchema: planningBlueprintOutputSchema,
		PlanningStrategy: planningStrategy{
			CoveragePriority: "complete_learning_arc_before_detail",
			Granularity:      granularity,
			BatchRole:        batchRole,
			CoverageRule:     "必须先覆盖从基础、练习、综合应用到复盘验收的完整学习闭环；容量不足时压缩每个阶段的深度并合并任务，绝不能只生成前半段。",
			GranularityRule:  granularityRule,
			BatchRule:        batchRule,
		},
		OutputRules: planningBlueprintOutputRules,
	})
	return "你是学习计划后台 Agent 的任务拆解器。请严格依据下面已经归一化的参数生成简体中文学习蓝图；后端会负责日期和时间排程，你只负责阶段、任务内容、工作量、顺序与前置关系。不得猜测、忽略或改写用户约束。" + ActivePromptPlaybookGuidance(context.Background()) + "\nPLANNING_BLUEPRINT_REQUEST_V2:\n" + string(brief)
}

func resolvedPlanningPromptScope(ctx PlanningContext) PlanningPromptScope {
	scope := ctx.PromptScope
	if scope.TotalPlanLearningDays <= 0 {
		scope.TotalPlanLearningDays = ctx.Input.Days
	}
	if scope.BatchLearningDays <= 0 {
		scope.BatchLearningDays = ctx.Input.Days
	}
	if scope.BatchIndex <= 0 {
		scope.BatchIndex = 1
	}
	return scope
}

func planningGranularity(totalPlanDays int) (string, string) {
	switch {
	case totalPlanDays <= 7:
		return "coarse_complete", "短周期计划应使用较少、较大的端到端任务，优先合并同类知识点；通常每项 60 到 120 分钟，并确保最后包含综合应用和验收。"
	case totalPlanDays <= 14:
		return "balanced", "中等周期计划使用平衡颗粒度，通常每项 45 到 90 分钟，在完整学习闭环内安排阶段练习和复盘。"
	default:
		return "fine_grained", "长周期计划可以细化知识点、练习和里程碑，通常每项 30 到 60 分钟，但每个批次仍须连续推进并服务于完整学习闭环。"
	}
}

func planningBatchStrategy(scope PlanningPromptScope) (string, string) {
	totalBatches := (scope.TotalPlanLearningDays + 9) / 10
	if totalBatches <= 1 {
		return "complete", "当前请求是单批计划，本批必须覆盖完整学习闭环。"
	}
	if scope.BatchIndex <= 1 {
		return "foundation", "这是长计划首批：建立必要基础并尽快进入可执行练习，为后续批次留下综合应用和复盘空间。"
	}
	if scope.BatchIndex >= totalBatches {
		return "completion", "这是长计划末批：不要重复前序基础；完成综合应用、查漏补缺、复盘和最终验收。"
	}
	return "progression", "这是长计划中间批：根据 previous_progress 衔接前序内容，推进更深入的练习、应用和阶段里程碑，不要从基础重新开始。"
}

func planningCapacityMinutes(input PlanGenerationInput) (int, int) {
	daily := input.HoursPerDay * 60
	if start, end, err := parsePlanningSlot(input.AvailableTimeSlot); err == nil && end-start < daily {
		daily = end - start
	}
	if daily < 0 {
		daily = 0
	}
	return daily, daily * input.Days
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
		return generatePlanningBlueprintBatchWithRepair(ctx, provider, planningContext, maxAttempts, 1)
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
		previousStageNames := make([]string, 0, minInt(len(combined.Stages), 12))
		startStage := len(combined.Stages) - 12
		if startStage < 0 {
			startStage = 0
		}
		for _, stage := range combined.Stages[startStage:] {
			previousStageNames = append(previousStageNames, stage.Name)
		}
		recentTaskTitles := make([]string, 0, minInt(len(combined.Tasks), 12))
		startTask := len(combined.Tasks) - 12
		if startTask < 0 {
			startTask = 0
		}
		for _, task := range combined.Tasks[startTask:] {
			recentTaskTitles = append(recentTaskTitles, task.Title)
		}
		batchContext.PromptScope = PlanningPromptScope{
			TotalPlanLearningDays: planningContext.Input.Days,
			BatchIndex:            batchIndex,
			CompletedLearningDays: completedDays,
			BatchLearningDays:     batchDays,
			PreviousStageNames:    previousStageNames,
			RecentTaskTitles:      recentTaskTitles,
		}
		batch, err := generatePlanningBlueprintBatchWithRepair(ctx, provider, batchContext, maxAttempts, batchIndex)
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

func generatePlanningBlueprintBatchWithRepair(ctx context.Context, provider AIProvider, planningContext PlanningContext, maxAttempts, batchIndex int) (PlanningBlueprint, error) {
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
		phase := "decomposing"
		if attempt > 1 {
			phase = "repairing"
		}
		attemptContext := WithAIInvocationStep(ctx, phase, batchIndex, attempt)
		raw, err := provider.GenerateContext(attemptContext, prompt, tokens)
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
	// A requested plan length is a learning cadence, not a packing hint. When
	// the model supplies enough tasks, reserve one task for every learning day
	// before filling remaining free capacity on those days.
	spreadAcrossRequestedDays := len(blueprint.Tasks) >= ctx.Input.Days
	reservedDays := 0
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
				if spreadAcrossRequestedDays && reservedDays < ctx.Input.Days {
					reservedDays++
					nextOffset++
					if reservedDays == ctx.Input.Days {
						nextOffset = 0
					}
				}
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
	preview := PlanPreview{Title: scheduledPlanTitle(blueprint.Title, tasks), Summary: strings.TrimSpace(blueprint.Summary), Rationale: strings.TrimSpace(blueprint.Rationale), Tasks: tasks}
	recomputePlanTotals(&preview)
	return preview, warnings, ValidatePlanPreview(preview, ctx.Input)
}

func scheduledPlanTitle(rawTitle string, tasks []PlanPreviewTask) string {
	title := strings.TrimSpace(planTitleDurationPattern.ReplaceAllString(rawTitle, ""))
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		title = "学习计划"
	}
	if !strings.Contains(title, "计划") {
		title += " 学习计划"
	}
	if len(tasks) == 0 {
		return title
	}
	first, firstErr := time.Parse(aiPlanDateLayout, tasks[0].Date)
	last, lastErr := time.Parse(aiPlanDateLayout, tasks[len(tasks)-1].Date)
	if firstErr != nil || lastErr != nil || last.Before(first) {
		return title
	}
	spanDays := int(last.Sub(first).Hours()/24) + 1
	return fmt.Sprintf("%s（安排跨度 %d 天）", title, spanDays)
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
