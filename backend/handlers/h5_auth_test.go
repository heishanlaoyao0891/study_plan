package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

func TestInvitationLifecycleAndMultipleH5Accounts(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "h5-test-secret"
	firstCode := createTestInvite(t, time.Now().Add(7*24*time.Hour), false)
	secondCode := createTestInvite(t, time.Now().Add(7*24*time.Hour), false)

	first := performJSONRequest(H5Register, map[string]string{
		"invite_code": firstCode, "username": "First_User", "nickname": "First User", "password": "password1",
	})
	if first.Code != http.StatusOK || recorderResponseCode(t, first) != 0 {
		t.Fatalf("register failed: status=%d body=%s", first.Code, first.Body.String())
	}
	second := performJSONRequest(H5Register, map[string]string{
		"invite_code": secondCode, "username": "second_user", "nickname": "Second User", "password": "password2",
	})
	if second.Code != http.StatusOK || recorderResponseCode(t, second) != 0 {
		t.Fatalf("second H5 account failed: status=%d body=%s", second.Code, second.Body.String())
	}

	var user models.User
	if err := db.DB.Where("username_normalized = ?", "first_user").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.OpenID != "" || user.Username != "First_User" || user.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte("password1")) != nil {
		t.Fatalf("unexpected H5 identity: %+v", user)
	}
	var invite models.RegistrationInvite
	if err := db.DB.Where("code_hash = ?", hashInviteCode(firstCode)).First(&invite).Error; err != nil {
		t.Fatal(err)
	}
	if invite.UsedAt == nil || invite.UserID == nil || *invite.UserID != user.ID {
		t.Fatalf("invitation was not consumed: %+v", invite)
	}
	reuse := performJSONRequest(H5Register, map[string]string{
		"invite_code": firstCode, "username": "third_user", "nickname": "Third User", "password": "password3",
	})
	if recorderResponseCode(t, reuse) != http.StatusBadRequest {
		t.Fatalf("used invitation accepted: %s", reuse.Body.String())
	}
}

func TestInvitationRejectsExpiredAndDisabled(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "invite-test-secret"
	for name, code := range map[string]string{
		"expired":  createTestInvite(t, time.Now().Add(-time.Minute), false),
		"disabled": createTestInvite(t, time.Now().Add(time.Hour), true),
	} {
		t.Run(name, func(t *testing.T) {
			response := performJSONRequest(H5Register, map[string]string{
				"invite_code": code, "username": name + "_user", "nickname": name + " User", "password": "password1",
			})
			if recorderResponseCode(t, response) != http.StatusBadRequest {
				t.Fatalf("invalid invitation accepted: %s", response.Body.String())
			}
		})
	}
}

