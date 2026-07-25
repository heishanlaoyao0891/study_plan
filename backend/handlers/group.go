package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"study_plan_backend/api"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

const maxActiveGroupMembers = 10

var (
	errAlreadyInActiveGroup = errors.New("already in an active study group")
	errGroupFull            = errors.New("group is full")
	errInvitationInvalid    = errors.New("invitation is invalid or expired")
)

type createGroupReq struct {
	Name    string `json:"name" binding:"required"`
	EndDate string `json:"end_date"`
}

type updateGroupReq struct {
	Name    *string `json:"name"`
	EndDate *string `json:"end_date"`
}

type transferGroupReq struct {
	UserID uint `json:"user_id" binding:"required"`
}

type inviteGroupReq struct {
	Days int `json:"days"`
}

type groupMemberView struct {
	UserID         uint   `json:"user_id"`
	Nickname       string `json:"nickname"`
	AvatarURL      string `json:"avatar_url"`
	Role           string `json:"role"`
	Level          int    `json:"level"`
	Streak         int    `json:"streak"`
	StudyMinutes   int    `json:"study_minutes"`
	CompletionRate int    `json:"completion_rate"`
	TodayCompleted bool   `json:"today_completed"`
}

func CreateStudyGroup(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req createGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if err := transitionExpiredStudyGroups(db.DB); err != nil {
		api.Fail(c, http.StatusInternalServerError, "expire groups failed: "+err.Error())
		return
	}
	if req.EndDate != "" {
		if _, err := time.Parse(dateLayout, req.EndDate); err != nil {
			api.Fail(c, http.StatusBadRequest, "invalid end_date, expect YYYY-MM-DD")
			return
		}
		if req.EndDate < shanghaiToday() {
			api.Fail(c, http.StatusBadRequest, "end_date cannot be in the past")
			return
		}
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "学习小组"
	}
	now := time.Now()
	group := models.StudyGroup{Name: name, LeaderUserID: uid, EndDate: req.EndDate, Status: models.StudyGroupStatusActive}
	member := models.StudyGroupMember{UserID: uid, Role: models.GroupMemberRoleLeader, Status: models.GroupMemberStatusActive, JoinedAt: now}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureNoActiveGroupTx(tx, uid); err != nil {
			return err
		}
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		member.GroupID = group.ID
		return tx.Create(&member).Error
	}); err != nil {
		if errors.Is(err, errAlreadyInActiveGroup) || isActiveMembershipConflict(err) {
			api.Fail(c, http.StatusConflict, errAlreadyInActiveGroup.Error())
			return
		}
		api.Fail(c, http.StatusInternalServerError, "create group failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"group": group, "member": member})
}

func CurrentStudyGroup(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	group, member, err := activeGroupForUser(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group failed: "+err.Error())
		return
	}
	if group.ID == 0 {
		api.OK(c, gin.H{"group": nil, "member": nil})
		return
	}
	api.OK(c, gin.H{"group": group, "member": member})
}

func GroupHistory(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	if err := transitionExpiredStudyGroups(db.DB); err != nil {
		api.Fail(c, http.StatusInternalServerError, "expire groups failed: "+err.Error())
		return
	}
	var groups []models.StudyGroup
	if err := db.DB.Table("study_groups").Joins("JOIN study_group_members ON study_group_members.group_id = study_groups.id").Where("study_group_members.user_id = ? AND study_groups.status = ?", uid, models.StudyGroupStatusEnded).Order("study_groups.ended_at DESC, study_groups.id DESC").Scan(&groups).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group history failed: "+err.Error())
		return
	}
	api.OK(c, groups)
}

func GroupMembers(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	group, _, err := activeGroupForUser(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group failed: "+err.Error())
		return
	}
	if group.ID == 0 {
		api.Fail(c, http.StatusNotFound, "group not found")
		return
	}
	members, err := buildGroupMemberViews(group.ID)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query members failed: "+err.Error())
		return
	}
	api.OK(c, members)
}

func GroupLeaderboard(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	group, _, err := activeGroupForUser(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group failed: "+err.Error())
		return
	}
	if group.ID == 0 {
		api.Fail(c, http.StatusNotFound, "group not found")
		return
	}
	scope := strings.TrimSpace(c.DefaultQuery("scope", "weekly"))
	if scope != "weekly" && scope != "all" {
		api.Fail(c, http.StatusBadRequest, "scope must be weekly or all")
		return
	}
	board, err := buildGroupLeaderboard(group.ID, scope)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query leaderboard failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"scope": scope, "rows": board})
}

func UpdateStudyGroup(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	group, member, err := activeGroupForUser(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group failed: "+err.Error())
		return
	}
	if group.ID == 0 {
		api.Fail(c, http.StatusNotFound, "group not found")
		return
	}
	if member.Role != models.GroupMemberRoleLeader {
		api.Fail(c, http.StatusForbidden, "leader only")
		return
	}
	var req updateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.EndDate != nil {
		if *req.EndDate != "" {
			if _, err := time.Parse(dateLayout, *req.EndDate); err != nil {
				api.Fail(c, http.StatusBadRequest, "invalid end_date, expect YYYY-MM-DD")
				return
			}
			if *req.EndDate < shanghaiToday() {
				api.Fail(c, http.StatusBadRequest, "end_date cannot be in the past")
				return
			}
		}
		updates["end_date"] = *req.EndDate
	}
	if len(updates) == 0 {
		api.OK(c, group)
		return
	}
	if err := db.DB.Model(&group).Updates(updates).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "update group failed: "+err.Error())
		return
	}
	api.OK(c, group)
}

func TransferGroupLeader(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	group, member, err := activeGroupForUser(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group failed: "+err.Error())
		return
	}
	if group.ID == 0 {
		api.Fail(c, http.StatusNotFound, "group not found")
		return
	}
	if member.Role != models.GroupMemberRoleLeader {
		api.Fail(c, http.StatusForbidden, "leader only")
		return
	}
	var req transferGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.UserID == uid {
		api.Fail(c, http.StatusBadRequest, "cannot transfer to yourself")
		return
	}
	var target models.StudyGroupMember
	if err := db.DB.Where("group_id = ? AND user_id = ? AND status = ?", group.ID, req.UserID, models.GroupMemberStatusActive).First(&target).Error; err != nil {
		api.Fail(c, http.StatusBadRequest, "target must be an active member")
		return
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.StudyGroup{}).Where("id = ?", group.ID).Updates(map[string]interface{}{"leader_user_id": req.UserID}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.StudyGroupMember{}).Where("group_id = ? AND user_id = ?", group.ID, uid).Update("role", models.GroupMemberRoleMember).Error; err != nil {
			return err
		}
		return tx.Model(&models.StudyGroupMember{}).Where("group_id = ? AND user_id = ?", group.ID, req.UserID).Update("role", models.GroupMemberRoleLeader).Error
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "transfer leader failed: "+err.Error())
		return
	}
	group.LeaderUserID = req.UserID
	api.OK(c, group)
}

