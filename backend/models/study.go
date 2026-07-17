package models

import "time"

const (
	TaskStatusPending    = "pending"
	TaskStatusInProgress = "in_progress"
	TaskStatusCompleted  = "completed"
)

type DailyTask struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	PlanID       uint       `gorm:"index;not null" json:"plan_id"`
	UserID       uint       `gorm:"index;not null" json:"user_id"`
	Date         string     `gorm:"size:10;not null;index" json:"date"`
	Title        string     `gorm:"size:128;not null" json:"title"`
	Description  string     `gorm:"size:1024" json:"description"`
	Status       string     `gorm:"size:24;default:pending;not null" json:"status"`
	ActualStart  *time.Time `json:"actual_start,omitempty"`
	ActualEnd    *time.Time `json:"actual_end,omitempty"`
	StudyMinutes int        `gorm:"default:0" json:"study_minutes"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (DailyTask) TableName() string { return "daily_tasks" }

type StudySession struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	TaskID      uint       `gorm:"index;not null" json:"task_id"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	DurationMin int        `gorm:"default:0" json:"duration_min"`
	Note        string     `gorm:"size:256" json:"note"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (StudySession) TableName() string { return "study_sessions" }
