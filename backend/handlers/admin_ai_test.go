package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestAIPlanningMetricsReportsQueueLatencyTokensAndFallback(t *testing.T) {
	setupTestDB(t)
	now := time.Now().UTC()
	jobs := []models.PlanningJob{
		{ID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", UserID: 1, RequestFingerprint: "a", Status: models.PlanningJobStatusQueued, Phase: models.PlanningJobStatusQueued, BaselinePreviewID: "p", BaselinePreviewVersion: 1, RequestJSON: `{}`, ExpiresAt: now.Add(time.Hour)},
		{ID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", UserID: 1, RequestFingerprint: "b", Status: models.PlanningJobStatusReady, Phase: models.PlanningJobStatusReady, BaselinePreviewID: "p", BaselinePreviewVersion: 1, RequestJSON: `{}`, Provider: "siliconflow", ModelName: "model", ProviderLatencyMS: 100, TotalTokens: 10, ExpiresAt: now.Add(time.Hour)},
		{ID: "cccccccccccccccccccccccccccccccc", UserID: 1, RequestFingerprint: "c", Status: models.PlanningJobStatusReady, Phase: models.PlanningJobStatusReady, BaselinePreviewID: "p", BaselinePreviewVersion: 1, RequestJSON: `{}`, ProviderLatencyMS: 200, TotalTokens: 20, ExpiresAt: now.Add(time.Hour)},
		{ID: "dddddddddddddddddddddddddddddddd", UserID: 1, RequestFingerprint: "d", Status: models.PlanningJobStatusFallback, Phase: models.PlanningJobStatusFallback, BaselinePreviewID: "p", BaselinePreviewVersion: 1, RequestJSON: `{}`, ProviderLatencyMS: 900, TotalTokens: 30, FailureReason: "invalid_blueprint", ExpiresAt: now.Add(time.Hour)},
	}
	if err := db.DB.Create(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	currentJobs := []models.AIPlanGenerationJob{
		{UserID: 1, RequestJSON: `{}`, RequestHash: "current-queued", Status: models.AIPlanJobStatusRunning, CreatedAt: now},
		{UserID: 1, RequestJSON: `{}`, RequestHash: "current-ai", Status: models.AIPlanJobStatusSucceeded, GenerationSource: "ai_decomposed", EnrichmentStatus: "success", Provider: "siliconflow", ModelName: "current-model", CreatedAt: now},
		{UserID: 1, RequestJSON: `{}`, RequestHash: "current-fallback", Status: models.AIPlanJobStatusSucceeded, GenerationSource: "local", EnrichmentStatus: "provider_error", EnrichmentReason: "provider_request_failed", Provider: "siliconflow", ModelName: "current-model", CreatedAt: now},
	}
	if err := db.DB.Create(&currentJobs).Error; err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/metrics", GetAIPlanningMetrics)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	var response struct {
		Data struct {
			QueueDepth   int64            `json:"queue_depth"`
			SuccessRate  float64          `json:"success_rate"`
			FallbackRate float64          `json:"fallback_rate"`
			P50          int64            `json:"p50_latency_ms"`
			P95          int64            `json:"p95_latency_ms"`
			TotalTokens  int64            `json:"total_tokens"`
			Reasons      map[string]int64 `json:"fallback_reasons"`
			Providers    map[string]int64 `json:"provider_models"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.QueueDepth != 2 || response.Data.SuccessRate != 3.0/5.0 || response.Data.FallbackRate != 2.0/5.0 || response.Data.P50 != 200 || response.Data.P95 != 900 || response.Data.TotalTokens != 60 || response.Data.Reasons["invalid_blueprint"] != 1 || response.Data.Reasons["provider_request_failed"] != 1 || response.Data.Providers["siliconflow/current-model"] != 2 {
		t.Fatalf("unexpected planning metrics: %+v", response.Data)
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
