package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"study_plan_backend/api"
	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
	"study_plan_backend/services"
)

func TestOnboardingStatusIsAccountScopedAndTerminal(t *testing.T) {
	setupGroupTestDB(t)
	user := createPasswordUser(t, "onboarding_user", "password1")
	response := performJSONRequestWithContext(UpdateOnboarding, map[string]string{"status": models.OnboardingStatusSkipped}, func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, user.ID)
	})
	if recorderResponseCode(t, response) != 0 {
		t.Fatal(response.Body.String())
	}
	if err := db.DB.First(&user, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if user.OnboardingStatus != models.OnboardingStatusSkipped || user.OnboardingVersion != models.CurrentOnboardingVersion || user.OnboardingCompletedAt == nil {
		t.Fatalf("onboarding terminal state was not persisted: %+v", user)
	}
	response = performJSONRequestWithContext(UpdateOnboarding, map[string]string{"status": models.OnboardingStatusCompleted}, func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, user.ID)
	})
	if recorderResponseCode(t, response) != 0 {
		t.Fatal(response.Body.String())
	}
	if err := db.DB.First(&user, user.ID).Error; err != nil || user.OnboardingStatus != models.OnboardingStatusSkipped {
		t.Fatalf("terminal onboarding state changed: %+v err=%v", user, err)
	}
}

func TestPasswordChangeReturnsFreshVersionedToken(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "password-change-secret"
	config.App.JWTExpireHours = 24
	user := createPasswordUser(t, "change_user", "password1")
	oldToken, err := services.SignToken(user.ID, user.OpenID, user.Role, user.SecurityVersion)
	if err != nil {
		t.Fatal(err)
	}
	response := performJSONRequestWithContext(ChangePassword, map[string]string{"current_password": "password1", "new_password": "password2"}, func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, user.ID)
	})
	if recorderResponseCode(t, response) != 0 {
		t.Fatal(response.Body.String())
	}
	var envelope struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	oldClaims, _ := services.ParseToken(oldToken)
	newClaims, err := services.ParseToken(envelope.Data.Token)
	if err != nil || newClaims.SecurityVersion != oldClaims.SecurityVersion+1 {
		t.Fatalf("fresh token did not carry incremented version: old=%+v new=%+v err=%v", oldClaims, newClaims, err)
	}
	if err := db.DB.First(&user, user.ID).Error; err != nil || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte("password2")) != nil {
		t.Fatal("new password was not persisted")
	}
}

func TestAuthRejectsStaleAndInactiveTokens(t *testing.T) {
	setupGroupTestDB(t)
	config.App.JWTSecret = "middleware-secret"
	config.App.JWTExpireHours = 24
	user := createPasswordUser(t, "token_user", "password1")
	token, _ := services.SignToken(user.ID, user.OpenID, user.Role, user.SecurityVersion)
	router := gin.New()
	router.GET("/me", middleware.Auth(), func(c *gin.Context) { c.Status(http.StatusOK) })
	assertAuthStatus := func(want int) {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/me", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(recorder, request)
		if recorder.Code != want {
			t.Fatalf("expected auth status %d, got %d: %s", want, recorder.Code, recorder.Body.String())
		}
	}
	assertAuthStatus(http.StatusOK)
	db.DB.Model(&user).Update("security_version", user.SecurityVersion+1)
	assertAuthStatus(http.StatusUnauthorized)
	user.SecurityVersion++
	token, _ = services.SignToken(user.ID, user.OpenID, user.Role, user.SecurityVersion)
	db.DB.Model(&user).Update("account_status", models.AccountStatusInactive)
	assertAuthStatus(http.StatusUnauthorized)
}

func TestAllowActiveUserRejectsInactiveAccount(t *testing.T) {
	setupGroupTestDB(t)
	user := createPasswordUser(t, "inactive_user", "password1")
	if err := db.DB.Model(&user).Update("account_status", models.AccountStatusInactive).Error; err != nil {
		t.Fatal(err)
	}
	user.AccountStatus = models.AccountStatusInactive
	response := performJSONRequestWithContext(func(c *gin.Context) {
		if allowActiveUser(c, &user) {
			api.OK(c, gin.H{"allowed": true})
		}
	}, nil, nil)
	if recorderResponseCode(t, response) != http.StatusForbidden {
		t.Fatalf("inactive account was allowed to log in: %s", response.Body.String())
	}
	if err := db.DB.First(&user, user.ID).Error; err != nil || user.AccountStatus != models.AccountStatusInactive {
		t.Fatalf("inactive account was reactivated: %+v err=%v", user, err)
	}
}

func TestPasswordResetCodeIsOneTimeAndInvalidatesTokens(t *testing.T) {
	setupGroupTestDB(t)
	user := createPasswordUser(t, "reset_user", "password1")
	create := performJSONRequestWithContext(CreatePasswordResetCode, nil, func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, uint(99))
		c.Params = gin.Params{{Key: "id", Value: strconv.FormatUint(uint64(user.ID), 10)}}
	})
	if recorderResponseCode(t, create) != 0 {
		t.Fatal(create.Body.String())
	}
	var envelope struct {
		Data struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &envelope); err != nil || envelope.Data.Code == "" {
		t.Fatalf("reset code was not returned once: %v %s", err, create.Body.String())
	}
	redeem := func() *httptest.ResponseRecorder {
		return performJSONRequest(RedeemPasswordReset, map[string]string{"username": user.Username, "code": envelope.Data.Code, "new_password": "password2"})
	}
	if response := redeem(); recorderResponseCode(t, response) != 0 {
		t.Fatal(response.Body.String())
	}
	if response := redeem(); recorderResponseCode(t, response) != http.StatusBadRequest {
		t.Fatalf("reset code was reusable: %s", response.Body.String())
	}
	if err := db.DB.First(&user, user.ID).Error; err != nil || user.SecurityVersion != 2 || bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte("password2")) != nil {
		t.Fatalf("reset did not update credentials and version: %+v err=%v", user, err)
	}
}

func createPasswordUser(t *testing.T, username, password string) models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	hashString := string(hash)
	user := models.User{
		Username: username, UsernameNormalized: username, Nickname: username, NicknameNormalized: username,
		PasswordHash: &hashString, AccountStatus: models.AccountStatusActive, Role: models.RoleUser,
	}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user
}
