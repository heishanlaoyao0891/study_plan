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
	ID                  uint                   `gorm:"primaryKey" json:"id"`
	UserID              uint                   `gorm:"index;not null" json:"user_id"`
	Title               string                 `gorm:"size:128;not null" json:"title"`
	Description         string                 `gorm:"size:1024" json:"description"`
	Status              string                 `gorm:"size:16;default:active;not null" json:"status"`
	WeeklyTargetHours   int                    `gorm:"default:0" json:"weekly_target_hours"`
	StartDate           string                 `gorm:"size:10" json:"start_date,omitempty"`
	EndDate             string                 `gorm:"size:10" json:"end_date,omitempty"`
	DefaultPlannedStart string                 `gorm:"size:5;default:20:00" json:"default_planned_start"`
	DefaultPlannedEnd   string                 `gorm:"size:5;default:21:00" json:"default_planned_end"`
	StudyWeekdays       []int                  `gorm:"serializer:json" json:"study_weekdays"`
	StudyDates          []string               `gorm:"serializer:json" json:"study_dates"`
	PublicToGroup       bool                   `gorm:"default:false" json:"public_to_group"`
	AIGenerated         bool                   `gorm:"default:false" json:"ai_generated"`
	GenerationSource    string                 `gorm:"size:32;not null;default:''" json:"generation_source"`
	IsShared            bool                   `gorm:"default:false" json:"is_shared"`
	SortOrder           int                    `gorm:"default:0" json:"sort_order"`
	ScheduleOverrides   []PlanScheduleOverride `gorm:"foreignKey:PlanID" json:"schedule_overrides"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
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

type PlanScheduleOverride struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	PlanID       uint      `gorm:"index;not null" json:"plan_id"`
	Weekday      int       `gorm:"default:0" json:"weekday,omitempty"`
	Date         string    `gorm:"size:10" json:"date,omitempty"`
	PlannedStart string    `gorm:"size:5;not null" json:"planned_start"`
	PlannedEnd   string    `gorm:"size:5;not null" json:"planned_end"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (PlanScheduleOverride) TableName() string { return "plan_schedule_overrides" }

type AIPlanCommit struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"uniqueIndex:idx_ai_plan_commit_user_key;not null" json:"user_id"`
	IdempotencyKey string    `gorm:"uniqueIndex:idx_ai_plan_commit_user_key;size:64;not null" json:"idempotency_key"`
	PlanID         uint      `gorm:"index;not null" json:"plan_id"`
	PreviewHash    string    `gorm:"size:64;not null" json:"preview_hash"`
	CreatedAt      time.Time `json:"created_at"`
}

func (AIPlanCommit) TableName() string { return "ai_plan_commits" }

const (
	AIPlanJobStatusPending   = "pending"
	AIPlanJobStatusRunning   = "running"
	AIPlanJobStatusSucceeded = "succeeded"
	AIPlanJobStatusFailed    = "failed"
)

type AIPlanGenerationJob struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	UserID           uint       `gorm:"not null;index" json:"-"`
	RequestJSON      string     `gorm:"type:text;not null" json:"-"`
	RequestHash      string     `gorm:"size:64;not null" json:"-"`
	IdempotencyKey   string     `gorm:"size:64;not null;default:''" json:"-"`
	Status           string     `gorm:"size:16;not null;index;check:chk_ai_plan_job_status,status IN ('pending','running','succeeded','failed')" json:"status"`
	Phase            string     `gorm:"size:32;not null;default:'queued'" json:"phase"`
	AttemptCount     int        `gorm:"not null;default:0" json:"attempt_count"`
	CheckpointJSON   string     `gorm:"type:text;not null;default:'{}'" json:"-"`
	NextAttemptAt    *time.Time `gorm:"index" json:"next_attempt_at,omitempty"`
	LeaseOwner       string     `gorm:"size:64;not null;default:''" json:"-"`
	LeaseExpiresAt   *time.Time `gorm:"index" json:"-"`
	ResultPlanID     *uint      `gorm:"index" json:"result_plan_id,omitempty"`
	ErrorCode        string     `gorm:"size:48;not null;default:''" json:"error_code,omitempty"`
	ErrorMessage     string     `gorm:"size:256;not null;default:''" json:"error_message,omitempty"`
	GenerationSource string     `gorm:"size:32;not null;default:''" json:"generation_source,omitempty"`
	Provider         string     `gorm:"size:32;not null;default:''" json:"provider,omitempty"`
	ModelName        string     `gorm:"size:128;not null;default:''" json:"model,omitempty"`
	EnrichmentStatus string     `gorm:"size:32;not null;default:''" json:"enrichment_status,omitempty"`
	EnrichmentReason string     `gorm:"size:64;not null;default:''" json:"enrichment_reason,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	ExpiresAt        time.Time  `gorm:"index" json:"expires_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (AIPlanGenerationJob) TableName() string { return "ai_plan_generation_jobs" }
