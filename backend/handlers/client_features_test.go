package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"study_plan_backend/db"
	"study_plan_backend/middleware"
	"study_plan_backend/models"
)

func TestClientFeaturesFailClosedAndExposeNoProviderDetails(t *testing.T) {
	setupTestDB(t)
	router := gin.New()
	router.GET("/features", GetClientFeatures)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/features", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected feature status: %d", recorder.Code)
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data["mini_program_ai_enabled"] != false || len(envelope.Data) != 1 {
		t.Fatalf("unexpected feature payload: %s", recorder.Body.String())
	}
	for _, secret := range []string{"provider", "base_url", "api_key", "model_name", "enabled"} {
		if _, exists := envelope.Data[secret]; exists {
			t.Fatalf("provider configuration leaked through feature endpoint: %s", recorder.Body.String())
		}
	}

	original := db.DB
	db.DB = nil
	t.Cleanup(func() { db.DB = original })
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/features", nil))
	if !strings.Contains(recorder.Body.String(), `"mini_program_ai_enabled":false`) {
		t.Fatalf("feature load failure did not fail closed: %s", recorder.Body.String())
	}
}

func TestMiniProgramAIJobGatePreservesH5AndLegacyClients(t *testing.T) {
	setupTestDB(t)
	var cfg models.AIConfig
	if err := db.DB.Order("id ASC").First(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	cfg.Provider = "mock"
	cfg.MiniProgramAIEnabled = false
	if err := db.DB.Save(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.POST("/jobs", func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, uint(7)) }, SubmitAIPlanJob)
	body := `{"goal":"Study Go","days":1,"hours_per_day":1,"start_date":"2026-08-01","available_time_slot":"20:00-21:00","idempotency_key":"platform_key_1234"}`

	miniRequest := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	miniRequest.Header.Set("Content-Type", "application/json")
	miniRequest.Header.Set("X-Client-Platform", miniProgramPlatform)
	miniResponse := httptest.NewRecorder()
	router.ServeHTTP(miniResponse, miniRequest)
	if miniResponse.Code != http.StatusForbidden || !strings.Contains(miniResponse.Body.String(), "mini-program AI plan generation is disabled") {
		t.Fatalf("disabled mini-program request was not rejected: status=%d body=%s", miniResponse.Code, miniResponse.Body.String())
	}

	for _, platform := range []string{"h5", ""} {
		request := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		if platform != "" {
			request.Header.Set("X-Client-Platform", platform)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusAccepted {
			t.Fatalf("%q client lost legacy compatibility: status=%d body=%s", platform, response.Code, response.Body.String())
		}
	}
}

func TestMiniProgramAISwitchAuditContainsOldAndNewValues(t *testing.T) {
	setupTestDB(t)
	adminID := uint(99)
	router := gin.New()
	router.PUT("/config", func(c *gin.Context) { c.Set(middleware.CtxUserIDKey, adminID) }, UpdateAIConfig)
	enabled := true
	body, _ := json.Marshal(aiConfigReq{Provider: "mock", RequestTimeoutSeconds: 30, InteractiveTargetSeconds: 2, BackgroundJobTimeoutSeconds: 300, DailyGenerationLimit: 5, Enabled: &enabled, MiniProgramAIEnabled: &enabled})
	request := httptest.NewRequest(http.MethodPut, "/config", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("switch update failed: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var count int64
	db.DB.Model(&models.AdminAuditLog{}).Count(&count)
	if count == 0 {
		t.Fatalf("switch update did not write audit: body=%s", recorder.Body.String())
	}
	var log models.AdminAuditLog
	if err := db.DB.Order("id DESC").First(&log).Error; err != nil {
		t.Fatal(err)
	}
	if log.ActionType != "update_ai_config" || log.AdminUserID != adminID || log.Reason != "mini_program_ai_enabled: false -> true" {
		t.Fatalf("switch change was not audited: %+v", log)
	}
}
