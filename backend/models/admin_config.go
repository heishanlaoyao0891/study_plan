package models

import "time"

type AdminAuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AdminUserID  uint      `gorm:"index;not null" json:"admin_user_id"`
	TargetUserID *uint     `gorm:"index" json:"target_user_id,omitempty"`
	ActionType   string    `gorm:"size:64;index;not null" json:"action_type"`
	Reason       string    `gorm:"size:512" json:"reason,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func (AdminAuditLog) TableName() string { return "admin_audit_logs" }

type AIConfig struct {
	ID                          uint      `gorm:"primaryKey" json:"id"`
	Provider                    string    `gorm:"size:32;default:mock;not null" json:"provider"`
	ModelName                   string    `gorm:"size:128" json:"model_name"`
	BaseURL                     string    `gorm:"size:512" json:"base_url"`
	RequestTimeoutSeconds       int       `gorm:"default:30" json:"request_timeout_seconds"`
	InteractiveTargetSeconds    int       `gorm:"default:2" json:"interactive_target_seconds"`
	BackgroundJobTimeoutSeconds int       `gorm:"default:300" json:"background_job_timeout_seconds"`
	DailyGenerationLimit        int       `gorm:"default:5" json:"daily_generation_limit"`
	Enabled                     bool      `gorm:"default:true" json:"enabled"`
	APIKeyCiphertext            string    `gorm:"size:2048" json:"-"`
	APIKeyEncrypted             bool      `gorm:"default:false" json:"-"`
	UpdatedBy                   *uint     `json:"updated_by,omitempty"`
	CreatedAt                   time.Time `json:"created_at"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

func (AIConfig) TableName() string { return "ai_configs" }

type SubscriptionMessageConfig struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	StudyStartTemplateID    string    `gorm:"size:128" json:"study_start_template_id"`
	CompletionTemplateID    string    `gorm:"size:128" json:"completion_template_id"`
	DecisionTemplateID      string    `gorm:"size:128" json:"decision_template_id"`
	MissedCheckinTemplateID string    `gorm:"size:128" json:"missed_checkin_template_id"`
	GroupNudgeTemplateID    string    `gorm:"size:128" json:"group_nudge_template_id"`
	SlackBalanceTemplateID  string    `gorm:"size:128" json:"slack_balance_template_id"`
	StudyStartEnabled       bool      `json:"study_start_enabled"`
	CompletionEnabled       bool      `json:"completion_enabled"`
	DecisionEnabled         bool      `json:"decision_enabled"`
	MissedCheckinEnabled    bool      `json:"missed_checkin_enabled"`
	GroupNudgeEnabled       bool      `json:"group_nudge_enabled"`
	SlackBalanceEnabled     bool      `json:"slack_balance_enabled"`
	StudyStartPage          string    `gorm:"size:256" json:"study_start_page"`
	CompletionPage          string    `gorm:"size:256" json:"completion_page"`
	DecisionPage            string    `gorm:"size:256" json:"decision_page"`
	MissedCheckinPage       string    `gorm:"size:256" json:"missed_checkin_page"`
	GroupNudgePage          string    `gorm:"size:256" json:"group_nudge_page"`
	SlackBalancePage        string    `gorm:"size:256" json:"slack_balance_page"`
	StudyStartFieldMapping  string    `gorm:"type:text" json:"study_start_field_mapping"`
	CompletionFieldMapping  string    `gorm:"type:text" json:"completion_field_mapping"`
	DecisionFieldMapping    string    `gorm:"type:text" json:"decision_field_mapping"`
	MissedCheckinMapping    string    `gorm:"type:text" json:"missed_checkin_field_mapping"`
	GroupNudgeFieldMapping  string    `gorm:"type:text" json:"group_nudge_field_mapping"`
	SlackBalanceMapping     string    `gorm:"type:text" json:"slack_balance_field_mapping"`
	UpdatedBy               *uint     `json:"updated_by,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func (SubscriptionMessageConfig) TableName() string { return "subscription_message_configs" }

type NotificationDeliveryLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	EventKey     string    `gorm:"size:191;index" json:"event_key"`
	UserID       uint      `gorm:"index" json:"user_id,omitempty"`
	ReminderType string    `gorm:"size:32;index;not null" json:"reminder_type"`
	Status       string    `gorm:"size:32;index;not null" json:"status"`
	Message      string    `gorm:"size:512" json:"message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (NotificationDeliveryLog) TableName() string { return "notification_delivery_logs" }

type NotificationSubscription struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"uniqueIndex:idx_user_reminder;not null" json:"user_id"`
	ReminderType string    `gorm:"uniqueIndex:idx_user_reminder;size:32;not null" json:"reminder_type"`
	TemplateID   string    `gorm:"size:128;not null;default:''" json:"template_id"`
	Subscribed   bool      `gorm:"default:true" json:"subscribed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (NotificationSubscription) TableName() string { return "notification_subscriptions" }

type AIGenerationUsage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	Provider    string    `gorm:"size:32" json:"provider"`
	Status      string    `gorm:"size:32;index;not null" json:"status"`
	ReferenceID string    `gorm:"size:64;index;not null;default:''" json:"reference_id,omitempty"`
	Message     string    `gorm:"size:512" json:"message,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (AIGenerationUsage) TableName() string { return "ai_generation_usage" }

type AIPromptPattern struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	PatternKey string    `gorm:"uniqueIndex;size:64;not null" json:"pattern_key"`
	Version    int       `gorm:"not null;default:1" json:"version"`
	Count      int64     `gorm:"not null;default:0" json:"count"`
	Guidance   string    `gorm:"size:512;not null" json:"guidance"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (AIPromptPattern) TableName() string { return "ai_prompt_patterns" }

type OpsContent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Kind      string    `gorm:"uniqueIndex;size:32;not null" json:"kind"`
	Title     string    `gorm:"size:128;not null" json:"title"`
	Body      string    `gorm:"type:text" json:"body"`
	UpdatedBy *uint     `json:"updated_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (OpsContent) TableName() string { return "ops_contents" }

type FeedbackReport struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"index;not null" json:"user_id"`
	Category       string     `gorm:"size:32;not null" json:"category"`
	Content        string     `gorm:"size:1024;not null" json:"content"`
	Contact        string     `gorm:"size:128" json:"contact"`
	Status         string     `gorm:"size:24;default:open;index" json:"status"`
	PublicResponse *string    `gorm:"type:text" json:"public_response,omitempty"`
	RespondedAt    *time.Time `json:"responded_at,omitempty"`
	RespondedBy    *uint      `gorm:"index" json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (FeedbackReport) TableName() string { return "feedback_reports" }

type AccountEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	EventType string    `gorm:"size:32;index;not null" json:"event_type"`
	Detail    string    `gorm:"size:512" json:"detail,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (AccountEvent) TableName() string { return "account_events" }