func EndStudyGroup(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	group, member, err := activeGroupForUser(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group failed: "+err.Error())
		return
	}
	if group.ID == 0 {
		api.Fail(c, http.StatusNotFound, "group not found")
		return
	}
	if member.Role != models.GroupMemberRoleLeader {
		api.Fail(c, http.StatusForbidden, "leader only")
		return
	}
	now := time.Now()
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.StudyGroup{}).Where("id = ? AND status = ?", group.ID, models.StudyGroupStatusActive).Updates(map[string]interface{}{"status": models.StudyGroupStatusEnded, "ended_at": &now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Model(&models.StudyGroupInvitation{}).Where("group_id = ? AND revoked_at IS NULL", group.ID).Update("revoked_at", &now).Error; err != nil {
			return err
		}
		return tx.Model(&models.StudyGroupMember{}).Where("group_id = ? AND status = ?", group.ID, models.GroupMemberStatusActive).Updates(map[string]interface{}{"status": models.GroupMemberStatusLeft, "left_at": &now}).Error
	}); err != nil {
		api.Fail(c, http.StatusInternalServerError, "end group failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"group_id": group.ID, "status": models.StudyGroupStatusEnded})
}

func LeaveStudyGroup(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	group, member, err := activeGroupForUser(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group failed: "+err.Error())
		return
	}
	if group.ID == 0 {
		api.Fail(c, http.StatusNotFound, "group not found")
		return
	}
	if member.Role == models.GroupMemberRoleLeader {
		api.Fail(c, http.StatusBadRequest, "leader must transfer leadership or end group before leaving")
		return
	}
	now := time.Now()
	result := db.DB.Model(&models.StudyGroupMember{}).Where("group_id = ? AND user_id = ? AND status = ?", group.ID, uid, models.GroupMemberStatusActive).Updates(map[string]interface{}{"status": models.GroupMemberStatusLeft, "left_at": &now})
	if result.Error != nil {
		api.Fail(c, http.StatusInternalServerError, "leave group failed: "+result.Error.Error())
		return
	}
	if result.RowsAffected != 1 {
		api.Fail(c, http.StatusConflict, "membership is no longer active")
		return
	}
	api.OK(c, gin.H{"group_id": group.ID, "status": models.GroupMemberStatusLeft})
}

func RemoveStudyGroupMember(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	group, member, err := activeGroupForUser(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group failed: "+err.Error())
		return
	}
	if group.ID == 0 {
		api.Fail(c, http.StatusNotFound, "group not found")
		return
	}
	if member.Role != models.GroupMemberRoleLeader {
		api.Fail(c, http.StatusForbidden, "leader only")
		return
	}
	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || targetID == 0 {
		api.Fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	if uint(targetID) == uid {
		api.Fail(c, http.StatusBadRequest, "cannot remove yourself")
		return
	}
	now := time.Now()
	result := db.DB.Model(&models.StudyGroupMember{}).Where("group_id = ? AND user_id = ? AND role <> ? AND status = ?", group.ID, targetID, models.GroupMemberRoleLeader, models.GroupMemberStatusActive).Updates(map[string]interface{}{"status": models.GroupMemberStatusRemoved, "left_at": &now})
	if result.Error != nil {
		api.Fail(c, http.StatusInternalServerError, "remove member failed: "+result.Error.Error())
		return
	}
	if result.RowsAffected != 1 {
		api.Fail(c, http.StatusConflict, "member is no longer removable")
		return
	}
	api.OK(c, gin.H{"group_id": group.ID, "removed_user_id": targetID})
}

func NudgeStudyGroupMember(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	group, _, err := activeGroupForUser(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group failed: "+err.Error())
		return
	}
	if group.ID == 0 {
		api.Fail(c, http.StatusNotFound, "group not found")
		return
	}
	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || targetID == 0 {
		api.Fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	if uint(targetID) == uid {
		api.Fail(c, http.StatusBadRequest, "cannot nudge yourself")
		return
	}
	var target models.StudyGroupMember
	if err := db.DB.Where("group_id = ? AND user_id = ? AND status = ?", group.ID, targetID, models.GroupMemberStatusActive).First(&target).Error; err != nil {
		api.Fail(c, http.StatusBadRequest, "target must be an active member")
		return
	}
	now := shanghaiNow()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var sameTarget int64
	db.DB.Model(&models.StudyGroupNudge{}).Where("group_id = ? AND sender_user_id = ? AND target_user_id = ? AND created_at >= ?", group.ID, uid, targetID, start).Count(&sameTarget)
	if sameTarget > 0 {
		api.Fail(c, http.StatusBadRequest, "already nudged this member today")
		return
	}
	var received int64
	db.DB.Model(&models.StudyGroupNudge{}).Where("group_id = ? AND target_user_id = ? AND created_at >= ?", group.ID, targetID, start).Count(&received)
	if received >= 3 {
		api.Fail(c, http.StatusBadRequest, "target has received too many nudges today")
		return
	}
	message := "小组成员提醒你开始学习"
	nudge := models.StudyGroupNudge{GroupID: group.ID, SenderUserID: uid, TargetUserID: uint(targetID), Status: "processing", Message: message}
	if err := db.DB.Create(&nudge).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "nudge failed: "+err.Error())
		return
	}
	var sender models.User
	db.DB.First(&sender, uid)
	delivery, _, deliverErr := services.DeliverNotification(db.DB, fmt.Sprintf("group_nudge:%d", nudge.ID), uint(targetID), "group_nudge", services.NotificationValues{Message: message, Sender: sender.Nickname}, services.SendSubscriptionMessage)
	if deliverErr != nil {
		delivery.Status, delivery.Message = "failed", deliverErr.Error()
	}
	nudge.Status, nudge.Message = delivery.Status, delivery.Message
	if nudge.Message == "" {
		nudge.Message = message
	}
	if err := db.DB.Save(&nudge).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "save nudge result failed: "+err.Error())
		return
	}
	api.OK(c, nudge)
}

