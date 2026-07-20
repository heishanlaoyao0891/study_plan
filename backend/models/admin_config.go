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
	ID                    uint      `gorm:"primaryKey" json:"id"`
	Provider              string    `gorm:"size:32;default:mock;not null" json:"provider"`
	ModelName             string    `gorm:"size:128" json:"model_name"`
	BaseURL               string    `gorm:"size:512" json:"base_url"`
	RequestTimeoutSeconds int       `gorm:"default:30" json:"request_timeout_seconds"`
	DailyGenerationLimit  int       `gorm:"default:20" json:"daily_generation_limit"`
	Enabled               bool      `gorm:"default:true" json:"enabled"`
	APIKeyCiphertext      string    `gorm:"size:2048" json:"-"`
	APIKeyEncrypted       bool      `gorm:"default:false" json:"-"`
	UpdatedBy             *uint     `json:"updated_by,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (AIConfig) TableName() string { return "ai_configs" }

type SubscriptionMessageConfig struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	StudyStartTemplateID    string    `gorm:"size:128" json:"study_start_template_id"`
	CompletionTemplateID    string    `gorm:"size:128" json:"completion_template_id"`
	DecisionTemplateID      string    `gorm:"size:128" json:"decision_template_id"`
	MissedCheckinTemplateID string    `gorm:"size:128" json:"missed_checkin_template_id"`
	StudyStartEnabled       bool      `gorm:"default:true" json:"study_start_enabled"`
	CompletionEnabled       bool      `gorm:"default:true" json:"completion_enabled"`
	DecisionEnabled         bool      `gorm:"default:true" json:"decision_enabled"`
	MissedCheckinEnabled    bool      `gorm:"default:true" json:"missed_checkin_enabled"`
	UpdatedBy               *uint     `json:"updated_by,omitempty"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func (SubscriptionMessageConfig) TableName() string { return "subscription_message_configs" }

type NotificationDeliveryLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index" json:"user_id,omitempty"`
	ReminderType string    `gorm:"size:32;index;not null" json:"reminder_type"`
	Status       string    `gorm:"size:32;index;not null" json:"status"`
	Message      string    `gorm:"size:512" json:"message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func (NotificationDeliveryLog) TableName() string { return "notification_delivery_logs" }
