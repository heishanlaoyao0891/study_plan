package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

func TestLoginMockCreatesUserAndBlockedUser(t *testing.T) {
	setupTestDB(t)
	config.App.WeChatLoginMock = true
	blockedAt := time.Now().Add(1 * time.Hour)
	if err := db.DB.Create(&models.User{OpenID: "mock_mock_blocked", Nickname: "B", BannedUntil: &blockedAt}).Error; err != nil {
		t.Fatal(err)
	}

	g := gin.New()
	g.POST("/login", Login)

	rec := httptest.NewRecorder()
	body, _ := json.Marshal(gin.H{"code": "mock_new"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	g.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	body, _ = json.Marshal(gin.H{"code": "mock_blocked"})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	g.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected blocked login, got %d", rec.Code)
	}
}

func TestCheckOverloadRequiresConfirmation(t *testing.T) {
	setupTestDB(t)
	uid := uint(1)
	for i := 0; i < 3; i++ {
		if err := db.DB.Create(&models.Plan{UserID: uid, Title: "P", Status: models.PlanStatusActive, WeeklyTargetHours: 20}).Error; err != nil {
			t.Fatal(err)
		}
	}
	_, err := checkOverload(uid, 20, false)
	if err == nil {
		t.Fatal("expected overload confirmation error")
	}
}

func TestGetPlanRejectsNonOwner(t *testing.T) {
	setupTestDB(t)
	owner := models.User{OpenID: "owner", Nickname: "owner"}
	other := models.User{OpenID: "other", Nickname: "other"}
	if err := db.DB.Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	plan := models.Plan{UserID: owner.ID, Title: "P"}
	if err := db.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}

	g := gin.New()
	g.GET("/plans/:id", func(c *gin.Context) {
		c.Set(middleware.CtxUserIDKey, other.ID)
	}, GetPlan)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plans/"+strconv.FormatUint(uint64(plan.ID), 10), nil)
	g.ServeHTTP(rec, req)
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected non-owner to be rejected, got code %d", resp.Code)
	}
}

func TestRequireAdminRejectsNonAdmin(t *testing.T) {
	g := gin.New()
	g.GET("/admin", func(c *gin.Context) {
		c.Set(middleware.CtxRoleKey, models.RoleUser)
	}, middleware.RequireAdmin(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	g.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected non-admin to be rejected, got %d", rec.Code)
	}
}

func TestRequireNicknameDoesNotExemptAdminRole(t *testing.T) {
	g := gin.New()
	g.GET("/plans", func(c *gin.Context) {
		c.Set(middleware.CtxRoleKey, models.RoleAdmin)
		c.Set(middleware.CtxUserKey, models.User{Role: models.RoleAdmin})
	}, middleware.RequireNickname(), func(c *gin.Context) { c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	g.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/plans", nil))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("nickname gate must apply to WeChat users regardless of role, got %d", recorder.Code)
	}
}

func TestRequireCompleteAccountRejectsMissingUsername(t *testing.T) {
	g := gin.New()
	hash := "hash"
	g.GET("/plans", func(c *gin.Context) {
		c.Set(middleware.CtxRoleKey, models.RoleUser)
		c.Set(middleware.CtxUserKey, models.User{NicknameNormalized: "user", PasswordHash: &hash})
	}, middleware.RequireCompleteAccount(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plans", nil)
	g.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected incomplete account to be rejected, got %d", rec.Code)
	}
}

func TestRequireCompleteAccountAllowsCompleteUser(t *testing.T) {
	hash := "hash"
	g := gin.New()
	g.GET("/plans", func(c *gin.Context) {
		c.Set(middleware.CtxRoleKey, models.RoleUser)
		c.Set(middleware.CtxUserKey, models.User{UsernameNormalized: "user", NicknameNormalized: "user", PasswordHash: &hash})
	}, middleware.RequireCompleteAccount(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/plans", nil)
	g.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected complete account to pass, got %d", rec.Code)
	}
}

func TestDailyTaskUniqueIndex(t *testing.T) {
	setupTestDB(t)
	user := models.User{OpenID: "u1", Nickname: "u1"}
	if err := db.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	plan := models.Plan{UserID: user.ID, Title: "P"}
	if err := db.DB.Create(&plan).Error; err != nil {
		t.Fatal(err)
	}
	task := models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-20", Title: "A"}
	if err := db.DB.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB.Create(&models.DailyTask{UserID: user.ID, PlanID: plan.ID, Date: "2026-07-20", Title: "B"}).Error; err == nil {
		t.Fatal("expected unique index to reject duplicate daily task")
	}
}

func TestAutoMigrateCreatesCriticalTablesAndIndexes(t *testing.T) {
	setupTestDB(t)
	for _, model := range []interface{}{
		&models.User{},
		&models.Plan{},
		&models.PlanScheduleOverride{},
		&models.DailyTask{},
		&models.DailyMotivation{},
		&models.Checkin{},
		&models.StudyGroup{},
		&models.OpsContent{},
		&models.AccountEvent{},
		&models.PasswordResetCode{},
	} {
		if !db.DB.Migrator().HasTable(model) {
			t.Fatalf("expected table for %T", model)
		}
	}
	if !db.DB.Migrator().HasIndex(&models.DailyTask{}, "idx_daily_tasks_user_plan_date") {
		t.Fatal("expected daily task unique index")
	}
	if !db.DB.Migrator().HasIndex(&models.Checkin{}, "idx_checkins_user_plan_date") {
		t.Fatal("expected checkin unique index")
	}
}