func TestAdminInvitationManagement(t *testing.T) {
	setupGroupTestDB(t)
	create := performJSONRequestWithContext(CreateRegistrationInvites, map[string]int{"count": 2}, func(c *gin.Context) {
		c.Set("user_id", uint(42))
	})
	if recorderResponseCode(t, create) != 0 {
		t.Fatalf("create invitations failed: %s", create.Body.String())
	}
	var created struct {
		Data struct {
			Codes       []string                    `json:"codes"`
			Invitations []models.RegistrationInvite `json:"invitations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if len(created.Data.Codes) != 2 || len(created.Data.Invitations) != 2 {
		t.Fatalf("unexpected generated invitations: %s", create.Body.String())
	}

	list := performJSONRequest(ListRegistrationInvites, nil)
	if recorderResponseCode(t, list) != 0 || strings.Contains(list.Body.String(), created.Data.Codes[0]) {
		t.Fatalf("listing leaked or failed: %s", list.Body.String())
	}

	disable := performJSONRequestWithContext(DisableRegistrationInvite, nil, func(c *gin.Context) {
		c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(created.Data.Invitations[0].ID), 10)}}
	})
	if recorderResponseCode(t, disable) != 0 {
		t.Fatalf("disable invitation failed: %s", disable.Body.String())
	}
	response := performJSONRequest(H5Register, map[string]string{
		"invite_code": created.Data.Codes[0], "username": "disabled_user", "nickname": "Disabled User", "password": "password1",
	})
	if recorderResponseCode(t, response) != http.StatusBadRequest {
		t.Fatalf("disabled generated invitation accepted: %s", response.Body.String())
	}
}

func TestH5LoginUsesGenericCredentialError(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "login-test-secret"
	code := createTestInvite(t, time.Now().Add(time.Hour), false)
	register := performJSONRequest(H5Register, map[string]string{
		"invite_code": code, "username": "Login_User", "nickname": "Login User", "password": "password1",
	})
	if recorderResponseCode(t, register) != 0 {
		t.Fatal(register.Body.String())
	}
	login := performJSONRequest(H5Login, map[string]string{"username": "login_user", "password": "password1"})
	if recorderResponseCode(t, login) != 0 {
		t.Fatalf("login failed: %s", login.Body.String())
	}
	for _, payload := range []map[string]string{
		{"username": "missing_user", "password": "password1"},
		{"username": "login_user", "password": "wrong-password"},
	} {
		response := performJSONRequest(H5Login, payload)
		var envelope struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Message != "invalid username or password" {
			t.Fatalf("credential response leaked account state: %s", response.Body.String())
		}
	}
}

func TestWechatUnknownReturnsRegistrationTokenWithoutOrphan(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "wechat-test-secret"
	config.App.WeChatLoginMock = true
	response := performJSONRequest(Login, map[string]string{"code": "new-user"})
	var envelope struct {
		Data struct {
			RegistrationRequired bool   `json:"registration_required"`
			RegistrationToken    string `json:"registration_token"`
			Token                string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Data.RegistrationRequired || envelope.Data.RegistrationToken == "" || envelope.Data.Token != "" {
		t.Fatalf("expected registration response: %s", response.Body.String())
	}
	var count int64
	db.DB.Model(&models.User{}).Count(&count)
	if count != 0 {
		t.Fatalf("wechat login created %d orphan users", count)
	}
	if _, err := services.ParseToken(envelope.Data.RegistrationToken); err == nil {
		t.Fatal("registration token was accepted as an application token")
	}
}

func TestWechatRegisterAndReturningLogin(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "wechat-register-secret"
	config.App.WeChatLoginMock = true
	login := performJSONRequest(Login, map[string]string{"code": "wechat-user"})
	var loginEnvelope struct {
		Data struct {
			RegistrationToken string `json:"registration_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &loginEnvelope); err != nil {
		t.Fatal(err)
	}
	code := createTestInvite(t, time.Now().Add(time.Hour), false)
	register := performJSONRequest(WeChatRegister, map[string]string{
		"registration_token": loginEnvelope.Data.RegistrationToken, "invite_code": code,
		"username": "wechat_user", "nickname": "Wechat User", "password": "password1",
	})
	if recorderResponseCode(t, register) != 0 {
		t.Fatalf("wechat registration failed: %s", register.Body.String())
	}
	var user models.User
	if err := db.DB.Where("open_id = ?", "mock_wechat-user").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	returning := performJSONRequest(Login, map[string]string{"code": "wechat-user"})
	var returningEnvelope struct {
		Data struct {
			Token                string `json:"token"`
			RegistrationRequired bool   `json:"registration_required"`
		} `json:"data"`
	}
	if err := json.Unmarshal(returning.Body.Bytes(), &returningEnvelope); err != nil {
		t.Fatal(err)
	}
	if returningEnvelope.Data.Token == "" || returningEnvelope.Data.RegistrationRequired {
		t.Fatalf("returning WeChat login did not authenticate: %s", returning.Body.String())
	}
}

func TestWechatLinksExistingH5AccountWithoutInvitation(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "wechat-link-secret"
	config.App.WeChatLoginMock = true
	code := createTestInvite(t, time.Now().Add(time.Hour), false)
	register := performJSONRequest(H5Register, map[string]string{
		"invite_code": code, "username": "linked_user", "nickname": "Linked User", "password": "password1",
	})
	if recorderResponseCode(t, register) != 0 {
		t.Fatal(register.Body.String())
	}
	login := performJSONRequest(Login, map[string]string{"code": "link-openid"})
	var envelope struct {
		Data struct {
			RegistrationToken string `json:"registration_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	link := performJSONRequest(WeChatLink, map[string]string{
		"registration_token": envelope.Data.RegistrationToken, "username": "LINKED_USER", "password": "password1",
	})
	if recorderResponseCode(t, link) != 0 {
		t.Fatalf("link failed: %s", link.Body.String())
	}
	var user models.User
	if err := db.DB.Where("username_normalized = ?", "linked_user").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.OpenID != "mock_link-openid" {
		t.Fatalf("openid not linked: %+v", user)
	}
}

func TestWechatLinkMigratesLegacyIncompleteOpenIDHolder(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "wechat-link-legacy-secret"
	openID := "mock_legacy-openid"
	if err := db.DB.Create(&models.User{OpenID: openID, Role: models.RoleUser, AccountStatus: models.AccountStatusActive}).Error; err != nil {
		t.Fatal(err)
	}
	code := createTestInvite(t, time.Now().Add(time.Hour), false)
	register := performJSONRequest(H5Register, map[string]string{
		"invite_code": code, "username": "legacy_link", "nickname": "Legacy Link", "password": "password1",
	})
	if recorderResponseCode(t, register) != 0 {
		t.Fatal(register.Body.String())
	}
	registrationToken, err := services.SignRegistrationToken(openID)
	if err != nil {
		t.Fatal(err)
	}
	link := performJSONRequest(WeChatLink, map[string]string{
		"registration_token": registrationToken, "username": "legacy_link", "password": "password1",
	})
	if recorderResponseCode(t, link) != 0 {
		t.Fatalf("link with legacy holder failed: %s", link.Body.String())
	}
	var user models.User
	if err := db.DB.Where("username_normalized = ?", "legacy_link").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.OpenID != openID {
		t.Fatalf("openid not moved to H5 account: %+v", user)
	}
	var legacy models.User
	if err := db.DB.Where("id <> ? AND account_status = ?", user.ID, models.AccountStatusInactive).First(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if legacy.OpenID != "" {
		t.Fatalf("legacy holder still owns openid: %+v", legacy)
	}
}

func TestWechatLinkReturnsChineseConflictForAlreadyLinkedIdentity(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "wechat-link-conflict-secret"
	hash, err := bcrypt.GenerateFromPassword([]byte("password1"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	existingHash := string(hash)
	if err := db.DB.Create(&models.User{OpenID: "already-linked-openid", Username: "other_user", UsernameNormalized: "other_user", Nickname: "Other User", NicknameNormalized: "other user", PasswordHash: &existingHash, Role: models.RoleUser, AccountStatus: models.AccountStatusActive}).Error; err != nil {
		t.Fatal(err)
	}
	code := createTestInvite(t, time.Now().Add(time.Hour), false)
	register := performJSONRequest(H5Register, map[string]string{
		"invite_code": code, "username": "conflict_link", "nickname": "Conflict Link", "password": "password1",
	})
	if recorderResponseCode(t, register) != 0 {
		t.Fatal(register.Body.String())
	}
	registrationToken, err := services.SignRegistrationToken("already-linked-openid")
	if err != nil {
		t.Fatal(err)
	}
	link := performJSONRequest(WeChatLink, map[string]string{
		"registration_token": registrationToken, "username": "conflict_link", "password": "password1",
	})
	if link.Code != http.StatusConflict || !strings.Contains(link.Body.String(), "当前微信身份已经绑定过账号") {
		t.Fatalf("expected localized identity conflict, status=%d body=%s", link.Code, link.Body.String())
	}
}

func createTestInvite(t *testing.T, expiresAt time.Time, disabled bool) string {
	t.Helper()
	var count int64
	if err := db.DB.Model(&models.RegistrationInvite{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	code := "invite-" + t.Name() + "-" + strconv.FormatInt(count, 10)
	invite := models.RegistrationInvite{CodeHash: hashInviteCode(code), CodePrefix: code[:8], ExpiresAt: expiresAt}
	if disabled {
		now := time.Now()
		invite.DisabledAt = &now
	}
	if err := db.DB.Create(&invite).Error; err != nil {
		t.Fatal(err)
	}
	return code
}

func performJSONRequest(handler gin.HandlerFunc, payload interface{}) *httptest.ResponseRecorder {
	return performJSONRequestWithContext(handler, payload, nil)
}

func performJSONRequestWithContext(handler gin.HandlerFunc, payload interface{}, setup func(*gin.Context)) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	if setup != nil {
		setup(context)
	}
	handler(context)
	return recorder
}

func recorderResponseCode(t *testing.T, recorder *httptest.ResponseRecorder) int {
	t.Helper()
	var envelope struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Code
}
