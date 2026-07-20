package models

import (
	"time"
)

const (
	PlanStatusActive   = "active"
	PlanStatusPaused   = "paused"
	PlanStatusArchived = "archived"
)

type Plan struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"index;not null" json:"user_id"`
	Title             string    `gorm:"size:128;not null" json:"title"`
	Description       string    `gorm:"size:1024" json:"description"`
	Status            string    `gorm:"size:16;default:active;not null" json:"status"`
	WeeklyTargetHours int       `gorm:"default:0" json:"weekly_target_hours"`
	StartDate         string    `gorm:"size:10" json:"start_date,omitempty"`
	EndDate           string    `gorm:"size:10" json:"end_date,omitempty"`
	PublicToGroup     bool      `gorm:"default:false" json:"public_to_group"`
	AIGenerated       bool      `gorm:"default:false" json:"ai_generated"`
	IsShared          bool      `gorm:"default:false" json:"is_shared"`
	SortOrder         int       `gorm:"default:0" json:"sort_order"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (Plan) TableName() string { return "plans" }

type PlanMember struct {
	ID       uint      `gorm:"primaryKey" json:"id"`
	PlanID   uint      `gorm:"index;not null" json:"plan_id"`
	UserID   uint      `gorm:"index;not null" json:"user_id"`
	Role     string    `gorm:"size:16;default:member;not null" json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

func (PlanMember) TableName() string { return "plan_members" }
