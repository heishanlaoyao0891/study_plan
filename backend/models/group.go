package models

import "time"

const (
	StudyGroupStatusActive = "active"
	StudyGroupStatusEnded  = "ended"

	GroupMemberRoleLeader = "leader"
	GroupMemberRoleMember = "member"

	GroupMemberStatusActive  = "active"
	GroupMemberStatusLeft    = "left"
	GroupMemberStatusRemoved = "removed"
)

type StudyGroup struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Name         string     `gorm:"size:128;not null" json:"name"`
	LeaderUserID uint       `gorm:"index;not null" json:"leader_user_id"`
	Status       string     `gorm:"size:16;default:active;not null;index" json:"status"`
	EndDate      string     `gorm:"size:10" json:"end_date,omitempty"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (StudyGroup) TableName() string { return "study_groups" }

type StudyGroupMember struct {
	ID       uint       `gorm:"primaryKey" json:"id"`
	GroupID  uint       `gorm:"index;not null" json:"group_id"`
	UserID   uint       `gorm:"index;not null" json:"user_id"`
	Role     string     `gorm:"size:16;default:member;not null" json:"role"`
	Status   string     `gorm:"size:16;default:active;not null;index" json:"status"`
	JoinedAt time.Time  `json:"joined_at"`
	LeftAt   *time.Time `json:"left_at,omitempty"`
}

func (StudyGroupMember) TableName() string { return "study_group_members" }

type StudyGroupInvitation struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	GroupID   uint       `gorm:"index;not null" json:"group_id"`
	Code      string     `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Scene     string     `gorm:"size:128" json:"scene,omitempty"`
	ShareLink string     `gorm:"size:256" json:"share_link,omitempty"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedBy uint       `gorm:"index;not null" json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (StudyGroupInvitation) TableName() string { return "study_group_invitations" }
