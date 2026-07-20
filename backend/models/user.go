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
)

type User struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	OpenID          string         `gorm:"uniqueIndex;size:128;not null" json:"openid"`
	Nickname        string         `gorm:"size:64" json:"nickname"`
	AvatarURL       string         `gorm:"size:512" json:"avatar_url"`
	PhoneNumber     string         `gorm:"size:32" json:"phone_number,omitempty"`
	PhoneVerifiedAt *time.Time     `json:"phone_verified_at,omitempty"`
	WeeklyHours     int            `gorm:"default:0" json:"weekly_hours"`
	SlackBalance    int            `gorm:"default:0" json:"slack_balance"`
	AccountStatus   string         `gorm:"size:16;default:active;not null" json:"account_status"`
	Role            string         `gorm:"size:16;default:user;not null" json:"role"`
	BannedUntil     *time.Time     `json:"banned_until,omitempty"`
	BannedReason    string         `gorm:"size:256" json:"banned_reason,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }
