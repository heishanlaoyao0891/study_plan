package models

import "time"

const (
	PlanningJobStatusQueued      = "queued"
	PlanningJobStatusDecomposing = "decomposing"
	PlanningJobStatusScheduling  = "scheduling"
	PlanningJobStatusReady       = "ready"
	PlanningJobStatusFallback    = "fallback"
	PlanningJobStatusCancelled   = "cancelled"
	PlanningJobStatusExpired     = "expired"
)

type PlanningPreviewVersion struct {
	ID                 uint       `gorm:"primaryKey" json:"-"`
	PreviewID          string     `gorm:"size:32;not null;uniqueIndex:idx_planning_preview_version" json:"preview_id"`
	Version            int        `gorm:"not null;uniqueIndex:idx_planning_preview_version" json:"version"`
	UserID             uint       `gorm:"not null;index" json:"-"`
	Source             string     `gorm:"size:32;not null" json:"source"`
	ContextFingerprint string     `gorm:"size:64;not null;index" json:"context_fingerprint"`
	ParentVersion      *int       `json:"parent_version,omitempty"`
	PreviewJSON        string     `gorm:"type:text;not null" json:"-"`
	InputJSON          string     `gorm:"type:text;not null;default:'{}'" json:"-"`
	ProvenanceToken    string     `gorm:"type:text;not null" json:"-"`
	CommittedAt        *time.Time `gorm:"index" json:"committed_at,omitempty"`
	ExpiresAt          time.Time  `gorm:"not null;index" json:"expires_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

func (PlanningPreviewVersion) TableName() string { return "planning_preview_versions" }

type PlanningJob struct {
	ID                      string     `gorm:"primaryKey;size:32" json:"id"`
	UserID                  uint       `gorm:"not null;index" json:"-"`
	RequestFingerprint      string     `gorm:"size:64;not null;index" json:"-"`
	Status                  string     `gorm:"size:24;not null;index" json:"status"`
	Phase                   string     `gorm:"size:24;not null" json:"phase"`
	Provider                string     `gorm:"size:32" json:"provider,omitempty"`
	ModelName               string     `gorm:"size:128" json:"model,omitempty"`
	BaselinePreviewID       string     `gorm:"size:32;not null;index" json:"preview_id"`
	BaselinePreviewVersion  int        `gorm:"not null" json:"baseline_version"`
	ResultPreviewVersion    *int       `json:"result_version,omitempty"`
	RequestJSON             string     `gorm:"type:text;not null" json:"-"`
	AttemptCount            int        `gorm:"not null;default:0" json:"attempt_count"`
	MaxAttempts             int        `gorm:"not null;default:2" json:"max_attempts"`
	LeaseOwner              string     `gorm:"size:64" json:"-"`
	LeaseExpiresAt          *time.Time `gorm:"index" json:"-"`
	CancelRequestedAt       *time.Time `json:"-"`
	FailureReason           string     `gorm:"size:64" json:"failure_reason,omitempty"`
	PhaseTimingsJSON        string     `gorm:"type:text" json:"-"`
	ProviderLatencyMS       int64      `gorm:"not null;default:0;index" json:"provider_latency_ms"`
	PromptTokens            int        `gorm:"not null;default:0" json:"prompt_tokens"`
	CompletionTokens        int        `gorm:"not null;default:0" json:"completion_tokens"`
	TotalTokens             int        `gorm:"not null;default:0" json:"total_tokens"`
	BackgroundBudgetSeconds int        `gorm:"not null;default:60" json:"background_budget_seconds"`
	StartedAt               *time.Time `json:"started_at,omitempty"`
	FinishedAt              *time.Time `json:"finished_at,omitempty"`
	ExpiresAt               time.Time  `gorm:"not null;index" json:"expires_at"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

func (PlanningJob) TableName() string { return "planning_jobs" }