func JoinStudyGroupByCode(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Fail(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if err := transitionExpiredStudyGroups(db.DB); err != nil {
		api.Fail(c, http.StatusInternalServerError, "expire groups failed: "+err.Error())
		return
	}
	now := time.Now()
	var member models.StudyGroupMember
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureNoActiveGroupTx(tx, uid); err != nil {
			return err
		}
		var inv models.StudyGroupInvitation
		if err := tx.Where("code = ? AND revoked_at IS NULL AND expires_at > ?", strings.TrimSpace(req.Code), now).First(&inv).Error; err != nil {
			return errInvitationInvalid
		}
		var group models.StudyGroup
		if err := tx.Where("id = ? AND status = ?", inv.GroupID, models.StudyGroupStatusActive).First(&group).Error; err != nil {
			return errInvitationInvalid
		}
		var activeCount int64
		if err := tx.Model(&models.StudyGroupMember{}).Where("group_id = ? AND status = ?", group.ID, models.GroupMemberStatusActive).Count(&activeCount).Error; err != nil {
			return err
		}
		if activeCount >= maxActiveGroupMembers {
			return errGroupFull
		}
		member = models.StudyGroupMember{GroupID: group.ID, UserID: uid, Role: models.GroupMemberRoleMember, Status: models.GroupMemberStatusActive, JoinedAt: now}
		return tx.Create(&member).Error
	})
	if err != nil {
		switch {
		case errors.Is(err, errInvitationInvalid):
			api.Fail(c, http.StatusNotFound, errInvitationInvalid.Error())
		case errors.Is(err, errGroupFull) || isGroupCapacityConflict(err):
			api.Fail(c, http.StatusConflict, errGroupFull.Error())
		case errors.Is(err, errAlreadyInActiveGroup) || isActiveMembershipConflict(err):
			api.Fail(c, http.StatusConflict, errAlreadyInActiveGroup.Error())
		default:
			api.Fail(c, http.StatusInternalServerError, "join group failed: "+err.Error())
		}
		return
	}
	api.OK(c, member)
}

func CreateStudyGroupInvitation(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	group, _, err := activeGroupForUser(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group failed: "+err.Error())
		return
	}
	if group.ID == 0 {
		api.Fail(c, http.StatusNotFound, "group not found")
		return
	}
	days := 7
	if raw := strings.TrimSpace(c.Query("days")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			days = n
		}
	}
	inv, err := upsertGroupInvitation(group.ID, uid, days)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "create invitation failed: "+err.Error())
		return
	}
	api.OK(c, inv)
}

func RevokeStudyGroupInvitation(c *gin.Context) {
	uid := c.GetUint(middleware.CtxUserIDKey)
	group, _, err := activeGroupForUser(uid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group failed: "+err.Error())
		return
	}
	if group.ID == 0 {
		api.Fail(c, http.StatusNotFound, "group not found")
		return
	}
	now := time.Now()
	if err := db.DB.Model(&models.StudyGroupInvitation{}).Where("group_id = ? AND revoked_at IS NULL", group.ID).Update("revoked_at", &now).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "revoke invitation failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"group_id": group.ID, "revoked": true})
}

func ensureNoActiveGroup(uid uint) error {
	group, _, err := activeGroupForUser(uid)
	if err != nil {
		return err
	}
	if group.ID != 0 {
		return errAlreadyInActiveGroup
	}
	return nil
}

func activeGroupForUser(uid uint) (models.StudyGroup, models.StudyGroupMember, error) {
	if err := transitionExpiredStudyGroups(db.DB); err != nil {
		return models.StudyGroup{}, models.StudyGroupMember{}, err
	}
	var member models.StudyGroupMember
	err := db.DB.Table("study_group_members").Joins("JOIN study_groups ON study_groups.id = study_group_members.group_id").Where("study_group_members.user_id = ? AND study_group_members.status = ? AND study_groups.status = ?", uid, models.GroupMemberStatusActive, models.StudyGroupStatusActive).Order("study_group_members.id DESC").First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.StudyGroup{}, models.StudyGroupMember{}, nil
	}
	if err != nil {
		return models.StudyGroup{}, models.StudyGroupMember{}, err
	}
	var group models.StudyGroup
	if err := db.DB.First(&group, member.GroupID).Error; err != nil {
		return models.StudyGroup{}, models.StudyGroupMember{}, err
	}
	return group, member, nil
}

