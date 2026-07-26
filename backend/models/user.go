package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"

	AccountStatusActive   = "active"
	AccountStatusInactive = "inactive"
	AccountStatusDeleted  = "deleted"

	OnboardingStatusNotStarted = "not_started"
	OnboardingStatusCompleted  = "completed"
	OnboardingStatusSkipped    = "skipped"
	CurrentOnboardingVersion   = 1
	PermanentBanYear           = 2099
)

func PermanentBanUntil() time.Time {
	return time.Date(PermanentBanYear, 12, 31, 23, 59, 59, 0, time.UTC)
}

func IsPermanentBan(until *time.Time) bool {
	return until != nil && !until.Before(time.Date(PermanentBanYear, 1, 1, 0, 0, 0, 0, time.UTC))
}

type User struct {
	ID                    uint           `gorm:"primaryKey" json:"id"`
	OpenID                string         `gorm:"size:128" json:"openid,omitempty"`
	Username              string         `gorm:"size:24" json:"username,omitempty"`
	UsernameNormalized    string         `gorm:"size:24" json:"-"`
	Nickname              string         `gorm:"size:64" json:"nickname"`
	NicknameNormalized    string         `gorm:"size:64" json:"-"`
	InviteTargetID        string         `gorm:"size:32" json:"-"`
	AvatarURL             string         `gorm:"size:512" json:"avatar_url"`
	PasswordHash          *string        `gorm:"size:255" json:"-"`
	WeeklyHours           int            `gorm:"default:0" json:"weekly_hours"`
	SlackBalance          int            `gorm:"default:0" json:"slack_balance"`
	AccountStatus         string         `gorm:"size:16;default:active;not null" json:"account_status"`
	OnboardingStatus      string         `gorm:"size:16;default:not_started;not null" json:"onboarding_status"`
	OnboardingVersion     int            `gorm:"default:1;not null" json:"onboarding_version"`
	OnboardingCompletedAt *time.Time     `json:"onboarding_completed_at,omitempty"`
	SecurityVersion       int            `gorm:"default:1;not null" json:"-"`
	Role                  string         `gorm:"size:16;default:user;not null" json:"role"`
	BannedUntil           *time.Time     `json:"banned_until,omitempty"`
	BannedReason          string         `gorm:"size:256" json:"banned_reason,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }

type RegistrationInvite struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	CodeHash         string     `gorm:"size:64;uniqueIndex;not null" json:"-"`
	CodePrefix       string     `gorm:"size:12;not null" json:"code_prefix"`
	ExpiresAt        time.Time  `gorm:"index;not null" json:"expires_at"`
	UsedAt           *time.Time `gorm:"index" json:"used_at,omitempty"`
	UserID           *uint      `gorm:"index" json:"user_id,omitempty"`
	DisabledAt       *time.Time `gorm:"index" json:"disabled_at,omitempty"`
	CreatedAt        time.Time  `gorm:"index" json:"created_at"`
	CreatedByAdminID *uint      `gorm:"index" json:"created_by_admin_id,omitempty"`
}

func (RegistrationInvite) TableName() string { return "registration_invites" }

type PasswordResetCode struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	UserID           uint       `gorm:"index;not null" json:"user_id"`
	CodeHash         string     `gorm:"size:64;index;not null" json:"-"`
	ExpiresAt        time.Time  `gorm:"index;not null" json:"expires_at"`
	ConsumedAt       *time.Time `gorm:"index" json:"consumed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	CreatedByAdminID uint       `gorm:"index;not null" json:"created_by_admin_id"`
}

func (PasswordResetCode) TableName() string { return "password_reset_codes" }

type PlanActionLayout struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	Direct    []string  `gorm:"serializer:json" json:"direct"`
	Overflow  []string  `gorm:"serializer:json" json:"overflow"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PlanActionLayout) TableName() string { return "plan_action_layouts" }
