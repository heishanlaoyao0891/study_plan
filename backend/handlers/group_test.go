package handlers

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"study_plan_backend/config"
	"study_plan_backend/db"
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