func activeGroupMemberCount(groupID uint) (int64, error) {
	var count int64
	if err := db.DB.Model(&models.StudyGroupMember{}).Where("group_id = ? AND status = ?", groupID, models.GroupMemberStatusActive).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func buildGroupMemberViews(groupID uint) ([]groupMemberView, error) {
	return buildGroupMemberViewsForScope(groupID, "all")
}

func buildGroupMemberViewsForScope(groupID uint, scope string) ([]groupMemberView, error) {
	var members []models.StudyGroupMember
	if err := db.DB.Where("group_id = ? AND status = ?", groupID, models.GroupMemberStatusActive).Order("id ASC").Find(&members).Error; err != nil {
		return nil, err
	}
	rows := make([]groupMemberView, 0, len(members))
	for _, member := range members {
		row, err := buildGroupMemberView(member, scope)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func buildGroupLeaderboard(groupID uint, scope string) ([]groupMemberView, error) {
	rows, err := buildGroupMemberViewsForScope(groupID, scope)
	if err != nil {
		return nil, err
	}
	if scope == "all" {
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].Level != rows[j].Level {
				return rows[i].Level > rows[j].Level
			}
			if rows[i].StudyMinutes != rows[j].StudyMinutes {
				return rows[i].StudyMinutes > rows[j].StudyMinutes
			}
			if rows[i].CompletionRate != rows[j].CompletionRate {
				return rows[i].CompletionRate > rows[j].CompletionRate
			}
			return rows[i].Streak > rows[j].Streak
		})
		return rows, nil
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Streak != rows[j].Streak {
			return rows[i].Streak > rows[j].Streak
		}
		if rows[i].StudyMinutes != rows[j].StudyMinutes {
			return rows[i].StudyMinutes > rows[j].StudyMinutes
		}
		if rows[i].CompletionRate != rows[j].CompletionRate {
			return rows[i].CompletionRate > rows[j].CompletionRate
		}
		return rows[i].Level > rows[j].Level
	})
	return rows, nil
}

func buildGroupMemberView(member models.StudyGroupMember, scope string) (groupMemberView, error) {
	var user models.User
	if err := db.DB.First(&user, member.UserID).Error; err != nil {
		return groupMemberView{}, err
	}
	streak, err := memberCurrentStreak(member.UserID)
	if err != nil {
		return groupMemberView{}, err
	}
	studyMinutes, completionRate, todayCompleted, err := memberGroupMetrics(member.UserID, scope)
	if err != nil {
		return groupMemberView{}, err
	}
	return groupMemberView{
		UserID:         user.ID,
		Nickname:       user.Nickname,
		AvatarURL:      user.AvatarURL,
		Role:           member.Role,
		Level:          memberLevel(streak),
		Streak:         streak,
		StudyMinutes:   studyMinutes,
		CompletionRate: completionRate,
		TodayCompleted: todayCompleted,
	}, nil
}

func memberCurrentStreak(uid uint) (int, error) {
	streak, _, err := consecutiveCheckins(db.DB, uid)
	return streak, err
}

func memberGroupMetrics(uid uint, scope string) (int, int, bool, error) {
	var totalTasks, completedTasks int64
	if err := groupMetricTasks(uid, scope).Count(&totalTasks).Error; err != nil {
		return 0, 0, false, err
	}
	if err := groupMetricTasks(uid, scope).Where("status = ?", models.TaskStatusCompleted).Count(&completedTasks).Error; err != nil {
		return 0, 0, false, err
	}
	studyMinutes := 0
	if scope == "weekly" {
		start, end := currentShanghaiWeekTimes()
		var seconds int64
		if err := db.DB.Model(&models.StudySession{}).Where("user_id = ? AND start_time >= ? AND start_time < ?", uid, start, end).Select("COALESCE(SUM(duration_sec),0)").Scan(&seconds).Error; err != nil {
			return 0, 0, false, err
		}
		studyMinutes = int(seconds / 60)
	} else {
		var minutes int64
		if err := groupMetricTasks(uid, scope).Select("COALESCE(SUM(study_minutes),0)").Scan(&minutes).Error; err != nil {
			return 0, 0, false, err
		}
		studyMinutes = int(minutes)
	}
	completionRate := 0
	if totalTasks > 0 {
		completionRate = int(completedTasks * 100 / totalTasks)
	}
	today := shanghaiToday()
	var todayCompleted int64
	if err := db.DB.Model(&models.DailyCheckin{}).Where("user_id = ? AND date = ? AND completed = ?", uid, today, true).Count(&todayCompleted).Error; err != nil {
		return 0, 0, false, err
	}
	return studyMinutes, completionRate, todayCompleted > 0, nil
}

