package models

import (
	"time"
)

type Checkin struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	PlanID    uint      `gorm:"index;not null" json:"plan_id"`
	Date      string    `gorm:"size:10;not null;index" json:"date"` // YYYY-MM-DD
	Completed bool      `gorm:"default:false" json:"completed"`
	Rewarded  bool      `gorm:"default:false" json:"rewarded"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CheckinUnique 唯一索引：同一个用户同一天同一个计划只能有一条打卡记录
// 使用 GORM 的 composite unique index tag
func (Checkin) TableName() string { return "checkins" }
