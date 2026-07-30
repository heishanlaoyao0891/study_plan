package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

func TestListUsersSearchesLoginUsername(t *testing.T) {
	setupGroupTestDB(t)
	user := createPasswordUser(t, "directory_login", "password1")
	if err := db.DB.Model(&user).Updates(map[string]interface{}{"nickname": "目录昵称", "open_id": "directory-openid"}).Error; err != nil {
		t.Fatal(err)
	}

	recorder := httptestAdminUserRequest(ListUsers, 99, http.MethodGet, strconv.FormatUint(uint64(user.ID), 10), "directory_login")
	if recorderResponseCode(t, recorder) != 0 {
		t.Fatalf("username search failed: %s", recorder.Body.String())
	}
	var envelope struct {
		Data struct {
			Users []models.User `json:"users"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data.Users) != 1 || envelope.Data.Users[0].ID != user.ID || envelope.Data.Users[0].Username != user.Username {
		t.Fatalf("username match missing from list: %+v", envelope.Data.Users)
	}
}

func TestDeleteAdminUserErasesNormalAccountAndAudits(t *testing.T) {
	setupGroupTestDB(t)
	admin := models.User{OpenID: "admin-delete", Username: "admin_delete", UsernameNormalized: "admin_delete", Nickname: "Admin Delete", NicknameNormalized: "admin delete", Role: models.RoleAdmin, AccountStatus: models.AccountStatusActive}
	if err := db.DB.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	target := createPasswordUser(t, "erase_target", "password1")
	if err := db.DB.Create(&models.Plan{UserID: target.ID, Title: "to erase", Status: models.PlanStatusActive}).Error; err != nil {
		t.Fatal(err)
	}

	recorder := httptestAdminUserRequest(DeleteAdminUser, admin.ID, http.MethodDelete, strconv.FormatUint(uint64(target.ID), 10), "")
	if recorderResponseCode(t, recorder) != 0 {
		t.Fatalf("admin delete failed: %s", recorder.Body.String())
	}
	var erased models.User
	if err := db.DB.First(&erased, target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if erased.AccountStatus != models.AccountStatusDeleted || erased.Username != "" || erased.Nickname != "" || erased.PasswordHash != nil || erased.SecurityVersion != target.SecurityVersion+1 {
		t.Fatalf("account was not anonymized: %+v", erased)
	}
	var plans int64
	db.DB.Model(&models.Plan{}).Where("user_id = ?", target.ID).Count(&plans)
	if plans != 0 {
		t.Fatalf("user learning data remained after delete: %d plans", plans)
	}
	var audit models.AdminAuditLog
	if err := db.DB.Where("admin_user_id = ? AND target_user_id = ? AND action_type = ?", admin.ID, target.ID, "delete_user").First(&audit).Error; err != nil {
		t.Fatalf("delete audit missing: %v", err)
	}
}

func TestDeleteAdminUserRejectsProtectedAccounts(t *testing.T) {
	setupGroupTestDB(t)
	admin := models.User{OpenID: "admin-guard", Username: "admin_guard", UsernameNormalized: "admin_guard", Nickname: "Admin Guard", NicknameNormalized: "admin guard", Role: models.RoleAdmin, AccountStatus: models.AccountStatusActive}
	otherAdmin := models.User{OpenID: "other-admin-guard", Username: "other_admin", UsernameNormalized: "other_admin", Nickname: "Other Admin", NicknameNormalized: "other admin", Role: models.RoleAdmin, AccountStatus: models.AccountStatusActive}
	if err := db.DB.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&otherAdmin).Error; err != nil {
		t.Fatal(err)
	}

	for _, targetID := range []uint{admin.ID, otherAdmin.ID} {
		recorder := httptestAdminUserRequest(DeleteAdminUser, admin.ID, http.MethodDelete, strconv.FormatUint(uint64(targetID), 10), "")
		if recorderResponseCode(t, recorder) != http.StatusBadRequest {
			t.Fatalf("protected account %d was not rejected: %s", targetID, recorder.Body.String())
		}
		var reloaded models.User
		if err := db.DB.First(&reloaded, targetID).Error; err != nil || reloaded.AccountStatus != models.AccountStatusActive {
			t.Fatalf("protected account changed: %+v err=%v", reloaded, err)
		}
	}
}

func httptestAdminUserRequest(handler gin.HandlerFunc, adminID uint, method, userID, search string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, "/?search="+search, nil)
	context.Params = gin.Params{{Key: "id", Value: userID}}
	context.Set(middleware.CtxUserIDKey, adminID)
	handler(context)
	return recorder
}