func groupMetricTasks(uid uint, scope string) *gorm.DB {
	query := db.DB.Model(&models.DailyTask{}).Where("user_id = ?", uid)
	if scope == "weekly" {
		start, end := currentShanghaiWeek()
		query = query.Where("date >= ? AND date < ?", start, end)
	}
	return query
}

func currentShanghaiWeek() (string, string) {
	start, end := currentShanghaiWeekTimes()
	return start.Format(dateLayout), end.Format(dateLayout)
}

func currentShanghaiWeekTimes() (time.Time, time.Time) {
	now := shanghaiNow()
	daysSinceMonday := (int(now.Weekday()) + 6) % 7
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -daysSinceMonday)
	return start, start.AddDate(0, 0, 7)
}

func transitionExpiredStudyGroups(database *gorm.DB) error {
	today := shanghaiToday()
	now := time.Now()
	return database.Transaction(func(tx *gorm.DB) error {
		var groupIDs []uint
		if err := tx.Model(&models.StudyGroup{}).Where("status = ? AND end_date <> '' AND end_date < ?", models.StudyGroupStatusActive, today).Pluck("id", &groupIDs).Error; err != nil {
			return err
		}
		if len(groupIDs) == 0 {
			return nil
		}
		if err := tx.Model(&models.StudyGroup{}).Where("id IN ? AND status = ?", groupIDs, models.StudyGroupStatusActive).Updates(map[string]interface{}{"status": models.StudyGroupStatusEnded, "ended_at": &now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.StudyGroupInvitation{}).Where("group_id IN ? AND revoked_at IS NULL", groupIDs).Update("revoked_at", &now).Error; err != nil {
			return err
		}
		return tx.Model(&models.StudyGroupMember{}).Where("group_id IN ? AND status = ?", groupIDs, models.GroupMemberStatusActive).Updates(map[string]interface{}{"status": models.GroupMemberStatusLeft, "left_at": &now}).Error
	})
}

func ensureNoActiveGroupTx(tx *gorm.DB, uid uint) error {
	var count int64
	if err := tx.Model(&models.StudyGroupMember{}).Where("user_id = ? AND status = ?", uid, models.GroupMemberStatusActive).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errAlreadyInActiveGroup
	}
	return nil
}

func isActiveMembershipConflict(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_study_group_members_active_user") || strings.Contains(message, "unique constraint failed: study_group_members.user_id")
}

func isGroupCapacityConflict(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "study group is full")
}

func memberLevel(streak int) int {
	switch {
	case streak >= 30:
		return 5
	case streak >= 14:
		return 4
	case streak >= 7:
		return 3
	case streak >= 3:
		return 2
	default:
		return 1
	}
}

func studyGroupInvitationByCode(code string) (models.StudyGroupInvitation, error) {
	var inv models.StudyGroupInvitation
	if err := db.DB.Where("code = ? AND revoked_at IS NULL AND expires_at > ?", strings.TrimSpace(code), time.Now()).First(&inv).Error; err != nil {
		return models.StudyGroupInvitation{}, err
	}
	return inv, nil
}

func activeGroupByID(id uint) (models.StudyGroup, error) {
	var group models.StudyGroup
	if err := db.DB.Where("id = ? AND status = ?", id, models.StudyGroupStatusActive).First(&group).Error; err != nil {
		return models.StudyGroup{}, err
	}
	return group, nil
}

func upsertGroupInvitation(groupID, createdBy uint, days int) (models.StudyGroupInvitation, error) {
	if days <= 0 {
		days = 7
	}
	code, err := randomInviteCode()
	if err != nil {
		return models.StudyGroupInvitation{}, err
	}
	var inv models.StudyGroupInvitation
	inv = models.StudyGroupInvitation{
		GroupID:   groupID,
		Code:      code,
		Scene:     "group:" + strconv.FormatUint(uint64(groupID), 10),
		ShareLink: groupShareLink(code),
		ExpiresAt: time.Now().AddDate(0, 0, days),
		CreatedBy: createdBy,
	}
	now := time.Now()
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.StudyGroupInvitation{}).Where("group_id = ? AND revoked_at IS NULL", groupID).Update("revoked_at", &now).Error; err != nil {
			return err
		}
		return tx.Create(&inv).Error
	}); err != nil {
		return models.StudyGroupInvitation{}, err
	}
	return inv, nil
}

func groupShareLink(code string) string {
	return "/pages/group/join?code=" + code
}

func randomInviteCode() (string, error) {
	buf := make([]byte, 9)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.TrimRight(base64.RawURLEncoding.EncodeToString(buf), "=")[:12], nil
}
