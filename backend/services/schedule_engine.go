package services

import (
	"fmt"
	"sort"
	"time"
)

func PlanningLoadWarnings(activePlanCount, existingWeeklyHours, newWeeklyHours, maxActivePlans, maxWeeklyHours int) []string {
	warnings := []string{}
	if activePlanCount >= maxActivePlans {
		warnings = append(warnings, fmt.Sprintf("已有 %d 个活跃计划，不建议同时进行过多学习计划", activePlanCount))
	}
	total := existingWeeklyHours + newWeeklyHours
	if total > maxWeeklyHours {
		warnings = append(warnings, fmt.Sprintf("所有计划每周总学时已达 %d 小时，压力可能过大", total))
	}
	return warnings
}

type ScheduleInterval struct {
	Start int
	End   int
}

func ParseScheduleRange(startValue, endValue string) (ScheduleInterval, error) {
	start, startErr := time.Parse("15:04", startValue)
	end, endErr := time.Parse("15:04", endValue)
	if startErr != nil || endErr != nil || !end.After(start) {
		return ScheduleInterval{}, fmt.Errorf("schedule range must be a same-day increasing HH:mm-HH:mm range")
	}
	return ScheduleInterval{Start: start.Hour()*60 + start.Minute(), End: end.Hour()*60 + end.Minute()}, nil
}

func MergeScheduleIntervals(intervals []ScheduleInterval) []ScheduleInterval {
	if len(intervals) == 0 {
		return nil
	}
	rows := append([]ScheduleInterval(nil), intervals...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Start < rows[j].Start })
	merged := []ScheduleInterval{rows[0]}
	for _, interval := range rows[1:] {
		last := &merged[len(merged)-1]
		if interval.Start <= last.End {
			if interval.End > last.End {
				last.End = interval.End
			}
			continue
		}
		merged = append(merged, interval)
	}
	return merged
}

func FirstFreeScheduleSlot(availableStart, availableEnd, duration int, occupied []ScheduleInterval) (int, bool) {
	rows := MergeScheduleIntervals(occupied)
	candidate := availableStart
	for _, interval := range rows {
		if interval.End <= candidate || interval.Start >= availableEnd {
			continue
		}
		if candidate+duration <= interval.Start {
			return candidate, true
		}
		if interval.End > candidate {
			candidate = interval.End
		}
	}
	return candidate, candidate+duration <= availableEnd
}

func ScheduleIntervalsOverlap(left, right ScheduleInterval) bool {
	return left.Start < right.End && right.Start < left.End
}
