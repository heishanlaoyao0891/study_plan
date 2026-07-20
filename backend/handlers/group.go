package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
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
)

const maxActiveGroupMembers = 10

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
	if err := ensureNoActiveGroup(uid); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.EndDate != "" {
		if _, err := time.Parse(dateLayout, req.EndDate); err != nil {
			api.Fail(c, http.StatusBadRequest, "invalid end_date, expect YYYY-MM-DD")
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
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		member.GroupID = group.ID
		return tx.Create(&member).Error
	}); err != nil {
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
		if err := tx.Model(&models.StudyGroup{}).Where("id = ?", group.ID).Updates(map[string]interface{}{"status": models.StudyGroupStatusEnded, "ended_at": &now}).Error; err != nil {
			return err
		}
		return tx.Model(&models.StudyGroupMember{}).Where("group_id = ? AND status = ?", group.ID, models.GroupMemberStatusActive).Update("status", models.GroupMemberStatusLeft).Error
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
	if err := db.DB.Model(&models.StudyGroupMember{}).Where("group_id = ? AND user_id = ?", group.ID, uid).Updates(map[string]interface{}{"status": models.GroupMemberStatusLeft, "left_at": &now}).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "leave group failed: "+err.Error())
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
	if err := db.DB.Model(&models.StudyGroupMember{}).Where("group_id = ? AND user_id = ? AND status = ?", group.ID, targetID, models.GroupMemberStatusActive).Updates(map[string]interface{}{"status": models.GroupMemberStatusRemoved, "left_at": &now}).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "remove member failed: "+err.Error())
		return
	}
	api.OK(c, gin.H{"group_id": group.ID, "removed_user_id": targetID})
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
	if err := ensureNoActiveGroup(uid); err != nil {
		api.Fail(c, http.StatusBadRequest, err.Error())
		return
	}
	inv, err := studyGroupInvitationByCode(req.Code)
	if err != nil {
		api.Fail(c, http.StatusNotFound, "invitation not found")
		return
	}
	group, err := activeGroupByID(inv.GroupID)
	if err != nil || group.ID == 0 || group.Status != models.StudyGroupStatusActive {
		api.Fail(c, http.StatusNotFound, "group not found")
		return
	}
	activeCount, err := activeGroupMemberCount(group.ID)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "query group size failed: "+err.Error())
		return
	}
	if activeCount >= maxActiveGroupMembers {
		api.Fail(c, http.StatusBadRequest, "group is full")
		return
	}
	now := time.Now()
	member := models.StudyGroupMember{GroupID: group.ID, UserID: uid, Role: models.GroupMemberRoleMember, Status: models.GroupMemberStatusActive, JoinedAt: now}
	if err := db.DB.Create(&member).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "join group failed: "+err.Error())
		return
	}
	api.OK(c, member)
}

func CreateStudyGroupInvitation(c *gin.Context) {
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
		return errors.New("already in an active study group")
	}
	return nil
}

func activeGroupForUser(uid uint) (models.StudyGroup, models.StudyGroupMember, error) {
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
	var members []models.StudyGroupMember
	if err := db.DB.Where("group_id = ? AND status = ?", groupID, models.GroupMemberStatusActive).Order("id ASC").Find(&members).Error; err != nil {
		return nil, err
	}
	rows := make([]groupMemberView, 0, len(members))
	for _, member := range members {
		row, err := buildGroupMemberView(member)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func buildGroupLeaderboard(groupID uint, scope string) ([]groupMemberView, error) {
	rows, err := buildGroupMemberViews(groupID)
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

func buildGroupMemberView(member models.StudyGroupMember) (groupMemberView, error) {
	var user models.User
	if err := db.DB.First(&user, member.UserID).Error; err != nil {
		return groupMemberView{}, err
	}
	streak, err := memberCurrentStreak(member.UserID)
	if err != nil {
		return groupMemberView{}, err
	}
	studyMinutes, completionRate, todayCompleted, err := memberGroupMetrics(member.UserID)
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
	var plans []models.Plan
	if err := db.DB.Where("user_id = ? AND status = ?", uid, models.PlanStatusActive).Find(&plans).Error; err != nil {
		return 0, err
	}
	if len(plans) == 0 {
		return 0, nil
	}
	planIDs := make([]uint, 0, len(plans))
	for _, p := range plans {
		planIDs = append(planIDs, p.ID)
	}
	streak := 0
	date := time.Now()
	for {
		dateStr := date.Format(dateLayout)
		var cnt int64
		db.DB.Model(&models.Checkin{}).Where("user_id = ? AND date = ? AND plan_id IN ? AND completed = ?", uid, dateStr, planIDs, true).Count(&cnt)
		if int(cnt) < len(planIDs) {
			break
		}
		streak++
		date = date.AddDate(0, 0, -1)
		if streak > 366 {
			break
		}
	}
	return streak, nil
}

func memberGroupMetrics(uid uint) (int, int, bool, error) {
	var totalTasks, completedTasks, studyMinutes int64
	if err := db.DB.Model(&models.DailyTask{}).Where("user_id = ?", uid).Count(&totalTasks).Error; err != nil {
		return 0, 0, false, err
	}
	if err := db.DB.Model(&models.DailyTask{}).Where("user_id = ? AND status = ?", uid, models.TaskStatusCompleted).Count(&completedTasks).Error; err != nil {
		return 0, 0, false, err
	}
	if err := db.DB.Model(&models.DailyTask{}).Where("user_id = ?", uid).Select("COALESCE(SUM(study_minutes),0)").Scan(&studyMinutes).Error; err != nil {
		return 0, 0, false, err
	}
	completionRate := 0
	if totalTasks > 0 {
		completionRate = int(completedTasks * 100 / totalTasks)
	}
	today := time.Now().Format(dateLayout)
	var activePlans []models.Plan
	if err := db.DB.Where("user_id = ? AND status = ?", uid, models.PlanStatusActive).Find(&activePlans).Error; err != nil {
		return 0, 0, false, err
	}
	if len(activePlans) == 0 {
		return int(studyMinutes), completionRate, false, nil
	}
	planIDs := make([]uint, 0, len(activePlans))
	for _, p := range activePlans {
		planIDs = append(planIDs, p.ID)
	}
	var todayCompleted int64
	if err := db.DB.Model(&models.Checkin{}).Where("user_id = ? AND date = ? AND plan_id IN ? AND completed = ?", uid, today, planIDs, true).Count(&todayCompleted).Error; err != nil {
		return 0, 0, false, err
	}
	return int(studyMinutes), completionRate, int(todayCompleted) == len(planIDs), nil
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
	now := time.Now()
	if err := db.DB.Model(&models.StudyGroupInvitation{}).Where("group_id = ? AND revoked_at IS NULL", groupID).Update("revoked_at", &now).Error; err != nil {
		return models.StudyGroupInvitation{}, err
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
	if err := db.DB.Create(&inv).Error; err != nil {
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
