package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

type banEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		AccountBanned bool   `json:"account_banned"`
		Reason        string `json:"reason"`
		BannedUntil   string `json:"banned_until"`
		Permanent     bool   `json:"permanent"`
		ServerNow     string `json:"server_now"`
	} `json:"data"`
}

func TestH5LoginReturnsTimedBanEnvelope(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "ban-h5-secret"
	user := createPasswordUser(t, "timed_ban_user", "password1")
	until := time.Now().Add(90 * time.Minute).UTC().Truncate(time.Second)
	if err := db.DB.Model(&user).Updates(map[string]interface{}{"banned_until": until, "banned_reason": "请稍后再试"}).Error; err != nil {
		t.Fatal(err)
	}

	response := performJSONRequest(H5Login, map[string]string{"username": user.Username, "password": "password1"})
	envelope := decodeBanEnvelope(t, response)
	if response.Code != http.StatusForbidden || envelope.Code != http.StatusForbidden || !envelope.Data.AccountBanned || envelope.Data.Permanent {
		t.Fatalf("unexpected timed ban response: status=%d body=%s", response.Code, response.Body.String())
	}
	if envelope.Data.Reason != "请稍后再试" || envelope.Data.BannedUntil != until.Format(time.RFC3339) {
		t.Fatalf("timed metadata mismatch: %+v", envelope.Data)
	}
	assertRFC3339(t, envelope.Data.ServerNow)
	assertSafeBanFields(t, response)
}

func TestH5LoginReturnsPermanentBanMetadata(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "ban-h5-permanent-secret"
	user := createPasswordUser(t, "permanent_h5_user", "password1")
	until := models.PermanentBanUntil()
	if err := db.DB.Model(&user).Update("banned_until", until).Error; err != nil {
		t.Fatal(err)
	}

	response := performJSONRequest(H5Login, map[string]string{"username": user.Username, "password": "password1"})
	envelope := decodeBanEnvelope(t, response)
	if response.Code != http.StatusForbidden || !envelope.Data.Permanent || envelope.Data.BannedUntil != until.Format(time.RFC3339) {
		t.Fatalf("unexpected H5 permanent ban response: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestWechatMockLoginReturnsPermanentBanEnvelope(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "ban-wechat-secret"
	config.App.WeChatLoginMock = true
	user := createPasswordUser(t, "permanent_ban_user", "password1")
	user.OpenID = "mock_banned-wechat"
	until := models.PermanentBanUntil()
	if err := db.DB.Model(&user).Updates(map[string]interface{}{"open_id": user.OpenID, "banned_until": until, "banned_reason": "规则复核中"}).Error; err != nil {
		t.Fatal(err)
	}

	response := performJSONRequest(Login, map[string]string{"code": "banned-wechat"})
	envelope := decodeBanEnvelope(t, response)
	if response.Code != http.StatusForbidden || !envelope.Data.Permanent || envelope.Data.BannedUntil != until.Format(time.RFC3339) {
		t.Fatalf("unexpected permanent ban response: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestExpiredBanClearsAndLoginProceeds(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "expired-ban-secret"
	user := createPasswordUser(t, "expired_ban_user", "password1")
	past := time.Now().Add(-time.Minute)
	if err := db.DB.Model(&user).Updates(map[string]interface{}{"banned_until": past, "banned_reason": "expired"}).Error; err != nil {
		t.Fatal(err)
	}

	response := performJSONRequest(H5Login, map[string]string{"username": user.Username, "password": "password1"})
	if response.Code != http.StatusOK || recorderResponseCode(t, response) != 0 {
		t.Fatalf("expired ban blocked login: %s", response.Body.String())
	}
	var reloaded models.User
	if err := db.DB.First(&reloaded, user.ID).Error; err != nil || reloaded.BannedUntil != nil || reloaded.BannedReason != "" {
		t.Fatalf("expired ban was not cleared: %+v err=%v", reloaded, err)
	}
}

func TestAuthMiddlewareReturnsBanAndClearsExpiredBan(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "ban-middleware-secret"
	config.App.JWTExpireHours = 24
	user := createPasswordUser(t, "middleware_ban_user", "password1")
	token, err := services.SignToken(user.ID, user.OpenID, user.Role, user.SecurityVersion)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/me", middleware.Auth(), func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	request := func() *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(recorder, req)
		return recorder
	}

	until := time.Now().Add(time.Hour)
	db.DB.Model(&user).Updates(map[string]interface{}{"banned_until": until, "banned_reason": "middleware reason"})
	banned := request()
	if banned.Code != http.StatusForbidden || !decodeBanEnvelope(t, banned).Data.AccountBanned {
		t.Fatalf("middleware did not return ban envelope: %s", banned.Body.String())
	}
	past := time.Now().Add(-time.Second)
	db.DB.Model(&user).Updates(map[string]interface{}{"banned_until": past, "banned_reason": "expired"})
	if allowed := request(); allowed.Code != http.StatusOK {
		t.Fatalf("middleware did not allow expired ban: %d %s", allowed.Code, allowed.Body.String())
	}
}

func TestWechatLinkDoesNotMutateBannedAccount(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "ban-link-secret"
	user := createPasswordUser(t, "banned_link_user", "password1")
	until := time.Now().Add(time.Hour)
	db.DB.Model(&user).Update("banned_until", until)
	registrationToken, err := services.SignRegistrationToken("new-open-id")
	if err != nil {
		t.Fatal(err)
	}
	response := performJSONRequest(WeChatLink, map[string]string{
		"registration_token": registrationToken, "username": user.Username, "password": "password1",
	})
	if response.Code != http.StatusForbidden || !decodeBanEnvelope(t, response).Data.AccountBanned {
		t.Fatalf("link did not return ban: %s", response.Body.String())
	}
	if err := db.DB.First(&user, user.ID).Error; err != nil || user.OpenID != "" {
		t.Fatalf("banned link mutated identity: %+v err=%v", user, err)
	}
}

func decodeBanEnvelope(t *testing.T, response *httptest.ResponseRecorder) banEnvelope {
	t.Helper()
	var envelope banEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope
}

func assertRFC3339(t *testing.T, value string) {
	t.Helper()
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		t.Fatalf("%q is not RFC3339: %v", value, err)
	}
}

func assertSafeBanFields(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	var envelope struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"account_banned": true, "reason": true, "banned_until": true, "permanent": true, "server_now": true}
	if len(envelope.Data) != len(allowed) {
		t.Fatalf("ban response exposed unexpected fields: %v", envelope.Data)
	}
	for key := range envelope.Data {
		if !allowed[key] {
			t.Fatalf("ban response exposed unexpected field %q", key)
		}
	}
}
