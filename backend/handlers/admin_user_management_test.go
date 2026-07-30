package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"study_plan_backend/config"
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

func TestListUsersDefaultsToNormalAndFiltersDeleted(t *testing.T) {
	setupGroupTestDB(t)
	normal := createPasswordUser(t, "normal_directory", "password1")
	deleted := createPasswordUser(t, "deleted_directory", "password1")
	banned := createPasswordUser(t, "banned_directory", "password1")
	inactive := createPasswordUser(t, "inactive_directory", "password1")
	if err := db.DB.Model(&deleted).Update("account_status", models.AccountStatusDeleted).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Model(&banned).Update("banned_until", models.PermanentBanUntil()).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Model(&inactive).Update("account_status", models.AccountStatusInactive).Error; err != nil {
		t.Fatal(err)
	}

	assertUsers := func(status string, want ...uint) {
		t.Helper()
		recorder := httptestAdminUserRequest(ListUsers, 99, http.MethodGet, "", "", status)
		if recorderResponseCode(t, recorder) != 0 {
			t.Fatalf("list users for %q failed: %s", status, recorder.Body.String())
		}
		var envelope struct {
			Data struct {
				Users []models.User `json:"users"`
			} `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		got := map[uint]bool{}
		for _, user := range envelope.Data.Users {
			got[user.ID] = true
		}
		for _, id := range want {
			if !got[id] {
				t.Fatalf("status %q did not return user %d: %+v", status, id, envelope.Data.Users)
			}
		}
		if len(got) != len(want) {
			t.Fatalf("status %q returned unexpected users: %+v", status, envelope.Data.Users)
		}
	}

	assertUsers("", normal.ID)
	assertUsers("banned", banned.ID)
	assertUsers("deleted", deleted.ID)
	assertUsers("all", normal.ID, deleted.ID, banned.ID, inactive.ID)
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

func TestCreateAdminUserGeneratesOneTimeInitialPasswordAndAudits(t *testing.T) {
	setupGroupTestDB(t)
	admin := models.User{OpenID: "admin-create", Username: "admin_create", UsernameNormalized: "admin_create", Nickname: "Admin Create", NicknameNormalized: "admin create", Role: models.RoleAdmin, AccountStatus: models.AccountStatusActive}
	if err := db.DB.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	response := performJSONRequestWithContext(CreateAdminUser, map[string]string{"username": "created_user", "nickname": "Created User"}, func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, admin.ID)
	})
	if recorderResponseCode(t, response) != 0 {
		t.Fatalf("admin create failed: %s", response.Body.String())
	}
	var envelope struct {
		Data struct {
			User            models.User `json:"user"`
			InitialPassword string      `json:"initial_password"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.InitialPassword == "" || envelope.Data.User.Username != "created_user" || envelope.Data.User.InviteTargetID != "" {
		t.Fatalf("unsafe admin create response: %+v", envelope.Data)
	}
	var created models.User
	if err := db.DB.First(&created, envelope.Data.User.ID).Error; err != nil {
		t.Fatal(err)
	}
	if created.InviteTargetID == "" || created.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*created.PasswordHash), []byte(envelope.Data.InitialPassword)) != nil {
		t.Fatalf("created user is incomplete or password was not hashed: %+v", created)
	}
	var audit models.AdminAuditLog
	if err := db.DB.Where("admin_user_id = ? AND target_user_id = ? AND action_type = ?", admin.ID, created.ID, "create_user").First(&audit).Error; err != nil {
		t.Fatalf("create audit missing: %v", err)
	}
	duplicate := performJSONRequestWithContext(CreateAdminUser, map[string]string{"username": "created_user", "nickname": "Another User"}, func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, admin.ID)
	})
	if recorderResponseCode(t, duplicate) != http.StatusConflict {
		t.Fatalf("duplicate username was accepted: %s", duplicate.Body.String())
	}
}

func TestUpdateUsernameChangesLoginIdentityAndRefreshesSession(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "update-username-secret"
	config.App.JWTExpireHours = 24
	user := createPasswordUser(t, "original_user", "password1")
	if err := db.DB.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	originalSecurityVersion := user.SecurityVersion
	response := performJSONRequestWithContext(UpdateUsername, map[string]string{"username": "Renamed_User"}, func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, user.ID)
	})
	if recorderResponseCode(t, response) != 0 {
		t.Fatalf("username update failed: %s", response.Body.String())
	}
	var envelope struct {
		Data struct {
			Token string      `json:"token"`
			User  models.User `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.Token == "" || envelope.Data.User.Username != "Renamed_User" {
		t.Fatalf("username update response is incomplete: %+v", envelope.Data)
	}
	if err := db.DB.First(&user, user.ID).Error; err != nil || user.Username != "Renamed_User" || user.UsernameNormalized != "renamed_user" || user.SecurityVersion != originalSecurityVersion+1 {
		t.Fatalf("username update was not persisted: %+v err=%v", user, err)
	}
	occupied := createPasswordUser(t, "occupied_user", "password1")
	conflict := performJSONRequestWithContext(UpdateUsername, map[string]string{"username": occupied.Username}, func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, user.ID)
	})
	if recorderResponseCode(t, conflict) != http.StatusConflict {
		t.Fatalf("occupied username was accepted: %s", conflict.Body.String())
	}
}

func TestUpdateUsernameLimitsChangesPerCalendarMonth(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "username-change-limit-secret"
	user := createPasswordUser(t, "limit_user", "password1")
	update := func(username string) *httptest.ResponseRecorder {
		return performJSONRequestWithContext(UpdateUsername, map[string]string{"username": username}, func(c *gin.Context) {
			c.Set(middleware.CtxUserIDKey, user.ID)
		})
	}
	for _, username := range []string{"limit_user_1", "limit_user_2", "limit_user_3"} {
		if response := update(username); recorderResponseCode(t, response) != 0 {
			t.Fatalf("allowed username update %q failed: %s", username, response.Body.String())
		}
	}
	if response := update("limit_user_4"); recorderResponseCode(t, response) != http.StatusTooManyRequests {
		t.Fatalf("fourth monthly username update was not rejected: %s", response.Body.String())
	}
	var count int64
	if err := db.DB.Model(&models.AccountEvent{}).Where("user_id = ? AND event_type = ?", user.ID, "username_change").Count(&count).Error; err != nil || count != 3 {
		t.Fatalf("username change ledger is incorrect: count=%d err=%v", count, err)
	}
}

func httptestAdminUserRequest(handler gin.HandlerFunc, adminID uint, method, userID, search string, status ...string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	path := "/?search=" + search
	if len(status) > 0 {
		path += "&status=" + status[0]
	}
	context.Request = httptest.NewRequest(method, path, nil)
	context.Params = gin.Params{{Key: "id", Value: userID}}
	context.Set(middleware.CtxUserIDKey, adminID)
	handler(context)
	return recorder
}
