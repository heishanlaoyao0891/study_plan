package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"study_plan_backend/config"
	"study_plan_backend/db"
	"study_plan_backend/models"
)

func TestAIProviderRequiresNewKeyForDifferentOrigin(t *testing.T) {
	setupTestDB(t)
	var cfg models.AIConfig
	if err := db.DB.Order("id ASC").First(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	cfg.Provider = "openai_compatible"
	cfg.ModelName = "stored-model"
	cfg.BaseURL = "https://provider.example"
	cfg.APIKeyCiphertext = "stored-secret-key"
	cfg.Enabled = true
	if err := db.DB.Save(&cfg).Error; err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.POST("/test", TestAIProvider)
	body, _ := json.Marshal(gin.H{"base_url": "http://127.0.0.1:12345", "model_name": "test-model", "provider": "openai_compatible"})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if strings.Contains(recorder.Body.String(), "stored-secret-key") || !strings.Contains(recorder.Body.String(), "new API key is required") {
		t.Fatalf("expected different-origin test to reject stored-key reuse without disclosure: %s", recorder.Body.String())
	}
}

func TestUpdateAIConfigNormalizesLegacyProviderAlias(t *testing.T) {
	setupTestDB(t)
	config.App.AIKeySecret = "test-only-encryption-secret"
	router := gin.New()
	router.PUT("/config", UpdateAIConfig)
	enabled := true
	body, _ := json.Marshal(aiConfigReq{Provider: "openai-compatible", ModelName: "model", BaseURL: "https://93.184.216.34", RequestTimeoutSeconds: 30, DailyGenerationLimit: 5, Enabled: &enabled, APIKey: "new-key"})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/config", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	var cfg models.AIConfig
	if err := db.DB.Order("id ASC").First(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "openai_compatible" {
		t.Fatalf("expected normalized provider, got %q; response: %s", cfg.Provider, recorder.Body.String())
	}
}
