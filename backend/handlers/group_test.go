package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

func setupGroupTestDB(t *testing.T) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	config.App = &config.Config{DBPath: dsn}
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.DB = gdb
	if err := db.AutoMigrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

func TestEnsureNoActiveGroupBlocksSecondGroup(t *testing.T) {
	setupGroupTestDB(t)
	user := models.User{OpenID: "u1", Nickname: "u1"}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	group := models.StudyGroup{Name: "G1", LeaderUserID: user.ID, Status: models.StudyGroupStatusActive}
	if err := db.DB.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&models.StudyGroupMember{GroupID: group.ID, UserID: user.ID, Role: models.GroupMemberRoleLeader, Status: models.GroupMemberStatusActive, JoinedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureNoActiveGroup(user.ID); err == nil {
		t.Fatal("expected active group guard to block a second group")
	}
}

func TestUpsertGroupInvitationReplacesOldCode(t *testing.T) {
	setupGroupTestDB(t)
	user := models.User{OpenID: "u1", Nickname: "u1"}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	group := models.StudyGroup{Name: "G1", LeaderUserID: user.ID, Status: models.StudyGroupStatusActive}
	if err := db.DB.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	first, err := upsertGroupInvitation(group.ID, user.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	second, err := upsertGroupInvitation(group.ID, user.ID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if first.Code == second.Code {
		t.Fatal("expected regenerated invitation code")
	}
	var revoked models.StudyGroupInvitation
	if err := db.DB.Where("code = ?", first.Code).First(&revoked).Error; err != nil {
		t.Fatal(err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("expected old invitation to be revoked")
	}
}

func TestExpiredGroupTransitionsBeforeReadAndAllowsFutureGroup(t *testing.T) {
	setupGroupTestDB(t)
	user := models.User{OpenID: "expired-user", Nickname: "expired-user"}
	db.DB.Create(&user)
	group := models.StudyGroup{Name: "Expired", LeaderUserID: user.ID, Status: models.StudyGroupStatusActive, EndDate: shanghaiNow().AddDate(0, 0, -1).Format(dateLayout)}
	db.DB.Create(&group)
	membership := models.StudyGroupMember{GroupID: group.ID, UserID: user.ID, Role: models.GroupMemberRoleLeader, Status: models.GroupMemberStatusActive, JoinedAt: time.Now()}
	db.DB.Create(&membership)
	invitation := models.StudyGroupInvitation{GroupID: group.ID, Code: "expired-code", ExpiresAt: time.Now().Add(time.Hour), CreatedBy: user.ID}
	db.DB.Create(&invitation)

	current, _, err := activeGroupForUser(user.ID)
	if err != nil || current.ID != 0 {
		t.Fatalf("expired group must not remain current: group=%+v err=%v", current, err)
	}
	db.DB.First(&group, group.ID)
	db.DB.First(&membership, membership.ID)
	db.DB.First(&invitation, invitation.ID)
	if group.Status != models.StudyGroupStatusEnded || group.EndedAt == nil || membership.Status != models.GroupMemberStatusLeft || membership.LeftAt == nil || invitation.RevokedAt == nil {
		t.Fatalf("expired lifecycle was not closed: group=%+v membership=%+v invitation=%+v", group, membership, invitation)
	}
	if err := ensureNoActiveGroup(user.ID); err != nil {
		t.Fatalf("expired membership should permit a future group: %v", err)
	}
}

func TestFutureEndDateGroupRemainsActive(t *testing.T) {
	setupGroupTestDB(t)
	user := models.User{OpenID: "future-user", Nickname: "future-user"}
	db.DB.Create(&user)
	group := models.StudyGroup{Name: "Future", LeaderUserID: user.ID, Status: models.StudyGroupStatusActive, EndDate: shanghaiNow().AddDate(0, 0, 1).Format(dateLayout)}
	db.DB.Create(&group)
	db.DB.Create(&models.StudyGroupMember{GroupID: group.ID, UserID: user.ID, Role: models.GroupMemberRoleLeader, Status: models.GroupMemberStatusActive, JoinedAt: time.Now()})
	current, _, err := activeGroupForUser(user.ID)
	if err != nil || current.ID != group.ID {
		t.Fatalf("future group should remain active: group=%+v err=%v", current, err)
	}
}

func TestMemberGroupMetricsUseCurrentShanghaiWeek(t *testing.T) {
	setupGroupTestDB(t)
	user := models.User{OpenID: "metrics-user", Nickname: "metrics-user"}
	db.DB.Create(&user)
	start, _ := currentShanghaiWeek()
	weekStart, _ := time.Parse(dateLayout, start)
	tasks := []models.DailyTask{
		{UserID: user.ID, PlanID: 1, Date: weekStart.Format(dateLayout), Title: "done", Status: models.TaskStatusCompleted, StudyMinutes: 30},
		{UserID: user.ID, PlanID: 1, Date: weekStart.AddDate(0, 0, 1).Format(dateLayout), Title: "pending", Status: models.TaskStatusPending, StudyMinutes: 10},
		{UserID: user.ID, PlanID: 1, Date: weekStart.AddDate(0, 0, -1).Format(dateLayout), Title: "old", Status: models.TaskStatusCompleted, StudyMinutes: 60},
	}
	if err := db.DB.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	weekStartAt, _ := currentShanghaiWeekTimes()
	if err := db.DB.Create(&models.StudySession{UserID: user.ID, TaskID: 1, StartTime: weekStartAt.Add(time.Minute), EndTime: timePointer(weekStartAt.Add(31 * time.Minute)), DurationMin: 30, DurationSec: 1800}).Error; err != nil {
		t.Fatal(err)
	}
	weeklyMinutes, weeklyRate, _, _ := memberGroupMetrics(user.ID, "weekly")
	allMinutes, allRate, _, _ := memberGroupMetrics(user.ID, "all")
	if weeklyMinutes != 30 || weeklyRate != 50 || allMinutes != 100 || allRate != 66 {
		t.Fatalf("unexpected metrics: weekly=%d/%d all=%d/%d", weeklyMinutes, weeklyRate, allMinutes, allRate)
	}
}

func TestAutoMigrateRejectsEleventhActiveGroupMember(t *testing.T) {
	setupGroupTestDB(t)
	group := models.StudyGroup{Name: "Capacity", LeaderUserID: 1, Status: models.StudyGroupStatusActive}
	if err := db.DB.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	for userID := uint(1); userID <= maxActiveGroupMembers; userID++ {
		member := models.StudyGroupMember{GroupID: group.ID, UserID: userID, Role: models.GroupMemberRoleMember, Status: models.GroupMemberStatusActive, JoinedAt: time.Now()}
		if err := db.DB.Create(&member).Error; err != nil {
			t.Fatalf("member %d should fit: %v", userID, err)
		}
	}
	extra := models.StudyGroupMember{GroupID: group.ID, UserID: 99, Role: models.GroupMemberRoleMember, Status: models.GroupMemberStatusActive, JoinedAt: time.Now()}
	if err := db.DB.Create(&extra).Error; err == nil {
		t.Fatal("expected database capacity guard to reject eleventh active member")
	}
}

func timePointer(value time.Time) *time.Time { return &value }

func TestActiveMemberCanCreateAndRevokeInvitation(t *testing.T) {
	setupGroupTestDB(t)
	leader := models.User{OpenID: "invite-leader", Nickname: "leader"}
	memberUser := models.User{OpenID: "invite-member", Nickname: "member"}
	db.DB.Create(&leader)
	db.DB.Create(&memberUser)
	group := models.StudyGroup{Name: "Invites", LeaderUserID: leader.ID, Status: models.StudyGroupStatusActive}
	db.DB.Create(&group)
	db.DB.Create(&models.StudyGroupMember{GroupID: group.ID, UserID: leader.ID, Role: models.GroupMemberRoleLeader, Status: models.GroupMemberStatusActive, JoinedAt: time.Now()})
	db.DB.Create(&models.StudyGroupMember{GroupID: group.ID, UserID: memberUser.ID, Role: models.GroupMemberRoleMember, Status: models.GroupMemberStatusActive, JoinedAt: time.Now()})
	path := "/groups/" + strconv.FormatUint(uint64(group.ID), 10)
	if response := callGroupHandler(t, CreateStudyGroupInvitation, memberUser.ID, http.MethodPost, path+"/invitations"); response.Code != 0 {
		t.Fatalf("active member invitation failed: %+v", response)
	}
	var invitation models.StudyGroupInvitation
	db.DB.Where("group_id = ? AND created_by = ? AND revoked_at IS NULL", group.ID, memberUser.ID).First(&invitation)
	if response := callGroupHandler(t, RevokeStudyGroupInvitation, memberUser.ID, http.MethodPost, path+"/invitations/revoke"); response.Code != 0 {
		t.Fatalf("active member revocation failed: %+v", response)
	}
	db.DB.First(&invitation, invitation.ID)
	if invitation.RevokedAt == nil {
		t.Fatal("expected invitation to be revoked")
	}
}

type groupHandlerResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func callGroupHandler(t *testing.T, handler gin.HandlerFunc, userID uint, method, path string) groupHandlerResponse {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, path, func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, userID) }, handler)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewReader(nil))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	var response groupHandlerResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return response
}
