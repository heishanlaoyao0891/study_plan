package handlers

import (
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"study_plan_backend/models"
)

const maxCoveredMinutesExclusive = 60

type coveredInterval struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type invalidScheduleTask struct {
	TaskID           uint                   `json:"task_id"`
	PlanID           uint                   `json:"plan_id"`
	PlanTitle        string                 `json:"plan_title"`
	Title            string                 `json:"title"`
	Date             string                 `json:"date"`
	CoveredMinutes   int                    `json:"covered_minutes"`
	CoveredIntervals []coveredInterval      `json:"covered_intervals"`
	ConflictingTasks []scheduleConflictTask `json:"conflicting_tasks"`
}

type scheduleConflictTask struct {
	TaskID    uint   `json:"task_id"`
	PlanID    uint   `json:"plan_id"`
	PlanTitle string `json:"plan_title"`
	Title     string `json:"title"`
	Start     string `json:"start"`
	End       string `json:"end"`
}

type scheduleConflictError struct {
	InvalidTasks []invalidScheduleTask `json:"invalid_tasks"`
}

func (e *scheduleConflictError) Error() string { return "planned schedule overlap reaches 60 minutes" }

func (e *scheduleConflictError) Metadata() map[string]interface{} {
	return map[string]interface{}{
		"invalid_tasks":                 e.InvalidTasks,
		"max_covered_minutes_exclusive": maxCoveredMinutesExclusive,
	}
}

type minuteInterval struct{ start, end int }

func validateScheduleTasks(tasks []models.DailyTask) error {
	return validateScheduleTasksWithPlanTitles(tasks, nil)
}

func validateScheduleTasksWithPlanTitles(tasks []models.DailyTask, planTitles map[uint]string) error {
	byDate := map[string][]models.DailyTask{}
	for _, task := range tasks {
		if plannedRangeMinutes(task.PlannedStart, task.PlannedEnd) > 0 {
			byDate[task.Date] = append(byDate[task.Date], task)
		}
	}
	invalid := make([]invalidScheduleTask, 0)
	for date, rows := range byDate {
		for index, task := range rows {
			taskStart, _ := minuteOfDay(task.PlannedStart)
			taskEnd, _ := minuteOfDay(task.PlannedEnd)
			covered := make([]minuteInterval, 0)
			conflicting := make([]scheduleConflictTask, 0)
			for otherIndex, other := range rows {
				if index == otherIndex {
					continue
				}
				otherStart, _ := minuteOfDay(other.PlannedStart)
				otherEnd, _ := minuteOfDay(other.PlannedEnd)
				start, end := maxIntValue(taskStart, otherStart), minIntValue(taskEnd, otherEnd)
				if end > start {
					covered = append(covered, minuteInterval{start: start, end: end})
					conflicting = append(conflicting, scheduleConflictTask{TaskID: other.ID, PlanID: other.PlanID, PlanTitle: schedulePlanTitle(other, planTitles), Title: other.Title, Start: formatMinute(start), End: formatMinute(end)})
				}
			}
			merged := mergeMinuteIntervals(covered)
			minutes := 0
			intervals := make([]coveredInterval, 0, len(merged))
			for _, interval := range merged {
				minutes += interval.end - interval.start
				intervals = append(intervals, coveredInterval{Start: formatMinute(interval.start), End: formatMinute(interval.end)})
			}
			if minutes >= maxCoveredMinutesExclusive {
				invalid = append(invalid, invalidScheduleTask{TaskID: task.ID, PlanID: task.PlanID, PlanTitle: schedulePlanTitle(task, planTitles), Title: task.Title, Date: date, CoveredMinutes: minutes, CoveredIntervals: intervals, ConflictingTasks: conflicting})
			}
		}
	}
	if len(invalid) > 0 {
		sort.Slice(invalid, func(i, j int) bool {
			if invalid[i].Date != invalid[j].Date {
				return invalid[i].Date < invalid[j].Date
			}
			return invalid[i].TaskID < invalid[j].TaskID
		})
		return &scheduleConflictError{InvalidTasks: invalid}
	}
	return nil
}

func schedulePlanTitle(task models.DailyTask, planTitles map[uint]string) string {
	if title := planTitles[task.PlanID]; title != "" {
		return title
	}
	return task.Title
}

func validateScheduleMutation(tx *gorm.DB, uid uint, proposed []models.DailyTask) error {
	if len(proposed) == 0 {
		return nil
	}
	dates := make([]string, 0)
	seenDates := map[string]bool{}
	replaced := map[uint]bool{}
	for _, task := range proposed {
		if !seenDates[task.Date] {
			dates = append(dates, task.Date)
			seenDates[task.Date] = true
		}
		if task.ID != 0 {
			replaced[task.ID] = true
		}
	}
	var persisted []models.DailyTask
	if err := tx.Where("user_id = ? AND date IN ?", uid, dates).Find(&persisted).Error; err != nil {
		return err
	}
	result := make([]models.DailyTask, 0, len(persisted)+len(proposed))
	for _, task := range persisted {
		if !replaced[task.ID] {
			result = append(result, task)
		}
	}
	result = append(result, proposed...)
	planTitles := map[uint]string{}
	planIDs := make([]uint, 0)
	seenPlanIDs := map[uint]bool{}
	for _, task := range result {
		if task.PlanID == 0 || seenPlanIDs[task.PlanID] {
			continue
		}
		seenPlanIDs[task.PlanID] = true
		planIDs = append(planIDs, task.PlanID)
	}
	if len(planIDs) > 0 {
		var plans []models.Plan
		if err := tx.Select("id", "title").Where("id IN ?", planIDs).Find(&plans).Error; err != nil {
			return err
		}
		for _, plan := range plans {
			planTitles[plan.ID] = plan.Title
		}
	}
	return validateScheduleTasksWithPlanTitles(result, planTitles)
}

func minuteOfDay(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func mergeMinuteIntervals(intervals []minuteInterval) []minuteInterval {
	if len(intervals) == 0 {
		return nil
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i].start < intervals[j].start })
	merged := []minuteInterval{intervals[0]}
	for _, interval := range intervals[1:] {
		last := &merged[len(merged)-1]
		if interval.start <= last.end {
			if interval.end > last.end {
				last.end = interval.end
			}
			continue
		}
		merged = append(merged, interval)
	}
	return merged
}

func formatMinute(value int) string { return fmt.Sprintf("%02d:%02d", value/60, value%60) }
func minIntValue(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxIntValue(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func respondScheduleError(c interface{ JSON(int, interface{}) }, err error) bool {
	conflict, ok := err.(*scheduleConflictError)
	if !ok {
		return false
	}
	c.JSON(409, map[string]interface{}{"code": 409, "message": conflict.Error(), "data": conflict.Metadata()})
	return true
}
