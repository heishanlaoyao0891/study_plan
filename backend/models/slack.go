package models

import "time"

type SlackConfig struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          *uint     `gorm:"index" json:"user_id,omitempty"`
	CheckinMinutes  int       `gorm:"default:10" json:"checkin_minutes"`
	MakeupCostRatio float64   `gorm:"default:1" json:"makeup_cost_ratio"`
	StreakBonus     int       `gorm:"default:0" json:"streak_bonus"`
	QualityBonus    int       `gorm:"default:0" json:"quality_bonus"`
	UpdatedBy       *uint     `json:"updated_by,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (SlackConfig) TableName() string { return "slack_configs" }

type SlackRecord struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"index;not null" json:"user_id"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	DurationMin int        `gorm:"default:0" json:"duration_min"`
	DeltaMin    int        `gorm:"default:0" json:"delta_min"`
	Activity    string     `gorm:"size:128" json:"activity"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (SlackRecord) TableName() string { return "slack_records" }
